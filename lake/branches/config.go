package branches

import (
	"github.com/brimdata/zed/pkg/nano"
	"github.com/segmentio/ksuid"
)

type Config struct {
	Ts     nano.Ts     `zng:"ts"`
	Name   string      `zng:"name"`
	Commit ksuid.KSUID `zng:"commit"`

	// audit info
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
