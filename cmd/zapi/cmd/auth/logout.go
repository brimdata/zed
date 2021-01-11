package auth

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/mccanne/charm"
)

var Logout = &charm.Spec{
	Name:  "logout",
	Usage: "auth logout",
	Short: "remove saved credentials for zqd service",
	Long:  ``,
	New:   NewLogoutCommand,
}

type LogoutCommand struct {
	*Command
	LaunchBrowser bool
}

func NewLogoutCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &LogoutCommand{Command: parent.(*Command)}, nil
}

func (c *LogoutCommand) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	if len(args) > 0 {
		return errors.New("logout command takes no arguments")
	}

	cpath, err := cmd.UserStdCredentialsPath()
	if err != nil {
		return err
	}
	svccreds, err := cmd.LoadCredentials(cpath)
	if err != nil {
		return fmt.Errorf("failed to load credentials file: %w", err)
	}
	svccreds.RemoveTokens(c.Host)
	if err := cmd.SaveCredentials(cpath, svccreds); err != nil {
		return fmt.Errorf("failed to save credentials file: %w", err)
	}
	fmt.Printf("Logout successful, cleared credentials for %s in %s\n", c.Host, cpath)
	return nil
}
