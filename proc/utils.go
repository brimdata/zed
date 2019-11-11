package proc

// This file contains utilties for writing unit tests of procs,
// it shouldn't include any actual product code.

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/looky-cloud/lookytalk/ast"
	"github.com/looky-cloud/lookytalk/parser"
)

func compileProc(code string, ctx *Context, parent Proc) (Proc, error) {
	// XXX If we use a newer version of pigeon, we can just compile
	// with "proc" as the terminal symbol.
	// But for now, we have to compile a complete flowgraph.
	// A simple proc isn't a valid query, so make up a program with
	// a wildcard filter and then pull out just the proc we care
	// about below.
	prog := fmt.Sprintf("* | %s", code)
	parsed, err := parser.Parse("", []byte(prog))
	if err != nil {
		return nil, err
	}

	sp, ok := parsed.(*ast.SequentialProc)
	if !ok {
		return nil, errors.New("expected SequentialProc")
	}
	if len(sp.Procs) != 2 {
		return nil, errors.New("expected 2 procs")
	}

	proc, err := CompileProc(nil, sp.Procs[1], ctx, parent)
	if err != nil {
		return nil, err
	}

	if len(proc) != 1 {
		return nil, errors.New("expected 1 proc")
	}

	return proc[0], nil
}

// TestSource implements the Proc interface but outputs a fixed set of
// batches.  Used as the parent of a proc to be tested to control the
// batches fed into the proc under test.
type TestSource struct {
	records []zson.Batch
	idx int
}

func (t *TestSource) Pull() (zson.Batch, error) {
	if t.idx >= len(t.records) {
		return nil, nil
	}

	b := t.records[t.idx]
	t.idx += 1
	return b, nil
}

func (t *TestSource) Done() { }
func (t *TestSource) Parents() []Proc { return nil }

// Helper for testing an individual proc.
// To use this, first call NewTestProc() with all the records that should
// flow through the proc.  Then use the Expect* methods to verify the
// output of the proc.  Always end a test case with Finish() to ensure
// there weren't any unexpected records or warnings.
type ProcTest struct {
	ctx          Context
	compiledProc Proc
	eos          bool
}

func NewProcTest(code string, resolver *resolver.Table, inRecords []zson.Batch) (*ProcTest, error) {
	ctx := Context{
		Context:  context.Background(),
		Resolver: resolver,
		Warnings: make(chan string, 5),
	}

	src := TestSource{inRecords, 0}

	compiledProc, err := compileProc(code, &ctx, &src)
	if err != nil {
		return nil, err
	}

	return &ProcTest{ctx, compiledProc, false}, nil
}

func (t *ProcTest) Pull() (zson.Batch, error) {
	if t.eos {
		return nil, errors.New("called Pull() after EOS")
	}

	b, err := t.compiledProc.Pull()
	if b == nil && err == nil {
		t.eos = true
	}
	return b, err
}

func (t *ProcTest) ExpectEOS() error {
	b, err := t.Pull()
	if err != nil {
		return err
	}
	if b != nil {
		return errors.New("got more data after ExpectEOS()")
	}
	return nil
}

func (t *ProcTest) Expect(data zson.Batch) error {
	b, err := t.Pull()
	if err != nil {
		return err
	}
	if b == nil {
		return errors.New("got EOS while expecting more data")
	}

	n := data.Length()
	if b.Length() != n {
		return fmt.Errorf("expected %d records, got %d", n, b.Length())
	}

	for i := 0; i < n; i++ {
		received := b.Index(i)
		expected := data.Index(i)

		if received.Descriptor != expected.Descriptor {
			return fmt.Errorf("descriptor mismatch in record %d", i)
		}
		if bytes.Compare(received.Raw, expected.Raw) != 0 {
			return fmt.Errorf("mismatch in record %d: %s vs %s", i, received.Raw, expected.Raw)
		}
	}

	return nil
}

func (t *ProcTest) ExpectWarning(expected string) error {
	select {
	case warning := <-t.ctx.Warnings:
		if warning == expected {
			return nil
		} else {
			return fmt.Errorf("mismatch in warning: got \"%s\", expected \"%s\"", warning, expected)
		}
	default:
		return errors.New("did not receive expected warning")
	}
}

func (t *ProcTest) Finish() error {
	if !t.eos {
		return errors.New("finished test before EOS")
	}

	// XXX warnings channel is never closed, just ensure there's
	// nothing there...
	select {
	case warning := <-t.ctx.Warnings:
		return fmt.Errorf("got unexpected warning \"%s\"", warning)
	default:
		return nil
	}
}
