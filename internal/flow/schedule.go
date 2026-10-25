package flow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ScheduledJob describes a recurring automation flow job
type ScheduledJob struct {
	ID         string    `json:"id"`
	FlowName   string    `json:"flow_name"`
	FlowPath   string    `json:"flow_path,omitempty"`
	CronExpr   string    `json:"cron_expr"`
	Enabled    bool      `json:"enabled"`
	InFlight   bool      `json:"in_flight"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastRun    time.Time `json:"last_run,omitempty"`
	NextRun    time.Time `json:"next_run"`
	RunCount   int       `json:"run_count"`
	LastStatus string    `json:"last_status,omitempty"` // SUCCESS, FAILED
	LastError  string    `json:"last_error,omitempty"`
}

// ScheduleStore persists scheduled jobs to disk in ~/.shellix/schedules.json
type ScheduleStore struct {
	filePath string
	mu       sync.Mutex
}

// NewScheduleStore initializes a ScheduleStore, creating ~/.shellix directory if missing
func NewScheduleStore(customPath string) (*ScheduleStore, error) {
	path := customPath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("unable to determine home directory: %w", err)
		}
		path = filepath.Join(home, ".shellix", "schedules.json")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create schedules directory: %w", err)
	}

	return &ScheduleStore{filePath: path}, nil
}

// Load reads all scheduled jobs from disk
func (s *ScheduleStore) Load() ([]ScheduledJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return []ScheduledJob{}, nil
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schedules file: %w", err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return []ScheduledJob{}, nil
	}

	var jobs []ScheduledJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, fmt.Errorf("failed to parse schedules: %w", err)
	}

	return jobs, nil
}

// Save persists scheduled jobs to disk atomically with 0600 permissions
func (s *ScheduleStore) Save(jobs []ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize schedules: %w", err)
	}

	// Atomic write via temp file
	tmpPath := fmt.Sprintf("%s.tmp.%d", s.filePath, time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed writing temporary schedules file: %w", err)
	}

	return os.Rename(tmpPath, s.filePath)
}

// Add creates and persists a new scheduled flow
func (s *ScheduleStore) Add(flowName, flowPath, cronExpr string) (*ScheduledJob, error) {
	sched, err := ParseCronExpression(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	jobs, err := s.Load()
	if err != nil {
		return nil, err
	}

	// Check if already exists with same flow name or ID
	for _, j := range jobs {
		if strings.EqualFold(j.FlowName, flowName) {
			return nil, fmt.Errorf("schedule for flow %q already exists (id: %s)", flowName, j.ID)
		}
	}

	randBytes := make([]byte, 4)
	_, _ = rand.Read(randBytes)
	id := fmt.Sprintf("job_%s", hex.EncodeToString(randBytes))

	now := time.Now()
	next := sched.NextTrigger(now)

	job := ScheduledJob{
		ID:        id,
		FlowName:  flowName,
		FlowPath:  flowPath,
		CronExpr:  cronExpr,
		Enabled:   true,
		InFlight:  false,
		CreatedAt: now,
		NextRun:   next,
		RunCount:  0,
	}

	jobs = append(jobs, job)
	if err := s.Save(jobs); err != nil {
		return nil, err
	}

	return &job, nil
}

// Remove deletes a job by its ID or FlowName
func (s *ScheduleStore) Remove(idOrName string) (bool, error) {
	jobs, err := s.Load()
	if err != nil {
		return false, err
	}

	filtered := make([]ScheduledJob, 0, len(jobs))
	removed := false

	for _, j := range jobs {
		if strings.EqualFold(j.ID, idOrName) || strings.EqualFold(j.FlowName, idOrName) {
			removed = true
			continue
		}
		filtered = append(filtered, j)
	}

	if removed {
		err = s.Save(filtered)
	}

	return removed, err
}

// Toggle enables or disables a scheduled job
func (s *ScheduleStore) Toggle(idOrName string, enable bool) (*ScheduledJob, error) {
	jobs, err := s.Load()
	if err != nil {
		return nil, err
	}

	var target *ScheduledJob
	for i := range jobs {
		if strings.EqualFold(jobs[i].ID, idOrName) || strings.EqualFold(jobs[i].FlowName, idOrName) {
			jobs[i].Enabled = enable
			if enable {
				if sched, err := ParseCronExpression(jobs[i].CronExpr); err == nil {
					jobs[i].NextRun = sched.NextTrigger(time.Now())
				}
			}
			target = &jobs[i]
			break
		}
	}

	if target == nil {
		return nil, fmt.Errorf("job %q not found", idOrName)
	}

	if err := s.Save(jobs); err != nil {
		return nil, err
	}

	return target, nil
}

// MarkInFlight updates the execution status to guard against overlapping runs
func (s *ScheduleStore) MarkInFlight(id string, inFlight bool) error {
	jobs, err := s.Load()
	if err != nil {
		return err
	}

	for i := range jobs {
		if jobs[i].ID == id {
			jobs[i].InFlight = inFlight
			if inFlight {
				jobs[i].StartedAt = time.Now()
			}
			break
		}
	}

	return s.Save(jobs)
}

// UpdateResult records execution outcome, clears in-flight flag, and calculates next run time
func (s *ScheduleStore) UpdateResult(id string, success bool, errMsg string) error {
	jobs, err := s.Load()
	if err != nil {
		return err
	}

	for i := range jobs {
		if jobs[i].ID == id {
			jobs[i].InFlight = false
			jobs[i].LastRun = time.Now()
			jobs[i].RunCount++
			if success {
				jobs[i].LastStatus = "SUCCESS"
				jobs[i].LastError = ""
			} else {
				jobs[i].LastStatus = "FAILED"
				jobs[i].LastError = errMsg
			}

			if sched, err := ParseCronExpression(jobs[i].CronExpr); err == nil {
				jobs[i].NextRun = sched.NextTrigger(time.Now())
			}
			break
		}
	}

	return s.Save(jobs)
}

// RunDaemon starts the scheduler loop with overlapping guards and logger
func (s *ScheduleStore) RunDaemon(ctx context.Context, pollInterval time.Duration, logger *SchedulerLogger, runFn func(job ScheduledJob) error) error {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	heartbeatTicker := time.NewTicker(5 * time.Second)
	defer heartbeatTicker.Stop()

	if logger != nil {
		logger.Info("scheduler", "Daemon event loop active (poll interval: %s)", pollInterval)
	}

	for {
		select {
		case <-ctx.Done():
			if logger != nil {
				logger.Info("scheduler", "Scheduler daemon shutting down (context cancelled)")
			}
			return ctx.Err()

		case <-heartbeatTicker.C:
			UpdateHeartbeat()

		case <-ticker.C:
			now := time.Now()
			jobs, err := s.Load()
			if err != nil {
				if logger != nil {
					logger.Error("scheduler", "Failed loading schedules: %v", err)
				}
				continue
			}

			for _, job := range jobs {
				if !job.Enabled || job.NextRun.IsZero() {
					continue
				}

				if now.After(job.NextRun) || now.Equal(job.NextRun) {
					// 1. Guard against overlapping executions
					if job.InFlight {
						if logger != nil {
							logger.Warn(job.ID, "Flow '%s' is still in-flight (started %s ago). Skipping overlapping execution.",
								job.FlowName, time.Since(job.StartedAt).Round(time.Second))
						}
						continue
					}

					// 2. Mark in-flight
					_ = s.MarkInFlight(job.ID, true)
					if logger != nil {
						logger.Info(job.ID, "Triggering flow '%s' (cron: %s)", job.FlowName, job.CronExpr)
					}

					go func(targetJob ScheduledJob) {
						start := time.Now()
						runErr := runFn(targetJob)
						dur := time.Since(start)

						errMsg := ""
						if runErr != nil {
							errMsg = runErr.Error()
							if logger != nil {
								logger.Error(targetJob.ID, "Flow '%s' failed in %s: %v", targetJob.FlowName, dur.Round(time.Millisecond), runErr)
							}
						} else {
							if logger != nil {
								logger.Success(targetJob.ID, "Flow '%s' finished successfully in %s", targetJob.FlowName, dur.Round(time.Millisecond))
							}
						}
						_ = s.UpdateResult(targetJob.ID, runErr == nil, errMsg)
					}(job)
				}
			}
		}
	}
}
