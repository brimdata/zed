package pools

import (
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/segmentio/ksuid"
)

type Config struct {
	Ts        nano.Ts      `zng:"ts"`
	Name      string       `zng:"name"`
	ID        ksuid.KSUID  `zng:"id"`
	Layout    order.Layout `zng:"layout"`
	Threshold int64        `zng:"threshold"`
}

var _ journal.Entry = (*Config)(nil)

func NewConfig(name string, layout order.Layout, thresh int64) *Config {
	if thresh == 0 {
		thresh = data.DefaultThreshold
	}
	return &Config{
		Ts:        nano.Now(),
		Name:      name,
		ID:        ksuid.New(),
		Layout:    layout,
		Threshold: thresh,
	}
}

func (p *Config) Key() string {
	return p.Name
}

func (p *Config) Path(root *storage.URI) *storage.URI {
	return root.AppendPath(p.ID.String())
}
