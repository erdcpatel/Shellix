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

// Save persists scheduled jobs to disk with 0600 permissions
func (s *ScheduleStore) Save(jobs []ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize schedules: %w", err)
	}

	return os.WriteFile(s.filePath, data, 0600)
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

// UpdateResult records execution outcome and calculates next run time
func (s *ScheduleStore) UpdateResult(id string, success bool, errMsg string) error {
	jobs, err := s.Load()
	if err != nil {
		return err
	}

	for i := range jobs {
		if jobs[i].ID == id {
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

// RunDaemon starts the scheduler loop, triggering due jobs
func (s *ScheduleStore) RunDaemon(ctx context.Context, pollInterval time.Duration, runFn func(job ScheduledJob) error) error {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			now := time.Now()
			jobs, err := s.Load()
			if err != nil {
				continue
			}

			for _, job := range jobs {
				if !job.Enabled || job.NextRun.IsZero() {
					continue
				}

				if now.After(job.NextRun) || now.Equal(job.NextRun) {
					runErr := runFn(job)
					errMsg := ""
					if runErr != nil {
						errMsg = runErr.Error()
					}
					_ = s.UpdateResult(job.ID, runErr == nil, errMsg)
				}
			}
		}
	}
}
