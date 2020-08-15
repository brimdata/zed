package proc

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
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// SortMemMaxBytes specifies the maximum amount of memory that each sort proc
// will consume.
var SortMemMaxBytes = 128 * 1024 * 1024

type Sort struct {
	Base
	dir        int
	nullsFirst bool
	fields     []ast.FieldExpr

	fieldResolvers     []expr.FieldExprResolver
	once               sync.Once
	resultCh           chan Result
	compareFn          expr.CompareFn
	unseenFieldTracker *unseenFieldTracker
}

func CompileSortProc(c *Context, parent Proc, node *ast.SortProc) (*Sort, error) {
	fieldResolvers, err := expr.CompileFieldExprs(node.Fields)
	if err != nil {
		return nil, err
	}
	return &Sort{
		Base:               Base{Context: c, Parent: parent},
		dir:                node.SortDir,
		nullsFirst:         node.NullsFirst,
		fields:             node.Fields,
		fieldResolvers:     fieldResolvers,
		resultCh:           make(chan Result),
		unseenFieldTracker: newUnseenFieldTracker(node.Fields, fieldResolvers),
	}, nil
}

func (s *Sort) Pull() (zbuf.Batch, error) {
	s.once.Do(func() { go s.sortLoop() })
	if r, ok := <-s.resultCh; ok {
		return r.Batch, r.Err
	}
	return nil, s.Context.Err()
}

func (s *Sort) sortLoop() {
	defer close(s.resultCh)
	firstRunRecs, eof, err := s.recordsForOneRun()
	if err != nil || len(firstRunRecs) == 0 {
		s.sendResult(nil, err)
		return
	}
	s.setCompareFn(firstRunRecs[0])
	if eof {
		// Just one run so do an in-memory sort.
		s.warnAboutUnseenFields()
		expr.SortStable(firstRunRecs, s.compareFn)
		array := zbuf.NewArray(firstRunRecs)
		s.sendResult(array, nil)
		return
	}
	// Multiple runs so do an external merge sort.
	runManager, err := s.createRuns(firstRunRecs)
	if err != nil {
		s.sendResult(nil, err)
		return
	}
	defer runManager.cleanup()
	s.warnAboutUnseenFields()
	for s.Context.Err() == nil {
		// Reading from runManager merges the runs.
		b, err := zbuf.ReadBatch(runManager, 100)
		s.sendResult(b, err)
		if b == nil || err != nil {
			return
		}
	}
}

func (s *Sort) sendResult(b zbuf.Batch, err error) {
	select {
	case s.resultCh <- Result{Batch: b, Err: err}:
	case <-s.Context.Done():
	}
}

func (s *Sort) recordsForOneRun() ([]*zng.Record, bool, error) {
	var nbytes int
	var recs []*zng.Record
	for {
		batch, err := s.Get()
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
			s.unseenFieldTracker.update(rec)
			nbytes += len(rec.Raw)
			recs = append(recs, rec)
		}
		batch.Unref()
		if nbytes >= SortMemMaxBytes {
			return recs, false, nil
		}
	}
}

func (s *Sort) createRuns(firstRunRecs []*zng.Record) (*runManager, error) {
	rm, err := newRunManager(s.compareFn)
	if err != nil {
		return nil, err
	}
	if err := rm.createRun(firstRunRecs); err != nil {
		rm.cleanup()
		return nil, err
	}
	for {
		recs, eof, err := s.recordsForOneRun()
		if err != nil {
			rm.cleanup()
			return nil, err
		}
		if recs != nil {
			if err := rm.createRun(recs); err != nil {
				rm.cleanup()
				return nil, err
			}
		}
		if eof {
			return rm, nil
		}
	}
}

func (s *Sort) warnAboutUnseenFields() {
	for _, f := range s.unseenFieldTracker.unseen() {
		s.Warnings <- fmt.Sprintf("Sort field %s not present in input", expr.FieldExprToString(f))
	}
}

func (s *Sort) setCompareFn(r *zng.Record) {
	resolvers := s.fieldResolvers
	if resolvers == nil {
		fld := guessSortField(r)
		resolver := func(r *zng.Record) zng.Value {
			e, err := r.Access(fld)
			if err != nil {
				return zng.Value{}
			}
			return e
		}
		resolvers = []expr.FieldExprResolver{resolver}
	}
	nullsMax := !s.nullsFirst
	if s.dir < 0 {
		nullsMax = !nullsMax
	}
	compareFn := expr.NewCompareFn(nullsMax, resolvers...)
	if s.dir < 0 {
		s.compareFn = func(a, b *zng.Record) int { return compareFn(b, a) }
	} else {
		s.compareFn = compareFn
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

func guessSortField(rec *zng.Record) string {
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
type runManager struct {
	runs       []*runFile
	runIndices map[*runFile]int
	compareFn  expr.CompareFn
	tempDir    string
	zctx       *resolver.Context
}

// newRunManager creates a temporary directory.  Call cleanup to remove it.
func newRunManager(compareFn expr.CompareFn) (*runManager, error) {
	tempDir, err := ioutil.TempDir("", "zq-sort-")
	if err != nil {
		return nil, err
	}
	return &runManager{
		runIndices: make(map[*runFile]int),
		compareFn:  compareFn,
		tempDir:    tempDir,
		zctx:       resolver.NewContext(),
	}, nil
}

func (r *runManager) cleanup() {
	for _, run := range r.runs {
		run.closeAndRemove()
	}
	os.RemoveAll(r.tempDir)
}

// createRun creates a new run containing the records in recs.
func (r *runManager) createRun(recs []*zng.Record) error {
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
func (r *runManager) Peek() (*zng.Record, error) {
	if r.Len() == 0 {
		return nil, nil
	}
	return r.runs[0].nextRecord, nil
}

// Read returns the smallest record (per Less) from among the next records in
// each run.  It implements the merge operation for an external merge sort.
func (r *runManager) Read() (*zng.Record, error) {
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

func (r *runManager) Len() int { return len(r.runs) }

func (r *runManager) Less(i, j int) bool {
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

func (r *runManager) Swap(i, j int) { r.runs[i], r.runs[j] = r.runs[j], r.runs[i] }

func (r *runManager) Push(x interface{}) { r.runs = append(r.runs, x.(*runFile)) }

func (r *runManager) Pop() interface{} {
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
