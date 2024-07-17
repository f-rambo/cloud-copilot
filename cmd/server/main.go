package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/internal/server"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "github.com/joho/godotenv/autoload"
	_ "go.uber.org/automaxprocs"

	"gopkg.in/natefinch/lumberjack.v2"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string = "ocean"
	// Version is the version of the compiled software.
	Version string = "0.1.0"
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "configs", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server, otherServer *server.OtherServer) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(gs, hs, otherServer),
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
	oceanConf := bc.Ocean
	logConf := oceanConf.Log

	logger := log.With(log.NewStdLogger(&lumberjack.Logger{
		Filename:   fmt.Sprintf("%s/%s", logConf.Path, logConf.Filename),
		MaxSize:    int(logConf.MaxSize), // megabytes
		MaxBackups: int(logConf.MaxBackups),
		MaxAge:     int(logConf.MaxAge), //days
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
