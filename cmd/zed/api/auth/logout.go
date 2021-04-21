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
	_, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) > 0 {
		return errors.New("logout command takes no arguments")
	}
	creds, err := c.LocalConfig.LoadCredentials()
	if err != nil {
		return fmt.Errorf("failed to load credentials file: %w", err)
	}
	creds.RemoveTokens(c.Host)
	if err := c.LocalConfig.SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials file: %w", err)
	}
	fmt.Printf("Logout successful, cleared credentials for %s\n", c.Host)
	return nil
}
