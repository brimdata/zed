package filedb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/zqd/auth"
	"github.com/brimsec/zq/ppl/zqd/db/filedb/oldconfig"
	"github.com/brimsec/zq/ppl/zqd/db/schema"
	"github.com/brimsec/zq/zqe"
	"go.uber.org/zap"
)

const currentVersion = 6

type dbdata struct {
	Version    int                `json:"version"`
	SpaceRows  []schema.SpaceRow  `json:"space_rows"`
	IntakeRows []schema.IntakeRow `json:"intake_rows"`
}

func migrateOldConfig(ctx context.Context, logger *zap.Logger, root iosrc.URI) error {
	dburi := root.AppendPath(dbname)
	exists, err := iosrc.Exists(ctx, dburi)
	if err != nil || exists {
		return err
	}

	// Since the dbfile doesn't exist, we check if we need to migrate the older
	// per-space config files into a new dbfile.
	configs, err := oldconfig.LoadConfigs(ctx, logger, root)
	if err != nil {
		return err
	}
	var rows []rowV4
	for id, config := range configs {
		datauri := config.DataURI
		if datauri.IsZero() {
			datauri = root.AppendPath(string(id))
		}
		rows = append(rows, rowV4{
			ID:      id,
			Name:    config.Name,
			DataURI: datauri,
			Storage: config.Storage,
		})
	}
	return iosrc.Replace(ctx, dburi, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(dbdataV4{
			Version:   4,
			SpaceRows: rows,
		})
	})
}

func migrateFileDatabase(ctx context.Context, dburi iosrc.URI) error {
	data, err := iosrc.ReadFile(ctx, dburi)
	if err != nil {
		if zqe.IsNotFound(err) {
			return iosrc.Replace(ctx, dburi, func(w io.Writer) error {
				return json.NewEncoder(w).Encode(dbdata{
					Version: currentVersion,
				})
			})
		}
		return err
	}
	var vc struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &vc); err != nil {
		return err
	}
	var migrator func([]byte) (int, []byte, error)
	version := vc.Version
	for version < currentVersion {
		switch version {
		case 4:
			migrator = migrateV5
		case 5:
			migrator = migrateV6
		default:
			return fmt.Errorf("unsupported database version %d", version)
		}
		var err error
		if version, data, err = migrator(data); err != nil {
			return err
		}
	}
	return iosrc.WriteFile(ctx, dburi, data)
}

type dbdataV4 struct {
	Version   int     `json:"version"`
	SpaceRows []rowV4 `json:"space_rows"`
}

type rowV4 struct {
	ID       api.SpaceID       `json:"id"`
	DataURI  iosrc.URI         `json:"data_uri"`
	Name     string            `json:"name"`
	ParentID api.SpaceID       `json:"parent_id"`
	Storage  api.StorageConfig `json:"storage"`
}

func migrateV5(data []byte) (int, []byte, error) {
	var dbv4 dbdataV4
	if err := json.Unmarshal(data, &dbv4); err != nil {
		return 0, nil, err
	}
	var db dbdataV5
	db.Version = 5
	db.SpaceRows = make([]rowV5, 0, len(dbv4.SpaceRows))
	for _, r := range dbv4.SpaceRows {
		db.SpaceRows = append(db.SpaceRows, rowV5{
			TenantID: auth.AnonymousTenantID,
			ID:       r.ID,
			DataURI:  r.DataURI,
			Name:     r.Name,
			ParentID: r.ParentID,
			Storage:  r.Storage,
		})
	}
	data, err := json.Marshal(db)
	return 5, data, err
}

type dbdataV5 struct {
	Version   int     `json:"version"`
	SpaceRows []rowV5 `json:"space_rows"`
}

type rowV5 struct {
	ID       api.SpaceID       `json:"id"`
	DataURI  iosrc.URI         `json:"data_uri"`
	Name     string            `json:"name"`
	ParentID api.SpaceID       `json:"parent_id"`
	Storage  api.StorageConfig `json:"storage"`
	TenantID auth.TenantID     `json:"tenant_id"`
}

func migrateV6(data []byte) (int, []byte, error) {
	var dbv5 dbdataV5
	if err := json.Unmarshal(data, &dbv5); err != nil {
		return 0, nil, err
	}
	var db dbdata
	db.Version = 6
	db.SpaceRows = make([]schema.SpaceRow, 0, len(dbv5.SpaceRows))
	for _, r := range dbv5.SpaceRows {
		db.SpaceRows = append(db.SpaceRows, schema.SpaceRow{
			ID:       r.ID,
			DataURI:  r.DataURI,
			Name:     r.Name,
			Storage:  r.Storage,
			TenantID: r.TenantID,
		})
	}
	data, err := json.Marshal(db)
	return 6, data, err
}

type dbdataV6 struct {
	Version   int               `json:"version"`
	SpaceRows []schema.SpaceRow `json:"space_rows"`
}

func migrateV7(data []byte) (int, []byte, error) {
	var dbv6 dbdataV6
	if err := json.Unmarshal(data, &dbv6); err != nil {
		return 0, nil, err
	}
	var db dbdata
	db.Version = 7
	db.SpaceRows = dbv6.SpaceRows
	data, err := json.Marshal(db)
	return 7, data, err
}
