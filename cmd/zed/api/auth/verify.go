package auth

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/pkg/charm"
)

var Verify = &charm.Spec{
	Name:  "verify",
	Usage: "auth verify",
	Short: "verify authentication credentials",
	Long:  ``,
	New:   NewVerify,
}

type VerifyCommand struct {
	*Command
}

func NewVerify(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &VerifyCommand{Command: parent.(*Command)}, nil
}

func (c *VerifyCommand) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) > 0 {
		return errors.New("verify command takes no arguments")
	}
	creds, err := c.LocalConfig.LoadCredentials()
	if err != nil {
		return err
	}
	tokens, ok := creds.ServiceTokens(c.Host)
	if !ok {
		return fmt.Errorf("no stored credentials for %v", c.Host)
	}
	c.Conn.SetAuthToken(tokens.Access)

	res, err := c.Conn.AuthIdentity(ctx)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
