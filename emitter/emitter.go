package emitter

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mccanne/zq/pkg/zsio/raw"
	"github.com/mccanne/zq/pkg/zsio/table"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/proc"
)

type Emitter struct {
	writer   zson.WriteCloser
	warnings io.Writer
}

func NewEmitter(w zson.WriteCloser) *Emitter {
	return &Emitter{
		writer: w,
	}
}

func (e *Emitter) SetWarningsWriter(w io.Writer) {
	e.warnings = w
}

func (e *Emitter) send(cid int, arr zson.Batch) error {
	for _, r := range arr.Records() {
		err := e.writer.Write(r)
		if err == table.ErrTooManyLines {
			fmt.Fprintln(os.Stderr, "output table too big")
			os.Exit(1)
		}
		if err != nil {
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
	if raw, ok := e.writer.(*raw.Raw); ok {
		return raw.WriteRaw(res)
	}
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
