package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/lake/api"
	zedfs "github.com/brimdata/zed/pkg/fs"
	"github.com/brimdata/zed/pkg/storage"
)

const credsFileName = "credentials.json"

type LakeFlags struct {
	ConfigDir string
	Lake      string
}

func (l *LakeFlags) SetFlags(fs *flag.FlagSet) {
	dir, _ := os.UserHomeDir()
	if dir != "" {
		dir = filepath.Join(dir, ".zed")
	}
	fs.StringVar(&l.ConfigDir, "configdir", dir, "configuration and credentials directory")
	lake := "http://localhost:9867"
	if s, ok := os.LookupEnv("ZED_LAKE"); ok {
		lake = s
	}
	fs.StringVar(&l.Lake, "lake", lake, "lake location (env: ZED_LAKE)")
}

func (l *LakeFlags) Connection() (*client.Connection, error) {
	uri, err := l.URI()
	if err != nil {
		return nil, nil
	}
	if !api.IsRemoteLake(uri) {
		return nil, errors.New("cannot open connection on local lake")
	}
	creds, err := l.LoadCredentials()
	if err != nil {
		return nil, err
	}
	conn := client.NewConnectionTo(uri.String())
	if token, ok := creds.ServiceTokens(uri.String()); ok {
		conn.SetAuthToken(token.Access)
	}
	return conn, nil
}

func (l *LakeFlags) Open(ctx context.Context) (api.Interface, error) {
	uri, err := l.URI()
	if err != nil {
		return nil, err
	}
	if api.IsRemoteLake(uri) {
		conn, err := l.Connection()
		if err != nil {
			return nil, err
		}
		return api.NewRemoteWithConnection(conn), nil
	}
	return api.OpenLocalLake(ctx, uri)
}

func (l *LakeFlags) URI() (*storage.URI, error) {
	if l.Lake == "" {
		return nil, errors.New("lake location must be set (either with the -lake flag or ZED_LAKE environment variable)")
	}
	u, err := storage.ParseURI(l.Lake)
	if err != nil {
		err = fmt.Errorf("error parsing lake location: %w", err)
	}
	return u, err
}

func (l *LakeFlags) LoadCredentials() (*Credentials, error) {
	path := filepath.Join(l.ConfigDir, credsFileName)
	var creds Credentials
	if err := zedfs.UnmarshalJSONFile(path, &creds); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Credentials{}, nil
		}
		return nil, err
	}
	return &creds, nil
}

func (l *LakeFlags) SaveCredentials(creds *Credentials) error {
	if err := os.MkdirAll(l.ConfigDir, 0700); err != nil {
		return err
	}
	return zedfs.MarshalJSONFile(creds, filepath.Join(l.ConfigDir, credsFileName), 0600)
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
