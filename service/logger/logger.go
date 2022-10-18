package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Path string `yaml:"path"`
	// If Path is a file, Mode will determine how the log file is managed.
	// FileModeAppend is the default if value is undefined.
	Mode    FileMode      `yaml:"mode,omitempty"`
	Level   zapcore.Level `yaml:"level"`
	DevMode bool          `yaml:"devmode"`
}

func New(conf Config) (*zap.Logger, error) {
	w, err := OpenFile(conf.Path, conf.Mode)
	if err != nil {
		return nil, err
	}
	core := zapcore.NewCore(jsonEncoder(), w, conf.Level)
	opts := []zap.Option{zap.AddStacktrace(zapcore.WarnLevel)}
	// If the development mode is on, calls to logger.DPanic will cause a panic
	// whereas in production would result in an error.
	if conf.DevMode {
		opts = append(opts, zap.Development())
	}
	return zap.New(core, opts...), nil
}

func jsonEncoder() zapcore.Encoder {
	conf := zap.NewProductionEncoderConfig()
	conf.CallerKey = ""
	return zapcore.NewJSONEncoder(conf)
}
