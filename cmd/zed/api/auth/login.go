package auth

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zq/api"
	zedapi "github.com/brimdata/zq/cmd/zed/api"
	"github.com/brimdata/zq/cmd/zed/api/auth/devauth"
	"github.com/brimdata/zq/pkg/charm"
	"github.com/pkg/browser"
)

var Login = &charm.Spec{
	Name:  "login",
	Usage: "auth login",
	Short: "login and save credentials for zqd service",
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
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	if len(args) > 0 {
		return errors.New("login command takes no arguments")
	}
	conn := c.Connection()

	method, err := conn.AuthMethod(c.Context())
	if err != nil {
		return fmt.Errorf("failed to obtain supported auth method: %w", err)
	}
	switch method.Kind {
	case api.AuthMethodAuth0:
	case api.AuthMethodNone:
		return fmt.Errorf("zqd service at %v does not support authentication", c.Host)
	default:
		return fmt.Errorf("zqd service at %v supports unhandled authentication method: %v", c.Host, method.Kind)
	}

	dar, err := devauth.DeviceAuthorizationFlow(c.Context(), devauth.Config{
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

	creds, err := c.LocalConfig.LoadCredentials()
	if err != nil {
		return fmt.Errorf("failed to load credentials file: %w", err)
	}
	creds.AddTokens(c.Host, zedapi.ServiceTokens{
		Access:  dar.AccessToken,
		ID:      dar.IDToken,
		Refresh: dar.RefreshToken,
	})
	if err := c.LocalConfig.SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials file: %w", err)
	}
	fmt.Printf("Login successful, stored credentials for %s\n", c.Host)
	return nil
}
