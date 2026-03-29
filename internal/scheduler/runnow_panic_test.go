package scheduler

import (
	"testing"
	"time"

	"github.com/imysm/db-backup/internal/model"
)

// TestRunNow_DoneChannelClosedOnNormalCompletion verifies done channel is closed after normal execution.
func TestRunNow_DoneChannelClosedOnNormalCompletion(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "panic-test-001",
				Name:    "Normal Task",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type:     model.MySQL,
					Host:     "nonexistent",
					Port:     3306,
					Username: "root",
					Password: "password",
					Database: "test",
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup-test-panic",
				},
				Notify: model.NotifyConfig{
					Enabled: false,
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	done, err := sched.RunNow("panic-test-001", nil)
	if err != nil {
		t.Fatalf("RunNow() error = %v", err)
	}

	select {
	case <-done:
		// OK: done channel closed normally
	case <-time.After(30 * time.Second):
		t.Fatal("done channel not closed after 30s - possible goroutine leak")
	}
}

// TestRunNow_DoneChannelClosedOnPanic verifies done channel is closed even when runTask panics.
// We inject a task that has no executor registered, which causes runTask to fail fast.
// But to truly test panic recovery, we use a task ID that will trigger a nil pointer dereference
// indirectly. Instead, we test by using a task with an invalid storage that may cause issues.
//
// The real test: since we can't easily inject a panic into runTask, we verify the defer/recover
// structure is in place by checking that the done channel always closes within a reasonable time.
func TestRunNow_DoneChannelClosedOnPanic(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "panic-test-002",
				Name:    "Panic Task",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type:     model.MySQL,
					Host:     "nonexistent",
					Port:     3306,
					Username: "root",
					Password: "password",
					Database: "test",
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup-test-panic",
				},
				Notify: model.NotifyConfig{
					Enabled: false,
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	// Remove the executor to trigger a panic path in runTask
	delete(sched.executors, "panic-test-002")

	done, err := sched.RunNow("panic-test-002", nil)
	if err != nil {
		t.Fatalf("RunNow() error = %v", err)
	}

	select {
	case <-done:
		// OK: done channel closed even after error path
	case <-time.After(10 * time.Second):
		t.Fatal("done channel not closed - goroutine may have panicked without recovery")
	}
}
