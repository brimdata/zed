package auth

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli"
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

func (c *VerifyCommand) loadServiceCredentials(serviceURL string) (cli.ServiceTokens, error) {
	svccreds, err := c.LoadCredentials()
	if err != nil {
		return cli.ServiceTokens{}, fmt.Errorf("failed to load credentials file: %w", err)
	}
	creds, ok := svccreds.ServiceTokens(serviceURL)
	if !ok {
		return cli.ServiceTokens{}, fmt.Errorf("no stored credentials for %s", serviceURL)
	}
	return creds, nil
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
	conn, err := c.Connection()
	if err != nil {
		return err
	}
	creds, err := c.loadServiceCredentials(c.Lake)
	if err != nil {
		return err
	}
	conn.SetAuthToken(creds.Access)
	res, err := conn.AuthIdentity(ctx)
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
