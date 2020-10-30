package logger

import (
	"go.uber.org/zap/zapcore"
)

type nameFilterCore struct {
	zapcore.Core
	name string
}

func newNameFilterCore(next zapcore.Core, name string) zapcore.Core {
	return &nameFilterCore{next, name}
}

func (core *nameFilterCore) With(fields []zapcore.Field) zapcore.Core {
	return &nameFilterCore{core.Core.With(fields), core.name}
}

func (core *nameFilterCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if core.name == e.LoggerName {
		return core.Core.Check(e, ce)
	}
	// skip entry
	return ce
}
