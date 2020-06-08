package space

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqe"
)

const PcapIndexFile = "packets.idx.json"

type fileSpace struct {
	spaceBase
	path string
	conf config
}

func (s *fileSpace) Info(ctx context.Context) (api.SpaceInfo, error) {
	si, err := s.spaceBase.Info(ctx)
	if err != nil {
		return api.SpaceInfo{}, err
	}
	pcapsize, err := s.PcapSize()
	if err != nil {
		return api.SpaceInfo{}, err
	}

	si.Name = s.conf.Name
	si.DataPath = s.conf.DataPath
	si.PcapSize = pcapsize
	si.PcapSupport = s.PcapPath() != ""
	si.PcapPath = s.PcapPath()
	return si, nil
}

func (s *fileSpace) Update(req api.SpacePutRequest) error {
	if req.Name == "" {
		return zqe.E(zqe.Invalid, "cannot set name to an empty string")
	}
	s.conf.Name = req.Name
	return s.conf.save(s.path)
}

func (s *fileSpace) delete() error {
	if err := s.sg.acquireForDelete(); err != nil {
		return err
	}
	if err := os.RemoveAll(s.path); err != nil {
		return err
	}
	return os.RemoveAll(s.conf.DataPath)
}

func (s *fileSpace) PcapIndexPath() string {
	return filepath.Join(s.conf.DataPath, PcapIndexFile)
}

// PcapSize returns the size in bytes of the packet capture in the space.
func (s *fileSpace) PcapSize() (int64, error) {
	return filesize(s.PcapPath())
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

func (s *fileSpace) SetPcapPath(pcapPath string) error {
	s.conf.PcapPath = pcapPath
	return s.conf.save(s.path)
}

func (s *fileSpace) PcapPath() string {
	return s.conf.PcapPath
}
