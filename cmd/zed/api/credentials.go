package api

import (
	"os"
	"path/filepath"

	"github.com/brimdata/zed/pkg/fs"
)

const (
	configDirName = "zapi"
	credsFileName = "credentials.json"
)

func (c *Command) zapiPath() (string, error) {
	if c.configDir != "" {
		if err := os.MkdirAll(c.configDir, 0777); err != nil {
			return "", err
		}
		return filepath.Abs(c.configDir)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	zcd := filepath.Join(dir, configDirName)
	if err := os.MkdirAll(zcd, 0777); err != nil {
		return "", err
	}
	return zcd, nil
}

func (c *Command) LoadCredentials() (*Credentials, error) {
	path := filepath.Join(c.configDir, credsFileName)
	var creds Credentials
	if err := fs.UnmarshalJSONFile(path, &creds); err != nil {
		if os.IsNotExist(err) {
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
	return fs.MarshalJSONFile(creds, filepath.Join(c.configDir, credsFileName), 0600)
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
