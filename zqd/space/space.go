package space

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/pkg/fs"
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
	return filepath.Join(s.path, "config.json")
}

func (s *Space) SetPacketPath(pcapPath string) error {
	s.conf.PacketPath = pcapPath
	return s.conf.save(s.path)
}

func (s Space) PacketPath() string {
	return s.conf.PacketPath
}

type config struct {
	DataPath   string `json:"data_path"`
	PacketPath string `json:"packet_path"`
}

// loadConfig loads the contents of config.json in a space's path.
func loadConfig(name string) (config, error) {
	var c config
	b, err := ioutil.ReadFile(filepath.Join(name, "config.json"))
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	return c, nil
}

func (c config) save(spacePath string) error {
	path := filepath.Join(spacePath, "config.json")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(c)
}
