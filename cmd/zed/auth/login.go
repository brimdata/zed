package auth

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client/auth0"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/pkg/browser"
)

const deviceCodeScope = "openid email profile offline_access"

var Login = &charm.Spec{
	Name:  "login",
	Usage: "auth login",
	Short: "log in to Zed lake service and save credentials",
	Long:  ``,
	New:   NewLoginCommand,
}

type LoginCommand struct {
	*Command
	launchBrowser bool
}

func NewLoginCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &LoginCommand{Command: parent.(*Command)}
	f.BoolVar(&c.launchBrowser, "launchbrowser", true, "automatically launch browser for verification")
	return c, nil
}

func (c *LoginCommand) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) > 0 {
		return errors.New("login command takes no arguments")
	}
	if err := os.MkdirAll(c.ConfigDir, 0700); err != nil {
		return err
	}
	conn, err := c.Connection()
	if err != nil {
		return err
	}
	method, err := conn.AuthMethod(ctx)
	if err != nil {
		return fmt.Errorf("failed to obtain authentication method: %w", err)
	}
	switch method.Kind {
	case api.AuthMethodAuth0:
	case api.AuthMethodNone:
		return fmt.Errorf("Zed service at %s does not support authentication", c.Lake)
	default:
		return fmt.Errorf("Zed service at %s requires unknown authentication method %s", c.Lake, method.Kind)
	}
	fmt.Println("method", method.Auth0.ClientID)
	fmt.Println("domain", method.Auth0.Domain)
	fmt.Println("audience", method.Auth0.Audience)
	auth0client, err := auth0.NewClient(*method.Auth0)
	if err != nil {
		return err
	}
	dcr, err := auth0client.GetDeviceCode(ctx, deviceCodeScope)
	if err != nil {
		return err
	}
	fmt.Println("Complete authentication at:", dcr.VerificationURIComplete)
	fmt.Println("User verification code:", dcr.UserCode)
	if c.launchBrowser {
		browser.OpenURL(dcr.VerificationURIComplete)
	}
	tokens, err := c.pollForTokens(ctx, auth0client, dcr)
	if err != nil {
		return err
	}
	if err := c.AuthStore().SetLakeTokens(c.Lake, tokens); err != nil {
		return fmt.Errorf("failed to save credentials file: %w", err)
	}
	fmt.Printf("Login successful, stored credentials for %s\n", c.Lake)
	return nil
}

func (c *LoginCommand) pollForTokens(ctx context.Context, client *auth0.Client, dcr auth0.DeviceCodeResponse) (auth0.Tokens, error) {
	delay := time.Duration(dcr.Interval) * time.Second
	if delay <= 0 {
		delay = time.Second
	}
	for {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return auth0.Tokens{}, ctx.Err()
		}
		tokens, err := client.GetAuthTokens(ctx, dcr)
		if err != nil {
			var aerr *auth0.APIError
			if errors.As(err, &aerr) && aerr.Kind == "authorization_pending" {
				continue
			}
		}
		return tokens, err
	}
}
