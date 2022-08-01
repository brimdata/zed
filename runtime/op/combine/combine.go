// A combine proc merges multiple upstream inputs into one output.
package combine

import (
	"context"
	"sync"

	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
)

type Proc struct {
	ctx      context.Context
	once     sync.Once
	parents  []*puller
	queue    <-chan *puller
	waitCh   <-chan struct{}
	nblocked int
}

func New(pctx *op.Context, parents []zbuf.Puller) *Proc {
	ctx := pctx.Context
	queue := make(chan *puller, len(parents))
	pullers := make([]*puller, 0, len(parents))
	waitCh := make(chan struct{})
	for _, parent := range parents {
		pullers = append(pullers, newPuller(ctx, waitCh, parent, queue))
	}
	return &Proc{
		ctx:     ctx,
		parents: pullers,
		queue:   queue,
		waitCh:  waitCh,
	}
}

func (p *Proc) Pull(done bool) (zbuf.Batch, error) {
	p.once.Do(func() {
		for _, parent := range p.parents {
			go parent.run()
		}
	})
	if done {
		return nil, p.propagateDone()
	}
	for {
		next, err := p.next()
		if err != nil {
			return nil, err
		}
		if next == nil {
			// Everything is blocked due to EOS received
			// on all paths.  We unblock everything to get
			// ready for the next platoon and send an EOS
			// downstream representing the fact that all fan-in
			// legs hit their EOS.
			return nil, p.unwait()
		}
		select {
		case result := <-next.resultCh:
			if result.Err != nil {
				return nil, result.Err
			}
			if result.Batch == nil {
				p.block(next)
				continue
			}
			return result.Batch, nil
		case <-p.ctx.Done():
			return nil, p.ctx.Err()
		}
	}
}

func (p *Proc) next() (*puller, error) {
	if p.nblocked >= len(p.parents) {
		return nil, nil
	}
	select {
	case parent := <-p.queue:
		return parent, nil
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	}
}

func (p *Proc) unwait() error {
	if len(p.parents) != p.nblocked {
		panic("unwait called without all parents blocked")
	}
	for _, parent := range p.parents {
		select {
		case <-p.waitCh:
		case <-p.ctx.Done():
			return p.ctx.Err()
		}
		parent.blocked = false
	}
	p.nblocked = 0
	return nil
}

func (p *Proc) block(parent *puller) {
	if !parent.blocked {
		parent.blocked = true
		p.nblocked++
	}
}

func (p *Proc) propagateDone() error {
	var wg sync.WaitGroup
	for _, parent := range p.parents {
		if parent.blocked {
			continue
		}
		parent := parent
		wg.Add(1)
		// We use a goroutine here because sending to parents[i].doneCh
		// can block until we've sent to parents[i+1].doneCh, as with
		// "fork (=> count() => pass) | head".
		go func() {
			defer wg.Done()
		again:
			select {
			case <-p.queue:
				// If a parent is waiting on the queue, we need to
				// read the queue to avoid deadlock.  Since we
				// are going to throw away the batch anyway, we can
				// simply ignore which parent it is as we will hit all
				// of them eventually as we loop over each unblocked parent.
				goto again
			case parent.doneCh <- struct{}{}:
				p.block(parent)
			case <-p.ctx.Done():
			}
		}()
	}
	wg.Wait()
	// Make sure all the dones that canceled pending queue entries
	// are clear.  Otherwise, this will block the queue on the next
	// platoon.
drain:
	select {
	case <-p.queue:
		goto drain
	default:
	}
	// Now that everyone is blocked either because they sent us an EOS,
	// we sent them a done, or and EOS/done collided at the same time,
	// we can unblock everything.
	return p.unwait()
}

type puller struct {
	zbuf.Puller
	ctx      context.Context
	resultCh chan op.Result
	doneCh   chan struct{}
	waitCh   chan<- struct{}
	queue    chan<- *puller
	// used only by Proc
	blocked bool
}

func newPuller(ctx context.Context, waitCh chan<- struct{}, parent zbuf.Puller, q chan<- *puller) *puller {
	return &puller{
		Puller:   op.NewCatcher(parent),
		ctx:      ctx,
		resultCh: make(chan op.Result),
		doneCh:   make(chan struct{}),
		waitCh:   waitCh,
		queue:    q,
	}
}

func (p *puller) run() {
	for {
		batch, err := p.Pull(false)
		p.queue <- p
		select {
		case p.resultCh <- op.Result{batch, err}:
			if err != nil {
				return
			}
			if batch == nil {
				// We just sent an EOS, so we'll wait until
				// all the other paths are done before pulling
				// again.  We also are guaranteed here that the
				// combiner has our EOS and knows we're done and
				// will mark us blocked and not raise our doneCh.
				p.wait()
			}
		case <-p.doneCh:
			if batch == nil {
				// Combiner tells us we're done but we just
				// received an EOS from upstream, so we don't want
				// to call Pull(true) as they would break the contract.
				// Since the combiner thinks we're done and our parent
				// thinks we're done, there's nothing to do.
				// Just continue the loop and reach for the next
				// platoon.
				if !p.wait() {
					return
				}
				continue
			}
			batch.Unref()
			// Drop the pending batch and initiate a done...
			batch, _ := p.Pull(true) // do something with err
			if batch != nil {
				panic("non-nil done batch")
			}
			// After we propagate Pull to our parent, we wait
			// for the propagation to finish across all pullers
			// so we finish as a group and don't start the next
			// platoon on our leg before the other legs have finished.
			if !p.wait() {
				return
			}
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *puller) wait() bool {
	select {
	case p.waitCh <- struct{}{}:
		return true
	case <-p.ctx.Done():
		return false
	}
}
