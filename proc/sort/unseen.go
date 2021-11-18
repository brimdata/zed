package sort

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
)

type unseenFieldTracker struct {
	unseenFields map[expr.Evaluator]struct{}
	seenTypes    map[zed.Type]bool
}

func newUnseenFieldTracker(fields []expr.Evaluator) *unseenFieldTracker {
	unseen := make(map[expr.Evaluator]struct{})
	// We start out withe unseen map full of all the fields and take
	// them out for each record type we encounter.
	for _, f := range fields {
		unseen[f] = struct{}{}
	}
	return &unseenFieldTracker{
		unseenFields: unseen,
		seenTypes:    make(map[zed.Type]bool),
	}
}

func (u *unseenFieldTracker) update(zv *zed.Value) {
	typ := zv.Type
	if len(u.unseenFields) == 0 || u.seenTypes[typ] {
		// Either have seen this type or nothing to unsee anymore.
		return
	}
	u.seenTypes[typ] = true
	for field := range u.unseenFields {
		v, _ := field.Eval(zv)
		if !v.IsNil() {
			delete(u.unseenFields, field)
		}
	}
}

func (u *unseenFieldTracker) unseen() []expr.Evaluator {
	var fields []expr.Evaluator
	for f := range u.unseenFields {
		fields = append(fields, f)
	}
	return fields
}
