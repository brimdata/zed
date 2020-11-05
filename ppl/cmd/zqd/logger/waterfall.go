package logger

import (
	"go.uber.org/multierr"
	"go.uber.org/zap/zapcore"
)

type waterfallCore []zapcore.Core

// NewWaterfall creates a new core that distributes logs to the underlying cores
// in a waterfall pattern; for each log entry the core will iterate through each
// core, in order, stopping when it finds a core that will accept the log entry.
func NewWaterfall(cores ...zapcore.Core) zapcore.Core {
	switch len(cores) {
	case 0:
		return zapcore.NewNopCore()
	case 1:
		return cores[0]
	default:
		return waterfallCore(cores)
	}
}

func (w waterfallCore) With(fields []zapcore.Field) zapcore.Core {
	clone := make(waterfallCore, len(w))
	for i := range w {
		clone[i] = w[i].With(fields)
	}
	return clone
}

func (w waterfallCore) Enabled(lvl zapcore.Level) bool {
	for i := range w {
		if w[i].Enabled(lvl) {
			return true
		}
	}
	return false
}

func (w waterfallCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	for i := range w {
		if out := w[i].Check(ent, nil); out != nil {
			return ce.AddCore(ent, w[i])
		}
	}
	return ce
}

func (w waterfallCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	var err error
	for i := range w {
		err = multierr.Append(err, w[i].Write(ent, fields))
	}
	return err
}

func (w waterfallCore) Sync() error {
	var err error
	for i := range w {
		err = multierr.Append(err, w[i].Sync())
	}
	return err
}
