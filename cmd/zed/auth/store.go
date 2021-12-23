package auth

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/brimdata/zed/api/client/auth0"
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
	_, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) > 0 {
		return errors.New("store command takes no arguments")
	}
	if err := os.MkdirAll(c.ConfigDir, 0700); err != nil {
		return err
	}
	if _, err := c.Connection(); err != nil {
		// The Connection call here is to verify we're operating on a remote lake.
		return err
	}
	store := c.AuthStore()
	tokens, err := store.LakeTokens(c.Lake)
	if err != nil {
		return fmt.Errorf("failed to load authentication store: %w", err)
	}
	if tokens == nil {
		tokens = &auth0.Tokens{}
	}
	tokens.Access = c.accessToken
	if err := store.SetLakeTokens(c.Lake, *tokens); err != nil {
		return fmt.Errorf("failed to update authentication: %w", err)
	}
	return nil
}
