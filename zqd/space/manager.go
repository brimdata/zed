package space

import (
	"context"
	"errors"
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
	mapLock  sync.Mutex
	spaces   map[api.SpaceID]Space
	logger   *zap.Logger
}

func NewManager(root string, logger *zap.Logger) (*Manager, error) {
	mgr := &Manager{
		rootPath: root,
		spaces:   make(map[api.SpaceID]Space),
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

		spaces, err := loadSpaces(path, config)
		if err != nil {
			return nil, err
		}
		for _, s := range spaces {
			mgr.spaces[s.ID()] = s
		}
	}

	return mgr, nil
}

func (m *Manager) Create(req api.SpacePostRequest) (Space, error) {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()

	if req.Name == "" && req.DataPath == "" {
		return nil, zqe.E(zqe.Invalid, "must supply non-empty name or dataPath")
	}

	var storecfg storage.Config
	if req.Storage != nil {
		storecfg = *req.Storage
	}
	if storecfg.Kind == storage.UnknownStore {
		storecfg.Kind = storage.FileStore
	}

	if req.Name == "" {
		req.Name = filepath.Base(req.DataPath)
	}
	id := newSpaceID()
	path := filepath.Join(m.rootPath, string(id))
	if err := os.Mkdir(path, 0755); err != nil {
		return nil, err
	}
	if req.DataPath == "" {
		req.DataPath = path
	}
	c := config{
		Name:     req.Name,
		DataPath: req.DataPath,
		Storage:  storecfg,
	}
	if err := c.save(path); err != nil {
		os.RemoveAll(path)
		return nil, err
	}

	if _, exists := m.spaces[id]; exists {
		m.logger.Error("created duplicate space id", zap.String("id", string(id)))
		return nil, errors.New("created duplicate space id (this should not happen)")
	}

	spaces, err := loadSpaces(path, c)
	if err != nil {
		return nil, err
	}
	if len(spaces) != 1 {
		panic("multiple spaces created during space create")
	}
	m.spaces[id] = spaces[0]
	return spaces[0], nil
}

func (m *Manager) CreateSubspace(parent Space, req api.SubspacePostRequest) (Space, error) {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()

	as, ok := parent.(*archiveSpace)
	if !ok {
		return nil, zqe.E(zqe.Invalid, "space does not support creating subspaces")
	}
	s, err := as.CreateSubspace(req)
	if err != nil {
		return nil, err
	}
	m.spaces[s.ID()] = s
	return s, nil
}

func (m *Manager) Get(id api.SpaceID) (Space, error) {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()
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

	if err := space.delete(); err != nil {
		return err
	}

	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	delete(m.spaces, id)
	return nil
}

func (m *Manager) List(ctx context.Context) ([]api.SpaceInfo, error) {
	result := []api.SpaceInfo{}

	m.mapLock.Lock()
	defer m.mapLock.Unlock()
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
