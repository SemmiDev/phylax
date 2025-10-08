package logger

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLogger(t *testing.T) {
	Convey("Given the Logger package", t, func() {
		Convey("New function", func() {
			Convey("When creating a logger with console output only", func() {
				logger, err := New("info", "")

				Convey("It should create a logger successfully", func() {
					So(err, ShouldBeNil)
					So(logger, ShouldNotBeNil)

					// Verify logging works by checking if it doesn't panic
					So(func() { logger.Info("Test log") }, ShouldNotPanic)
				})
			})

			Convey("When creating a logger with a valid log file", func() {
				// Create a unique temporary directory for the log file
				tempDir, err := os.MkdirTemp("", "logger_test")
				So(err, ShouldBeNil)
				defer os.RemoveAll(tempDir) // Clean up the directory after the test

				logFile := filepath.Join(tempDir, "test.log")

				logger, err := New("debug", logFile)

				Convey("It should create a logger and log file successfully", func() {
					So(err, ShouldBeNil)
					So(logger, ShouldNotBeNil)

					// Write a log to ensure the file is created
					logger.Debug("Test debug log")
					logger.Sync() // Force flush to ensure file is written

					// Verify the log file exists
					_, err := os.Stat(logFile)
					So(err, ShouldBeNil)

					// Clean up
					logger.Close()
				})
			})

			Convey("When creating a logger with an invalid log level", func() {
				logger, err := New("invalid", "")

				Convey("It should default to Info level and create a logger", func() {
					So(err, ShouldBeNil)
					So(logger, ShouldNotBeNil)

					// Verify logging works at default level
					So(func() { logger.Info("Test info log") }, ShouldNotPanic)
					So(func() { logger.Debug("Test debug log") }, ShouldNotPanic) // Debug should not log
				})
			})

			Convey("When creating a logger with an invalid log file path", func() {
				// Use an invalid path (e.g., a directory we can't create)
				logFile := "/invalid/path/test.log"

				logger, err := New("info", logFile)

				Convey("It should return an error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "failed to create log directory")
					So(logger, ShouldBeNil)
				})
			})
		})

		Convey("Close method", func() {
			Convey("When closing a logger with file output", func() {
				// Create a unique temporary directory for the log file
				tempDir, err := os.MkdirTemp("", "logger_test")
				So(err, ShouldBeNil)
				defer os.RemoveAll(tempDir) // Clean up the directory after the test

				logFile := filepath.Join(tempDir, "test.log")

				logger, err := New("info", logFile)
				So(err, ShouldBeNil)
				So(logger, ShouldNotBeNil)

				// Write a log to ensure the file is created
				logger.Info("Test info log")
				logger.Sync() // Force flush to ensure file is written

				Convey("It should close without error", func() {
					So(func() { logger.Close() }, ShouldNotPanic)

					// Verify the log file exists
					_, err := os.Stat(logFile)
					So(err, ShouldBeNil)
				})
			})

			Convey("When closing a logger with console output only", func() {
				logger, err := New("info", "")
				So(err, ShouldBeNil)
				So(logger, ShouldNotBeNil)

				Convey("It should close without error", func() {
					So(func() { logger.Close() }, ShouldNotPanic)
				})
			})
		})
	})
}
