package kvs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zngbytes"
	"github.com/brimdata/zed/zson"
)

const maxRetries = 10

var (
	ErrRetriesExceeded = fmt.Errorf("config journal unavailable after %d attempts", maxRetries)
	ErrKeyExists       = errors.New("key already exists")
	ErrNoSuchKey       = errors.New("no such key")
	ErrConstraint      = errors.New("constraint failed")
)

type Store struct {
	journal     *journal.Queue
	unmarshaler *zson.UnmarshalZNGContext
	table       map[string]interface{}
	at          journal.ID
	loadTime    time.Time
}

type Entry struct {
	Key   string
	Value interface{}
}

func newStore(path *storage.URI, valTypes []interface{}) *Store {
	u := zson.NewZNGUnmarshaler()
	u.Bind(valTypes...)
	u.Bind(Entry{})
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
	r, err := s.journal.OpenAsZNG(ctx, head, 0)
	if err != nil {
		return err
	}
	table := make(map[string]interface{})
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
		var e Entry
		if err := s.unmarshaler.Unmarshal(rec.Value, &e); err != nil {
			return err
		}
		if e.Value == nil {
			delete(table, e.Key)
		} else {
			table[e.Key] = e.Value
		}
	}
}

func (s *Store) stale() bool {
	if s.loadTime.IsZero() {
		return true
	}
	return time.Now().Sub(s.loadTime) > time.Second
}

func (s *Store) Keys(ctx context.Context, key string) ([]string, error) {
	if err := s.load(ctx); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(s.table))
	for key := range s.table {
		keys = append(keys, key)
	}
	return keys, nil
}

func (s *Store) Values(ctx context.Context) ([]interface{}, error) {
	if err := s.load(ctx); err != nil {
		return nil, err
	}
	vals := make([]interface{}, 0, len(s.table))
	for _, val := range s.table {
		vals = append(vals, val)
	}
	return vals, nil
}

func (s *Store) All(ctx context.Context) ([]Entry, error) {
	if err := s.load(ctx); err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(s.table))
	for key, val := range s.table {
		entries = append(entries, Entry{key, val})
	}
	return entries, nil
}

func (s *Store) Lookup(ctx context.Context, key string) (interface{}, error) {
	var fresh bool
	if s.stale() {
		if err := s.load(ctx); err != nil {
			return nil, err
		}
		fresh = true
	}
	val, ok := s.table[key]
	if !ok {
		if fresh {
			return nil, ErrNoSuchKey
		}
		// If we didn't load the table, try loading it
		// then re-checking for the key.
		if err := s.load(ctx); err != nil {
			return nil, err
		}
		val, ok = s.table[key]
		if !ok {
			return nil, ErrNoSuchKey
		}
	}
	return val, nil
}

func (s *Store) Insert(ctx context.Context, key string, value interface{}) error {
	e := Entry{
		Key:   key,
		Value: value,
	}
	b, err := e.serialize()
	if err != nil {
		return err
	}
	for attempts := 0; attempts < maxRetries; attempts++ {
		if err := s.load(ctx); err != nil {
			return err
		}
		if _, ok := s.table[key]; ok {
			return ErrKeyExists
		}
		if err := s.journal.CommitAt(ctx, s.at, b); err != nil {
			if os.IsExist(err) {
				time.Sleep(time.Millisecond)
				continue
			}
			return err
		}
		// Force a re-load after a change.
		s.at = 0
		return nil
	}
	return ErrRetriesExceeded
}

type Constraint func(interface{}) bool

func (s *Store) Delete(ctx context.Context, key string, c Constraint) error {
	e := Entry{Key: key}
	b, err := e.serialize()
	if err != nil {
		return err
	}
	for attempts := 0; attempts < maxRetries; attempts++ {
		if err := s.load(ctx); err != nil {
			return err
		}
		val, ok := s.table[key]
		if !ok {
			return ErrNoSuchKey
		}
		if c != nil && !c(val) {
			return ErrConstraint
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

func (s *Store) Move(ctx context.Context, oldKey, newKey string, newVal interface{}) error {
	remove := Entry{
		Key:   oldKey,
		Value: nil,
	}
	add := Entry{
		Key:   newKey,
		Value: newVal,
	}
	serializer := zngbytes.NewSerializer()
	if err := serializer.Write(&remove); err != nil {
		return err
	}
	if err := serializer.Write(&add); err != nil {
		return err
	}
	if err := serializer.Close(); err != nil {
		return err
	}
	b := serializer.Bytes()
	for attempts := 0; attempts < maxRetries; attempts++ {
		if err := s.load(ctx); err != nil {
			return err
		}
		if _, ok := s.table[oldKey]; !ok {
			return ErrNoSuchKey
		}
		if _, ok := s.table[newKey]; ok {
			return ErrKeyExists
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

func (e Entry) serialize() ([]byte, error) {
	serializer := zngbytes.NewSerializer()
	if err := serializer.Write(&e); err != nil {
		return nil, err
	}
	if err := serializer.Close(); err != nil {
		return nil, err
	}
	return serializer.Bytes(), nil
}

func Open(ctx context.Context, engine storage.Engine, path *storage.URI, keyTypes []interface{}) (*Store, error) {
	s := newStore(path, keyTypes)
	var err error
	s.journal, err = journal.Open(ctx, engine, path)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func Create(ctx context.Context, engine storage.Engine, path *storage.URI, keyTypes []interface{}) (*Store, error) {
	s := newStore(path, keyTypes)
	j, err := journal.Create(ctx, engine, path)
	if err != nil {
		return nil, err
	}
	s.journal = j
	return s, nil
}
