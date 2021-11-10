package auth

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/api"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/cmd/zed/lake/auth/devauth"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/pkg/browser"
)

var Login = &charm.Spec{
	Name:  "login",
	Usage: "auth login",
	Short: "log in to Zed lake service and save credentials",
	Long:  ``,
	New:   NewLoginCommand,
}

type LoginCommand struct {
	*Command
	LaunchBrowser bool
}

func NewLoginCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &LoginCommand{Command: parent.(*Command)}
	f.BoolVar(&c.LaunchBrowser, "launchbrowser", true, "automatically launch browser for verification")
	return c, nil
}

func (c *LoginCommand) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) > 0 {
		return errors.New("login command takes no arguments")
	}
	conn, err := c.lake.Connection()
	if err != nil {
		return err
	}
	method, err := conn.AuthMethod(ctx)
	if err != nil {
		return fmt.Errorf("failed to obtain supported auth method: %w", err)
	}
	switch method.Kind {
	case api.AuthMethodAuth0:
	case api.AuthMethodNone:
		return fmt.Errorf("Zed lake service at %s does not support authentication", c.lake.Host)
	default:
		return fmt.Errorf("Zed lake service at %s requires unknown authentication method %s", c.lake.Host, method.Kind)
	}
	fmt.Println("method", method.Auth0.ClientID)
	fmt.Println("domain", method.Auth0.Domain)
	fmt.Println("audience", method.Auth0.Audience)
	dar, err := devauth.DeviceAuthorizationFlow(ctx, devauth.Config{
		Audience: method.Auth0.Audience,
		Domain:   method.Auth0.Domain,
		ClientID: method.Auth0.ClientID,
		Scope:    "openid profile email offline_access",
		UserPrompt: func(res devauth.UserCodePrompt) error {
			fmt.Println("Complete authentication at:", res.VerificationURL)
			fmt.Println("User verification code:", res.UserCode)
			if c.LaunchBrowser {
				browser.OpenURL(res.VerificationURL)
			}
			return nil
		},
	})
	if err != nil {
		return err
	}
	creds, err := c.lake.LoadCredentials()
	if err != nil {
		return fmt.Errorf("failed to load credentials file: %w", err)
	}
	creds.AddTokens(c.lake.Host, zedapi.ServiceTokens{
		Access:  dar.AccessToken,
		ID:      dar.IDToken,
		Refresh: dar.RefreshToken,
	})
	if err := c.lake.SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials file: %w", err)
	}
	fmt.Printf("Login successful, stored credentials for %s\n", c.lake.Host)
	return nil
}
