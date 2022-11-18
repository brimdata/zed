package lakemanage

import (
	"context"
	"fmt"
	"time"

	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

func IndexScan(ctx context.Context, lk api.Interface, pool, branch string, thresh time.Duration,
	rules []index.Rule, ch chan<- ObjectIndexes) (*time.Time, error) {
	it, err := newIndexIterator(ctx, lk, pool, branch)
	if err != nil {
		return nil, err
	}
	defer it.reader.Close()
	var nextcold *time.Time
	for {
		o, err := it.Next()
		if o == nil || err != nil {
			return nextcold, err
		}
		// XXX An object's create timestamp is currently derived from the
		// timestamp in its ksuid ID when it should really be the commit
		// timestamp since this is when the object officially exists from the
		// lake's perspective.
		ts := o.Object.ID.Time()
		if time.Since(ts) < thresh {
			coldts := ts.Add(thresh)
			if nextcold == nil || (*nextcold).After(coldts) {
				nextcold = &coldts
			}
			continue
		}
	loop:
		for _, r := range rules {
			for _, rid := range o.RuleIDs {
				if r.RuleID() == rid {
					continue loop
				}
			}
			o.NeedsIndex = append(o.NeedsIndex, r.RuleID().String())
		}
		if len(o.NeedsIndex) > 0 {
			select {
			case ch <- *o:
			case <-ctx.Done():
				return nextcold, ctx.Err()
			}
		}
	}
}

type ObjectIndexes struct {
	Object     data.Object   `zed:"object"`
	RuleIDs    []ksuid.KSUID `zed:"rule_ids"`
	NeedsIndex []string
}

const indexQueryFmt = `
from (
	pool '%s'@'%s':objects => sort id | yield { object: this }
	pool '%s'@'%s':indexes => sort id | yield { index: this }
)
| left join on object.id = index.id rule_id := index.rule.id
| object := any(object), rule_ids := collect(rule_id) by id := object.id `

type indexIterator struct {
	reader      zio.ReadCloser
	unmarshaler *zson.UnmarshalZNGContext
}

func newIndexIterator(ctx context.Context, lk api.Interface, pool, branch string) (*indexIterator, error) {
	query := fmt.Sprintf(indexQueryFmt, pool, branch, pool, branch)
	r, err := lk.Query(ctx, nil, query)
	if err != nil {
		return nil, err
	}
	return &indexIterator{
		reader:      r,
		unmarshaler: zson.NewZNGUnmarshaler(),
	}, nil
}

func (p *indexIterator) Next() (*ObjectIndexes, error) {
	val, err := p.reader.Read()
	if val == nil || err != nil {
		return nil, err
	}
	var o ObjectIndexes
	if err := p.unmarshaler.Unmarshal(val, &o); err != nil {
		return nil, err
	}
	return &o, nil
}
