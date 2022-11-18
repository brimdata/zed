package lakemanage

import (
	"context"
	"fmt"
	"time"

	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lakeparse"
)

type Config struct {
	Compact CompactConfig `yaml:"compact"`
	Index   IndexConfig   `yaml:"index"`
}

type CompactConfig struct {
	Disabled      bool          `yaml:"disabled"`
	ColdThreshold time.Duration `yaml:"coldthresh"`
}

type IndexConfig struct {
	Disabled      bool          `yaml:"disabled"`
	ColdThreshold time.Duration `yaml:"coldthresh"`
	RuleNames     []string      `yaml:"rules"`

	rules []index.Rule
}

func (c *IndexConfig) getRules(ctx context.Context, lk api.Interface) error {
	if !c.Enabled() {
		return nil
	}
	rules, err := api.GetIndexRules(ctx, lk)
	if err != nil {
		return err
	}
loop:
	for _, id := range lakeparse.FormatIDs(c.RuleNames) {
		for _, r := range rules {
			switch id {
			case r.RuleName(), r.RuleID().String():
				c.rules = append(c.rules, r)
				continue loop
			}
		}
		return fmt.Errorf("could not find index rule %q", id)
	}
	return nil
}

func (c *IndexConfig) Enabled() bool {
	return !c.Disabled && len(c.RuleNames) > 0
}
