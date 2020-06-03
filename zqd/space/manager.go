package space

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/storage"
	"github.com/brimsec/zq/zqe"
	"go.uber.org/zap"
)

type Manager struct {
	rootPath string
	spacesMu sync.Mutex
	spaces   map[api.SpaceID]Space
	names    map[string]api.SpaceID
	logger   *zap.Logger
}

func NewManager(root string, logger *zap.Logger) (*Manager, error) {
	mgr := &Manager{
		rootPath: root,
		spaces:   make(map[api.SpaceID]Space),
		names:    make(map[string]api.SpaceID),
		logger:   logger,
	}

	dirs, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		path := filepath.Join(root, dir.Name())
		config, err := loadConfig(path)
		if err != nil {
			logger.Error("Error loading config", zap.Error(err))
			continue
		}

		if config.Version < 1 {
			config.Name = safeName(mgr.names, config.Name)
			for _, sub := range config.Subspaces {
				sub.Name = safeName(mgr.names, sub.Name)
			}
			config.Version = configVersion
			if err := writeConfig(path, config); err != nil {
				logger.Error("error migrating config", zap.Error(err))
				continue
			}
		}

		spaces, err := loadSpaces(path, config)
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

func (m *Manager) Create(req api.SpacePostRequest) (Space, error) {
	m.spacesMu.Lock()
	defer m.spacesMu.Unlock()
	if req.Name == "" && req.DataPath == "" {
		return nil, zqe.E(zqe.Invalid, "must supply non-empty name or dataPath")
	}
	// If name is not set then derrive name from DataPath; removing and
	// replacing invalid characters.
	if req.Name == "" {
		req.Name = safeName(m.names, req.DataPath)
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
	id := newSpaceID()
	path := filepath.Join(m.rootPath, string(id))
	if err := os.Mkdir(path, 0755); err != nil {
		return nil, err
	}
	if req.DataPath == "" {
		req.DataPath = path
	}
	conf := config{Name: req.Name, DataPath: req.DataPath, Storage: storecfg}
	if err := writeConfig(path, conf); err != nil {
		os.RemoveAll(path)
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
