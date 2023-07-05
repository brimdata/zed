package zfmt

import (
	"fmt"
	"strings"

	"github.com/brimdata/zed/compiler/ast"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr/agg"
	"github.com/brimdata/zed/runtime/expr/function"
	"github.com/brimdata/zed/zson"
	"golang.org/x/exp/slices"
)

func AST(p ast.Seq) string {
	c := &canon{canonZed: canonZed{formatter{tab: 2}}, head: true, first: true}
	if scope, ok := p[0].(*ast.Scope); ok {
		c.scope(scope, false)
	} else {
		c.seq(p)
	}
	c.flush()
	return c.String()
}

type canon struct {
	canonZed
	head  bool
	first bool
}

func (c *canon) open(args ...interface{}) {
	c.formatter.open(args...)
}

func (c *canon) close() {
	c.formatter.close()
}

func (c *canon) assignments(assignments []ast.Assignment) {
	for k, a := range assignments {
		if k > 0 {
			c.write(",")
		}
		c.assignment(a)
	}
}

func (c *canon) assignment(a ast.Assignment) {
	if a.LHS != nil {
		c.expr(a.LHS, "")
		c.write(":=")
	}
	c.expr(a.RHS, "")
}

func (c *canon) defs(defs []ast.Def, separator string) {
	for k, d := range defs {
		if k > 0 {
			c.write(separator)
		}
		c.def(d)
	}
}

func (c *canon) def(d ast.Def) {
	c.write(d.Name)
	c.write("=")
	c.expr(d.Expr, "")
}

func (c *canon) exprs(exprs []ast.Expr) {
	for k, e := range exprs {
		if k > 0 {
			c.write(", ")
		}
		c.expr(e, "")
	}
}

func (c *canon) expr(e ast.Expr, parent string) {
	switch e := e.(type) {
	case nil:
		c.write("null")
	case *ast.Agg:
		c.write("%s(", e.Name)
		if e.Expr != nil {
			c.expr(e.Expr, "")
		}
		c.write(")")
		if e.Where != nil {
			c.write(" where ")
			c.expr(e.Where, "")
		}
	case *ast.Assignment:
		c.assignment(*e)
	case *astzed.Primitive:
		c.literal(*e)
	case *ast.ID:
		c.write(e.Name)
	case *ast.UnaryExpr:
		c.write(e.Op)
		c.expr(e.Operand, "not")
	case *ast.BinaryExpr:
		c.binary(e, parent)
	case *ast.Conditional:
		c.write("(")
		c.expr(e.Cond, "")
		c.write(") ? ")
		c.expr(e.Then, "")
		c.write(" : ")
		c.expr(e.Else, "")
	case *ast.Call:
		c.write("%s(", e.Name)
		c.exprs(e.Args)
		c.write(")")
	case *ast.Cast:
		c.expr(e.Type, "")
		c.write("(")
		c.expr(e.Expr, "")
		c.write(")")
	case *ast.SQLExpr:
		c.sql(e)
	case *astzed.TypeValue:
		c.write("<")
		c.typ(e.Value)
		c.write(">")
	case *ast.Regexp:
		c.write("/%s/", e.Pattern)
	case *ast.Glob:
		c.write(e.Pattern)
	case *ast.Grep:
		c.write("grep(")
		switch p := e.Pattern.(type) {
		case *ast.Regexp:
			c.write("/%s/", p.Pattern)
		case *ast.Glob:
			c.write(p.Pattern)
		case *ast.String:
			c.write(zson.QuotedString([]byte(p.Text)))
		default:
			c.open("(unknown grep pattern %T)", p)
		}
		if !isThis(e.Expr) {
			c.write(",")
			c.expr(e.Expr, "")
		}
		c.write(")")
	case *ast.Term:
		c.write(e.Text)
	case *ast.RecordExpr:
		c.write("{")
		for k, elem := range e.Elems {
			if k != 0 {
				c.write(",")
			}
			switch e := elem.(type) {
			case *ast.Field:
				c.write(zson.QuotedName(e.Name))
				c.write(":")
				c.expr(e.Value, "")
			case *ast.ID:
				c.write(zson.QuotedName(e.Name))
			case *ast.Spread:
				c.write("...")
				c.expr(e.Expr, "")
			default:
				c.write("zfmt: unknown record elem type: %T", e)
			}
		}
		c.write("}")
	case *ast.ArrayExpr:
		c.write("[")
		c.vectorElems(e.Elems)
		c.write("]")
	case *ast.SetExpr:
		c.write("|[")
		c.vectorElems(e.Elems)
		c.write("]|")
	case *ast.MapExpr:
		c.write("|{")
		for k, e := range e.Entries {
			if k != 0 {
				c.write(",")
			}
			c.expr(e.Key, "")
			c.write(":")
			c.expr(e.Value, "")
		}
		c.write("}|")
	case *ast.OverExpr:
		c.open("(")
		c.ret()
		c.write("over ")
		c.exprs(e.Exprs)
		if len(e.Locals) > 0 {
			c.write(" with ")
			c.defs(e.Locals, ", ")
		}
		c.seq(e.Body)
		c.close()
		c.ret()
		c.flush()
		c.write(")")
	default:
		c.write("(unknown expr %T)", e)
	}
}

