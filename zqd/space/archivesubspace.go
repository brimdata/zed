package space

import (
	"context"
	"fmt"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zqd/api"
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
	var dp iosrc.URI
	err = s.findConfig(func(i int) error {
		name = s.parent.conf.Subspaces[i].Name
		dp = s.parent.conf.DataURI
		return nil
	})
	return api.SpaceInfo{
		ID:          s.id,
		Name:        name,
		DataPath:    dp,
		StorageKind: sum.Kind,
		Size:        sum.DataBytes,
		Span:        span,
		ParentID:    s.parent.ID(),
	}, err
}

func (s *archiveSubspace) update(req api.SpacePutRequest) error {
	return s.findConfig(func(i int) error {
		conf := s.parent.conf.clone()
		conf.Subspaces[i].Name = req.Name
		return s.parent.updateConfigWithLock(conf)
	})
}

func (s *archiveSubspace) delete() error {
	if err := s.sg.acquireForDelete(); err != nil {
		return err
	}

	return s.findConfig(func(i int) error {
		conf := s.parent.conf.clone()
		conf.Subspaces = append(conf.Subspaces[:i], conf.Subspaces[i+1:]...)
		return s.parent.updateConfigWithLock(conf)
	})
}

func (s *archiveSubspace) Name() string {
	var name string
	err := s.findConfig(func(i int) error {
		name = s.parent.conf.Subspaces[i].Name
		return nil
	})
	if err != nil {
		panic(err)
	}
	return name
}

func (s *archiveSubspace) findConfig(fn func(int) error) error {
	s.parent.confMu.Lock()
	defer s.parent.confMu.Unlock()
	i := s.parent.conf.subspaceIndex(s.id)
	if i == -1 {
		// should not happen
		return fmt.Errorf("subspace %s cannot find conf in parent %s", s.id, s.parent.id)
	}
	return fn(i)
}
