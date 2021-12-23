package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/api/client/auth0"
	"github.com/brimdata/zed/lake/api"
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
		return nil, err
	}
	if !api.IsLakeService(uri) {
		return nil, errors.New("cannot open connection on local lake")
	}
	tokens, err := l.AuthStore().Tokens(uri.String())
	if err != nil {
		return nil, err
	}
	conn := client.NewConnectionTo(uri.String())
	if tokens != nil {
		conn.SetAuthToken(tokens.Access)
	}
	return conn, nil
}

func (l *LakeFlags) Open(ctx context.Context) (api.Interface, error) {
	uri, err := l.URI()
	if err != nil {
		return nil, err
	}
	if api.IsLakeService(uri) {
		conn, err := l.Connection()
		if err != nil {
			return nil, err
		}
		return api.NewRemoteWithConnection(conn), nil
	}
	return api.OpenLocalLake(ctx, uri)
}

func (l *LakeFlags) AuthStore() *auth0.Store {
	return auth0.NewStore(filepath.Join(l.ConfigDir, credsFileName))
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