func (c *canon) vectorElems(elems []ast.VectorElem) {
	for k, elem := range elems {
		if k > 0 {
			c.write(",")
		}
		switch elem := elem.(type) {
		case *ast.Spread:
			c.write("...")
			c.expr(elem.Expr, "")
		case *ast.VectorValue:
			c.expr(elem.Expr, "")
		}
	}
}

func (c *canon) binary(e *ast.BinaryExpr, parent string) {
	switch e.Op {
	case ".":
		if !isThis(e.LHS) {
			c.expr(e.LHS, "")
			c.write(".")
		}
		c.expr(e.RHS, "")
	case "[":
		if isThis(e.LHS) {
			c.write("this")
		} else {
			c.expr(e.LHS, "")
		}
		c.write("[")
		c.expr(e.RHS, "")
		c.write("]")
	case "and", "or", "in":
		parens := needsparens(parent, e.Op)
		c.maybewrite("(", parens)
		c.expr(e.LHS, e.Op)
		c.write(" %s ", e.Op)
		c.expr(e.RHS, e.Op)
		c.maybewrite(")", parens)
	default:
		parens := needsparens(parent, e.Op)
		c.maybewrite("(", parens)
		// do need parens calc
		c.expr(e.LHS, e.Op)
		c.write("%s", e.Op)
		c.expr(e.RHS, e.Op)
		c.maybewrite(")", parens)
	}
}

func needsparens(parent, op string) bool {
	return precedence(parent)-precedence(op) < 0
}

func precedence(op string) int {
	switch op {
	case "not":
		return 1
	case "^":
		return 2
	case "*", "/", "%":
		return 3
	case "+", "-":
		return 4
	case "<", "<=", ">", ">=", "==", "!=", "in":
		return 5
	case "and":
		return 6
	case "or":
		return 7
	default:
		return 100
	}
}

func (c *canon) sql(e *ast.SQLExpr) {
	if e.Select == nil {
		c.write(" SELECT *")
	} else {
		c.write(" SELECT")
		c.assignments(e.Select)
	}
}

func isThis(e ast.Expr) bool {
	if id, ok := e.(*ast.ID); ok {
		return id.Name == "this"
	}
	return false
}

func (c *canon) maybewrite(s string, do bool) {
	if do {
		c.write(s)
	}
}

func (c *canon) next() {
	if c.first {
		c.first = false
	} else {
		c.write("\n")
	}
	c.needRet = false
	c.writeTab()
	if c.head {
		c.head = false
	} else {
		c.write("| ")
	}
}

func (c *canon) decl(d ast.Decl) {
	switch d := d.(type) {
	case *ast.ConstDecl:
		c.write("const %s = ", d.Name)
		c.expr(d.Expr, "")
	case *ast.FuncDecl:
		c.write("func %s(", d.Name)
		for i := range d.Params {
			if i != 0 {
				c.write(", ")
			}
			c.write(d.Params[i])
		}
		c.open("): (")
		c.ret()
		c.expr(d.Expr, d.Name)
		c.close()
		c.ret()
		c.flush()
		c.write(")")
	case *ast.OpDecl:
		c.write("op %s(", d.Name)
		for k, p := range d.Params {
			if k > 0 {
				c.write(", ")
			}
			c.write(p)
		}
		c.open("): (")
		c.ret()
		c.flush()
		c.head = true
		c.seq(d.Body)
		c.close()
		c.ret()
		c.flush()
		c.write(")")
		c.head, c.first = true, true
	default:
		c.open("unknown decl: %T", d)
		c.close()
	}

}
func (c *canon) seq(seq ast.Seq) {
	for _, p := range seq {
		c.op(p)
	}
}

