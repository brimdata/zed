package lakeflags

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
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/storage"
)

var ErrNoHEAD = errors.New("HEAD not specified: indicate with -use or run the \"use\" command")

const credsFileName = "credentials.json"

type Flags struct {
	ConfigDir string
	// LakeSpecified is set to true if the lake is explicitly set via either
	// command line flag or environment variable.
	LakeSpecified bool
	Lake          string
	Quiet         bool
	defaultHead   string
}

func (l *Flags) SetFlags(fs *flag.FlagSet) {
	fs.BoolVar(&l.Quiet, "q", false, "quiet mode")
	defaultHead, _ := readHead()
	fs.StringVar(&l.defaultHead, "use", defaultHead, "commit to use, i.e., pool, pool@branch, or pool@commit")
	dir, _ := os.UserHomeDir()
	if dir != "" {
		dir = filepath.Join(dir, ".zed")
	}
	fs.StringVar(&l.ConfigDir, "configdir", dir, "configuration and credentials directory")
	l.Lake = "http://localhost:9867"
	if s, ok := os.LookupEnv("ZED_LAKE"); ok {
		l.Lake = s
		l.LakeSpecified = true
	}
	fs.Func("lake", fmt.Sprintf("lake location (env ZED_LAKE) (default %s)", l.Lake), func(s string) error {
		l.Lake = s
		l.LakeSpecified = true
		return nil
	})
}

func (f *Flags) HEAD() (*lakeparse.Commitish, error) {
	c, err := lakeparse.ParseCommitish(f.defaultHead)
	if err != nil {
		return nil, err
	}
	if c.Pool == "" {
		return nil, errors.New("pool unspecified")
	}
	if c.Branch == "" {
		c.Branch = "main"
	}
	return c, nil
}

func (l *Flags) Connection() (*client.Connection, error) {
	uri, err := l.URI()
	if err != nil {
		return nil, err
	}
	if !api.IsLakeService(uri.String()) {
		return nil, errors.New("cannot open connection on local lake")
	}
	conn := client.NewConnectionTo(uri.String())
	if err := conn.SetAuthStore(l.AuthStore()); err != nil {
		return nil, err
	}
	return conn, nil
}

func (l *Flags) Open(ctx context.Context) (api.Interface, error) {
	uri, err := l.URI()
	if err != nil {
		return nil, err
	}
	if api.IsLakeService(uri.String()) {
		conn, err := l.Connection()
		if err != nil {
			return nil, err
		}
		return api.NewRemoteLake(conn), nil
	}
	return api.OpenLocalLake(ctx, uri.String())
}

func (l *Flags) AuthStore() *auth0.Store {
	return auth0.NewStore(filepath.Join(l.ConfigDir, credsFileName))
}

func (l *Flags) URI() (*storage.URI, error) {
	if l.Lake == "" {
		return nil, errors.New("lake location must be set (either with the -lake flag or ZED_LAKE environment variable)")
	}
	u, err := storage.ParseURI(l.Lake)
	if err != nil {
		err = fmt.Errorf("error parsing lake location: %w", err)
	}
	return u, err
}
