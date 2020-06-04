package space

import (
	"context"
	"os"
	"sync"

	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqe"
)

type archiveSpace struct {
	spaceBase

	// confMutex protects changes to configuration changes.
	confMutex sync.Mutex
	path      string
	conf      config
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
	s.confMutex.Lock()
	defer s.confMutex.Unlock()

	if req.Name == "" {
		return zqe.E(zqe.Invalid, "cannot set name to an empty string")
	}

	s.conf.Name = req.Name
	return s.conf.save(s.path)
}

func (s *archiveSpace) delete() error {
	s.confMutex.Lock()
	defer s.confMutex.Unlock()

	if len(s.conf.Subspaces) != 0 {
		return zqe.E(zqe.Conflict, "unable to delete space with subspaces")
	}

	if err := s.sg.acquireForDelete(); err != nil {
		return err
	}
	if err := os.RemoveAll(s.path); err != nil {
		return err
	}
	return os.RemoveAll(s.conf.DataPath)
}
