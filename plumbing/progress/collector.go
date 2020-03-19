package progress

import (
	"context"
	"io"
	"time"
)

// CountingUpdate holds the part and whole for calculating percentage of progress
type CountingUpdate struct {
	Type  UpdateType
	Count uint32
	Max   uint32
}

// Collector collects data to send progress updates
type Collector struct {
	receiveObject chan *CountingUpdate
	resolveDelta  chan *CountingUpdate
	accumulate    chan int
	Reader        io.Reader
	seeker        io.Seeker
	isSeekable    bool
	pr            *Reporter
}

// NewCollector creates a Collector
func NewCollector(reader io.Reader, pr *Reporter) *Collector {
	seeker, ok := reader.(io.ReadSeeker)
	return &Collector{
		receiveObject: make(chan *CountingUpdate),
		resolveDelta:  make(chan *CountingUpdate),
		accumulate:    make(chan int),
		Reader:        reader,
		seeker:        seeker,
		isSeekable:    ok,
		pr:            pr,
	}
}

// Start begins a background process that will send periodic progress updates
func (pc *Collector) Start(ctx context.Context) {
	go func() {
		defer close(pc.pr.Receive)

		startedReceivingObjects := false
		startedResolvingDeltas := false
		doneReceivingObjects := false
		doneResolvingDeltas := false
		var lastObject *CountingUpdate = nil
		var lastDelta *CountingUpdate = nil

		total := uint64(0)
		rate := uint64(0)
		throughput := NewThroughput()

		updateTicker := time.NewTicker(pc.pr.UpdatePeriod)
		rateTicker := time.NewTicker(pc.pr.RatePeriod)
		defer updateTicker.Stop()
		defer rateTicker.Stop()

		sendReceivingObjects := func() {
			if lastObject != nil && !doneReceivingObjects {
				pc.pr.Receive <- &Update{
					Type:          ReceivingObjects,
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
				pc.pr.Receive <- &Update{
					Type:  ResolvingDeltas,
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
func (pc *Collector) ReceiveObject(count, max uint32) {
	pc.receiveObject <- &CountingUpdate{
		Type:  ReceivingObjects,
		Count: count,
		Max:   max,
	}
}

// ResolveDelta sends an object to the background process to be tracked
func (pc *Collector) ResolveDelta(count, max uint32) {
	pc.resolveDelta <- &CountingUpdate{
		Type:  ResolvingDeltas,
		Count: count,
		Max:   max,
	}
}

// Read satisfies the io.Reader interface; sends bytes read to the background process to be tracked
func (pc *Collector) Read(b []byte) (int, error) {
	bytesRead, err := pc.Reader.Read(b)
	pc.accumulate <- bytesRead
	return bytesRead, err
}

// Seek satisfies the io.Seeker interface
func (pc *Collector) Seek(offset int64, whence int) (int64, error) {
	return pc.seeker.Seek(offset, whence)
}
