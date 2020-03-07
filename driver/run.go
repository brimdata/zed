package driver

import (
	"fmt"
	"io"
	"time"

	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
)

type Driver struct {
	writers  []zbuf.Writer
	warnings io.Writer
}

func New(w ...zbuf.Writer) *Driver {
	return &Driver{
		writers: w,
	}
}

func (d *Driver) SetWarningsWriter(w io.Writer) {
	d.warnings = w
}

func (d *Driver) Write(cid int, arr zbuf.Batch) error {
	if len(d.writers) == 1 {
		cid = 0
	}
	for _, r := range arr.Records() {
		if err := d.writers[cid].Write(r); err != nil {
			return err
		}
	}
	return nil
}

func (d *Driver) WriteWarning(msg string) error {
	if d.warnings != nil {
		_, err := fmt.Fprintln(d.warnings, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Driver) Run(out *proc.MuxOutput) error {
	if len(d.writers) != 1 && len(d.writers) != out.N() {
		return fmt.Errorf("Driver.Run(): Mismatched channels and writer counts")
	}

	for !out.Complete() {
		res := out.Pull(time.After(time.Second * 10))
		if res.Err == proc.ErrTimeout {
			continue
		}
		if res.Err != nil {
			return res.Err
		}
		if res.Warning != "" {
			d.WriteWarning(res.Warning)
		}
		if res.Batch != nil {
			if err := d.Write(res.ID, res.Batch); err != nil {
				return err
			}
		}
	}
	return nil
}
