package lakemanage

import (
	"time"

	"github.com/brimdata/zed/lake/pools"
)

const DefaultInterval = time.Minute

type Config struct {
	Interval *time.Duration `yaml:"interval"`
	Vectors  bool           `yaml:"vectors"`
	Pools    []PoolConfig   `yaml:"pools"`
}

func (c *Config) poolConfig(p *pools.Config) PoolConfig {
	for _, pconf := range c.Pools {
		if p.Name != pconf.Pool && p.ID.String() != pconf.Pool {
			continue
		}
		if pconf.Interval == nil {
			pconf.Interval = c.Interval
		}
		if pconf.Branch == "" {
			pconf.Branch = "main"
		}
		return pconf
	}
	return PoolConfig{
		Pool:     p.Name,
		Branch:   "main",
		Interval: c.Interval,
		Vectors:  c.Vectors,
	}
}

type PoolConfig struct {
	Pool     string         `yaml:"pool"`
	Branch   string         `yaml:"branch"`
	Interval *time.Duration `yaml:"interval"`
	Vectors  bool           `yaml:"vectors"`
}

func (c *PoolConfig) interval() time.Duration {
	if c.Interval == nil {
		return DefaultInterval
	}
	return *c.Interval
}
