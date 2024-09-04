package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/internal/server"
	"github.com/f-rambo/ocean/utils"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "github.com/joho/godotenv/autoload"
	_ "go.uber.org/automaxprocs"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string
	// shipVersion is the version of the ship.
	shipVersion string = "0.0.1"

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "configs", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server, internalLogic *server.InternalLogic) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{
			"service":      Name,
			"version":      Version,
			"ship_version": shipVersion,
			"runtime":      runtime.Version(),
			"os":           runtime.GOOS,
			"arch":         runtime.GOARCH,
			"conf":         flagconf,
		}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
			internalLogic,
		),
	)
}

func main() {
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
	conf := bc
	Name = conf.Server.Name
	Version = conf.Server.Version
	if conf.Server.ShipVersion != "" {
		shipVersion = conf.Server.ShipVersion
	}
	logConf := conf.Log
	logPath, err := utils.GetPackageStorePathByNames("log")
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(logPath, 0755)
	if err != nil {
		panic(err)
	}
	logger := log.With(log.NewStdLogger(&lumberjack.Logger{
		Filename:   filepath.Join(logPath, fmt.Sprintf("%s.log", Name)),
		MaxSize:    int(logConf.MaxSize), // megabytes
		MaxBackups: int(logConf.MaxBackups),
		MaxAge:     int(logConf.MaxAge), // days
		Compress:   logConf.Compress,    // disabled by default
		LocalTime:  logConf.LocalTime,
	}),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)
	app, cleanup, err := wireApp(
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
