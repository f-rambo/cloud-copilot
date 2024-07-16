package test

import (
	"fmt"
	"testing"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	gomock "github.com/golang/mock/gomock"
)

func TestOCean(t *testing.T) {
	fmt.Println("Starting TestOCean.....")
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	fmt.Println("New Controller.....")
	c := config.New(
		config.WithSource(
			file.NewSource("../../configs/"),
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
	app, cleanup, err := wireApp(
		ctl,
		&bc,
		log.DefaultLogger,
	)
	if err != nil {
		panic(err)
	}

	defer cleanup()
	if err := app.Run(); err != nil {
		panic(err)
	}
}
