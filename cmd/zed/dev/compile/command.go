package compile

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/brimdata/zed/cli/clierrors"
	"github.com/brimdata/zed/cmd/zed/dev"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/zfmt"
)

var Cmd = &charm.Spec{
	Name:  "compile",
	Usage: "compile [ options ] zed",
	Short: "inspect Zed language abstract syntax trees and compiler stages",
	Long: `
The "zed dev compile" command parses a Zed expression and prints the resulting abstract syntax
tree as JSON object to standard output.  If you have installed the
shortcuts, "zc" is a short cut for the "zed dev compile" command.

"zed dev compile" is a tool for dev and test,
and is also useful to advanced users for understanding how Zed syntax is
translated into an analytics requests sent to the "zed server" search endpoint.

The -O flag is handy for turning on and off the compiler, which lets you see
how the parsed AST is transformed into a runtime object comprised of the
Zed kernel operators.
`,
	New: New,
}

func init() {
	dev.Cmd.Add(Cmd)
}

type Command struct {
	*root.Command
	pigeon   bool
	proc     bool
	canon    bool
	semantic bool
	optimize bool
	parallel int
	n        int
	includes includes
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.BoolVar(&c.pigeon, "pigeon", false, "run pigeon version of peg parser")
	f.BoolVar(&c.proc, "proc", false, "run pigeon version of peg parser and marshal into ast.Op")
	f.BoolVar(&c.semantic, "s", false, "display semantically analyzed AST (implies -proc)")
	f.BoolVar(&c.optimize, "O", false, "display optimized, non-filter AST (implies -proc)")
	f.IntVar(&c.parallel, "P", 0, "display parallelized AST (implies -proc)")
	f.BoolVar(&c.canon, "C", false, "display AST in Zed canonical format (implies -proc)")
	f.Var(&c.includes, "I", "source file containing Zed query text (may be repeated)")
	return c, nil
}

type includes []string

func (i includes) String() string {
	return strings.Join(i, ",")
}

func (i *includes) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 && len(c.includes) == 0 {
		return charm.NeedHelp
	}
	c.n = 0
	if c.pigeon {
		c.n++
	}
	if c.proc {
		c.n++
	}
	if c.semantic {
		c.n++
	}
	if c.optimize {
		c.n++
	}
	if c.parallel > 0 {
		c.n++
	}
	if c.n == 0 {
		if c.canon {
			c.proc = true
		} else {
			c.pigeon = true
		}
	}
	set, err := parser.ConcatSource(c.includes, strings.Join(args, " "))
	if err != nil {
		return err
	}
	var lk *lake.Root
	if c.semantic || c.optimize || c.parallel != 0 {
		lakeAPI, err := c.LakeFlags.Open(ctx)
		if err == nil {
			lk = lakeAPI.Root()
		}
	}
	return c.parse(set, lk)
}

func (c *Command) header(msg string) {
	if c.n > 1 {
		bars := strings.Repeat("=", len(msg))
		fmt.Printf("/%s\\\n", bars)
		fmt.Printf("|%s|\n", msg)
		fmt.Printf("\\%s/\n", bars)
	}
}

func (c *Command) parse(set *parser.SourceSet, lk *lake.Root) error {
	if c.pigeon {
		s, err := parsePigeon(set)
		if err != nil {
			return err
		}
		c.header("pigeon")
		fmt.Println(s)
	}
	if c.proc {
		seq, err := compiler.Parse(string(set.Contents))
		if err != nil {
			return err
		}
		c.header("proc")
		c.writeAST(seq)
	}
	if c.semantic {
		runtime, err := c.compile(set, lk)
		if err != nil {
			return err
		}
		c.header("semantic")
		c.writeDAG(runtime.Entry())
	}
	if c.optimize {
		runtime, err := c.compile(set, lk)
		if err != nil {
			return err
		}
		if err := runtime.Optimize(); err != nil {
			return err
		}
		c.header("optimized")
		c.writeDAG(runtime.Entry())
	}
	if c.parallel > 0 {
		runtime, err := c.compile(set, lk)
		if err != nil {
			return err
		}
		if err := runtime.Optimize(); err != nil {
			return err
		}
		if err := runtime.Parallelize(c.parallel); err != nil {
			return err
		}
		c.header("parallelized")
		c.writeDAG(runtime.Entry())
	}
	return nil
}

func (c *Command) writeAST(seq ast.Seq) {
	s, err := astFmt(seq, c.canon)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(s)
	}
}

func (c *Command) writeDAG(seq dag.Seq) {
	s, err := dagFmt(seq, c.canon)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(s)
	}
}

func (c *Command) compile(set *parser.SourceSet, lk *lake.Root) (*compiler.Job, error) {
	p, err := compiler.Parse(string(set.Contents))
	if err != nil {
		return nil, clierrors.Format(set, err)
	}
	job, err := compiler.NewJob(runtime.DefaultContext(), p, data.NewSource(nil, lk), nil)
	if err != nil {
		return nil, clierrors.Format(set, err)
	}
	return job, nil
}

func normalize(b []byte) (string, error) {
	var v interface{}
	err := json.Unmarshal(b, &v)
	if err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(v, "", "    ")
	return string(out), err
}

func astFmt(seq ast.Seq, canon bool) (string, error) {
	if canon {
		return zfmt.AST(seq), nil
	}
	seqJSON, err := json.Marshal(seq)
	if err != nil {
		return "", err
	}
	return normalize(seqJSON)
}

func dagFmt(seq dag.Seq, canon bool) (string, error) {
	if canon {
		return zfmt.DAG(seq), nil
	}
	dagJSON, err := json.Marshal(seq)
	if err != nil {
		return "", err
	}
	return normalize(dagJSON)
}

func parsePigeon(set *parser.SourceSet) (string, error) {
	ast, err := parser.ParseZed(string(set.Contents))
	if err != nil {
		return "", clierrors.Format(set, err)
	}
	goPEGJSON, err := json.Marshal(ast)
	if err != nil {
		return "", errors.New("go peg parser returned bad value for: " + string(set.Contents))
	}
	return normalize(goPEGJSON)
}
