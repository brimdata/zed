package sort

import (
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
)

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
