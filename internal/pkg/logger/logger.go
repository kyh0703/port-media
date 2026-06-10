package logger

import (
	"fmt"

	"github.com/kyh0703/portfoilo-media/configs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(cfg *configs.Config) (*zap.Logger, error) {
	zapCfg := zap.NewProductionConfig()
	if cfg.Log.Development {
		zapCfg = zap.NewDevelopmentConfig()
	}

	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Log.Level)); err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}

	zapCfg.Level = zap.NewAtomicLevelAt(level)
	zapCfg.InitialFields = map[string]any{
		"service": cfg.App.Name,
		"env":     cfg.App.Env,
		"version": cfg.App.Version,
	}
	return zapCfg.Build()
}
