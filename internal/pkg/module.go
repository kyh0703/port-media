package pkg

import (
	"github.com/kyh0703/portfoilo-media/internal/pkg/cache"
	"github.com/kyh0703/portfoilo-media/internal/pkg/health"
	"github.com/kyh0703/portfoilo-media/internal/pkg/logger"
	"github.com/kyh0703/portfoilo-media/internal/pkg/validate"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"pkg",
	fx.Provide(
		cache.NewRedisClient,
		health.NewChecker,
		logger.NewLogger,
		validate.NewValidator,
	),
)
