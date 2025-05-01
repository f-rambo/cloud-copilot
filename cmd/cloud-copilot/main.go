package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"runtime"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/internal/data"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	_ "github.com/joho/godotenv/autoload"
	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "configs", "config path, eg: -conf config.yaml")
}

func newApp(ctx context.Context, logger log.Logger, gs *grpc.Server, hs *http.Server, dr *data.Data) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Context(ctx),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{
			utils.ServiceNameKey.String():    Name,
			utils.ServiceVersionKey.String(): Version,
			utils.RuntimeKey.String():        runtime.Version(),
			utils.OSKey.String():             runtime.GOOS,
			utils.ArchKey.String():           runtime.GOARCH,
			utils.ConfKey.String():           flagconf,
			utils.ConfDirKey.String():        filepath.Dir(flagconf),
		}),
		kratos.Logger(logger),
		kratos.Server(gs, hs, dr),
	)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			panic(r)
		}
	}()

	// config
	flag.Parse()
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	Name = bc.Server.Name
	Version = bc.Server.Version

	// logger
	utilLog := utils.NewLog(&bc)
	defer utilLog.Close()
	logger := log.With(utilLog, utilLog.GetLogContenteKeyvals()...)

	log.SetLogger(logger)

	app, cleanup, err := wireApp(
		context.Background(),
		&bc,
		logger,
	)
	if err != nil {
		panic(err)
	}

	defer cleanup()
	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
