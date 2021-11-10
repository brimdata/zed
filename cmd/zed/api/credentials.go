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
	if c.configPath != "" {
		if err := os.MkdirAll(c.configPath, 0777); err != nil {
			return "", err
		}
		return filepath.Abs(c.configPath)
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
	path, err := c.zapiPath()
	if err != nil {
		return nil, err
	}
	path = filepath.Join(path, credsFileName)
	var cf Credentials
	if err := fs.UnmarshalJSONFile(path, &cf); err != nil {
		if os.IsNotExist(err) {
			return &Credentials{}, nil
		}
		return nil, err
	}
	return &cf, nil
}

func (c *Command) SaveCredentials(cf *Credentials) error {
	path, err := c.zapiPath()
	if err != nil {
		return err
	}
	return fs.MarshalJSONFile(cf, filepath.Join(path, credsFileName), 0600)
}

type ServiceInfo struct {
	Endpoint string        `json:"endpoint"`
	Tokens   ServiceTokens `json:"tokens"`
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
		if s.Endpoint == url {
			return s.Tokens, true
		}
	}
	return ServiceTokens{}, false
}

func (c *Credentials) AddTokens(u string, sc ServiceTokens) {
	svcs := make([]ServiceInfo, 0, len(c.Services)+1)
	for _, s := range c.Services {
		if s.Endpoint != u {
			svcs = append(svcs, s)
		}
	}
	svcs = append(svcs, ServiceInfo{
		Endpoint: u,
		Tokens:   sc,
	})
	c.Services = svcs
}

func (c *Credentials) RemoveTokens(u string) {
	svcs := make([]ServiceInfo, 0, len(c.Services))
	for _, s := range c.Services {
		if s.Endpoint != u {
			svcs = append(svcs, s)
		}
	}
	c.Services = svcs
}
