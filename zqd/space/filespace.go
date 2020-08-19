package space

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/pcapstorage"
	"github.com/brimsec/zq/zqe"
)

type fileSpace struct {
	spaceBase
	path iosrc.URI

	confMu sync.Mutex
	conf   config
}

func (s *fileSpace) Info(ctx context.Context) (api.SpaceInfo, error) {
	si, err := s.spaceBase.Info(ctx)
	if err != nil {
		return api.SpaceInfo{}, err
	}
	du := s.conf.DataURI
	if du.IsZero() {
		du = s.path
	}

	si.Name = s.conf.Name
	si.DataPath = du
	return si, nil
}

func (s *fileSpace) PcapStorage() *pcapstorage.Store {
	return s.pcapstore
}

func (s *fileSpace) dataURI() iosrc.URI {
	du := s.conf.DataURI
	if du.IsZero() {
		du = s.path
	}
	return du
}

func (s *fileSpace) Name() string {
	s.confMu.Lock()
	defer s.confMu.Unlock()
	return s.conf.Name
}

func (s *fileSpace) update(req api.SpacePutRequest) error {
	if req.Name == "" {
		return zqe.E(zqe.Invalid, "cannot set name to an empty string")
	}
	s.confMu.Lock()
	defer s.confMu.Unlock()
	conf := s.conf.clone()
	conf.Name = req.Name
	return s.updateConfigWithLock(conf)
}

func (s *fileSpace) updateConfigWithLock(conf config) error {
	if err := writeConfig(s.path, conf); err != nil {
		return err
	}
	s.conf = conf
	return nil
}

func (s *fileSpace) delete() error {
	if err := s.sg.acquireForDelete(); err != nil {
		return err
	}
	if err := iosrc.RemoveAll(s.path); err != nil {
		return err
	}
	return iosrc.RemoveAll(s.conf.DataURI)
}

func filesize(path string) (int64, error) {
	f, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	return f.Size(), nil
}
