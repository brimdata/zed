package split

import (
	"sync"

	"github.com/brimdata/zq/proc"
	"github.com/brimdata/zq/zbuf"
)

// Splitter splits its input into multiple proc outputs.  Since procs run from the
// receiver backward via Pull(), SplitProc pulls data from upstream when all the
// outputs are ready, then sends the data downstream.
//
// This scheme implements flow control since the SplitProc prevents any of
// the downstream from running ahead, esentially running the parallel paths
// at the rate of the slowest consumer.
type Splitter struct {
	parent   proc.Interface
	once     sync.Once
	n        int
	requests chan (chan<- proc.Result)
}

func New(parent proc.Interface) *Splitter {
	return &Splitter{
		parent:   parent,
		requests: make(chan (chan<- proc.Result)),
	}
}

func (s *Splitter) Add() chan (chan<- proc.Result) {
	s.n++
	return s.requests
}

func (s *Splitter) gather(strip []chan<- proc.Result) []chan<- proc.Result {
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

func (s *Splitter) run() {
	// This loop is started by the first downstream SplitChannel as
	// long as there are active downstream consumers.
	// If the downstream proc isn't ready, it's request won't have arrived
	// and this thread will block in the gather loop waiting for all requests.
	// Once all the requests are available (or null requests are received
	// indicating the downstream proc is done), then data is pulled from
	// the upstream path and a reference-counted batch is transmitted to
	// each requesting entity.
	strip := make([]chan<- proc.Result, 0, s.n)
	for {
		flight := s.gather(strip)
		if s.n == 0 {
			break
		}
		batch, err := s.parent.Pull()
		send(flight, proc.Result{batch, err})
		if batch != nil {
			batch.Unref()
		}
	}
	s.parent.Done()
}

func send(flight []chan<- proc.Result, result proc.Result) {
	for _, ch := range flight {
		if result.Batch != nil {
			result.Batch.Ref()
		}
		ch <- result
	}
}

type Proc struct {
	request chan chan<- proc.Result
	ch      chan proc.Result
	parent  *Splitter
}

func (s *Splitter) NewProc() *Proc {
	p := &Proc{
		ch:     make(chan proc.Result),
		parent: s,
	}
	p.request = s.Add()
	return p
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	p.parent.once.Do(func() {
		go p.parent.run()
	})
	if p.ch == nil {
		return nil, nil
	}
	// Send SplitProc a request, then read the result. On EOS we send a nil
	// request to let SplitProc know we're done, which will cause it to exit
	// when it sees that all of us SplitChannels are gone.
	p.request <- p.ch
	result := <-p.ch
	if proc.EOS(result.Batch, result.Err) {
		p.Done()
	}
	return result.Batch, result.Err
}

func (p *Proc) Done() {
	// Signal to SplitProc that this path is done by sending a nil channel
	// object.  We go ahead and mark the Proc done by setting
	// it's channel to nil in case a spurious Pull() is called, but this
	// should not happen.
	if p.ch != nil {
		var null chan proc.Result
		p.request <- null
		p.ch = nil
	}
}
