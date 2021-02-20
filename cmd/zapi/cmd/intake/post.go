package intake

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/mccanne/charm"
)

var Post = &charm.Spec{
	Name:  "post",
	Usage: "intake post",
	Short: "post data to intake",
	Long:  ``,
	New:   NewPost,
}

type PostCommand struct {
	*Command
}

func NewPost(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &PostCommand{Command: parent.(*Command)}
	return c, nil
}

func (c *PostCommand) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	if len(args) != 2 {
		return fmt.Errorf("expected arguments: <intake name or id> <file or '-'>")
	}
	intake, err := c.lookupIntake(args[0])
	if err != nil {
		return err
	}
	var rc io.ReadCloser
	if args[1] == "-" {
		rc = ioutil.NopCloser(os.Stdin)
	} else {
		var uri iosrc.URI
		uri, err = iosrc.ParseURI(args[1])
		if err != nil {
			return err
		}
		rc, err = iosrc.NewReader(c.Context(), uri)
		if err != nil {
			return err
		}
	}
	defer rc.Close()
	return c.Connection().IntakePostData(c.Context(), intake.ID, rc)
}
