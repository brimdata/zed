package space

import (
	"context"
	"sync"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zqd/storage"
	"github.com/brimsec/zq/zqd/storage/archivestore"
	"github.com/brimsec/zq/zqe"
	"go.uber.org/zap"
)

type archiveSpace struct {
	spaceBase
	path iosrc.URI

	confMu sync.Mutex
	conf   config
}

func (s *archiveSpace) Info(ctx context.Context) (api.SpaceInfo, error) {
	si, err := s.spaceBase.Info(ctx)
	if err != nil {
		return api.SpaceInfo{}, err
	}

	s.confMu.Lock()
	defer s.confMu.Unlock()
	si.Name = s.conf.Name
	si.DataPath = s.conf.DataURI
	return si, nil
}

func (s *archiveSpace) Name() string {
	s.confMu.Lock()
	defer s.confMu.Unlock()
	return s.conf.Name
}

func (s *archiveSpace) update(req api.SpacePutRequest) error {
	if req.Name == "" {
		return zqe.E(zqe.Invalid, "cannot set name to an empty string")
	}

	s.confMu.Lock()
	defer s.confMu.Unlock()

	conf := s.conf.clone()
	conf.Name = req.Name
	return s.updateConfigWithLock(conf)
}

func (s *archiveSpace) updateConfigWithLock(conf config) error {
	if err := writeConfig(s.path, conf); err != nil {
		return err
	}
	s.conf = conf
	return nil
}

func (s *archiveSpace) delete(ctx context.Context) error {
	s.confMu.Lock()
	defer s.confMu.Unlock()

	if len(s.conf.Subspaces) != 0 {
		return zqe.E(zqe.Conflict, "cannot delete space with subspaces")
	}

	if err := s.sg.acquireForDelete(); err != nil {
		return err
	}
	if err := iosrc.RemoveAll(ctx, s.path); err != nil {
		return err
	}
	return iosrc.RemoveAll(ctx, s.conf.DataURI)
}

func (s *archiveSpace) CreateSubspace(ctx context.Context, req api.SubspacePostRequest) (*archiveSubspace, error) {
	s.confMu.Lock()
	defer s.confMu.Unlock()

	substore, err := archivestore.Load(ctx, s.conf.DataURI, &storage.ArchiveConfig{
		OpenOptions: &req.OpenOptions,
	})
	if err != nil {
		return nil, err
	}

	subcfg := subspaceConfig{
		ID:          newSpaceID(),
		Name:        req.Name,
		OpenOptions: req.OpenOptions,
	}
	newconf := s.conf.clone()
	newconf.Subspaces = append(newconf.Subspaces, subcfg)
	if err := s.updateConfigWithLock(newconf); err != nil {
		return nil, err
	}

	logger := s.logger.With(zap.String("space_id", string(subcfg.ID)))
	return &archiveSubspace{
		spaceBase: spaceBase{subcfg.ID, substore, nil, newGuard(), logger},
		parent:    s,
	}, nil
}