func (c *canon) op(p ast.Op) {
	switch p := p.(type) {
	case *ast.Scope:
		c.scope(p, true)
	case *ast.Parallel:
		c.next()
		c.open("fork (")
		for _, p := range p.Paths {
			c.ret()
			c.write("=>")
			c.open()
			c.head = true
			c.seq(p)
			c.close()
		}
		c.close()
		c.ret()
		c.flush()
		c.write(")")
		if p.MergeBy != nil {
			c.write(" merge-by ")
			c.fieldpath(p.MergeBy)
		}
		if p.MergeReverse {
			c.write(" rev")
		}
	case *ast.Switch:
		c.next()
		c.write("switch ")
		if p.Expr != nil {
			c.expr(p.Expr, "")
			c.write(" ")
		}
		c.open("(")
		for _, k := range p.Cases {
			c.ret()
			if k.Expr != nil {
				c.write("case ")
				c.expr(k.Expr, "")
			} else {
				c.write("default")
			}
			c.write(" =>")
			c.open()
			c.head = true
			c.seq(k.Path)
			c.close()
		}
		c.close()
		c.ret()
		c.flush()
		c.write(")")
	case *ast.From:
		//XXX cleanup for len(Trunks) = 1
		c.next()
		c.open("from (")
		for _, trunk := range p.Trunks {
			c.ret()
			c.source(trunk.Source)
			if trunk.Seq != nil {
				c.write(" =>")
				c.open()
				c.head = true
				c.seq(trunk.Seq)
				c.close()
			}
		}
		c.close()
		c.ret()
		c.flush()
		c.write(")")
	case *ast.SQLExpr:
		c.next()
		c.open("SELECT ")
		if p.Select == nil {
			c.write("*")
		} else {
			c.assignments(p.Select)
		}
		if p.From != nil {
			c.ret()
			c.write("FROM ")
			c.expr(p.From.Table, "")
			if p.From.Alias != nil {
				c.write(" AS ")
				c.expr(p.From.Alias, "")
			}
		}
		for _, join := range p.Joins {
			c.ret()
			switch join.Style {
			case "left":
				c.write("LEFT ")
			case "right":
				c.write("RIGHT ")
			}
			c.write("JOIN ")
			c.expr(join.Table, "")
			if join.Alias != nil {
				c.write(" AS ")
				c.expr(join.Alias, "")
			}
			c.write(" ON ")
			c.expr(join.LeftKey, "")
			c.write("=")
			c.expr(join.RightKey, "")
		}
		if p.Where != nil {
			c.ret()
			c.write("WHERE ")
			c.expr(p.Where, "")
		}
		if p.GroupBy != nil {
			c.ret()
			c.write("GROUP BY ")
			c.exprs(p.GroupBy)
		}
		if p.Having != nil {
			c.ret()
			c.write("HAVING ")
			c.expr(p.Having, "")
		}
		if p.OrderBy != nil {
			c.ret()
			c.write("ORDER BY ")
			c.exprs(p.OrderBy.Keys)
			c.write(" ")
			c.write(strings.ToUpper(p.OrderBy.Order.String()))
		}
		if p.Limit != 0 {
			c.ret()
			c.write("LIMIT %d", p.Limit)
		}
	case *ast.Summarize:
		c.next()
		c.open("summarize")
		c.ret()
		c.open()
		c.assignments(p.Aggs)
		if len(p.Keys) != 0 {
			c.write(" by ")
			c.assignments(p.Keys)
		}
		if p.Limit != 0 {
			c.write(" -with limit %d", p.Limit)
		}
		c.close()
		c.close()
	case *ast.Cut:
		c.next()
		c.write("cut ")
		c.assignments(p.Args)
	case *ast.Drop:
		c.next()
		c.write("drop ")
		c.exprs(p.Args)
	case *ast.Sort:
		c.next()
		c.write("sort")
		if p.Order == order.Desc {
			c.write(" -r")
		}
		if p.NullsFirst {
			c.write(" -nulls first")
		}
		if len(p.Args) > 0 {
			c.space()
			c.exprs(p.Args)
		}
	case *ast.Load:
		c.next()
		c.write("load %s", zson.QuotedString([]byte(p.Pool)))
		if p.Branch != "" {
			c.write("@%s", p.Branch)
		}
		if p.Author != "" {
			c.write(" author %s", p.Author)
		}
		if p.Message != "" {
			c.write(" message %s", p.Message)
		}
		if p.Meta != "" {
			c.write(" meta %s", p.Meta)
		}
	case *ast.Head:
		c.next()
		c.open("head ")
		c.expr(p.Count, "")
		c.close()
	case *ast.Tail:
		c.next()
		c.open("tail ")
		c.expr(p.Count, "")
		c.close()
	case *ast.Uniq:
		c.next()
		c.write("uniq")
		if p.Cflag {
			c.write(" -c")
		}
	case *ast.Pass:
		c.next()
		c.write("pass")
	case *ast.OpExpr:
		if agg := isAggFunc(p.Expr); agg != nil {
			c.op(agg)
			return
		}
		c.next()
		var which string
		e := p.Expr
		if IsSearch(e) {
			which = "search "
		} else if IsBool(e) {
			which = "where "
		} else if _, ok := e.(*ast.Call); !ok {
			which = "yield "
		}
		// Since we can't determine whether the expression is a func call or
		// an op call until the semantic pass, leave this ambiguous.
		// XXX (nibs) - I don't think we should be doing this kind introspection
		// here. This is why we have the semantic pass and canonical zed here
		// should reflect the ambiguous nature of the expression.
		if which != "" {
			c.open(which)
			defer c.close()
		}
		c.expr(e, "")
	case *ast.Search:
		c.next()
		c.open("search ")
		c.expr(p.Expr, "")
		c.close()
	case *ast.Where:
		c.next()
		c.open("where ")
		c.expr(p.Expr, "")
		c.close()
	case *ast.Top:
		c.next()
		c.write("top limit=%d flush=%t ", p.Limit, p.Flush)
		c.exprs(p.Args)
	case *ast.Put:
		c.next()
		c.write("put ")
		c.assignments(p.Args)
	case *ast.Rename:
		c.next()
		c.write("rename ")
		c.assignments(p.Args)
	case *ast.Fuse:
		c.next()
		c.write("fuse")
	case *ast.Join:
		c.next()
		c.write("join ")
		if p.RightInput != nil {
			c.open("(")
			c.head = true
			c.seq(p.RightInput)
			c.close()
			c.ret()
			c.flush()
			c.write(") ")
		}
		c.write("on ")
		c.expr(p.LeftKey, "")
		c.write("=")
		c.expr(p.RightKey, "")
		if p.Args != nil {
			c.write(" ")
			c.assignments(p.Args)
		}
	case *ast.OpAssignment:
		c.next()
		which := "put "
		if isAggAssignments(p.Assignments) {
			which = "summarize "
		}
		c.open(which)
		c.assignments(p.Assignments)
		c.close()
	//case *ast.SqlExpression:
	//	//XXX TBD
	//	c.open("sql")
	//	c.close()
	case *ast.Merge:
		c.next()
		c.write("merge ")
		c.expr(p.Expr, "")
	case *ast.Over:
		c.over(p)
	case *ast.Yield:
		c.next()
		c.write("yield ")
		c.exprs(p.Exprs)
	default:
		c.open("unknown proc: %T", p)
		c.close()
	}
}

