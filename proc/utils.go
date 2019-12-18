package proc

// This file contains utilties for writing unit tests of procs
// XXX It should go in a test framework instead of dangling here.  TBD.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zsio/detector"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/zql"
	"github.com/stretchr/testify/require"
)

func CompileTestProc(code string, ctx *Context, parent Proc) (Proc, error) {
	// XXX If we use a newer version of pigeon, we can just compile
	// with "proc" as the terminal symbol.
	// But for now, we have to compile a complete flowgraph.
	// A simple proc isn't a valid query, so make up a program with
	// a wildcard filter and then pull out just the proc we care
	// about below.
	prog := fmt.Sprintf("* | %s", code)
	parsed, err := zql.Parse("", []byte(prog))
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
	idx     int
}

func NewTestSource(batches []zson.Batch) *TestSource {
	return &TestSource{records: batches}
}

func (t *TestSource) Pull() (zson.Batch, error) {
	if t.idx >= len(t.records) {
		return nil, nil
	}

	b := t.records[t.idx]
	t.idx += 1
	return b, nil
}

func (t *TestSource) Done()           {}
func (t *TestSource) Parents() []Proc { return nil }

// Helper for testing an individual proc.
// To use this, first call NewTestProc() with all the records that should
// flow through the proc.  Then use the Expect* methods to verify the
// output of the proc.  Always end a test case with Finish() to ensure
// there weren't any unexpected records or warnings.
type ProcTest struct {
	ctx          *Context
	compiledProc Proc
	eos          bool
}

func NewProcTest(proc Proc, ctx *Context) *ProcTest {
	return &ProcTest{ctx, proc, false}
}

func NewTestContext(res *resolver.Table) *Context {
	if res == nil {
		res = resolver.NewTable()
	}
	return &Context{
		Context:  context.Background(),
		Resolver: res,
		Warnings: make(chan string, 5),
	}
}

func NewProcTestFromSource(code string, resolver *resolver.Table, inRecords []zson.Batch) (*ProcTest, error) {
	ctx := NewTestContext(resolver)
	src := TestSource{inRecords, 0}
	compiledProc, err := CompileTestProc(code, ctx, &src)
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

func parse(resolver *resolver.Table, src string) (*zson.Array, error) {
	reader := detector.LookupReader("zson", strings.NewReader(src), resolver)
	records := make([]*zson.Record, 0)
	for {
		rec, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		records = append(records, rec)
	}

	return zson.NewArray(records, nano.MaxSpan), nil
}

// TestOneProcWithWarnings runs one test of a proc by compiling cmd as a proc,
// then Parsing zsonin, running the resulting records through the proc, and
// asserting that the output matches zsonout.  It also asserts that the
// given warning(s) are emitted.
func TestOneProcWithWarnings(t *testing.T, zsonin, zsonout string, warnings []string, cmd string) {
	resolver := resolver.NewTable()
	recsin, err := parse(resolver, zsonin)
	require.NoError(t, err)
	recsout, err := parse(resolver, zsonout)
	require.NoError(t, err)

	test, err := NewProcTestFromSource(cmd, resolver, []zson.Batch{recsin})
	require.NoError(t, err)

	var result zson.Batch
	if recsout.Length() > 0 {
		result, err = test.Pull()
		require.NoError(t, err)
	}
	require.NoError(t, test.ExpectEOS())
	for _, w := range warnings {
		require.NoError(t, test.ExpectWarning(w))
	}
	require.NoError(t, test.Finish())

	if recsout.Length() > 0 {
		require.Equal(t, recsout.Length(), result.Length(), "Got correct number of output records")
		for i := 0; i < result.Length(); i++ {
			r1 := recsout.Index(i)
			r2 := result.Index(i)
			// XXX could print something a lot pretter if/when this fails.
			require.Equalf(t, r2.Raw, r1.Raw, "Expected record %d to match", i)
		}
	}
}

// TestOneProc runs one test of a proc by compiling cmd as a proc, then
// Parsing zsonin, running the resulting records through the proc, and
// finally asserting that the output matches zsonout.
func TestOneProc(t *testing.T, zsonin, zsonout string, cmd string) {
	resolver := resolver.NewTable()
	recsin, err := parse(resolver, zsonin)
	require.NoError(t, err)
	recsout, err := parse(resolver, zsonout)
	require.NoError(t, err)

	test, err := NewProcTestFromSource(cmd, resolver, []zson.Batch{recsin})
	require.NoError(t, err)

	result, err := test.Pull()
	require.NoError(t, err)
	require.NoError(t, test.ExpectEOS())
	require.NoError(t, test.Finish())

	require.Equal(t, recsout.Length(), result.Length(), "Got correct number of output records")
	for i := 0; i < result.Length(); i++ {
		r1 := recsout.Index(i)
		r2 := result.Index(i)
		// XXX could print something a lot pretter if/when this fails.
		require.Equalf(t, r2.Raw, r1.Raw, "Expected record %d to match", i)
	}
}

// TestOneProcUnsorted is similar to TestOneProc, except ordering of
// records in the proc output is not important.  That is, the expected
// output records must all be present, but they may appear in any order.
func TestOneProcUnsorted(t *testing.T, zsonin, zsonout string, cmd string) {
	resolver := resolver.NewTable()
	recsin, err := parse(resolver, zsonin)
	require.NoError(t, err)
	recsout, err := parse(resolver, zsonout)
	require.NoError(t, err)

	test, err := NewProcTestFromSource(cmd, resolver, []zson.Batch{recsin})
	require.NoError(t, err)

	result, err := test.Pull()
	require.NoError(t, err)
	require.NoError(t, test.ExpectEOS())
	require.NoError(t, test.Finish())

	require.Equal(t, recsout.Length(), result.Length(), "Got correct number of output records")
	res := result.Records()
	sort.Slice(res, func(i, j int) bool { return bytes.Compare(res[i].Raw, res[j].Raw) > 0 })
	expected := recsout.Records()
	sort.Slice(expected, func(i, j int) bool { return bytes.Compare(expected[i].Raw, expected[j].Raw) > 0 })
	for i := 0; i < len(res); i++ {
		// XXX could print something a lot pretter if/when this fails.
		require.Equalf(t, expected[i].Raw, res[i].Raw, "Expected record %d to match", i)
	}
}
