//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/internal/data"
	"github.com/f-rambo/cloud-copilot/internal/interfaces"
	"github.com/f-rambo/cloud-copilot/internal/server"
	thirdpackage "github.com/f-rambo/cloud-copilot/third_package"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Bootstrap, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, thirdpackage.ProviderSet, data.ProviderSet, biz.ProviderSet, interfaces.ProviderSet, newApp))
}