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

func (mc waterfallCore) With(fields []zapcore.Field) zapcore.Core {
	clone := make(waterfallCore, len(mc))
	for i := range mc {
		clone[i] = mc[i].With(fields)
	}
	return clone
}

func (mc waterfallCore) Enabled(lvl zapcore.Level) bool {
	for i := range mc {
		if mc[i].Enabled(lvl) {
			return true
		}
	}
	return false
}

func (mc waterfallCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	for i := range mc {
		if out := mc[i].Check(ent, nil); out != nil {
			return ce.AddCore(ent, mc[i])
		}
	}
	return ce
}

func (mc waterfallCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	var err error
	for i := range mc {
		err = multierr.Append(err, mc[i].Write(ent, fields))
	}
	return err
}

func (mc waterfallCore) Sync() error {
	var err error
	for i := range mc {
		err = multierr.Append(err, mc[i].Sync())
	}
	return err
}
