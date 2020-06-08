package space

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqe"
	"go.uber.org/zap"
)

type Manager struct {
	rootPath string
	spacesMu sync.Mutex
	spaces   map[api.SpaceID]*Space
	logger   *zap.Logger
}

func NewManager(root string, logger *zap.Logger) (*Manager, error) {
	mgr := &Manager{
		rootPath: root,
		spaces:   make(map[api.SpaceID]*Space),
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

		space, err := loadSpace(path, config)
		if err != nil {
			return nil, err
		}
		mgr.spaces[space.ID()] = space
	}

	return mgr, nil
}

func (m *Manager) Create(name, dataPath string) (*Space, error) {
	m.spacesMu.Lock()
	defer m.spacesMu.Unlock()

	if name == "" && dataPath == "" {
		return nil, zqe.E(zqe.Invalid, "must supply non-empty name or dataPath")
	}
	if name == "" {
		name = filepath.Base(dataPath)
	}
	id := newSpaceID()
	path := filepath.Join(m.rootPath, string(id))
	if err := os.Mkdir(path, 0755); err != nil {
		return nil, err
	}
	if dataPath == "" {
		dataPath = path
	}
	c := config{
		Name:     name,
		DataPath: dataPath,
	}
	if err := c.save(path); err != nil {
		os.RemoveAll(path)
		return nil, err
	}

	if _, exists := m.spaces[id]; exists {
		m.logger.Error("created duplicate space id", zap.String("id", string(id)))
		return nil, errors.New("created duplicate space id (this should not happen)")
	}

	sp, err := loadSpace(path, c)
	if err != nil {
		return nil, err
	}
	m.spaces[id] = sp
	return sp, nil
}

func (m *Manager) Get(id api.SpaceID) (*Space, error) {
	m.spacesMu.Lock()
	defer m.spacesMu.Unlock()
	space, exists := m.spaces[id]
	if !exists {
		return nil, ErrSpaceNotExist
	}

	return space, nil
}

func (m *Manager) Delete(id api.SpaceID) error {
	m.spacesMu.Lock()
	defer m.spacesMu.Unlock()

	space, exists := m.spaces[id]
	if !exists {
		return ErrSpaceNotExist
	}

	err := space.delete()
	if err != nil {
		return err
	}

	delete(m.spaces, id)
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
