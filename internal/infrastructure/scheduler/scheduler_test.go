package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestScheduler(t *testing.T) {
	Convey("Given a Scheduler", t, func() {
		Convey("New function", func() {
			scheduler := New()

			Convey("It should create a new scheduler successfully", func() {
				So(scheduler, ShouldNotBeNil)
				So(scheduler.cron, ShouldNotBeNil)
			})
		})

		Convey("AddJob function", func() {
			scheduler := New()

			Convey("When adding a job with a valid cron spec", func() {
				// Create a temporary file to verify job execution
				tempDir, err := os.MkdirTemp("", "scheduler_test")
				So(err, ShouldBeNil)
				defer os.RemoveAll(tempDir)

				logFile := filepath.Join(tempDir, "job.log")
				job := func(ctx context.Context) error {
					return os.WriteFile(logFile, []byte("executed"), 0644)
				}

				err = scheduler.AddJob("* * * * * *", job) // Every second

				Convey("It should add the job successfully", func() {
					So(err, ShouldBeNil)

					// Start the scheduler and wait briefly to allow job execution
					scheduler.Start()
					time.Sleep(2 * time.Second) // Wait for at least one execution
					scheduler.Stop()

					// Verify the job executed by checking the log file
					_, err := os.Stat(logFile)
					So(err, ShouldBeNil)
					content, err := os.ReadFile(logFile)
					So(err, ShouldBeNil)
					So(string(content), ShouldEqual, "executed")
				})
			})

			Convey("When adding a job with an invalid cron spec", func() {
				job := func(ctx context.Context) error { return nil }
				err := scheduler.AddJob("invalid spec", job)

				Convey("It should return an error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "expected exactly 6 fields")
				})
			})
		})

		Convey("Start and Stop methods", func() {
			scheduler := New()

			Convey("When starting and stopping the scheduler", func() {
				// Create a temporary file to verify job execution
				tempDir, err := os.MkdirTemp("", "scheduler_test")
				So(err, ShouldBeNil)
				defer os.RemoveAll(tempDir)

				logFile := filepath.Join(tempDir, "job.log")
				job := func(ctx context.Context) error {
					return os.WriteFile(logFile, []byte("executed"), 0644)
				}

				err = scheduler.AddJob("* * * * * *", job) // Every second
				So(err, ShouldBeNil)

				Convey("It should start and stop without error", func() {
					So(func() { scheduler.Start() }, ShouldNotPanic)

					// Wait briefly to ensure the job runs at least once
					time.Sleep(2 * time.Second)

					// Verify the job executed
					_, err := os.Stat(logFile)
					So(err, ShouldBeNil)

					// Stop the scheduler and ensure it stops cleanly
					So(func() { scheduler.Stop() }, ShouldNotPanic)

					// Verify no further executions after stopping
					os.Remove(logFile) // Clear the file
					time.Sleep(2 * time.Second)
					_, err = os.Stat(logFile)
					So(os.IsNotExist(err), ShouldBeTrue) // File should not be recreated
				})
			})
		})
	})
}
