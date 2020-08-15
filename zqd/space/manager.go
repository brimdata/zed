package space

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"sync"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/s3io"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/storage"
	"github.com/brimsec/zq/zqe"
	"go.uber.org/zap"
)

type Manager struct {
	rootPath iosrc.URI
	spacesMu sync.Mutex
	spaces   map[api.SpaceID]Space
	names    map[string]api.SpaceID
	logger   *zap.Logger
}

func NewManager(root iosrc.URI, logger *zap.Logger) (*Manager, error) {
	mgr := &Manager{
		rootPath: root,
		spaces:   make(map[api.SpaceID]Space),
		names:    make(map[string]api.SpaceID),
		logger:   logger,
	}
	var err error
	var dirs []iosrc.URI
	switch root.Scheme {
	case "file":
		if dirs, err = filespaces(root); err != nil {
			return nil, err
		}
	case "s3":
		if dirs, err = s3spaces(root); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%s: unsupported scheme", root.Scheme)
	}

	for _, dir := range dirs {
		config, err := loadConfig(dir)
		if err != nil {
			logger.Error("Error loading config", zap.Error(err))
			continue
		}

		spaces, err := loadSpaces(dir, config)
		if err != nil {
			return nil, err
		}
		for _, s := range spaces {
			mgr.spaces[s.ID()] = s
			mgr.names[s.Name()] = s.ID()
		}
	}

	return mgr, nil
}

func s3spaces(root iosrc.URI) ([]iosrc.URI, error) {
	prefixes, err := s3io.ListCommonPrefixes(root.String(), nil)
	if err != nil {
		return nil, err
	}
	var uris []iosrc.URI
	for _, p := range prefixes {
		u := root
		u.Path = p
		uris = append(uris, u)
	}
	return uris, nil
}

func filespaces(root iosrc.URI) ([]iosrc.URI, error) {
	dirs, err := ioutil.ReadDir(root.Filepath())
	if err != nil {
		return nil, err
	}
	var uris []iosrc.URI
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		uris = append(uris, root.AppendPath(dir.Name()))
	}
	return uris, nil
}

func (m *Manager) Create(req api.SpacePostRequest) (Space, error) {
	m.spacesMu.Lock()
	defer m.spacesMu.Unlock()
	if req.Name == "" && req.DataPath == "" {
		return nil, zqe.E(zqe.Invalid, "must supply non-empty name or dataPath")
	}
	var datapath iosrc.URI
	if req.DataPath != "" {
		var err error
		datapath, err = iosrc.ParseURI(req.DataPath)
		if err != nil {
			return nil, err
		}
	}
	// If name is not set then derive name from DataPath, removing and
	// replacing invalid characters.
	if req.Name == "" {
		req.Name = safeName(path.Base(datapath.Path))
		req.Name = uniqueName(m.names, req.Name)
	}
	if err := validateName(m.names, req.Name); err != nil {
		return nil, err
	}
	var storecfg storage.Config
	if req.Storage != nil {
		storecfg = *req.Storage
	}
	if storecfg.Kind == storage.UnknownStore {
		storecfg.Kind = storage.FileStore
	}
	if storecfg.Kind == storage.FileStore && m.rootPath.Scheme != "file" {
		return nil, zqe.E(zqe.Invalid, "cannot create file storage space on non-file backed data path")
	}
	id := newSpaceID()
	path := m.rootPath.AppendPath(string(id))
	src, err := iosrc.GetSource(path)
	if err != nil {
		return nil, err
	}
	if dirmk, ok := src.(iosrc.DirMaker); ok {
		if err := dirmk.MkdirAll(path, 0754); err != nil {
			return nil, err
		}
	}
	if req.DataPath == "" {
		datapath = path
	}
	conf := config{Version: configVersion, Name: req.Name, DataURI: datapath, Storage: storecfg}
	if err := writeConfig(path, conf); err != nil {
		iosrc.RemoveAll(path)
		return nil, err
	}
	spaces, err := loadSpaces(path, conf)
	if err != nil {
		return nil, err
	}
	s := spaces[0]
	m.spaces[s.ID()] = s
	m.names[s.Name()] = s.ID()
	return s, err
}

func (m *Manager) CreateSubspace(parent Space, req api.SubspacePostRequest) (Space, error) {
	m.spacesMu.Lock()
	defer m.spacesMu.Unlock()
	if err := validateName(m.names, req.Name); err != nil {
		return nil, err
	}
	as, ok := parent.(*archiveSpace)
	if !ok {
		return nil, zqe.E(zqe.Invalid, "space does not support creating subspaces")
	}

	s, err := as.CreateSubspace(req)
	if err != nil {
		return nil, err
	}
	m.spaces[s.ID()] = s
	m.names[s.Name()] = s.ID()
	return s, nil
}

func (m *Manager) UpdateSpace(space Space, req api.SpacePutRequest) error {
	m.spacesMu.Lock()
	defer m.spacesMu.Unlock()
	if err := validateName(m.names, req.Name); err != nil {
		return err
	}

	// Right now you can only update a name in a SpacePutRequest but eventually
	// there will be other options.
	oldname := space.Name()
	if oldname == req.Name {
		return nil
	}
	if err := space.update(req); err != nil {
		return err
	}
	delete(m.names, oldname)
	m.names[space.Name()] = space.ID()
	return nil
}

func (m *Manager) Get(id api.SpaceID) (Space, error) {
	m.spacesMu.Lock()
	defer m.spacesMu.Unlock()

	space, exists := m.spaces[id]
	if !exists {
		return nil, ErrSpaceNotExist
	}

	return space, nil
}

func (m *Manager) Delete(id api.SpaceID) error {
	space, err := m.Get(id)
	if err != nil {
		return err
	}

	m.spacesMu.Lock()
	defer m.spacesMu.Unlock()
	name := space.Name()
	if err := space.delete(); err != nil {
		return err
	}

	delete(m.spaces, id)
	delete(m.names, name)
	return nil
}

func (m *Manager) List(ctx context.Context) ([]api.SpaceInfo, error) {
	result := []api.SpaceInfo{}

	m.spacesMu.Lock()
	defer m.spacesMu.Unlock()
	for id := range m.spaces {
		sp := m.spaces[id]
		info, err := sp.Info(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, info)
	}
	return result, nil
}
