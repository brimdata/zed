package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/ztest"
	"github.com/mccanne/charm"
)

var Zrun = &charm.Spec{
	Name:  "zrun",
	Usage: "zrun [ options ] yaml-file",
	Short: "tool for Z command sequences and tests",
	Long: `
`,
	New: func(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
		return New(flags)
	},
}

func init() {
	Zrun.Add(charm.Help)
}

type Command struct {
	debug  bool
	tmpdir string
}

func New(f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	f.BoolVar(&c.debug, "x", false, "put ./bin on the path first or running the ztest")
	f.StringVar(&c.tmpdir, "d", "./zrun", "working directory for zrun job (a temporary subdir is created within this directory)")
	//f.BoolVar(&c.js, "js", false, "run javascript version of peg parser")
	//f.BoolVar(&c.pigeon, "pigeon", true, "run pigeon version of peg parser")
	//f.BoolVar(&c.ast, "ast", false, "run pigeon version of peg parser and show marshaled ast")
	//f.BoolVar(&c.all, "all", false, "run all and show variants")
	//f.BoolVar(&c.optimize, "O", true, "run semantic optimizer on ast version")
	//f.BoolVar(&c.debug, "D", false, "display ast version as lisp-y debugger output")
	//f.BoolVar(&c.canon, "C", false, "display canonical version")
	//f.Var(&c.includes, "I", "source file containing Z query text (may be repeated)")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return Zrun.Exec(c, []string{"help"})
	}
	if len(args) != 1 {
		return errors.New("zrun takes a single file input")
	}
	path := args[0]
	zt, err := ztest.FromYAMLFile(path)
	if err != nil {
		return fmt.Errorf("zrun: %w", err)
	}
	shellPath := os.Getenv("PATH")
	if shellPath == "" {
		return errors.New("could not access $PATH from shell environment")
	}
	if c.debug {
		shellPath = fmt.Sprintf("./bin:%s", shellPath)
	}
	dirname := filepath.Dir(path)
	filename := filepath.Base(path)
	_, err = zt.RunTest(c.tmpdir, shellPath, dirname, filename)
	if err == nil {
		fmt.Printf("%s: success\n", path)
	}
	return err
}
