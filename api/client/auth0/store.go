package auth0

import (
	"errors"
	"io/fs"

	zedfs "github.com/brimdata/zed/pkg/fs"
	"github.com/brimdata/zed/pkg/storage"
)

func normalizeLakeURI(lake string) string {
	u := storage.MustParseURI(lake)
	return u.String()
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path}
}

func (s Store) LakeTokens(lake string) (*Tokens, error) {
	creds, err := s.load()
	if err != nil || creds == nil {
		return nil, err
	}
	return creds.ServiceTokens(lake), nil
}

func (s Store) SetLakeTokens(lake string, tokens Tokens) error {
	creds, err := s.load()
	if err != nil {
		return err
	}
	creds.AddTokens(lake, tokens)
	return s.save(creds)
}

func (s Store) RemoveLakeTokens(lake string) error {
	creds, err := s.load()
	if err != nil {
		return err
	}
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

func (c *Credentials) ServiceTokens(url string) *Tokens {
	for _, s := range c.Services {
		if s.URL == url {
			return &s.Tokens
		}
	}
	return nil
}

func (c *Credentials) AddTokens(lake string, tokens Tokens) {
	lake = normalizeLakeURI(lake)
	svcs := make([]ServiceInfo, 0, len(c.Services)+1)
	for _, s := range c.Services {
		if s.URL != lake {
			svcs = append(svcs, s)
		}
	}
	svcs = append(svcs, ServiceInfo{
		URL:    lake,
		Tokens: tokens,
	})
	c.Services = svcs
}

func (c *Credentials) RemoveTokens(lake string) {
	lake = normalizeLakeURI(lake)
	svcs := make([]ServiceInfo, 0, len(c.Services))
	for _, s := range c.Services {
		if s.URL != lake {
			svcs = append(svcs, s)
		}
	}
	c.Services = svcs
}
