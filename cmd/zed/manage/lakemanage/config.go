package lakemanage

import (
	"fmt"
	"sort"
	"time"

	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/lakeparse"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slices"
)

const (
	defaultCompactColdThresh = 5 * time.Minute
	defaultIndexColdThresh   = 10 * time.Minute
)

type Config struct {
	Compact CompactConfig `yaml:"compact"`
	Index   IndexConfig   `yaml:"index"`
	Pools   []PoolConfig  `yaml:"pools"`
}

func (c *Config) poolConfig(p *pools.Config, indexes []index.Rule) (string, CompactConfig, IndexConfig, error) {
	var branch string
	compact := c.Compact
	index := c.Index.Clone()
	for _, pc := range c.Pools {
		if p.Name != pc.Pool && p.ID.String() != pc.Pool {
			continue
		}
		branch = pc.Branch
		if pc.Compact != nil {
			compact = *pc.Compact
			if compact.ColdThreshold == nil {
				compact.ColdThreshold = c.Compact.ColdThreshold
			}
		}
		if pc.Index != nil {
			index = pc.Index.IndexConfig
			if pc.Index.InheritRules {
				index.RuleNames = append(index.RuleNames, c.Index.RuleNames...)
			}
			if index.ColdThreshold == nil {
				index.ColdThreshold = c.Index.ColdThreshold
			}
		}
		break
	}
	if branch == "" {
		branch = "main"
	}
	err := index.fillRules(indexes)
	return branch, compact, index, err
}

type PoolConfig struct {
	Pool   string `yaml:"pool"`
	Branch string `yaml:"branch"`
	// Compact specifies the compaction options for this pool. If nil the Compact
	// options from the global settings will be used.
	Compact *CompactConfig `yaml:"compact"`
	// Index specifies the indexing options for this pool. If nil the Index
	// options from the global settings will be used.
	Index *PoolIndexConfig `yaml:"index"`

	pool pools.Config
}

type CompactConfig struct {
	Disabled      bool           `yaml:"disabled"`
	ColdThreshold *time.Duration `yaml:"cold_threshold"`
}

func (c *CompactConfig) coldThreshold() time.Duration {
	if c.ColdThreshold == nil {
		return defaultCompactColdThresh
	}
	return *c.ColdThreshold
}

func (c *CompactConfig) MarshalLogObject(o zapcore.ObjectEncoder) error {
	o.AddBool("enabled", !c.Disabled)
	o.AddDuration("cold_threshold", c.coldThreshold())
	return nil
}

type PoolIndexConfig struct {
	IndexConfig  `yaml:",inline"`
	InheritRules bool `yaml:"inherit_rules"`
}

type IndexConfig struct {
	Disabled      bool           `yaml:"disabled"`
	ColdThreshold *time.Duration `yaml:"cold_threshold"`
	RuleNames     []string       `yaml:"rules"`

	rules []index.Rule
}

func (c *IndexConfig) Enabled() bool {
	return !c.Disabled && len(c.RuleNames) > 0
}

func (c *IndexConfig) Clone() IndexConfig {
	out := *c
	out.RuleNames = slices.Clone(c.RuleNames)
	return out
}

func (c *IndexConfig) coldThreshold() time.Duration {
	if c.ColdThreshold == nil {
		return defaultIndexColdThresh
	}
	return *c.ColdThreshold
}

func (c *IndexConfig) fillRules(rules []index.Rule) error {
	if !c.Enabled() {
		return nil
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

	c.rules = dedupeIndexes(c.rules)
	return nil
}

func dedupeIndexes(rules []index.Rule) []index.Rule {
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].RuleName() < rules[j].RuleName()
	})
	out := rules[:0]
	var prev index.Rule
	for _, rule := range rules {
		if prev == nil || rule.RuleID() != prev.RuleID() {
			out = append(out, rule)
			prev = rule
		}
	}
	return out
}

func (c *IndexConfig) MarshalLogObject(o zapcore.ObjectEncoder) error {
	o.AddBool("enabled", c.Enabled())
	o.AddDuration("cold_threshold", c.coldThreshold())
	o.AddArray("rules", zapcore.ArrayMarshalerFunc(func(a zapcore.ArrayEncoder) error {
		for _, r := range c.rules {
			a.AppendString(r.RuleName())
		}
		return nil
	}))
	return nil
}
