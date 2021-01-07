package user

import (
	"context"
	"flag"
	"fmt"

	"github.com/brimsec/zq/ppl/cmd/zident/root"
	"github.com/mccanne/charm"
	"gopkg.in/auth0.v5/management"
)

var Search = &charm.Spec{
	Name:  "search",
	Usage: "zident user [global options] search",
	Short: "search for users",
	Long: `
Searches for users in Auth0.

See 'zident help user' for required authentication.
`,
	New: NewSearchCommand,
}

type SearchCommand struct {
	*root.Command
	a0cfg       auth0ClientConfig
	searchFlags searchFlags
}

func NewSearchCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &SearchCommand{Command: parent.(*Command).Command}
	c.searchFlags.SetFlags(f)
	return c, nil
}

func (c *SearchCommand) Run(_ []string) error {
	if err := c.a0cfg.FromEnv(); err != nil {
		return err
	}

	m, err := management.New(c.a0cfg.domain.String(),
		management.WithClientCredentials(c.a0cfg.clientId, c.a0cfg.clientSecret))
	if err != nil {
		return err
	}

	return streamUsers(context.TODO(), m, c.searchFlags, true, func(user *management.User) error {
		fmt.Printf("%+v\n", userOutput(user))
		return nil
	})
}
