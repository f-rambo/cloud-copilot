package sailor

import (
	"context"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	sailorV1alpha1 "github.com/f-rambo/sailor/api/v1alpha1"
	"github.com/go-kratos/kratos/v2/log"
)

type SailorClient struct {
	log *log.Helper
	c   *conf.Bootstrap
}

func NewSailorClient(c *conf.Bootstrap, logger log.Logger) biz.SailorRepo {
	return &SailorClient{
		log: log.NewHelper(logger),
		c:   c,
	}
}

func (s *SailorClient) Create(ctx context.Context, deployedApp *biz.DeployApp) error {
	restConfig, err := getKubeConfig()
	if err != nil {
		return err
	}
	sailorApp, err := newForConfig(restConfig)
	if err != nil {
		return err
	}
	app := buildAppResource(deployedApp.Namespace, deployedApp.ReleaseName,
		sailorV1alpha1.AppSpec{Manifest: deployedApp.Manifest})
	_, err = sailorApp.apps(deployedApp.Namespace).Create(ctx, &app)
	if err != nil {
		return err
	}
	return nil
}
