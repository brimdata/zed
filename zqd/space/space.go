package space

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	ErrSpaceNotExist = errors.New("space does not exist")
)

type Space struct {
	name     string
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
	return &Space{name: name, path: path, dataPath: dataPath}, nil
}

func Create(root, name, dataPath string) (*Space, error) {
	path := filepath.Join(root, name)
	if err := os.Mkdir(path, 0755); err != nil {
		return nil, err
	}
	if dataPath == "" {
		dataPath = path
	}
	if err := saveConfig(path, dataPath); err != nil {
		os.RemoveAll(path)
		return nil, err
	}
	return &Space{name, path, dataPath}, nil
}

func (s Space) Name() string {
	return s.name
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
