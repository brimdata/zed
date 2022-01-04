package display

import (
	"bytes"
	"context"
	"io"
	"sync"
	"time"

	"github.com/gosuri/uilive"
)

type Displayer interface {
	Display(io.Writer) bool
}

type Display struct {
	live     *uilive.Writer
	interval time.Duration
	updater  Displayer
	buffer   *bytes.Buffer
	ctx      context.Context
	cancel   context.CancelFunc
	done     sync.WaitGroup
}

func New(updater Displayer, interval time.Duration, out io.Writer) *Display {
	live := uilive.New()
	live.Out = out
	ctx, cancel := context.WithCancel(context.Background())
	return &Display{
		ctx:      ctx,
		cancel:   cancel,
		live:     live,
		interval: interval,
		updater:  updater,
		buffer:   bytes.NewBuffer(nil),
	}
}

func (d *Display) update() bool {
	d.buffer.Reset()
	cont := d.updater.Display(d.buffer)
	// Ignore any errors.
	_, _ = io.Copy(d.live, d.buffer)
	_ = d.live.Flush()
	return cont
}

func (d *Display) Run() {
	d.done.Add(1)
	for {
		if !d.update() {
			d.cancel()
		}
		select {
		case <-d.ctx.Done():
			d.done.Done()
			return
		case <-time.After(d.interval):
		}
	}
}

func (d *Display) Bypass() io.Writer {
	return d.live.Bypass()
}

func (d *Display) Close() {
	d.cancel()
	d.done.Wait()
	d.update()
}

func (d *Display) Wait() {
	d.done.Wait()
}

func (d *Display) Done() {
	d.done.Wait()
}
