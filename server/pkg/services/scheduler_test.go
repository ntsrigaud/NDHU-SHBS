package services_test

import (
	"sync/atomic"
	"testing"
	"time"

	"shbs-server/pkg/services"
)

func TestSetupScheduler_JobRuns(t *testing.T) {
	var count atomic.Int32

	scheduler := services.SetupScheduler(50*time.Millisecond, func() {
		count.Add(1)
	})
	defer scheduler.Shutdown()

	// Wait long enough for the job to fire at least once.
	time.Sleep(200 * time.Millisecond)

	if count.Load() == 0 {
		t.Error("expected scheduler job to have run at least once")
	}
}

func TestSetupScheduler_Shutdown(t *testing.T) {
	var count atomic.Int32

	scheduler := services.SetupScheduler(50*time.Millisecond, func() {
		count.Add(1)
	})

	time.Sleep(120 * time.Millisecond)
	scheduler.Shutdown()
	snapshot := count.Load()

	// After shutdown, no more increments should occur.
	time.Sleep(150 * time.Millisecond)
	if count.Load() != snapshot {
		t.Errorf("job ran after Shutdown: count went from %d to %d", snapshot, count.Load())
	}
}
