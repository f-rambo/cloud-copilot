package infrastructure

import (
	"context"
	"testing"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

func TestMigrateToBostionHost(t *testing.T) {
	infrastructure := &ClusterInfrastructure{
		log: log.NewHelper(log.DefaultLogger),
		c:   &conf.Bootstrap{},
	}
	err := infrastructure.MigrateToBostionHost(context.Background(), &biz.Cluster{
		BostionHost: &biz.BostionHost{},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestInstall(t *testing.T) {
	infrastructure := &ClusterInfrastructure{
		log: log.NewHelper(log.DefaultLogger),
		c:   &conf.Bootstrap{},
	}
	err := infrastructure.Install(context.Background(), &biz.Cluster{
		BostionHost: &biz.BostionHost{},
	})
	if err != nil {
		t.Fatal(err)
	}
}
