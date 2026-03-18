package main

import (
	"fmt"
	"time"
)

func formatDate(t time.Time) string {
	return t.Format("20060102")
}

func parseTimestamp(ts int64) string {
	return time.Unix(ts, 0).UTC().Format("2006-01-02 15:04:05 UTC")
}

func workoutTypeName(mode, subMode int) string {
	if name, ok := workoutTypes[[2]int{mode, subMode}]; ok {
		return name
	}
	return fmt.Sprintf("Unknown (%d/%d)", mode, subMode)
}

func formatPace(avgSpeedSecPerKm float64) string {
	if avgSpeedSecPerKm == 0 {
		return "N/A"
	}
	total := int(avgSpeedSecPerKm)
	mins := total / 60
	secs := total % 60
	return fmt.Sprintf("%d:%02d/km", mins, secs)
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}
