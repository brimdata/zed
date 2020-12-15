package postgres

import (
	"net/url"

	"github.com/go-pg/pg/v10"
)

type Config struct {
	pg.Options
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
