package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"

	"github.com/brimsec/zq/zql"
	"github.com/mccanne/charm"
	"github.com/peterh/liner"
)

var Ast = &charm.Spec{
	Name:  "ast",
	Usage: "ast [ options ] zql",
	Short: "tool for inspecting zql abtract-syntax trees",
	Long: `
The ast command parses a zql expression and prints the resulting abstract-syntax
tree as JSON object to standard output.  This serves a tool for dev and test
but could also be used by power users trying to understand how zql syntax is
translated into the analytics requests that is sent to the zqd search endpoint.

By default, it runs the built-in PEG parser built into this go binary.
If you specify -js, it will try to run a javascript version of the parser
by execing node in the currrent directory running the javascript in ./zql/run.js.
`,
	New: func(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
		return New(flags)
	},
}

func init() {
	Ast.Add(charm.Help)
}

type Command struct {
	repl bool
	js   bool
	both bool
}

func New(f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	f.BoolVar(&c.repl, "repl", false, "enter repl")
	f.BoolVar(&c.js, "js", false, "run javascript version of peg parser")
	f.BoolVar(&c.both, "both", false, "run both javascript and go version of peg parser")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if c.repl {
		c.interactive()
		return nil
	}
	if len(args) == 0 {
		return Ast.Exec(c, []string{"help"})
	}
	result, err := c.parse(strings.Join(args, " "))
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}

func (c *Command) parse(z string) (string, error) {
	var result string
	if c.js || c.both {
		b, err := runNode("", z)
		result = string(b)
		if !c.both {
			return result, err
		}
	}
	query, err := zql.ParseProc(z)
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(query, "", "  ")
	if err != nil {
		return "", errors.New("couldn't format AST as json")
	}
	return result + string(b), nil
}

func (c *Command) interactive() {
	rl := liner.NewLiner()
	defer rl.Close()
	for {
		line, err := rl.Prompt("> ")
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		rl.AppendHistory(line)
		result, err := c.parse(line)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(result)
	}
}

func runNode(dir, line string) ([]byte, error) {
	cmd := exec.Command("node", "./zql/run.js", "-e", "start")
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdin = strings.NewReader(line)
	return cmd.Output()
}
