package postgresdb

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/go-pg/pg/v10"
)

type Config struct {
	pg.Options

	PasswordFile string
}

func (c *Config) Init() error {
	if c.Password != "" && c.PasswordFile != "" {
		return errors.New("db.postgres.password and db.postgres.passwordFile cannot both be set")
	}
	if c.PasswordFile != "" {
		b, err := ioutil.ReadFile(c.PasswordFile)
		if err != nil {
			return fmt.Errorf("error reading file specified in db.postgres.passwordFile: %w", err)
		}
		c.Password = string(b)
	}
	return nil
}

func (c *Config) SetFlagsWithPrefix(prefix string, fs *flag.FlagSet) {
	fs.Var(c, prefix+"url", "postgres url (postgres://[user[:password]@][netloc][:port]/[database])")
	fs.StringVar(&c.Addr, prefix+"addr", "localhost:5432", "postgres address")
	fs.StringVar(&c.User, prefix+"user", "", "postgres username")
	fs.StringVar(&c.Password, prefix+"password", "", "postgres password")
	fs.StringVar(&c.PasswordFile, prefix+"passwordFile", "", "path with file containing postgres password")
	fs.StringVar(&c.Database, prefix+"database", "", "postgres database name")
}

func (c *Config) SetFlags(fs *flag.FlagSet) {
	c.SetFlagsWithPrefix("", fs)
}

func (c *Config) Set(s string) error {
	opt, err := pg.ParseURL(s)
	if err == nil {
		c.Options = *opt
	}
	return err
}

func (c *Config) IsEmpty() bool {
	return c.Network == "" && c.Addr == "" && c.User == "" && c.Password == "" && c.Database == ""
}

// postgresql://[user[:password]@][netloc][:port]

func (c Config) String() string {
	if c.IsEmpty() {
		return ""
	}
	str := "postgres://"
	if c.User != "" {
		str += c.User
		if c.Password != "" {
			str += ":" + c.Password
		}
		str += "@"
	}
	str += c.Addr
	if c.Database != "" {
		str += "/" + c.Database
	}
	if c.TLSConfig == nil {
		params := url.Values{"sslmode": {"disable"}}
		str += "?" + params.Encode()
	}
	return str
}

// StringRedacted is the same as string except password and a username are
// redacted. This should be used in logs.
func (c Config) StringRedacted() string {
	c.Password = strings.Repeat("*", 5)
	c.User = strings.Repeat("*", 5)
	return c.String()
}
