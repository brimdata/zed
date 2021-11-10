package auth

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/pkg/charm"
)

var Store = &charm.Spec{
	Name:   "store",
	Usage:  "auth store",
	Short:  "store raw tokens",
	Long:   ``,
	New:    NewStore,
	Hidden: true,
}

type StoreCommand struct {
	*Command

	accessToken string
}

func NewStore(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &StoreCommand{Command: parent.(*Command)}
	f.StringVar(&c.accessToken, "access", "", "raw access token as string")
	return c, nil
}

func (c *StoreCommand) Run(args []string) error {
	_, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) > 0 {
		return errors.New("store command takes no arguments")
	}
	creds, err := c.lake.LoadCredentials()
	if err != nil {
		return fmt.Errorf("failed to load credentials file: %w", err)
	}
	tokens, _ := creds.ServiceTokens(c.lake.Host)
	tokens.Access = c.accessToken
	creds.AddTokens(c.lake.Host, tokens)
	if err := c.lake.SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials file: %w", err)
	}
	return nil
}
