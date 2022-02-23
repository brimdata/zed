package index

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zngbytes"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

const (
	maxRetries = 10
)

var (
	ErrRetriesExceeded = fmt.Errorf("config journal unavailable after %d attempts", maxRetries)
	ErrNoSuchRule      = errors.New("no such rule")
)

type Store struct {
	journal     *journal.Queue
	unmarshaler *zson.UnmarshalZNGContext
	table       map[ksuid.KSUID]Rule
	at          journal.ID
	loadTime    time.Time
}

type AddRule struct {
	Rule Rule
}

type DeleteRule struct {
	ID ksuid.KSUID
}

var RuleTypes = []interface{}{
	AddRule{},
	DeleteRule{},
	FieldRule{},
	TypeRule{},
	AggRule{},
}

func newStore(path *storage.URI) *Store {
	u := zson.NewZNGUnmarshaler()
	u.Bind(RuleTypes...)
	return &Store{
		unmarshaler: u,
	}
}

func (s *Store) load(ctx context.Context) error {
	head, err := s.journal.ReadHead(ctx)
	if err != nil {
		return err
	}
	if head == s.at {
		return nil
	}
	r, err := s.journal.OpenAsZNG(ctx, zed.NewContext(), head, 0)
	if err != nil {
		return err
	}
	defer r.Close()
	table := make(map[ksuid.KSUID]Rule)
	for {
		rec, err := r.Read()
		if err != nil {
			return err
		}
		if rec == nil {
			s.table = table
			s.loadTime = time.Now()
			s.at = head
			return nil

		}
		var rule interface{}
		if err := s.unmarshaler.Unmarshal(*rec, &rule); err != nil {
			return err
		}
		switch rule := rule.(type) {
		case *AddRule:
			id := rule.Rule.RuleID()
			if _, ok := table[id]; ok {
				return fmt.Errorf("system error: rule ID %s already exists in index journal", id)
			}
			table[id] = rule.Rule
		case *DeleteRule:
			id := rule.ID
			if _, ok := table[id]; !ok {
				return fmt.Errorf("system error: index rule ID %s does not exist in index journal", id)
			}
			delete(table, id)
		default:
			return fmt.Errorf("bad type record type in index store log: %T", rule)
		}
	}
}

func (s *Store) stale() bool {
	if s.loadTime.IsZero() {
		return true
	}
	return time.Now().Sub(s.loadTime) > time.Second
}

func (s *Store) All(ctx context.Context) ([]Rule, error) {
	if err := s.load(ctx); err != nil {
		return nil, err
	}
	rules := make([]Rule, 0, len(s.table))
	for _, r := range s.table {
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *Store) Names(ctx context.Context) ([]string, error) {
	if err := s.load(ctx); err != nil {
		return nil, err
	}
	table := make(map[string]struct{})
	for _, rule := range s.table {
		table[rule.RuleName()] = struct{}{}
	}
	names := make([]string, 0, len(table))
	for name := range table {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func (s *Store) Lookup(ctx context.Context, name string) ([]Rule, error) {
	var fresh bool
	if s.stale() {
		if err := s.load(ctx); err != nil {
			return nil, err
		}
		fresh = true
	}
again:
	var rules []Rule
	for _, rule := range s.table {
		if rule.RuleName() == name {
			rules = append(rules, rule)
		}
	}
	if len(rules) == 0 {
		if fresh {
			return nil, fmt.Errorf("rule set %q not found", name)
		}
		// If we didn't load a fresh table, try loading it
		// then re-checking for the name...
		if err := s.load(ctx); err != nil {
			return nil, err
		}
		fresh = true
		goto again
	}
	return rules, nil
}

func (s *Store) LookupByID(ctx context.Context, id ksuid.KSUID) (Rule, error) {
	var fresh bool
	if s.stale() {
		if err := s.load(ctx); err != nil {
			return nil, err
		}
		fresh = true
	}
again:
	rule, ok := s.table[id]
	if !ok {
		if fresh {
			return nil, fmt.Errorf("index rule %s: not found", id)
		}
		// If we didn't load a fresh table, try loading it
		// then re-checking for the name...
		if err := s.load(ctx); err != nil {
			return nil, err
		}
		fresh = true
		goto again
	}
	return rule, nil
}

func (s *Store) Add(ctx context.Context, rule Rule) error {
	add := &AddRule{Rule: rule}
	b, err := serialize(add)
	if err != nil {
		return err
	}
	id := rule.RuleID()
	for attempts := 0; attempts < maxRetries; attempts++ {
		if err := s.load(ctx); err != nil {
			return err
		}
		if _, ok := s.table[id]; ok {
			return fmt.Errorf("rule %q: ID in use", id)
		}
		if s.exists(rule) {
			return errors.New("index rule already exists")
		}
		err := s.journal.CommitAt(ctx, s.at, b)
		if err != nil {
			if os.IsExist(err) {
				time.Sleep(time.Millisecond)
				continue
			}
			return err
		}
		s.at = 0
		return nil
	}
	return ErrRetriesExceeded
}

func (s *Store) exists(rule Rule) bool {
	name := rule.RuleName()
	for _, existing := range s.table {
		if existing.RuleName() == name && Equivalent(rule, existing) {
			return true
		}
	}
	return false
}

func (s *Store) Delete(ctx context.Context, id ksuid.KSUID) (Rule, error) {
	b, err := serialize(&DeleteRule{ID: id})
	if err != nil {
		return nil, err
	}
	for attempts := 0; attempts < maxRetries; attempts++ {
		if err := s.load(ctx); err != nil {
			return nil, err
		}
		rule, ok := s.table[id]
		if !ok {
			return nil, fmt.Errorf("index rule %q does not exist", id)
		}
		err := s.journal.CommitAt(ctx, s.at, b)
		if err != nil {
			if os.IsExist(err) {
				time.Sleep(time.Millisecond)
				continue
			}
			return nil, err
		}
		s.at = 0
		return rule, nil
	}
	return nil, ErrRetriesExceeded
}

func (s *Store) IDs(ctx context.Context) ([]ksuid.KSUID, error) {
	if err := s.load(ctx); err != nil {
		return nil, err
	}
	var ids []ksuid.KSUID
	for id := range s.table {
		ids = append(ids, id)
	}
	return ids, nil
}

func serialize(r interface{}) ([]byte, error) {
	serializer := zngbytes.NewSerializer()
	serializer.Decorate(zson.StylePackage)
	if err := serializer.Write(r); err != nil {
		return nil, err
	}
	if err := serializer.Close(); err != nil {
		return nil, err
	}
	return serializer.Bytes(), nil
}

func OpenStore(ctx context.Context, engine storage.Engine, path *storage.URI) (*Store, error) {
	s := newStore(path)
	var err error
	s.journal, err = journal.Open(ctx, engine, path)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func CreateStore(ctx context.Context, engine storage.Engine, path *storage.URI) (*Store, error) {
	s := newStore(path)
	j, err := journal.Create(ctx, engine, path, journal.Nil)
	if err != nil {
		return nil, err
	}
	s.journal = j
	return s, nil
}
