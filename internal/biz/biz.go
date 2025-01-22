package biz

import (
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
	conf      *conf.Bootstrap
	log       *log.Helper
}

func NewBiz(clusterUc *ClusterUsecase, appUc *AppUsecase, servicesUc *ServicesUseCase, userUc *UserUseCase, projectUc *ProjectUsecase, conf *conf.Bootstrap, logger log.Logger) *Biz {
	return &Biz{
		clusterUc: clusterUc,
		appUc:     appUc,
		conf:      conf,
		log:       log.NewHelper(logger),
	}
}

func (b *Biz) BizRunners() []transport.Server {
	return []transport.Server{
		b.clusterUc,
		b.appUc,
	}
}
