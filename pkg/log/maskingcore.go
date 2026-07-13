package log

import "go.uber.org/zap/zapcore"

type maskingCore struct {
	zapcore.Core
}

func wrapSensitiveFieldCore(core zapcore.Core) zapcore.Core {
	if core == nil {
		return nil
	}
	return maskingCore{Core: core}
}

func (c maskingCore) With(fields []zapcore.Field) zapcore.Core {
	return maskingCore{Core: c.Core.With(sanitizeFields(fields))}
}

func (c maskingCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(entry.Level) {
		return checked
	}
	return checked.AddCore(entry, c)
}

func (c maskingCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return c.Core.Write(entry, sanitizeFields(fields))
}
