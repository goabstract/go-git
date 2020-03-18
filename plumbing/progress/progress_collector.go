package progress

import (
	"context"
	"io"
	"time"
)

// CountingProgressUpdate holds the part and whole for calculating percentage of progress
type CountingProgressUpdate struct {
	Type  ProgressType
	Count uint32
	Max   uint32
}

// ProgressCollector collects data to send progress updates
type ProgressCollector struct {
	receiveObject chan *CountingProgressUpdate
	resolveDelta  chan *CountingProgressUpdate
	accumulate    chan int
	Reader        io.Reader
	seeker        io.Seeker
	isSeekable    bool
	pr            *ProgressReporter
}

func NewProgressCollector(reader io.Reader, pr *ProgressReporter) *ProgressCollector {
	seeker, ok := reader.(io.ReadSeeker)
	return &ProgressCollector{
		receiveObject: make(chan *CountingProgressUpdate),
		resolveDelta:  make(chan *CountingProgressUpdate),
		accumulate:    make(chan int),
		Reader:        reader,
		seeker:        seeker,
		isSeekable:    ok,
		pr:            pr,
	}
}

// Start begins a background process that will send periodic progress updates
func (pc *ProgressCollector) Start(ctx context.Context) {
	if pc == nil || pc.pr == nil {
		return
	}
	go func() {
		defer close(pc.pr.Receive)

		startedReceivingObjects := false
		startedResolvingDeltas := false
		doneReceivingObjects := false
		doneResolvingDeltas := false
		var lastObject *CountingProgressUpdate = nil
		var lastDelta *CountingProgressUpdate = nil

		total := uint64(0)
		rate := uint64(0)
		throughput := NewThroughput()

		updateTicker := time.NewTicker(pc.pr.UpdatePeriod)
		rateTicker := time.NewTicker(pc.pr.RatePeriod)
		defer updateTicker.Stop()
		defer rateTicker.Stop()

		sendReceivingObjects := func() {
			if lastObject != nil && !doneReceivingObjects {
				pc.pr.Receive <- &ProgressUpdate{
					Type:          ProgressReceivingObjects,
					Count:         lastObject.Count,
					Max:           lastObject.Max,
					BytesReceived: total,
					Rate:          rate,
				}

				if lastObject.Count == lastObject.Max {
					doneReceivingObjects = true
				}
			}
		}

		sendResolvingDeltas := func() {
			if lastDelta != nil && !doneResolvingDeltas {
				pc.pr.Receive <- &ProgressUpdate{
					Type:  ProgressResolvingDeltas,
					Count: lastDelta.Count,
					Max:   lastDelta.Max,
				}

				if lastDelta.Count == lastDelta.Max {
					doneResolvingDeltas = true
				}
			}
		}

		for {
			select {
			case v := <-pc.receiveObject:
				lastObject = v
				if !startedReceivingObjects {
					startedReceivingObjects = true
					sendReceivingObjects()
				}
			case v := <-pc.resolveDelta:
				lastDelta = v
				if !startedResolvingDeltas {
					startedResolvingDeltas = true
					sendResolvingDeltas()
				}
			case v := <-pc.accumulate:
				total += uint64(v)
				throughput.AdvanceBytesRead(v)
			case <-rateTicker.C:
				rate = throughput.AdvanceTime(time.Now().UnixNano())
			case <-updateTicker.C:
				sendReceivingObjects()
				sendResolvingDeltas()
			case <-ctx.Done():
				rate = throughput.AdvanceTime(time.Now().UnixNano())
				sendReceivingObjects()
				sendResolvingDeltas()
				return
			}
		}
	}()
}

// ReceiveObject sends an object to the background process to be tracked
func (pc *ProgressCollector) ReceiveObject(count, max uint32) {
	pc.receiveObject <- &CountingProgressUpdate{
		Type:  ProgressReceivingObjects,
		Count: count,
		Max:   max,
	}
}

// ResolveDelta sends an object to the background process to be tracked
func (pc *ProgressCollector) ResolveDelta(count, max uint32) {
	pc.resolveDelta <- &CountingProgressUpdate{
		Type:  ProgressResolvingDeltas,
		Count: count,
		Max:   max,
	}
}

// Read satisfies the io.Reader interface; sends bytes read to the background process to be tracked
func (pc *ProgressCollector) Read(b []byte) (int, error) {
	bytesRead, err := pc.Reader.Read(b)
	pc.accumulate <- bytesRead
	return bytesRead, err
}

// Seeker satisfies the io.Seeker interface
func (pc *ProgressCollector) Seek(offset int64, whence int) (int64, error) {
	return pc.seeker.Seek(offset, whence)
}
