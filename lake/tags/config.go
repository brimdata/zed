package tags

import (
	"github.com/brimdata/zed/pkg/nano"
	"github.com/segmentio/ksuid"
)

type Config struct {
	Ts     nano.Ts     `zed:"ts"`
	Name   string      `zed:"name"`
	Commit ksuid.KSUID `zed:"commit"`
}

func NewConfig(name string, commit ksuid.KSUID) *Config {
	return &Config{
		Ts:     nano.Now(),
		Name:   name,
		Commit: commit,
	}
}

func (c *Config) Key() string {
	return c.Name
}
