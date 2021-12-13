package auth

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/pkg/charm"
)

var Logout = &charm.Spec{
	Name:  "logout",
	Usage: "auth logout",
	Short: "remove saved credentials for a Zed lake service",
	Long:  ``,
	New:   NewLogoutCommand,
}

type LogoutCommand struct {
	*Command
}

func NewLogoutCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &LogoutCommand{Command: parent.(*Command)}, nil
}

func (c *LogoutCommand) Run(args []string) error {
	_, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if _, err := c.Connection(); err != nil {
		// The Connection call here is to verify we're operating on a remote lake.
		return err
	}
	if len(args) > 0 {
		return errors.New("logout command takes no arguments")
	}
	creds, err := c.LoadCredentials()
	if err != nil {
		return fmt.Errorf("failed to load credentials file: %w", err)
	}
	creds.RemoveTokens(c.Lake)
	if err := c.SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials file: %w", err)
	}
	fmt.Printf("Logout successful, cleared credentials for %s\n", c.Lake)
	return nil
}
