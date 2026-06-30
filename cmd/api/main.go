package main

import (
	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/adapter/in/http"
	"github.com/kyh0703/portfoilo-media/internal/adapter/in/http/middleware"
	"github.com/kyh0703/portfoilo-media/internal/adapter/in/lifecycle"
	"github.com/kyh0703/portfoilo-media/internal/adapter/out/auth"
	"github.com/kyh0703/portfoilo-media/internal/adapter/out/persistence"
	"github.com/kyh0703/portfoilo-media/internal/adapter/out/repository"
	rtc "github.com/kyh0703/portfoilo-media/internal/adapter/out/webrtc"
	pkg "github.com/kyh0703/portfoilo-media/internal/pkg"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		configs.Module,
		pkg.Module,
		auth.Module,
		rtc.Module,
		persistence.Module,
		repository.Module,
		lifecycle.Module,
		middleware.Module,
		handler.Module,
		Module,
	).Run()
}
