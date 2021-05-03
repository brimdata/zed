package post

import (
	"errors"
	"flag"

	"github.com/brimdata/zed/api/client"
	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/ast"
)

type postFlags struct {
	apicmd.PoolCreateFlags
	force     bool
	shaper    string
	shaperAST ast.Proc
	cmd       *apicmd.Command
}

func (f *postFlags) SetFlags(fs *flag.FlagSet) {
	fs.BoolVar(&f.force, "f", false, "create pool if specified pool does not exist")
	fs.StringVar(&f.shaper, "z", "", "Z shaper script to apply to data before storing")
	f.PoolCreateFlags.SetFlags(fs)
}

func (f *postFlags) Init() error {
	c := f.cmd
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if f.shaper != "" {
		ast, err := compiler.ParseProc(f.shaper)
		if err != nil {
			return err
		}
		f.shaperAST = ast
	}
	if !f.force {
		return nil
	} else if c.PoolName == "" {
		return errors.New("if -f flag is enabled, a pool must specified")
	}
	_, err = f.PoolCreateFlags.Create(ctx, c.Conn, c.PoolName)
	if err != nil && !errors.Is(err, client.ErrPoolExists) {
		return err
	}
	return nil
}
