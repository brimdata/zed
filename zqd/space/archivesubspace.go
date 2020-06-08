package space

import (
	"context"
	"fmt"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqe"
)

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
