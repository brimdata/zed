package emitter

import (
	"fmt"
	"io"
	"time"

	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/proc"
)

type Emitter struct {
	writer   zson.Writer
	warnings io.Writer
}

func NewEmitter(w zson.Writer) *Emitter {
	return &Emitter{
		writer: w,
	}
}

func (e *Emitter) SetWarningsWriter(w io.Writer) {
	e.warnings = w
}

func (e *Emitter) send(cid int, arr zson.Batch) error {
	for _, r := range arr.Records() {
		if err := e.writer.Write(r); err != nil {
			return err
		}
	}
	return nil
}

func (e *Emitter) writeWarnings(msg string) error {
	_, err := fmt.Fprintln(e.warnings, msg)
	if err != nil {
		return err
	}
	return nil
}

func (e *Emitter) handle(res proc.MuxResult) error {
	if res.Warning != "" {
		e.writeWarnings(res.Warning)
		return nil
	}
	if res.Batch != nil {
		return e.send(res.ID, res.Batch)
	}
	return nil
}

func (e *Emitter) Run(out *proc.MuxOutput) error {
	for !out.Complete() {
		res := out.Pull(time.After(time.Second * 10))
		if res.Err == proc.ErrTimeout {
			continue
		}
		if res.Err != nil {
			return res.Err
		}
		if err := e.handle(res); err != nil {
			return err
		}
	}
	return nil
}
