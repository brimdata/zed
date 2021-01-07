package user

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/ppl/cmd/zident/root"
	"github.com/mccanne/charm"
	"gopkg.in/auth0.v5"
	"gopkg.in/auth0.v5/management"
)

var NewUser = &charm.Spec{
	Name:  "new",
	Usage: "zident user new",
	Short: "create new user",
	Long: `
Creates a new user in Auth0.

See 'zident help user' for required authentication.
`,
	New: NewNewCommand,
}

type NewCommand struct {
	*root.Command
	a0cfg         auth0ClientConfig
	email         string
	emailVerified bool
	tenantID      string
}

func NewNewCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &NewCommand{Command: parent.(*Command).Command}
	f.StringVar(&c.email, "email", "", "email address of new user")
	f.StringVar(&c.tenantID, "tenantid", "", "tenant_id to set instead of creating one")
	return c, nil
}

func (c *NewCommand) Run(_ []string) error {
	if err := c.a0cfg.FromEnv(); err != nil {
		return err
	}
	if c.email == "" {
		return errors.New("must specify an email address")
	}

	var tenantID string
	if c.tenantID != "" {
		if err := validTenantID(c.tenantID); err != nil {
			return err
		}
		tenantID = c.tenantID
	} else {
		tenantID = newTenantID()
	}

	userID := newUserID()

	m, err := management.New(c.a0cfg.domain.String(),
		management.WithClientCredentials(c.a0cfg.clientId, c.a0cfg.clientSecret))
	if err != nil {
		return err
	}
	u := &management.User{
		Connection: auth0.String(c.a0cfg.connection),
		Email:      auth0.String(c.email),
		// We don't send a verification email the users response to the password
		// reset email will verify their address.
		VerifyEmail: auth0.Bool(false),
		Password:    auth0.String(newPassword()),
		AppMetadata: map[string]interface{}{
			"brim_tenant_id": tenantID,
			"brim_user_id":   userID,
		},
	}
	if err := m.User.Create(u, management.Context(context.TODO())); err != nil {
		return err
	}
	// Ensure request password is cleared, in case this user struct is logged.
	u.Password = nil

	if err := triggerChangePassword(context.TODO(), c.a0cfg, c.email); err != nil {
		return err
	}

	fmt.Println("User created")
	return nil
}
