package biz

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewBiz, NewClusterUseCase, NewAppUsecase, NewServicesUseCase, NewUseUser, NewProjectUseCase, NewWorkspaceUsecase, NewAgentUsecase)

type Biz struct {
	clusterUc *ClusterUsecase
	appUc     *AppUsecase
	userUc    *UserUseCase
	conf      *conf.Bootstrap
	log       *log.Helper
}

func NewBiz(clusterUc *ClusterUsecase, appUc *AppUsecase, servicesUc *ServicesUseCase, userUc *UserUseCase, projectUc *ProjectUsecase, conf *conf.Bootstrap, logger log.Logger) *Biz {
	return &Biz{
		clusterUc: clusterUc,
		appUc:     appUc,
		userUc:    userUc,
		conf:      conf,
		log:       log.NewHelper(logger),
	}
}

func (b *Biz) Initialize(ctx context.Context) error {
	bizIntiFunc := []func(context.Context) error{
		b.appUc.Init,
		b.userUc.InitAdminUser,
	}
	for _, f := range bizIntiFunc {
		if err := f(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (b *Biz) BizRunners() []transport.Server {
	return []transport.Server{
		b.clusterUc,
		b.appUc,
	}
}
