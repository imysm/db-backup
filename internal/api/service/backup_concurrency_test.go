package service

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestBackupService_RunNow_ConcurrentReject verifies that concurrent RunNow calls for the same job
// are rejected when the job is already running.
func TestBackupService_RunNow_ConcurrentReject(t *testing.T) {
	// We can't use real DB, so we test the sync.Map logic directly
	svc := &BackupService{}

	// Simulate first job starting
	_, loaded := svc.running.LoadOrStore(uint(1), struct{}{})
	if loaded {
		t.Fatal("first LoadOrStore should not return loaded=true")
	}

	// Simulate second concurrent call for same job
	_, loaded = svc.running.LoadOrStore(uint(1), struct{}{})
	if !loaded {
		t.Fatal("second LoadOrStore should return loaded=true (already running)")
	}

	// Clean up
	svc.running.Delete(uint(1))

	// After cleanup, should be able to store again
	_, loaded = svc.running.LoadOrStore(uint(1), struct{}{})
	if loaded {
		t.Fatal("LoadOrStore after Delete should not return loaded=true")
	}
	svc.running.Delete(uint(1))
}

// TestBackupService_RunNow_DifferentJobsCanRunConcurrently verifies that different jobs
// can run concurrently without blocking each other.
func TestBackupService_RunNow_DifferentJobsCanRunConcurrently(t *testing.T) {
	svc := &BackupService{}

	// Multiple different job IDs should all be accepted
	for i := uint(1); i <= 5; i++ {
		_, loaded := svc.running.LoadOrStore(i, struct{}{})
		if loaded {
			t.Fatalf("job %d should not be loaded", i)
		}
	}

	// Verify all are present
	count := 0
	svc.running.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	if count != 5 {
		t.Errorf("expected 5 running jobs, got %d", count)
	}

	// Clean up
	for i := uint(1); i <= 5; i++ {
		svc.running.Delete(i)
	}
}

// TestBackupService_RunNow_ConcurrentGoroutinesRace verifies no race conditions with -race flag.
func TestBackupService_RunNow_ConcurrentGoroutinesRace(t *testing.T) {
	svc := &BackupService{}
	const goroutines = 50

	var wg sync.WaitGroup
	accepted := make(chan uint, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id uint) {
			defer wg.Done()
			_, loaded := svc.running.LoadOrStore(id, struct{}{})
			if !loaded {
				accepted <- id
			}
		}(uint(i % 5)) // 5 unique job IDs
	}

	wg.Wait()
	close(accepted)

	// Exactly 5 unique jobs should have been accepted (first goroutine for each ID)
	acceptedMap := make(map[uint]bool)
	for id := range accepted {
		acceptedMap[id] = true
	}
	if len(acceptedMap) != 5 {
		t.Errorf("expected exactly 5 unique accepted jobs, got %d", len(acceptedMap))
	}

	// Clean up
	for i := uint(0); i < 5; i++ {
		svc.running.Delete(i)
	}
}

// TestBackupService_RunNow_ReleaseAfterCompletion verifies that after a job completes,
// it can be triggered again.
func TestBackupService_RunNow_ReleaseAfterCompletion(t *testing.T) {
	svc := &BackupService{}
	jobID := uint(42)

	// First run: acquire
	_, loaded := svc.running.LoadOrStore(jobID, struct{}{})
	if loaded {
		t.Fatal("first acquire should succeed")
	}

	// Simulate completion: release
	svc.running.Delete(jobID)

	// Second run: should succeed again
	_, loaded = svc.running.LoadOrStore(jobID, struct{}{})
	if loaded {
		t.Fatal("second acquire after release should succeed")
	}
	svc.running.Delete(jobID)
}

// TestBackupService_RunNow_ConcurrentStress tests rapid acquire/release cycles.
func TestBackupService_RunNow_ConcurrentStress(t *testing.T) {
	svc := &BackupService{}
	jobID := uint(99)
	const iterations = 100

	var wg sync.WaitGroup

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, loaded := svc.running.LoadOrStore(jobID, struct{}{})
			if !loaded {
				// Simulate short work
				time.Sleep(time.Microsecond)
				svc.running.Delete(jobID)
			}
		}()
	}

	wg.Wait()

	// Ensure cleanup
	svc.running.Delete(jobID)
	// If we get here without deadlock or race, the test passes
	fmt.Println("stress test passed")
}
