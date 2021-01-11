package auth

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/mccanne/charm"
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

func LoadServiceCredentials(serviceURL string) (cmd.ServiceTokens, error) {
	cpath, err := cmd.UserStdCredentialsPath()
	if err != nil {
		return cmd.ServiceTokens{}, err
	}
	svccreds, err := cmd.LoadCredentials(cpath)
	if err != nil {
		return cmd.ServiceTokens{}, fmt.Errorf("failed to load credentials file: %w", err)
	}
	creds, ok := svccreds.ServiceTokens(serviceURL)
	if !ok {
		return cmd.ServiceTokens{}, fmt.Errorf("no stored credentials for %v", serviceURL)
	}
	return creds, nil
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

	creds, err := LoadServiceCredentials(c.Host)
	if err != nil {
		return err
	}
	conn.SetAuthToken(creds.Access)

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
