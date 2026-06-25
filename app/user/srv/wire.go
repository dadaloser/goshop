//go:build wireinject
// +build wireinject

package srv

import (
	"goshop/app/pkg/options"
	"goshop/app/user/srv/internal/controller/user"
	"goshop/app/user/srv/internal/data/v1/db"
	"goshop/app/user/srv/internal/service/v1"
	gapp "goshop/gmicro/app"
	"goshop/pkg/log"

	"github.com/google/wire"
)

func initApp(*options.NacosOptions, *log.Options, *options.ServerOptions, *options.RegistryOptions, *options.TelemetryOptions, *options.MySQLOptions) (*gapp.App, error) {
	wire.Build(ProviderSet, v1.ProviderSet, db.ProviderSet, user.ProviderSet)
	return &gapp.App{}, nil
}
