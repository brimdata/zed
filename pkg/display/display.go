package display

import (
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/gosuri/uilive"
)

type Displayer interface {
	Display(io.Writer) bool
}

type Display struct {
	close     chan struct{}
	closeOnce sync.Once
	live      *uilive.Writer
	interval  time.Duration
	updater   Displayer
	buffer    *bytes.Buffer
	done      sync.WaitGroup
}

func New(updater Displayer, interval time.Duration, out io.Writer) *Display {
	live := uilive.New()
	live.Out = out
	return &Display{
		close:    make(chan struct{}),
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
			d.closeOnce.Do(func() { close(d.close) })
		}
		select {
		case <-d.close:
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
	d.closeOnce.Do(func() { close(d.close) })
	d.done.Wait()
	d.update()
}

func (d *Display) Wait() {
	d.done.Wait()
}

func (d *Display) Done() {
	d.done.Wait()
}
