package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/brimsec/zq/zql"
	"github.com/mccanne/charm"
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
`,
	New: func(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
		return New(flags)
	},
}

func init() {
	Ast.Add(charm.Help)
}

type Command struct{}

func New(f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return Ast.Exec(c, []string{"help"})
	}
	z := strings.Join(args, " ")
	query, err := zql.ParseProc(z)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(query, "", "  ")
	if err != nil {
		fmt.Println("ast: couldn't format AST as json")
	} else {
		fmt.Println(string(b))
	}
	return nil
}
