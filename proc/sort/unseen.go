package sort

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
)

type unseenFieldTracker struct {
	unseenFields map[expr.Evaluator]struct{}
	seenTypes    map[*zed.TypeRecord]bool
}

func newUnseenFieldTracker(fields []expr.Evaluator) *unseenFieldTracker {
	unseen := make(map[expr.Evaluator]struct{})
	// We start with the unseen map full of all the fields and take
	// them out for each record type we encounter.
	for _, f := range fields {
		unseen[f] = struct{}{}
	}
	return &unseenFieldTracker{
		unseenFields: unseen,
		seenTypes:    make(map[*zed.TypeRecord]bool),
	}
}

func (u *unseenFieldTracker) update(ectx expr.Context, rec *zed.Value) {
	recType := zed.TypeRecordOf(rec.Type)
	if len(u.unseenFields) == 0 || u.seenTypes[recType] {
		// We've already seen every field or seen this type.
		return
	}
	u.seenTypes[recType] = true
	for field := range u.unseenFields {
		val := field.Eval(ectx, rec)
		if !val.IsMissing() {
			delete(u.unseenFields, field)
		}
	}
}

func (u *unseenFieldTracker) unseen() []expr.Evaluator {
	if len(u.seenTypes) == 0 {
		// We haven't seen any input.
		return nil
	}
	var fields []expr.Evaluator
	for f := range u.unseenFields {
		fields = append(fields, f)
	}
	return fields
}
