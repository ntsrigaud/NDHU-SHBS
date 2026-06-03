package services

import (
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
)

// SetupScheduler creates a gocron scheduler and registers a single recurring
// job that runs the provided function on the given interval. The scheduler is
// started immediately.
func SetupScheduler(interval time.Duration, job func()) gocron.Scheduler {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("Cannot create scheduler: %v", err)
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(interval),
		gocron.NewTask(job),
	)
	if err != nil {
		log.Fatalf("Cannot register scheduler job: %v", err)
	}

	scheduler.Start()
	return scheduler
}