func (c *canon) over(o *ast.Over) {
	c.next()
	c.write("over ")
	c.exprs(o.Exprs)
	if len(o.Locals) > 0 {
		c.write(" with ")
		c.defs(o.Locals, ", ")
	}
	if o.Body != nil {
		c.write(" => (")
		c.open()
		c.head = true
		c.seq(o.Body)
		c.close()
		c.ret()
		c.flush()
		c.write(")")
	}
}

func (c *canon) scope(s *ast.Scope, parens bool) {
	if parens {
		c.open("(")
		c.ret()
	}
	for _, d := range s.Decls {
		c.decl(d)
		c.ret()
	}
	//XXX functions?
	c.flush()
	c.seq(s.Body)
	if parens {
		c.close()
		c.ret()
		c.flush()
		c.write(")")
	}
}

func (c *canon) pool(p *ast.Pool) {
	//XXX TBD name, from, to, id etc
	var s string
	switch specPool := p.Spec.Pool.(type) {
	case nil:
		// This is a lake meta-query.
	case *ast.Glob:
		s = specPool.Pattern
	case *ast.Regexp:
		s = "/" + specPool.Pattern + "/"
	case *ast.String:
		s = zson.QuotedString([]byte(specPool.Text))
	default:
		s = fmt.Sprintf("(unknown pool type %T)", specPool)
	}
	if p.Spec.Commit != "" {
		s += "@" + p.Spec.Commit
	}
	if p.Spec.Meta != "" {
		s += ":" + p.Spec.Meta
	}
	if p.Spec.Tap {
		s += " tap"
	}
	c.write("pool %s", s)
}

