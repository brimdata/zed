package space

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/brimsec/zq/pkg/nano"
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

type archiveSubspace struct {
	spaceBase
	parent *archiveSpace
}

func (s *archiveSubspace) Info(ctx context.Context) (api.SpaceInfo, error) {
	sum, err := s.store.Summary(ctx)
	if err != nil {
		return api.SpaceInfo{}, err
	}
	var span *nano.Span
	if sum.Span.Dur > 0 {
		span = &sum.Span
	}
	var name string
	err = s.findConfig(func(i int) error {
		name = s.parent.conf.Subspaces[i].Name
		return nil
	})
	if err != nil {
		return api.SpaceInfo{}, err
	}
	spaceInfo := api.SpaceInfo{
		ID:          s.id,
		Name:        name,
		DataPath:    s.parent.conf.DataPath,
		StorageKind: sum.Kind,
		Size:        sum.DataBytes,
		Span:        span,
	}
	return spaceInfo, nil
}

func (s *archiveSubspace) Update(req api.SpacePutRequest) error {
	if req.Name == "" {
		return zqe.E(zqe.Invalid, "cannot set name to an empty string")
	}

	return s.findConfig(func(i int) error {
		s.parent.conf.Subspaces[i].Name = req.Name
		return s.parent.conf.save(s.parent.path)
	})
}

func (s *archiveSubspace) delete() error {
	if err := s.sg.acquireForDelete(); err != nil {
		return err
	}

	return s.findConfig(func(i int) error {
		s.parent.conf.Subspaces = append(s.parent.conf.Subspaces[:i], s.parent.conf.Subspaces[i+1:]...)
		return s.parent.conf.save(s.parent.path)
	})
}

func (s *archiveSubspace) findConfig(fn func(i int) error) error {
	s.parent.muConf.Lock()
	defer s.parent.muConf.Unlock()

	for i := range s.parent.conf.Subspaces {
		if s.parent.conf.Subspaces[i].ID == s.id {
			return fn(i)
		}
	}
	// should not happen
	return fmt.Errorf("subspace %s cannot find conf in parent %s", s.id, s.parent.id)
}
