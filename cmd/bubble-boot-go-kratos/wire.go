//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/biz"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/data"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/auth"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/server"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/service"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.App, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		server.ProviderSet,
		service.ProviderSet,
		biz.ProviderSet,
		data.ProviderSet,
		auth.ProviderSet,
		newApp,
	))
}
