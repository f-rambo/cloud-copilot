package main

import (
	"flag"
	"fmt"
	"os"
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

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server, internalLogic *server.InternalLogic) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{
			"service": Name,
			"version": Version,
			"runtime": runtime.Version(),
			"os":      runtime.GOOS,
			"arch":    runtime.GOARCH,
			"conf":    flagconf,
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
	defer func() {
		if r := recover(); r != nil {
			logStackTrace(r)
		}
	}()

	flag.Parse()
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		logFatalWithStack(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		logFatalWithStack(err)
	}

	utilLog, err := utils.NewLog(&bc)
	if err != nil {
		logFatalWithStack(err)
	}
	defer utilLog.Close()
	logger := log.With(utilLog, utilLog.GetLogContenteKeyvals()...)
	app, cleanup, err := wireApp(
		&bc,
		logger,
	)
	if err != nil {
		logFatalWithStack(err)
	}

	defer cleanup()
	// start and wait for stop signal
	if err := app.Run(); err != nil {
		logFatalWithStack(err)
	}
}

func logStackTrace(err any) {
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, true)
	fmt.Printf("Panic: %v\nStack trace:\n%s", err, buf[:n])
}

func logFatalWithStack(err error) {
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, true)
	log.Fatalf("Error: %v\nStack trace:\n%s", err, buf[:n])
}
