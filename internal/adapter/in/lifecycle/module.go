package lifecycle

import (
	sessionservice "github.com/kyh0703/portfoilo-media/internal/core/service/session"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"lifecycle",
	fx.Provide(
		NewMediaServerStateReporterOptions,
		sessionservice.NewMediaServerStateReporter,
		NewIdleRoomCleanupScheduler,
		NewMediaServerStateScheduler,
	),
	fx.Invoke(
		RegisterIdleRoomCleanup,
		RegisterMediaServerStateScheduler,
	),
)
