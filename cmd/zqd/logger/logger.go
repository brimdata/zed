package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Path string `yaml:"path"`
	// If Path is a file, Mode will determine how the log file is managed.
	// FileModeAppend is the default if value is undefined.
	Mode  FileMode      `yaml:"mode,omitempty"`
	Name  string        `yaml:"name"`
	Level zapcore.Level `yaml:"level"`
}

func New(confs ...Config) (zapcore.Core, error) {
	var cores []zapcore.Core
	for _, c := range confs {
		core, err := newCore(c)
		if err != nil {
			return nil, err
		}
		cores = append(cores, core)
	}
	return zapcore.NewTee(cores...), nil
}

func newCore(conf Config) (zapcore.Core, error) {
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
