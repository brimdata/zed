package dbmanage

import (
	"time"

	"github.com/superdb/super/db/tables"
)

const DefaultInterval = time.Minute

type Config struct {
	Interval *time.Duration `yaml:"interval"`
	Tables   []TableConfig  `yaml:"tables"`
}

func (c *Config) tableConfig(t *tables.Config) TableConfig {
	for _, pconf := range c.Tables {
		if t.Name != pconf.Table && t.ID.String() != pconf.Table {
			continue
		}
		if pconf.Branch == "" {
			pconf.Branch = "main"
		}
		return pconf
	}
	return TableConfig{
		Table:  t.Name,
		Branch: "main",
	}
}

func (c *Config) interval() time.Duration {
	if c.Interval == nil {
		return DefaultInterval
	}
	return *c.Interval
}

type TableConfig struct {
	Table  string `yaml:"table"`
	Branch string `yaml:"branch"`
}
