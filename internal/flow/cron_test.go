package flow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseCronExpression(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"wildcard", "* * * * *", false},
		{"every 5 mins", "*/5 * * * *", false},
		{"daily macro", "@daily", false},
		{"hourly macro", "@hourly", false},
		{"weekly macro", "@weekly", false},
		{"custom schedule", "30 4 1,15 * 1-5", false},
		{"invalid fields count", "* * * *", true},
		{"minute out of range", "60 * * * *", true},
		{"hour out of range", "* 24 * * *", true},
		{"month out of range", "* * * 13 *", true},
		{"invalid step", "*/0 * * * *", true},
		{"invalid range", "5-2 * * * *", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sched, err := ParseCronExpression(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseCronExpression(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
			if !tt.wantErr && sched == nil {
				t.Fatalf("expected non-nil schedule for %q", tt.expr)
			}
		})
	}
}

func TestNextTrigger(t *testing.T) {
	// Schedule: at minute 15 of every hour
	sched, err := ParseCronExpression("15 * * * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	baseTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	next := sched.NextTrigger(baseTime)

	expected := time.Date(2026, 1, 1, 10, 15, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("NextTrigger = %v, want %v", next, expected)
	}

	// When baseTime is 10:15, next should be 11:15
	nextAfter := sched.NextTrigger(next)
	expectedAfter := time.Date(2026, 1, 1, 11, 15, 0, 0, time.UTC)
	if !nextAfter.Equal(expectedAfter) {
		t.Errorf("NextTrigger = %v, want %v", nextAfter, expectedAfter)
	}
}

func TestScheduleStoreCRUD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shellix-sched-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "schedules.json")
	store, err := NewScheduleStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// 1. Initial list empty
	jobs, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}

	// 2. Add job
	job, err := store.Add("deploy-prod", "flows/deploy.yaml", "0 0 * * *")
	if err != nil {
		t.Fatalf("failed to add job: %v", err)
	}
	if job.FlowName != "deploy-prod" || !job.Enabled {
		t.Errorf("unexpected job: %+v", job)
	}

	// 3. Duplicate flow name should fail
	_, err = store.Add("deploy-prod", "flows/deploy.yaml", "@hourly")
	if err == nil {
		t.Fatalf("expected error adding duplicate flow name")
	}

	// 4. Toggle disabled
	toggled, err := store.Toggle(job.ID, false)
	if err != nil {
		t.Fatalf("failed to toggle: %v", err)
	}
	if toggled.Enabled {
		t.Errorf("expected job to be disabled")
	}

	// 5. Update run result
	err = store.UpdateResult(job.ID, true, "")
	if err != nil {
		t.Fatalf("failed to update result: %v", err)
	}

	jobs, _ = store.Load()
	if len(jobs) != 1 || jobs[0].RunCount != 1 || jobs[0].LastStatus != "SUCCESS" {
		t.Errorf("unexpected job state after update: %+v", jobs[0])
	}

	// 6. Remove job
	removed, err := store.Remove("deploy-prod")
	if err != nil || !removed {
		t.Fatalf("failed to remove job: %v", err)
	}

	jobs, _ = store.Load()
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs after deletion, got %d", len(jobs))
	}
}

func TestScheduleDaemon(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shellix-daemon-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "schedules.json")
	store, _ := NewScheduleStore(dbPath)

	job, err := store.Add("fast-task", "flow.yaml", "* * * * *")
	if err != nil {
		t.Fatal(err)
	}

	// Force NextRun to be now or past
	jobs, _ := store.Load()
	jobs[0].NextRun = time.Now().Add(-1 * time.Second)
	_ = store.Save(jobs)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	executed := false
	err = store.RunDaemon(ctx, 20*time.Millisecond, nil, func(j ScheduledJob) error {
		if j.ID == job.ID {
			executed = true
		}
		return nil
	})

	if err != context.DeadlineExceeded {
		t.Logf("daemon exited with: %v", err)
	}

	// Wait briefly for goroutine completion
	time.Sleep(50 * time.Millisecond)

	if !executed {
		t.Errorf("expected due job to be triggered by daemon")
	}
}

func TestProcessLiveness(t *testing.T) {
	currentPID := os.Getpid()
	if !IsProcessAlive(currentPID) {
		t.Errorf("expected current PID %d to be alive", currentPID)
	}

	// Non-existent PID
	fakePID := 9999999
	if IsProcessAlive(fakePID) {
		t.Errorf("expected fake PID %d to be dead", fakePID)
	}
}

func TestSchedulerLogger(t *testing.T) {
	logger, err := NewSchedulerLogger(false)
	if err != nil {
		t.Fatalf("failed creating logger: %v", err)
	}
	defer logger.Close()

	logger.Info("test", "Testing scheduler logger message: %s", "hello")
	logger.Success("test", "Success event")
	logger.Warn("test", "Warning event")

	logPath := GetSchedulerLogPath()
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed reading log: %v", err)
	}

	if !strings.Contains(string(content), "Testing scheduler logger message: hello") {
		t.Errorf("log file does not contain expected message: %s", string(content))
	}
}
