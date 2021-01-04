package apiserver

import (
	"flag"
	"fmt"
)

// XXX This file and all db related declarations should/will be moved to
// separate db package in ppl/zqd/db

// Implement the flag.Value interface for type DBKind

func (k *DBKind) Set(s string) error {
	switch s {
	case string(DBUnspecified), string(DBFile), string(DBPostgres):
		*k = DBKind(s)
		return nil
	}
	return fmt.Errorf("unsupported db kind: %s", s)
}

func (k DBKind) String() string {
	if k == "" {
		k = DBFile
	}
	return string(k)
}

func (d *DBConfig) SetFlags(fs *flag.FlagSet) {
	fs.Var(&d.Kind, "db.kind", "the kind of database backing space data (values: file, postgres)")
	fs.Var(&d.Postgres, "db.postgres.url", "postgres url (postgres://[user[:password]@][netloc][:port]/[database])")
	fs.StringVar(&d.Postgres.Addr, "db.postgres.addr", "localhost:5432", "postgres address")
	fs.StringVar(&d.Postgres.User, "db.postgres.user", "", "postgres username")
	fs.StringVar(&d.Postgres.Password, "db.postgres.password", "", "postgres password")
	fs.StringVar(&d.Postgres.Database, "db.postgres.database", "", "postgres database name")
}
