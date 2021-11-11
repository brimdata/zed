package api

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	zedfs "github.com/brimdata/zed/pkg/fs"
)

const credsFileName = "credentials.json"

func (c *Command) LoadCredentials() (*Credentials, error) {
	path := filepath.Join(c.configDir, credsFileName)
	var creds Credentials
	if err := zedfs.UnmarshalJSONFile(path, &creds); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Credentials{}, nil
		}
		return nil, err
	}
	return &creds, nil
}

func (c *Command) SaveCredentials(creds *Credentials) error {
	if err := os.MkdirAll(c.configDir, 0700); err != nil {
		return err
	}
	return zedfs.MarshalJSONFile(creds, filepath.Join(c.configDir, credsFileName), 0600)
}

type ServiceInfo struct {
	URL    string        `json:"url"`
	Tokens ServiceTokens `json:"tokens"`
}

type ServiceTokens struct {
	Access  string `json:"access"`
	ID      string `json:"id"`
	Refresh string `json:"refresh"`
}

type Credentials struct {
	Version  int           `json:"version"`
	Services []ServiceInfo `json:"services"`
}

func (c *Credentials) ServiceTokens(url string) (ServiceTokens, bool) {
	for _, s := range c.Services {
		if s.URL == url {
			return s.Tokens, true
		}
	}
	return ServiceTokens{}, false
}

func (c *Credentials) AddTokens(url string, tokens ServiceTokens) {
	svcs := make([]ServiceInfo, 0, len(c.Services)+1)
	for _, s := range c.Services {
		if s.URL != url {
			svcs = append(svcs, s)
		}
	}
	svcs = append(svcs, ServiceInfo{
		URL:    url,
		Tokens: tokens,
	})
	c.Services = svcs
}

func (c *Credentials) RemoveTokens(url string) {
	svcs := make([]ServiceInfo, 0, len(c.Services))
	for _, s := range c.Services {
		if s.URL != url {
			svcs = append(svcs, s)
		}
	}
	c.Services = svcs
}
