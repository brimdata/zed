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
	live     *uilive.Writer
	interval time.Duration
	updater  Displayer
	buffer   *bytes.Buffer
	close    chan struct{}
	done     sync.WaitGroup
}

func New(updater Displayer, interval time.Duration) *Display {
	return &Display{
		live:     uilive.New(),
		interval: interval,
		updater:  updater,
		buffer:   bytes.NewBuffer(nil),
		close:    make(chan struct{}),
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
			close(d.close)
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
	close(d.close)
	d.done.Wait()
	d.update()
}

func (d *Display) Wait() {
	d.done.Wait()
}

func (d *Display) Done() {
	return d.done.Wait()
}
