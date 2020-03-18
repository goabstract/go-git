package progress

const (
	ThroughputRateSlotCount = 8
)

// Throughput calculates the rate that data is read
type Throughput struct {
	CurrentTotal        uint64
	PreviousTotal       uint64
	PreviousNanoseconds uint64
	AverageBytes        uint64
	AverageMisecs       uint64   // a "misec" is 1/1024 sec
	LastBytes           []uint64 // size = 8
	LastMisecs          []uint64 // size = 8; "misec" is 1/1024 sec
	Index               uint32
}

// NewThroughput creates a new throughput
func NewThroughput() *Throughput {
	return &Throughput{
		LastBytes:  make([]uint64, ThroughputRateSlotCount),
		LastMisecs: make([]uint64, ThroughputRateSlotCount),
	}
}

// AdvanceTime tracks the progress and returns the current rate of bytes being read
// SEE: https://github.com/git/git/blob/be8661a3286c67a5d4088f4226cbd7f8b76544b0/progress.c#L186-L244
func (t *Throughput) AdvanceTime(now int64) uint64 {
	// a "misec" is 1/1024 of a second, useful for simplifying the
	// calculation of rate in IEC units
	misecs := ((uint64(now) - t.PreviousNanoseconds) * 4398) >> 32
	count := uint64(t.CurrentTotal - t.PreviousTotal)

	t.PreviousTotal = t.CurrentTotal
	t.PreviousNanoseconds = uint64(now)
	t.AverageBytes += count
	t.AverageMisecs += misecs

	rate := t.AverageBytes / t.AverageMisecs

	t.AverageBytes -= t.LastBytes[t.Index]
	t.AverageMisecs -= t.LastMisecs[t.Index]
	t.LastBytes[t.Index] = count
	t.LastMisecs[t.Index] = misecs
	t.Index = (t.Index + 1) % ThroughputRateSlotCount

	return rate
}

// AdvanceBytesRead tracks the number of bytes read
func (t *Throughput) AdvanceBytesRead(bytesRead int) {
	if t.PreviousTotal == 0 {
		t.PreviousTotal = uint64(bytesRead)
	}
	t.CurrentTotal += uint64(bytesRead)
}
