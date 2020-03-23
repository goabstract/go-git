package progress

import (
	"time"
)

// Reporter sends digested progress updates
type Reporter struct {
	Receive      chan *Update
	UpdatePeriod time.Duration
	RatePeriod   time.Duration
}

// NewReporter creates a new Reporter
func NewReporter(updatePeriod time.Duration, ratePeriod time.Duration) *Reporter {
	return &Reporter{
		Receive:      make(chan *Update),
		UpdatePeriod: updatePeriod,
		RatePeriod:   ratePeriod,
	}
}
