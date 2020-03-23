package progress

import (
	"time"
)

// ProgressReporter sends digested progress updates
type ProgressReporter struct {
	Receive      chan *ProgressUpdate
	UpdatePeriod time.Duration
	RatePeriod   time.Duration
}

// NewProgressReporter creates a new ProgressReporter
func NewProgressReporter(updatePeriod time.Duration, ratePeriod time.Duration) *ProgressReporter {
	return &ProgressReporter{
		Receive:      make(chan *ProgressUpdate),
		UpdatePeriod: updatePeriod,
		RatePeriod:   ratePeriod,
	}
}
