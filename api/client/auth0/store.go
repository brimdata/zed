package auth0

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	zedfs "github.com/brimdata/zed/pkg/fs"
	"github.com/brimdata/zed/pkg/storage"
)

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path}
}

func (s Store) Tokens(uri string) (*Tokens, error) {
	creds, err := s.load()
	if err != nil || creds == nil {
		return nil, err
	}
	return creds.Tokens(uri), nil
}

func (s Store) SetTokens(uri string, tokens Tokens) error {
	creds, err := s.load()
	if err != nil {
		return err
	}
	creds.AddTokens(uri, tokens)
	return s.save(creds)
}

func (s Store) RemoveTokens(uri string) error {
	creds, err := s.load()
	if err != nil {
		return err
	}
	creds.RemoveTokens(uri)
	return s.save(creds)
}

func (s Store) load() (*Credentials, error) {
	var creds Credentials
	if err := zedfs.UnmarshalJSONFile(s.path, &creds); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Credentials{}, nil
		}
		return nil, err
	}
	return &creds, nil
}

func (s Store) save(creds *Credentials) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	return zedfs.MarshalJSONFile(creds, s.path, 0600)
}

type ServiceInfo struct {
	URL    string `json:"url"`
	Tokens Tokens `json:"tokens"`
}

type Credentials struct {
	Version  int           `json:"version"`
	Services []ServiceInfo `json:"services"`
}

func (c *Credentials) Tokens(uri string) *Tokens {
	for _, s := range c.Services {
		if s.URL == uri {
			return &s.Tokens
		}
	}
	return nil
}

func (c *Credentials) AddTokens(uri string, tokens Tokens) {
	uri = normalizeURI(uri)
	svcs := make([]ServiceInfo, 0, len(c.Services)+1)
	for _, s := range c.Services {
		if s.URL != uri {
			svcs = append(svcs, s)
		}
	}
	svcs = append(svcs, ServiceInfo{
		URL:    uri,
		Tokens: tokens,
	})
	c.Services = svcs
}

func (c *Credentials) RemoveTokens(uri string) {
	uri = normalizeURI(uri)
	svcs := make([]ServiceInfo, 0, len(c.Services))
	for _, s := range c.Services {
		if s.URL != uri {
			svcs = append(svcs, s)
		}
	}
	c.Services = svcs
}

func normalizeURI(uri string) string {
	u := storage.MustParseURI(uri)
	return u.String()
}
