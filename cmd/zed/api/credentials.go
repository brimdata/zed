package api

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brimdata/zq/pkg/fs"
)

const (
	zapiConfigDirName = "zapi"
	credsFileName     = "credentials.json"
)

type LocalConfigFlags struct {
	configPath string
}

func (f *LocalConfigFlags) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&f.configPath, "configpath", "", "directory to store local configuration and credentials")
}

func (f *LocalConfigFlags) credentialsPath() (string, error) {
	if f.configPath != "" {
		return filepath.Abs(filepath.Join(f.configPath, credsFileName))
	}
	c, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("zapi: no user configuration directory available: %w", err)
	}
	return filepath.Join(c, zapiConfigDirName, credsFileName), nil
}

func (f *LocalConfigFlags) LoadCredentials() (*Credentials, error) {
	cpath, err := f.credentialsPath()
	if err != nil {
		return nil, err
	}
	var cf Credentials
	if err := fs.UnmarshalJSONFile(cpath, &cf); err != nil {
		if os.IsNotExist(err) {
			return &Credentials{}, nil
		}
		return nil, err
	}
	return &cf, nil
}

func (f *LocalConfigFlags) SaveCredentials(cf *Credentials) error {
	cpath, err := f.credentialsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cpath), 0700); err != nil {
		return err
	}
	return fs.MarshalJSONFile(cf, cpath, 0600)
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
