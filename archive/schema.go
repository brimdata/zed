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

func writeTempFile(dir, pattern string, b []byte) (name string, err error) {
	f, err := ioutil.TempFile(dir, pattern)
	if err != nil {
		return "", err
	}
	_, err = f.Write(b)
	if err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), f.Close()
}

func (c *Config) Write(path string) (err error) {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	tmp, err := writeTempFile(filepath.Dir(path), "."+configName+".*", b)
	if err != nil {
		return err
	}
	err = os.Rename(tmp, path)
	if err != nil {
		os.Remove(tmp)
	}
	return err
}

func ConfigRead(path string) (*Config, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var c Config
	return &c, json.NewDecoder(f).Decode(&c)
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
		return nil, errors.New("no archive directory specified")
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
		return nil, errors.New("no archive directory specified")
	}
	cfgpath := filepath.Join(path, configName)
	if _, err := os.Stat(cfgpath); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0700); err != nil {
				return nil, err
			}
			err = f.Config().Write(cfgpath)
		}
		if err != nil {
			return nil, err
		}
	}
	return OpenArchive(path)
}
