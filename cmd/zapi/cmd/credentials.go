package cmd

import (
	"os"
	"path"
	"path/filepath"

	"github.com/brimsec/zq/pkg/fs"
)

const (
	zapiConfigDirName = "zapi"
	credsFileName     = "credentials.json"
)

func LoadCredentials(path string) (*Credentials, error) {
	var cf Credentials
	if err := fs.UnmarshalJSONFile(path, &cf); err != nil {
		if os.IsNotExist(err) {
			return &Credentials{}, nil
		}
		return nil, err
	}
	return &cf, nil
}

func SaveCredentials(path string, cf *Credentials) error {
	return fs.MarshalJSONFile(cf, path, 0600)
}

func UserStdCredentialsPath() (string, error) {
	c, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	zcd := filepath.Join(c, zapiConfigDirName)
	if err := os.MkdirAll(zcd, 0777); err != nil {
		return "", err
	}
	return path.Join(zcd, credsFileName), nil
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
