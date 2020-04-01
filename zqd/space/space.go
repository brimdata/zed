package space

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zqd/api"
)

const (
	configFile  = "config.json"
	infoFile    = "info.json"
	AllBzngFile = "all.bzng"
)

var (
	ErrSpaceNotExist = errors.New("space does not exist")
	ErrSpaceExists   = errors.New("space exists")
)

type Space struct {
	path string
	conf config
}

func Open(root, name string) (*Space, error) {
	path := filepath.Join(root, name)
	c, err := loadConfig(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSpaceNotExist
		}
		return nil, err
	}
	return &Space{path, c}, nil
}

func Create(root, name, dataPath string) (*Space, error) {
	// XXX this should be validated before reaching here.
	if name == "" && dataPath == "" {
		return nil, errors.New("must supply non-empty name or dataPath")
	}
	var path string
	if name == "" {
		var err error
		if path, err = fs.UniqueDir(root, filepath.Base(dataPath)); err != nil {
			return nil, err
		}
	} else {
		path = filepath.Join(root, name)
		if err := os.Mkdir(path, 0700); err != nil {
			if os.IsExist(err) {
				return nil, ErrSpaceExists
			}
			return nil, err
		}
	}
	if dataPath == "" {
		dataPath = path
	}
	c := config{DataPath: dataPath}
	if err := c.save(path); err != nil {
		os.RemoveAll(path)
		return nil, err
	}
	return &Space{path, c}, nil
}

func (s Space) Name() string {
	return filepath.Base(s.path)
}

func (s Space) Info() (api.SpaceInfo, error) {
	f, err := s.OpenFile(AllBzngFile)
	if err != nil {
		return api.SpaceInfo{}, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return api.SpaceInfo{}, err
	}
	spaceInfo := api.SpaceInfo{
		Name:       s.Name(),
		Size:       stat.Size(),
		PacketPath: s.PacketPath(),
	}
	i, err := loadInfo(s.conf.DataPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return api.SpaceInfo{}, err
		}
		return spaceInfo, nil
	}

	spaceInfo.MinTime = &i.MinTime
	spaceInfo.MaxTime = &i.MaxTime

	return spaceInfo, nil
}

func (s Space) DataPath(elem ...string) string {
	return filepath.Join(append([]string{s.conf.DataPath}, elem...)...)
}

func (s Space) OpenFile(file string) (*os.File, error) {
	return os.Open(s.DataPath(file))
}

func (s Space) CreateFile(file string) (*os.File, error) {
	return os.Create(s.DataPath(file))
}

func (s Space) HasFile(file string) bool {
	info, err := os.Stat(s.DataPath(file))
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func (s Space) ConfigPath() string {
	return filepath.Join(s.path, configFile)
}

func (s *Space) SetPacketPath(pcapPath string) error {
	s.conf.PacketPath = pcapPath
	return s.conf.save(s.path)
}

func (s Space) PacketPath() string {
	return s.conf.PacketPath
}

// Delete removes the space's path and data dir (should the data dir be
// different then the space's path).
func (s Space) Delete() error {
	if err := os.RemoveAll(s.path); err != nil {
		return err
	}
	return os.RemoveAll(s.conf.DataPath)
}

type config struct {
	DataPath   string `json:"data_path"`
	PacketPath string `json:"packet_path"`
}

type info struct {
	MinTime nano.Ts `json:"min_time"`
	MaxTime nano.Ts `json:"max_time"`
}

func (s Space) SetTimes(minTs, maxTs nano.Ts) error {
	cur, err := loadInfo(s.conf.DataPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		cur = info{nano.MaxTs, nano.MinTs}
	}
	cur.MinTime = nano.Min(cur.MinTime, minTs)
	cur.MaxTime = nano.Max(cur.MaxTime, maxTs)
	return cur.save(s.conf.DataPath)
}

func (s Space) GetTimes() (*nano.Ts, *nano.Ts, error) {
	i, err := loadInfo(s.conf.DataPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, nil, err
		}
		return nil, nil, nil
	}
	return &i.MinTime, &i.MaxTime, nil
}

// loadConfig loads the contents of config.json in a space's path.
func loadConfig(spacePath string) (config, error) {
	var c config
	b, err := ioutil.ReadFile(filepath.Join(spacePath, configFile))
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	return c, nil
}

func (c config) save(spacePath string) error {
	path := filepath.Join(spacePath, configFile)
	tmppath := path + ".tmp"
	f, err := os.Create(tmppath)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(c); err != nil {
		f.Close()
		os.Remove(tmppath)
		return err
	}
	if err = f.Close(); err != nil {
		os.Remove(tmppath)
		return err
	}
	return os.Rename(tmppath, path)
}

func loadInfo(path string) (info, error) {
	var i info
	b, err := ioutil.ReadFile(filepath.Join(path, infoFile))
	if err != nil {
		return info{}, err
	}
	if err := json.Unmarshal(b, &i); err != nil {
		return i, err
	}
	return i, nil
}

func (i info) save(path string) error {
	path = filepath.Join(path, infoFile)
	tmppath := path + ".tmp"
	f, err := os.Create(tmppath)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(i); err != nil {
		f.Close()
		os.Remove(tmppath)
		return err
	}
	if err = f.Close(); err != nil {
		os.Remove(tmppath)
		return err
	}
	return os.Rename(tmppath, path)
}
