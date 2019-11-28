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

func unknownFormat(format string) error {
	return fmt.Errorf("unknown output format: %s", format)
}

func (e *Emitter) SetWarnaingsWriter(w io.Writer) {
	e.warnings = w
}

func (e *Emitter) Write(cid int, arr zson.Batch) error {
	for _, r := range arr.Records() {
		// for cid < 0, we keep the Channel that's already in
		// the record
		if cid >= 0 {
			r.Channel = uint16(cid)
		}
		if err := e.writer.Write(r); err != nil {
			return err
		}
	}
	return nil
}

func (e *Emitter) WriteWarning(msg string) error {
	_, err := fmt.Fprintln(e.warnings, msg)
	if err != nil {
		return err
	}
	return nil
}

//XXX this goes somewhere else
func (e *Emitter) Run(out *proc.MuxOutput) error {
	for !out.Complete() {
		res := out.Pull(time.After(time.Second * 10))
		if res.Err == proc.ErrTimeout {
			continue
		}
		if res.Err != nil {
			return res.Err
		}
		if res.Warning != "" {
			e.WriteWarning(res.Warning)
		}
		if res.Batch != nil {
			if err := e.Write(res.ID, res.Batch); err != nil {
				return err
			}
		}
	}
	return nil
}
