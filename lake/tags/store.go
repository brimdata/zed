package tags

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/pkg/storage"
)

var (
	ErrExists   = errors.New("tag already exists")
	ErrNotFound = errors.New("tag not found")
)

type Store struct {
	store *journal.Store
}

func CreateStore(ctx context.Context, engine storage.Engine, path *storage.URI) (*Store, error) {
	store, err := journal.CreateStore(ctx, engine, path, Config{})
	if err != nil {
		return nil, err
	}
	return &Store{store}, nil
}

func OpenStore(ctx context.Context, engine storage.Engine, path *storage.URI) (*Store, error) {
	store, err := journal.OpenStore(ctx, engine, path, Config{})
	if err != nil {
		return nil, err
	}
	return &Store{store}, nil
}

func (s *Store) All(ctx context.Context) ([]Config, error) {
	entries, err := s.store.All(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]Config, 0, len(entries))
	for _, entry := range entries {
		tag, ok := entry.(*Config)
		if !ok {
			return nil, errors.New("corrupt tag config journal")
		}
		list = append(list, *tag)
	}
	return list, nil
}

func (s *Store) LookupByName(ctx context.Context, name string) (*Config, error) {
	list, _ := s.All(ctx)
	for k, config := range list {
		if config.Name == name {
			return &list[k], nil
		}
	}
	return nil, fmt.Errorf("%q: %w", name, ErrNotFound)
}

func (s *Store) Add(ctx context.Context, config *Config) error {
	return s.store.Insert(ctx, config)
}

func (s *Store) Remove(ctx context.Context, config Config) error {
	err := s.store.Delete(ctx, config.Name, func(v journal.Entry) bool {
		return true
	})
	if err != nil {
		if err == journal.ErrNoSuchKey {
			return fmt.Errorf("%s: %w", config.Name, ErrNotFound)
		}
		return err
	}
	return nil
}
