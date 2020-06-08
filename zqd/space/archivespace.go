package space

import (
	"context"
	"os"
	"sync"

	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/storage"
	"github.com/brimsec/zq/zqd/storage/archivestore"
	"github.com/brimsec/zq/zqe"
)

type archiveSpace struct {
	spaceBase
	path string

	// muConf protects changes to configuration changes.
	muConf sync.Mutex
	conf   config
}

func (s *archiveSpace) Info(ctx context.Context) (api.SpaceInfo, error) {
	si, err := s.spaceBase.Info(ctx)
	if err != nil {
		return api.SpaceInfo{}, err
	}

	si.Name = s.conf.Name
	si.DataPath = s.conf.DataPath
	return si, nil
}

func (s *archiveSpace) Update(req api.SpacePutRequest) error {
	if req.Name == "" {
		return zqe.E(zqe.Invalid, "cannot set name to an empty string")
	}

	s.muConf.Lock()
	defer s.muConf.Unlock()

	s.conf.Name = req.Name
	return s.conf.save(s.path)
}

func (s *archiveSpace) delete() error {
	s.muConf.Lock()
	defer s.muConf.Unlock()

	if len(s.conf.Subspaces) != 0 {
		return zqe.E(zqe.Conflict, "cannot delete space with subspaces")
	}

	if err := s.sg.acquireForDelete(); err != nil {
		return err
	}
	if err := os.RemoveAll(s.path); err != nil {
		return err
	}
	return os.RemoveAll(s.conf.DataPath)
}

func (s *archiveSpace) CreateSubspace(req api.SubspacePostRequest) (*archiveSubspace, error) {
	if req.Name == "" {
		return nil, zqe.E(zqe.Invalid, "cannot set name to an empty string")
	}

	s.muConf.Lock()
	defer s.muConf.Unlock()

	substore, err := archivestore.Load(s.conf.DataPath, &storage.ArchiveConfig{
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
	s.conf.Subspaces = append(s.conf.Subspaces, subcfg)
	if err := s.conf.save(s.path); err != nil {
		return nil, err
	}

	return &archiveSubspace{
		spaceBase: spaceBase{subcfg.ID, substore, newGuard()},
		parent:    s,
	}, nil
}
