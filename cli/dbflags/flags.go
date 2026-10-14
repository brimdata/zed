package dbflags

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/superdb/super/api/client"
	"github.com/superdb/super/api/client/auth0"
	"github.com/superdb/super/cli/tableflags"
	"github.com/superdb/super/db"
	"github.com/superdb/super/db/api"
	"github.com/superdb/super/pkg/storage"
	"go.uber.org/zap"
)

var (
	ErrNoHEAD  = errors.New("HEAD not specified: indicate with -use or run the \"use\" command")
	ErrLocalDB = errors.New("cannot open connection on local database")
)

type Flags struct {
	ConfigDir string
	Quiet     bool
	attach    string
}

func (l *Flags) SetFlags(fs *flag.FlagSet) {
	fs.BoolVar(&l.Quiet, "q", false, "quiet mode")
	dir, _ := os.UserHomeDir()
	if dir != "" {
		dir = filepath.Join(dir, ".super")
	}
	fs.StringVar(&l.ConfigDir, "configdir", dir, "configuration and credentials directory")
	fs.Func("attach", "database for this command only (default: the attached database)",
		func(s string) error {
			l.attach = s
			if s == "" {
				return errors.New("please provide path of database or URI of service")
			}
			return nil
		})
}

func (l *Flags) Connection() (*client.Connection, error) {
	uri, err := l.URI()
	if err != nil {
		return nil, err
	}
	if !api.IsRemote(uri.String()) {
		return nil, ErrLocalDB
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
	if api.IsRemote(uri.String()) {
		conn, err := l.Connection()
		if err != nil {
			return nil, err
		}
		return api.NewRemoteDB(conn), nil
	}
	DB, err := api.Connect(ctx, zap.Must(zap.NewProduction()), uri.String())
	if errors.Is(err, db.ErrNotExist) {
		return nil, fmt.Errorf("%w\n(hint: run 'super db init' to initialize a database at this location)", err)
	}
	return DB, err
}

func (l *Flags) URI() (*storage.URI, error) {
	s := l.attach
	if s == "" {
		var err error
		if s, err = tableflags.ReadAttach(); err != nil {
			return nil, err
		}
	}
	if s == "" && pingLocalServer() {
		s = "http://localhost:9867"
	}
	if s == "" {
		s = os.Getenv("SUPER_DB")
	}
	if s == "" {
		if s = getDefaultDataDir(); s == "" {
			return nil, errors.New(`no database attached (run "super db attach <location>" or use -attach)`)
		}
	}
	return storage.ParseURI(strings.TrimRight(s, "/"))
}

func (l *Flags) AuthStore() *auth0.Store {
	return auth0.NewStore(filepath.Join(l.ConfigDir, "credentials.json"))
}

func pingLocalServer() bool {
	_, err := client.NewConnection().Ping(context.Background())
	return err == nil
}
