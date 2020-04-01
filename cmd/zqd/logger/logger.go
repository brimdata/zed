package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	TypeFile      Type = "file"
	TypeWaterfall      = "waterfall"
	TypeTee            = "tee"
)

type Type string

type Config struct {
	Path string `yaml:"path"`
	// If Path is a file, Mode will determine how the log file is managed.
	// FileModeAppend is the default if value is undefined.
	Mode  FileMode      `yaml:"mode,omitempty"`
	Name  string        `yaml:"name"`
	Level zapcore.Level `yaml:"level"`
	// Type specifies how log entries are handled. If Type is Waterfall or Tee
	// this will distribute log entries to child loggers; fields Mode, Name,
	// and Level will be ignored. If the value is Sink or empty the Children
	// field will be ignored.
	Type Type `yaml:"type,omitempty"`
	// Specifies underlying children loggers. Only applicable when Type is
	// TypeWaterfall or TypeTee.
	Children []Config `yaml:"children,omitempty"`
}

func NewCore(conf Config) (zapcore.Core, error) {
	switch conf.Type {
	case TypeFile, "":
		return newFileCore(conf)
	case TypeWaterfall:
		return newWaterfallCore(conf.Children...)
	case TypeTee:
		return newTeeCore(conf.Children...)
	default:
		return nil, fmt.Errorf("unsupported logger type: %s", conf.Type)
	}
}

func newCores(confs ...Config) ([]zapcore.Core, error) {
	var cores []zapcore.Core
	for _, c := range confs {
		core, err := NewCore(c)
		if err != nil {
			return nil, err
		}
		cores = append(cores, core)
	}
	return cores, nil
}

func newTeeCore(confs ...Config) (zapcore.Core, error) {
	cores, err := newCores(confs...)
	return zapcore.NewTee(cores...), err
}

func newWaterfallCore(conf ...Config) (zapcore.Core, error) {
	cores, err := newCores(conf...)
	return NewWaterfall(cores...), err
}

func newFileCore(conf Config) (zapcore.Core, error) {
	w, err := OpenFile(conf.Path, conf.Mode)
	if err != nil {
		return nil, err
	}
	core := zapcore.NewCore(jsonEncoder(), w, conf.Level)
	if conf.Name != "" {
		core = newNameFilterCore(core, conf.Name)
	}
	return core, nil
}

func jsonEncoder() zapcore.Encoder {
	conf := zap.NewProductionEncoderConfig()
	conf.CallerKey = ""
	return zapcore.NewJSONEncoder(conf)
}
