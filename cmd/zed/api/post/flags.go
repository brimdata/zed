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
	apicmd.SpaceCreateFlags
	force     bool
	shaper    string
	shaperAST ast.Proc
	cmd       *apicmd.Command
}

func (f *postFlags) SetFlags(fs *flag.FlagSet) {
	fs.BoolVar(&f.force, "f", false, "create space if specified space does not exist")
	fs.StringVar(&f.shaper, "z", "", "Z shaper script to apply to data before storing")
	f.SpaceCreateFlags.SetFlags(fs)
}

func (f *postFlags) Init() error {
	c := f.cmd
	ctx, cleanup, err := c.Init(&f.SpaceCreateFlags)
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
	} else if c.Spacename == "" {
		return errors.New("if -f flag is enabled, a space name must specified")
	}
	sp, err := f.SpaceCreateFlags.Create(ctx, c.Connection(), c.Spacename)
	if err != nil {
		if err == client.ErrSpaceExists {
			// Fetch space ID.
			_, err = c.SpaceID(ctx)
		}
		return err
	}
	c.SetSpaceID(sp.ID)
	return nil
}
