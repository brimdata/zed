package archive

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/pkg/fs"
)

const configName = "zar.json"

var DefaultConfig Config = Config{
	Version:          0,
	LogSizeThreshold: 500 * 1024 * 1024,
}

type Config struct {
	Version          int `json:"version"`
	LogSizeThreshold int `json:"log_size_threshold"`
}

func (c *Config) Write(path string) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	f, err := ioutil.TempFile(filepath.Dir(path), "."+configName+".*")
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err != nil {
		f.Close()
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func ConfigRead(path string) (*Config, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var c Config
	err = json.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

type CreateFlags struct {
	megaThresh int
	byteThresh int
}

func (f *CreateFlags) SetFlags(fs *flag.FlagSet) {
	fs.IntVar(&f.megaThresh, "s", DefaultConfig.LogSizeThreshold/(1024*1024), "target size of chopped files in MiB")
	fs.IntVar(&f.byteThresh, "b", 0, "target size of chopped files in bytes (overrides -s)")
}

func (f *CreateFlags) Config() *Config {
	cfg := DefaultConfig

	thresh := f.byteThresh
	if thresh == 0 {
		thresh = f.megaThresh * 1024 * 1024
	}
	cfg.LogSizeThreshold = thresh

	return &cfg
}

type Archive struct {
	Config *Config
	Root   string
}

func OpenArchive(path string) (*Archive, error) {
	if path == "" {
		return nil, errors.New("no ZAR_ROOT directory specified")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("ZAR_ROOT not a directory")
	}

	c, err := ConfigRead(filepath.Join(path, configName))
	if err != nil {
		return nil, err
	}

	return &Archive{
		Config: c,
		Root:   path,
	}, nil
}

func CreateOrOpenArchive(path string, f CreateFlags) (*Archive, error) {
	if path == "" {
		return nil, errors.New("no ZAR_ROOT directory specified")
	}
	if info, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path, 0700)
			if err != nil {
				return nil, err
			}
		}
	} else if !info.IsDir() {
		return nil, errors.New("ZAR_ROOT not a directory")
	}

	cfgpath := filepath.Join(path, configName)
	if _, err := os.Stat(cfgpath); err != nil {
		if os.IsNotExist(err) {
			err = f.Config().Write(cfgpath)
			if err != nil {
				return nil, err
			}
		}
	}
	return OpenArchive(path)
}
