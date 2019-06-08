package src

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.SugaredLogger

func init() {
	newLogger(zapcore.InfoLevel)
}

func EnableDebug() {
	newLogger(zapcore.DebugLevel)
}

func newLogger(lv zapcore.Level) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.Level = zap.NewAtomicLevelAt(lv)
	logger, err := cfg.Build(zap.AddCaller(), zap.AddStacktrace(zap.PanicLevel))
	if err != nil {
		panic(err)
	}
	Logger = logger.Sugar()
}
