package user

import (
	"context"
	"flag"
	"fmt"

	"github.com/brimsec/zq/ppl/cmd/zident/root"
	"github.com/mccanne/charm"
	"gopkg.in/auth0.v5/management"
)

var ResetPassword = &charm.Spec{
	Name:  "resetpassword",
	Usage: "zident user [global options] resetpassword",
	Short: "send change password email for user",
	Long: `
Triggers a password reset email for a user.

See 'zident help user' for required authentication.
`,
	New: NewResetPasswordCommand,
}

type ResetPasswordCommand struct {
	*root.Command
	a0cfg       auth0ClientConfig
	searchFlags searchFlags
}

func NewResetPasswordCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &ResetPasswordCommand{Command: parent.(*Command).Command}
	c.searchFlags.SetFlags(f)
	return c, nil
}

func (c *ResetPasswordCommand) Run(_ []string) error {
	if err := c.a0cfg.FromEnv(); err != nil {
		return err
	}

	m, err := management.New(c.a0cfg.domain.String(),
		management.WithClientCredentials(c.a0cfg.clientId, c.a0cfg.clientSecret))
	if err != nil {
		return err
	}

	u, err := findUser(context.TODO(), m, c.searchFlags)
	if err != nil {
		return err
	}
	user := userOutput(u)

	if err := triggerChangePassword(context.TODO(), c.a0cfg, user.Email); err != nil {
		return err
	}

	fmt.Println("Password change email triggered")
	return nil
}
