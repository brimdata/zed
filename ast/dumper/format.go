package dumper

import (
	"fmt"
	"strings"

	"github.com/brimsec/zq/ast"
)

type formatter struct {
	strings.Builder
	indent  int
	tab     int
	needTab bool
}

func Format(p ast.Proc) string {
	f := &formatter{tab: 2}
	f.proc(p)
	return f.String()
}

func (f *formatter) writeTab() {
	for k := 0; k < f.indent; k++ {
		f.WriteByte(' ')
	}
	f.needTab = false
}

func (f *formatter) write(args ...interface{}) {
	if f.needTab {
		f.writeTab()
	}
	format := args[0].(string)
	f.WriteString(fmt.Sprintf(format, args[1:]...))
}

func (f *formatter) open(args ...interface{}) {
	f.write("(")
	if len(args) > 0 {
		f.write(args...)
	}
	f.indent += f.tab
}

func (f *formatter) close() {
	f.write(")")
	f.indent -= f.tab
}

func (f *formatter) ret() {
	f.write("\n")
	f.needTab = true
}

func (f *formatter) space() {
	f.write(" ")
}

func (f *formatter) assignments(assignments []ast.Assignment) {
	for _, a := range assignments {
		f.open("= ")
		f.expr(a.LHS)
		f.write(" ")
		f.expr(a.RHS)
		f.close()
		f.ret()
	}
}

func (f *formatter) exprs(exprs []ast.Expression) {
	for _, e := range exprs {
		f.expr(e)
		f.space()
	}
}

func (f *formatter) expr(e ast.Expression) {
	switch e := e.(type) {
	case nil:
		f.write("(nil)")
	case *ast.Reducer:
		f.open(e.Operator)
		f.expr(e.Expr)
		if e.Where != nil {
			f.open("where ")
			f.expr(e.Where)
			f.close()
		}
		f.close()
	case *ast.Empty:
		f.write("(empty)")
	case *ast.Literal:
		f.open("literal ")
		f.write(e.Type)
		f.write(" ")
		f.write(e.Value)
		f.close()
	case *ast.Identifier:
		// If the identifier refers to a named variable in scope (like "$"),
		// then return a Var expression referring to the pointer to the value.
		// Note that constants may be accessed this way too by entering their
		// names into the global (outermost) scope in the Scope entity.
		f.open("id ")
		f.write(e.Name)
		f.close()
	case *ast.RootRecord:
		f.write(".")
	case *ast.UnaryExpression:
		f.open(e.Operator)
		f.write(" ")
		f.expr(e.Operand)
		f.close()
	case *ast.SelectExpression:
		f.write("(select)")
	case *ast.BinaryExpression:
		f.open(e.Operator)
		f.space()
		f.expr(e.LHS)
		f.space()
		f.expr(e.RHS)
		f.close()
	case *ast.ConditionalExpression:
		f.open("cond ")
		f.expr(e.Condition)
		f.space()
		f.expr(e.Then)
		f.space()
		f.expr(e.Else)
		f.close()
	case *ast.FunctionCall:
		f.open(e.Function)
		f.space()
		f.exprs(e.Args)
		f.close()
	case *ast.CastExpression:
		f.open("cast %s ", e.Type)
		f.expr(e.Expr)
		f.close()
	default:
		f.open("(unknown expr %T)", e)
		f.close()
		f.ret()
	}
}

//		Op           string       `json:"op"`
//		Duration     Duration     `json:"duration"`
//		InputSortDir int          `json:"input_sort_dir,omitempty"`
//		Limit        int          `json:"limit"`
//		Keys         []Assignment `json:"keys"`
//		Reducers     []Assignment `json:"reducers"`
//		ConsumePart  bool         `json:"consume_part,omitempty"`
//		EmitPart     bool         `json:"emit_part,omitempty"`

func (f *formatter) procs(procs []ast.Proc) {
	for _, p := range procs {
		f.proc(p)
	}
}

func (f *formatter) proc(p ast.Proc) {
	switch p := p.(type) {
	case *ast.SequentialProc:
		f.open("seq")
		f.ret()
		f.procs(p.Procs)
		f.close()
		f.ret()
	case *ast.ParallelProc:
		f.open("par")
		f.ret()
		f.procs(p.Procs)
		f.close()
		f.ret()
	case *ast.GroupByProc:
		f.open("groupby dur=%d dir=%d limit=%d", p.Duration, p.InputSortDir, p.Limit)
		if p.ConsumePart {
			f.write(" partials-in")
		}
		if p.EmitPart {
			f.write(" partials-out")
		}
		f.open("keys")
		f.assignments(p.Keys)
		f.close()
		f.ret()
		f.open("aggs")
		f.assignments(p.Reducers)
		f.close()
		f.close()
		f.ret()
	case *ast.CutProc:
		f.open("cut")
		f.assignments(p.Fields)
		f.close()
		f.ret()
	case *ast.PickProc:
		f.open("pick")
		f.assignments(p.Fields)
		f.close()
		f.ret()
	case *ast.DropProc:
		f.open("drop")
		f.exprs(p.Fields)
		f.close()
		f.ret()
	case *ast.SortProc:
		f.open(fmt.Sprintf("sort dir=%d nf=%t ", p.SortDir, p.NullsFirst))
		f.exprs(p.Fields)
		f.close()
		f.ret()
	case *ast.HeadProc:
		f.open("head %d", p.Count)
		f.close()
		f.ret()
	case *ast.TailProc:
		f.open("tail %d", p.Count)
		f.close()
		f.ret()
	case *ast.UniqProc:
		f.open("uniq")
		if p.Cflag {
			f.write(" -c")
		}
		f.close()
		f.ret()
	case *ast.PassProc:
		f.open("pass")
		f.ret()
	case *ast.FilterProc:
		f.open("filter ")
		f.expr(p.Filter)
		f.close()
		f.ret()
	case *ast.TopProc:
		f.open("top limit=%d flush=%t ", p.Limit, p.Flush)
		f.ret()
		f.exprs(p.Fields)
		f.close()
		f.ret()
	case *ast.PutProc:
		f.open("put ")
		f.ret()
		f.assignments(p.Clauses)
		f.close()
		f.ret()
	case *ast.RenameProc:
		f.open("put")
		f.ret()
		f.assignments(p.Fields)
		f.close()
		f.ret()
	case *ast.FuseProc:
		f.open("fuse")
		f.close()
		f.ret()
	case *ast.FunctionCall:
		f.open(p.Function)
		f.space()
		f.exprs(p.Args)
		f.close()
		f.ret()
	case *ast.JoinProc:
		f.open("join on ")
		f.expr(p.LeftKey)
		f.write(" = ")
		f.expr(p.RightKey)
		f.ret()
		f.open("cut")
		f.assignments(p.Clauses)
		f.close()
		f.ret()
	case *ast.SqlExpression:
		//XXX TBD
		f.open("sql")
		f.close()
		f.ret()
	default:
		f.open("unknown proc: %T", p)
		f.close()
		f.ret()
	}
}
