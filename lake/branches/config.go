package branches

import (
	"github.com/brimdata/zed/pkg/nano"
	"github.com/segmentio/ksuid"
)

// Config describes a branches configuration.
// swagger:model Branch
type Config struct {
	Ts     nano.Ts     `zed:"ts" json:"ts"`
	Name   string      `zed:"name" json:"name"`
	Commit ksuid.KSUID `zed:"commit" json:"commit"`

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