func isAggFunc(e ast.Expr) *ast.Summarize {
	call, ok := e.(*ast.Call)
	if !ok {
		return nil
	}
	if _, err := agg.NewPattern(call.Name, true); err != nil {
		return nil
	}
	return &ast.Summarize{
		Kind: "Summarize",
		Aggs: []ast.Assignment{{
			Kind: "Assignment",
			RHS:  call,
		}},
	}
}

func IsBool(e ast.Expr) bool {
	switch e := e.(type) {
	case *astzed.Primitive:
		return e.Type == "bool"
	case *ast.UnaryExpr:
		return IsBool(e.Operand)
	case *ast.BinaryExpr:
		switch e.Op {
		case "and", "or", "in", "==", "!=", "<", "<=", ">", ">=":
			return true
		default:
			return false
		}
	case *ast.Conditional:
		return IsBool(e.Then) && IsBool(e.Else)
	case *ast.Call:
		return function.HasBoolResult(e.Name)
	case *ast.Cast:
		if typval, ok := e.Type.(*astzed.TypeValue); ok {
			if typ, ok := typval.Value.(*astzed.TypePrimitive); ok {
				return typ.Name == "bool"
			}
		}
		return false
	case *ast.Grep, *ast.Regexp, *ast.Glob:
		return true
	default:
		return false
	}
}

func isAggAssignments(assigns []ast.Assignment) bool {
	return !slices.ContainsFunc(assigns, func(a ast.Assignment) bool {
		return isAggFunc(a.RHS) == nil
	})
}

func IsSearch(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.Regexp, *ast.Glob, *ast.Term:
		return true
	case *ast.BinaryExpr:
		switch e.Op {
		case "and", "or":
			return IsSearch(e.LHS) || IsSearch(e.RHS)
		default:
			return false
		}
	case *ast.UnaryExpr:
		return IsSearch(e.Operand)
	default:
		return false
	}
}

func (c *canon) http(p *ast.HTTP) {
	//XXX TBD other stuff
	c.write("get %s", p.URL)
	if p.Format != "" {
		c.write(" format %s", p.Format)
	}
	if p.Method != "" {
		c.write(" method %s", zson.QuotedName(p.Method))
	}
	if p.Headers != nil {
		c.write(" headers ")
		c.expr(p.Headers, "")
	}
	if p.Body != "" {
		c.write(" body %s", zson.QuotedName(p.Body))
	}
}

func (c *canon) file(p *ast.File) {
	//XXX TBD other stuff
	c.write("file %s", p.Path)
	if p.Format != "" {
		c.write(" format %s", p.Format)
	}
}

func (c *canon) source(src ast.Source) {
	switch src := src.(type) {
	case *ast.Pool:
		c.pool(src)
	case *ast.HTTP:
		c.http(src)
	case *ast.File:
		c.file(src)
	default:
		c.write("unknown source type: %T", src)
	}
}
