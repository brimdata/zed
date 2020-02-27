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
	path     string
	dataPath string
}

func Open(root, name string) (*Space, error) {
	path := filepath.Join(root, name)
	dataPath, err := loadConfig(filepath.Join(root, name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSpaceNotExist
		}
		return nil, err
	}
	return &Space{path: path, dataPath: dataPath}, nil
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
	if err := saveConfig(path, dataPath); err != nil {
		os.RemoveAll(path)
		return nil, err
	}
	return &Space{path, dataPath}, nil
}

func (s Space) Name() string {
	return filepath.Base(s.path)
}

func (s Space) DataPath(elem ...string) string {
	return filepath.Join(append([]string{s.dataPath}, elem...)...)
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

type config struct {
	DataPath string `json:"data_path"`
}

// loadConfig loads the contents of config.json. Currently DataPath is the only
// attribute in Config, so only return the DataPath field.
func loadConfig(name string) (string, error) {
	b, err := ioutil.ReadFile(filepath.Join(name, "config.json"))
	if err != nil {
		return "", err
	}
	c := config{}
	if err := json.Unmarshal(b, &c); err != nil {
		return "", err
	}
	return c.DataPath, nil
}

func saveConfig(name, dataPath string) error {
	path := filepath.Join(name, "config.json")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(config{dataPath})
}
