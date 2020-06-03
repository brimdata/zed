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

	// confmu protects changes to configuration changes.
	confmu sync.Mutex
	conf   config
}

func (s *archiveSpace) Info(ctx context.Context) (api.SpaceInfo, error) {
	si, err := s.spaceBase.Info(ctx)
	if err != nil {
		return api.SpaceInfo{}, err
	}

	s.confmu.Lock()
	defer s.confmu.Unlock()
	si.Name = s.conf.Name
	si.DataPath = s.conf.DataPath
	return si, nil
}

func (s *archiveSpace) Name() string {
	s.confmu.Lock()
	defer s.confmu.Unlock()
	return s.conf.Name
}

func (s *archiveSpace) update(req api.SpacePutRequest) error {
	if req.Name == "" {
		return zqe.E(zqe.Invalid, "cannot set name to an empty string")
	}

	s.confmu.Lock()
	defer s.confmu.Unlock()

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

func (s *archiveSpace) delete() error {
	s.confmu.Lock()
	defer s.confmu.Unlock()

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
	s.confmu.Lock()
	defer s.confmu.Unlock()

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
	newconf := s.conf.clone()
	newconf.Subspaces = append(newconf.Subspaces, subcfg)
	if err := s.updateConfigWithLock(newconf); err != nil {
		return nil, err
	}

	return &archiveSubspace{
		spaceBase: spaceBase{subcfg.ID, substore, newGuard()},
		parent:    s,
	}, nil
}
