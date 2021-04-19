package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/ppl/zqd/auth"
)

var CLI = &charm.Spec{
	Name:  "gentoken",
	Usage: "gentoken",
	Short: "generate access token to test zqd auth",
	New:   New,
}

type Command struct {
	charm.Command

	domain         string
	expiration     time.Duration
	privateKeyFile string
	keyID          string
	tenantID       string
	userID         string
}

func New(_ charm.Command, fs *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	fs.StringVar(&c.domain, "domain", "", "domain to use to generate token issuer")
	fs.DurationVar(&c.expiration, "expiration", 4*time.Hour, "expiry duration for generated token")
	fs.StringVar(&c.privateKeyFile, "privatekeyfile", "", "path of file containing private key (required)")
	fs.StringVar(&c.keyID, "keyid", "", "key identifier")
	fs.StringVar(&c.tenantID, "tenantid", "", "tenant ID claim in generated token")
	fs.StringVar(&c.userID, "userid", "", "user ID claim in generated token")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) > 0 {
		return errors.New("gentoken takes no arguments")
	}
	if c.privateKeyFile == "" {
		return errors.New("must specify a keyfile")
	}
	token, err := auth.GenerateAccessToken(c.keyID, c.privateKeyFile, c.expiration, c.domain, auth.TenantID(c.tenantID), auth.UserID(c.userID))
	if err != nil {
		return fmt.Errorf("GenerateAccessToken failed: %w", err)
	}
	fmt.Println(token)
	return nil
}

func main() {
	err := CLI.ExecRoot(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
