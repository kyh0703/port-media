package main

import (
	"github.com/kyh0703/portfoilo-media/configs"
	sessionservice "github.com/kyh0703/portfoilo-media/internal/core/service/session"
)

func NewSessionServiceOptions(cfg *configs.Config) sessionservice.ServiceOptions {
	if cfg == nil {
		return sessionservice.ServiceOptions{}
	}
	return sessionservice.ServiceOptions{
		RuntimeEventHistoryLimit: cfg.Realtime.RuntimeEventHistoryLimit,
	}
}
