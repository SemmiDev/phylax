package usecase

import (
	"fmt"
	"regexp"
	"time"
)

// extractTimestamp extracts timestamp from backup filename
// Format: {dbname}_{type}_{YYYYMMDD}_{HHMMSS}.{ext}
func extractTimestamp(filename string) (time.Time, error) {
	// Pattern: name_type_20060102_150405.ext or name_type_20060102_150405.sql.gz
	pattern := regexp.MustCompile(`(\d{8})_(\d{6})`)
	matches := pattern.FindStringSubmatch(filename)

	if len(matches) < 3 {
		return time.Time{}, fmt.Errorf("invalid filename format: no timestamp found")
	}

	timestampStr := matches[1] + "_" + matches[2]
	return time.Parse("20060102_150405", timestampStr)
}
