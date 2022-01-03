package proc

import (
	"sync"

	"github.com/brimdata/zed/zbuf"
)

type Selector interface {
	Forward(*Router, zbuf.Batch) error
}

type Router struct {
	pctx     *Context
	parent   Interface
	selector Selector
	routes   map[Interface]chan<- Result
	blocked  map[Interface]struct{}
	doneCh   chan Interface
	once     sync.Once
	backlog  []Interface
	// Router err protected by result channel
	err error
}

func NewRouter(pctx *Context, parent Interface) *Router {
	return &Router{
		pctx:    pctx,
		parent:  NewCatcher(parent),
		routes:  make(map[Interface]chan<- Result),
		blocked: make(map[Interface]struct{}),
		doneCh:  make(chan Interface),
	}
}

func (r *Router) Link(s Selector) {
	r.selector = s
}

func (r *Router) AddRoute() Interface {
	ch := make(chan Result)
	child := &route{
		router:   r,
		resultCh: ch,
	}
	r.routes[child] = ch
	return child
}

func (r *Router) run() {
	defer func() {
		// At double-EOS completion, we close all channels so the
		// downstream routes will automatically get their double-EOS completions.
		for _, ch := range r.routes {
			close(ch)
		}
	}()
	var eos bool
	for {
		batch, err := r.parent.Pull()
		if err != nil {
			r.err = err
			return
		}
		if batch == nil {
			if eos {
				return
			}
			r.sendEOS()
			eos = true
			continue
		}
		eos = false
		// The selectors decides what if any of the batch it
		// wants to send to which down stream procs by calling
		// back the Router.Send() method.  We Unref() the batch here
		// after calling Forward(), so it is up to the selector to Ref()
		// the batch if it wants to hold on to it.
		if err := r.selector.Forward(r, batch); err != nil {
			r.err = err
			return
		}
		batch.Unref()
		r.drain()
		if len(r.blocked) == len(r.routes) {
			// All downstream routes have indicated they are done,
			// so they have all received exactly one EOS.  Now call
			// halt to send done to our parent and advance to the
			// first EOS from upstream (which is already delivered
			// to all downstream routes).
			// We set EOS true, then repeat the loop exiting on
			// a second EOS or resuming the next batch group
			// to all downstream routes.
			if err := r.halt(); err != nil {
				r.err = err
				return
			}
			r.unblock()
			eos = true
		}
	}
}

// Send an EOS to each unblocked route and unblock the routes that
// were already blocked (and thus already received their first EOS
// for this group).
func (r *Router) sendEOS() {
	for p := range r.routes {
		if ok := r.Send(p, nil, nil); !ok {
			delete(r.blocked, p)
		}
	}

}

func (r *Router) unblock() {
	for p := range r.blocked {
		delete(r.blocked, p)
	}
}

func (r *Router) halt() error {
	r.parent.Done()
	for {
		b, err := r.parent.Pull()
		if b == nil || err != nil {
			return err
		}
		b.Unref()
	}
}

func (r *Router) Send(p Interface, b zbuf.Batch, err error) bool {
	if _, ok := r.blocked[p]; ok {
		if b != nil {
			b.Unref()
		}
		return false
	}
	for {
		select {
		case r.routes[p] <- Result{Batch: b, Err: err}:
			return true
		case p := <-r.doneCh:
			r.backlog = append(r.backlog, p)
		case <-r.pctx.Done():
			return false
		}
	}
}

func (r *Router) drain() {
	for len(r.backlog) > 0 {
		p := r.backlog[0]
		r.backlog = r.backlog[1:]
		r.block(p)
	}
	for {
		select {
		case p := <-r.doneCh:
			r.block(p)
		default:
			return
		}
	}
}

func (r *Router) block(p Interface) {
	// When we get a done, we block the channel to that route
	// if it's not already blocked.  Since the contract of
	// Done is the downstream initiator will pull until EOS,
	// we know we won't deadlock here.  If the channel is
	// already blocked, it will get its single EOS anyway
	// and then block in its Pull until we send it the first
	// batch of the next group or the second EOS.
	if _, ok := r.blocked[p]; !ok {
		r.Send(p, nil, nil)
		r.blocked[p] = struct{}{}
	}
}

func (r *Router) done(p Interface) {
	r.doneCh <- p
}

type route struct {
	router   *Router
	resultCh <-chan Result
}

func (r *route) Pull() (zbuf.Batch, error) {
	r.router.once.Do(func() {
		go r.router.run()
	})
	if result, ok := <-r.resultCh; ok {
		return result.Batch, result.Err
	}
	return nil, r.router.err
}

func (r *route) Done() {
	r.router.done(r)
}
