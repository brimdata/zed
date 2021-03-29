package auth

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/pkg/charm"
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
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	if len(args) > 0 {
		return errors.New("verify command takes no arguments")
	}
	conn := c.Connection()

	creds, err := c.LocalConfig.LoadCredentials()
	if err != nil {
		return err
	}
	tokens, ok := creds.ServiceTokens(c.Host)
	if !ok {
		return fmt.Errorf("no stored credentials for %v", c.Host)
	}
	conn.SetAuthToken(tokens.Access)

	res, err := conn.AuthIdentity(c.Context())
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
