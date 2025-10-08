package usecase

import (
	"fmt"
	"strings"
	"time"
)

func extractTimestamp(filename string) (time.Time, error) {
	// Remove extensions
	name := strings.TrimSuffix(filename, ".gz")
	name = strings.TrimSuffix(name, ".sql")
	name = strings.TrimSuffix(name, ".dump")
	name = strings.TrimSuffix(name, ".archive")

	// Split by underscore and get last two parts (date_time)
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid filename format")
	}

	dateStr := parts[len(parts)-2]
	timeStr := parts[len(parts)-1]
	timestampStr := dateStr + "_" + timeStr

	return time.Parse("20060102_150405", timestampStr)
}
