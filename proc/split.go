package proc

import (
	"sync"

	"github.com/mccanne/zq/pkg/zng"
)

// Split splits its input into multiple proc outputs.  Since procs run from the
// receiver backward via Pull(), SplitProc pulls data from upstream when all the
// outputs are ready, then sends the data downstream.
//
// This scheme implements flow control since the SplitProc prevents any of
// the downstream from running ahead, esentially running the parallel paths
// at the rate of the slowest consumer.
type Split struct {
	Base
	once     sync.Once
	n        int
	requests chan (chan<- Result)
}

func NewSplit(c *Context, parent Proc) *Split {
	s := &Split{Base: Base{Context: c, Parent: parent}}
	s.requests = make(chan (chan<- Result))
	return s
}

func (s *Split) Add(p *SplitChannel) chan (chan<- Result) {
	s.n++
	return s.requests
}

func (s *Split) gather(strip []chan<- Result) []chan<- Result {
	flight := strip[:0]
	for len(flight) < s.n {
		ch := <-s.requests
		if ch == nil {
			s.n--
		} else {
			flight = append(flight, ch)
		}
	}
	return flight
}

func (s *Split) run() {
	// This loop is started by the first downstream SplitChannel as
	// long as there are active downstream consumers.
	// If the downstream proc isn't ready, it's request won't have arrived
	// and this thread will block in the gather loop waiting for all requests.
	// Once all the requests are available (or null requests are received
	// indicating the downstream proc is done), then data is pulled from
	// the upstream path and a reference-counted batch is transmitted to
	// each requesting entity.
	strip := make([]chan<- Result, 0, s.n)
	for s.n > 0 {
		flight := s.gather(strip)
		batch, err := s.Get()
		s.send(flight, Result{batch, err})
		if batch != nil {
			batch.Unref()
		}
	}
	s.Done()
}

func (s *Split) send(flight []chan<- Result, result Result) {
	for _, ch := range flight {
		if result.Batch != nil {
			result.Batch.Ref()
		}
		ch <- result
	}
}

func (s *Split) Pull() (zng.Batch, error) {
	// never called
	return nil, nil
}

type SplitChannel struct {
	request chan chan<- Result
	ch      chan Result
	parent  *Split
}

func NewSplitChannel(parent *Split) *SplitChannel {
	s := &SplitChannel{
		ch:     make(chan Result),
		parent: parent,
	}
	s.request = parent.Add(s)
	return s
}

func (s *SplitChannel) Parents() []Proc {
	return []Proc{s.parent}
}

func (s *SplitChannel) Pull() (zng.Batch, error) {
	s.parent.once.Do(func() {
		go s.parent.run()
	})
	if s.ch == nil {
		return nil, nil
	}
	// Send SplitProc a request, then read the result.  If context is
	// canceled, we send a nil request to let SplitProc know we're done,
	// which will cause it to exit when it sees that all of us SplitChannels
	// are gone.  We don't want both SplitProc and SplitChannel listening
	// on context canceled as that could lead to deadlock.
	var err error
	s.request <- s.ch
	select {
	case result := <-s.ch:
		if result.Batch == nil && result.Err == nil {
			s.Done()
		}
		return result.Batch, result.Err
	case <-s.parent.Context.Done():
		err = s.parent.Context.Err()
	}
	s.Done()
	return nil, err
}

func (s *SplitChannel) Done() {
	// Signal to SplitProc that this path is done by sending a nil channel
	// object.  We go ahead and mark the SplitChannel done by setting
	// it's channel to nil in case a spurious Pull() is called, but this
	// should not happen.
	if s.ch != nil {
		var null chan Result
		s.request <- null
		s.ch = nil
	}
}
