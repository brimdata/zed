package info

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/brimsec/zq/pkg/colw"
	"github.com/mccanne/charm"
	"github.com/mccanne/charm/pkg/termwidth"
)

var Ls = &charm.Spec{
	Name:  "ls",
	Usage: "ls [-l] [glob1 glob2 ...]",
	Short: "list spaces or information about a space",
	Long: `The ls command lists the names and information about spaces known to the system.
When run with arguments, only the spaces that match the glob-style parameters are shown
much like the traditional unix ls command.  When used with "-l", each space specified
is listed with its detailed information as in the info command.`,
	New: NewLs,
}

type LsCommand struct {
	*cmd.Command
	lflag bool
}

func NewLs(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &LsCommand{Command: parent.(*cmd.Command)}
	f.BoolVar(&c.lflag, "l", false, "show detail information about each space listed")
	return c, nil
}

func fmtSelected(spaces []string, selected string) {
	for i, space := range spaces {
		if selected == space {
			spaces[i] = space + " \033[0;32m<-\033[0m" // make the selector green!
		}
	}
}

// Run lists all spaces in the current zqd host or if a parameter
// is provided (in glob style) lists the info about that space.
func (c *LsCommand) Run(args []string) error {
	api, err := c.API()
	if err != nil {
		return err
	}
	matches, err := cmd.SpaceGlob(api, args)
	if err != nil {
		return err
	}
	if matches == nil || len(matches) == 0 {
		return cmd.ErrNoMatch
	}
	if c.lflag {
		// print details about each space
		return printInfoList(api, matches)
	} else {
		// print listing laid out in columns like ls
		width := termwidth.Width()
		if !c.NoFancy {
			fmtSelected(matches, c.Spacename)
		}
		err := colw.Write(os.Stdout, matches, width, 3)
		if err == colw.ErrDoesNotFit {
			fmt.Println(strings.Join(matches, "\n"))
		} else if err != nil {
			return err
		}
	}
	return nil
}
