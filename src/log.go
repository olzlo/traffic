package src

import (
	"github.com/uber-go/zap"
	"github.com/uber-go/zap/zapcore"
)

var logger *zap.Logger = NewLogger()

func NewLogger() *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.Sampling = nil
	//cfg.OutputPaths = []string{"stdout"}
	return cfg.Build(zap.AddCaller())
}
