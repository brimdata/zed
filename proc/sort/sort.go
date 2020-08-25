package sort

import (
	"bufio"
	"container/heap"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// MemMaxBytes specifies the maximum amount of memory that each sort proc
// will consume.
var MemMaxBytes = 128 * 1024 * 1024

type Proc struct {
	ctx        *proc.Context
	parent     proc.Interface
	dir        int
	nullsFirst bool
	fields     []ast.FieldExpr

	fieldResolvers     []expr.FieldExprResolver
	once               sync.Once
	resultCh           chan proc.Result
	compareFn          expr.CompareFn
	unseenFieldTracker *unseenFieldTracker
}

func New(ctx *proc.Context, parent proc.Interface, node *ast.SortProc) (*Proc, error) {
	fieldResolvers, err := expr.CompileFieldExprs(node.Fields)
	if err != nil {
		return nil, err
	}
	return &Proc{
		ctx:                ctx,
		parent:             parent,
		dir:                node.SortDir,
		nullsFirst:         node.NullsFirst,
		fields:             node.Fields,
		fieldResolvers:     fieldResolvers,
		resultCh:           make(chan proc.Result),
		unseenFieldTracker: newUnseenFieldTracker(node.Fields, fieldResolvers),
	}, nil
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	p.once.Do(func() { go p.sortLoop() })
	if r, ok := <-p.resultCh; ok {
		return r.Batch, r.Err
	}
	return nil, p.ctx.Err()
}

func (p *Proc) Done() {
	p.parent.Done()
}

func (p *Proc) sortLoop() {
	defer close(p.resultCh)
	firstRunRecs, eof, err := p.recordsForOneRun()
	if err != nil || len(firstRunRecs) == 0 {
		p.sendResult(nil, err)
		return
	}
	p.setCompareFn(firstRunRecs[0])
	if eof {
		// Just one run so do an in-memory sort.
		p.warnAboutUnseenFields()
		expr.SortStable(firstRunRecs, p.compareFn)
		array := zbuf.NewArray(firstRunRecs)
		p.sendResult(array, nil)
		return
	}
	// Multiple runs so do an external merge sort.
	runManager, err := p.createRuns(firstRunRecs)
	if err != nil {
		p.sendResult(nil, err)
		return
	}
	defer runManager.Cleanup()
	p.warnAboutUnseenFields()
	for p.ctx.Err() == nil {
		// Reading from runManager merges the runs.
		b, err := zbuf.ReadBatch(runManager, 100)
		p.sendResult(b, err)
		if b == nil || err != nil {
			return
		}
	}
}

func (p *Proc) sendResult(b zbuf.Batch, err error) {
	select {
	case p.resultCh <- proc.Result{Batch: b, Err: err}:
	case <-p.ctx.Done():
	}
}

func (p *Proc) recordsForOneRun() ([]*zng.Record, bool, error) {
	var nbytes int
	var recs []*zng.Record
	for {
		batch, err := p.parent.Pull()
		if err != nil {
			return nil, false, err
		}
		if batch == nil {
			return recs, true, nil
		}
		l := batch.Length()
		for i := 0; i < l; i++ {
			rec := batch.Index(i)
			rec.CopyBody()
			p.unseenFieldTracker.update(rec)
			nbytes += len(rec.Raw)
			recs = append(recs, rec)
		}
		batch.Unref()
		if nbytes >= MemMaxBytes {
			return recs, false, nil
		}
	}
}

func (p *Proc) createRuns(firstRunRecs []*zng.Record) (*RunManager, error) {
	rm, err := NewRunManager(p.compareFn)
	if err != nil {
		return nil, err
	}
	if err := rm.CreateRun(firstRunRecs); err != nil {
		rm.Cleanup()
		return nil, err
	}
	for {
		recs, eof, err := p.recordsForOneRun()
		if err != nil {
			rm.Cleanup()
			return nil, err
		}
		if recs != nil {
			if err := rm.CreateRun(recs); err != nil {
				rm.Cleanup()
				return nil, err
			}
		}
		if eof {
			return rm, nil
		}
	}
}

func (p *Proc) warnAboutUnseenFields() {
	for _, f := range p.unseenFieldTracker.unseen() {
		p.ctx.Warnings <- fmt.Sprintf("Sort field %s not present in input", expr.FieldExprToString(f))
	}
}

func (p *Proc) setCompareFn(r *zng.Record) {
	resolvers := p.fieldResolvers
	if resolvers == nil {
		fld := GuessSortKey(r)
		resolver := func(r *zng.Record) zng.Value {
			e, err := r.Access(fld)
			if err != nil {
				return zng.Value{}
			}
			return e
		}
		resolvers = []expr.FieldExprResolver{resolver}
	}
	nullsMax := !p.nullsFirst
	if p.dir < 0 {
		nullsMax = !nullsMax
	}
	compareFn := expr.NewCompareFn(nullsMax, resolvers...)
	if p.dir < 0 {
		p.compareFn = func(a, b *zng.Record) int { return compareFn(b, a) }
	} else {
		p.compareFn = compareFn
	}
}

func firstOf(typ *zng.TypeRecord, which []zng.Type) string {
	for _, col := range typ.Columns {
		for _, t := range which {
			if zng.SameType(col.Type, t) {
				return col.Name
			}
		}
	}
	return ""
}

func firstNot(typ *zng.TypeRecord, which zng.Type) string {
	for _, col := range typ.Columns {
		if !zng.SameType(col.Type, which) {
			return col.Name
		}
	}
	return ""
}

var intTypes = []zng.Type{
	zng.TypeInt16,
	zng.TypeUint16,
	zng.TypeInt32,
	zng.TypeUint32,
	zng.TypeInt64,
	zng.TypeUint64,
}

func GuessSortKey(rec *zng.Record) string {
	typ := rec.Type
	if fld := firstOf(typ, intTypes); fld != "" {
		return fld
	}
	if fld := firstOf(typ, []zng.Type{zng.TypeFloat64}); fld != "" {
		return fld
	}
	if fld := firstNot(typ, zng.TypeTime); fld != "" {
		return fld
	}
	return "ts"
}

// runManager manages runs (files of sorted records).
type RunManager struct {
	runs       []*runFile
	runIndices map[*runFile]int
	compareFn  expr.CompareFn
	tempDir    string
	zctx       *resolver.Context
}

// NewRunManager creates a temporary directory.  Call Cleanup to remove it.
func NewRunManager(compareFn expr.CompareFn) (*RunManager, error) {
	tempDir, err := ioutil.TempDir("", "zq-sort-")
	if err != nil {
		return nil, err
	}
	return &RunManager{
		runIndices: make(map[*runFile]int),
		compareFn:  compareFn,
		tempDir:    tempDir,
		zctx:       resolver.NewContext(),
	}, nil
}

func (r *RunManager) Cleanup() {
	for _, run := range r.runs {
		run.closeAndRemove()
	}
	os.RemoveAll(r.tempDir)
}

// CreateRun creates a new run containing the records in recs.
func (r *RunManager) CreateRun(recs []*zng.Record) error {
	expr.SortStable(recs, r.compareFn)
	index := len(r.runIndices)
	filename := filepath.Join(r.tempDir, strconv.Itoa(index))
	runFile, err := newRunFile(filename, recs, r.zctx)
	if err != nil {
		return err
	}
	r.runIndices[runFile] = index
	heap.Push(r, runFile)
	return nil
}

// Peek returns the next record without advancing the reader.  The record stops
// being valid at the next read call.
func (r *RunManager) Peek() (*zng.Record, error) {
	if r.Len() == 0 {
		return nil, nil
	}
	return r.runs[0].nextRecord, nil
}

// Read returns the smallest record (per Less) from among the next records in
// each run.  It implements the merge operation for an external merge sort.
func (r *RunManager) Read() (*zng.Record, error) {
	for {
		if r.Len() == 0 {
			return nil, nil
		}
		rec, eof, err := r.runs[0].read()
		if err != nil {
			return nil, err
		}
		if eof {
			r.runs[0].closeAndRemove()
			heap.Pop(r)
		} else {
			heap.Fix(r, 0)
		}
		if rec != nil {
			return rec, nil
		}
	}
}

func (r *RunManager) Len() int { return len(r.runs) }

func (r *RunManager) Less(i, j int) bool {
	v := r.compareFn(r.runs[i].nextRecord, r.runs[j].nextRecord)
	switch {
	case v < 0:
		return true
	case v == 0:
		// Maintain stability.
		return r.runIndices[r.runs[i]] < r.runIndices[r.runs[j]]
	default:
		return false
	}
}

func (r *RunManager) Swap(i, j int) { r.runs[i], r.runs[j] = r.runs[j], r.runs[i] }

func (r *RunManager) Push(x interface{}) { r.runs = append(r.runs, x.(*runFile)) }

func (r *RunManager) Pop() interface{} {
	x := r.runs[len(r.runs)-1]
	r.runs = r.runs[:len(r.runs)-1]
	return x
}

// runFile represents a run as as a readable file containing a sorted sequence
// of records.
type runFile struct {
	file       *os.File
	nextRecord *zng.Record
	zr         zbuf.Reader
}

// newRunFile writes sorted to filename and returns a runFile that reads the
// file using zctx.
func newRunFile(filename string, sorted []*zng.Record, zctx *resolver.Context) (*runFile, error) {
	f, err := fs.Create(filename)
	if err != nil {
		return nil, err
	}
	r := &runFile{file: f}
	if err := writeZng(f, sorted); err != nil {
		r.closeAndRemove()
		return nil, err
	}
	if _, err := f.Seek(0, 0); err != nil {
		r.closeAndRemove()
		return nil, err
	}
	zr := zngio.NewReader(bufio.NewReader(f), zctx)
	rec, err := zr.Read()
	if err != nil {
		r.closeAndRemove()
		return nil, err
	}
	return &runFile{
		file:       f,
		nextRecord: rec,
		zr:         zr,
	}, nil
}

// closeAndRemove closes and removes the underlying file.
func (r *runFile) closeAndRemove() {
	r.file.Close()
	os.Remove(r.file.Name())
}

// read returns the next record along with a boolean that is true at EOF.
func (r *runFile) read() (*zng.Record, bool, error) {
	rec := r.nextRecord
	if rec != nil {
		rec = rec.Keep()
	}
	var err error
	r.nextRecord, err = r.zr.Read()
	eof := r.nextRecord == nil && err == nil
	return rec, eof, err
}

// writeZng writes records to w as a zng stream.
func writeZng(w io.Writer, records []*zng.Record) error {
	bw := bufio.NewWriter(w)
	zw := zngio.NewWriter(bw, zio.WriterFlags{})
	for _, rec := range records {
		if err := zw.Write(rec); err != nil {
			return err
		}
	}
	if err := zw.Flush(); err != nil {
		return nil
	}
	return bw.Flush()
}

type unseenFieldTracker struct {
	unseenFields map[ast.FieldExpr]expr.FieldExprResolver
	seenTypes    map[*zng.TypeRecord]bool
}

func newUnseenFieldTracker(fields []ast.FieldExpr, resolvers []expr.FieldExprResolver) *unseenFieldTracker {
	unseen := make(map[ast.FieldExpr]expr.FieldExprResolver)
	for i, r := range resolvers {
		unseen[fields[i]] = r
	}
	return &unseenFieldTracker{
		unseenFields: unseen,
		seenTypes:    make(map[*zng.TypeRecord]bool),
	}
}

func (u *unseenFieldTracker) update(rec *zng.Record) {
	if len(u.unseenFields) == 0 || u.seenTypes[rec.Type] {
		return
	}
	u.seenTypes[rec.Type] = true
	for field, fieldResolver := range u.unseenFields {
		if !fieldResolver(rec).IsNil() {
			delete(u.unseenFields, field)
		}
	}
}

func (u *unseenFieldTracker) unseen() []ast.FieldExpr {
	var fields []ast.FieldExpr
	for f := range u.unseenFields {
		fields = append(fields, f)
	}
	return fields
}
