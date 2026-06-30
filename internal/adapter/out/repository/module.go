package repository

import "go.uber.org/fx"

var Module = fx.Module(
	"repository",
	fx.Provide(
		NewBunMediaSessionRecordRepository,
		NewMemoryRoomRuntimeRepository,
		NewRedisMediaSessionStateRepository,
		NewRedisMediaServerStateRepository,
	),
)
