package biz

import (
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewBiz, NewClusterUseCase, NewAppUsecase, NewServicesUseCase, NewUseUser, NewProjectUseCase, NewWorkspaceUsecase, NewAgentUsecase)

type Biz struct {
	clusterUc *ClusterUsecase
	appUc     *AppUsecase
}

func NewBiz(clusterUc *ClusterUsecase, appUc *AppUsecase) *Biz {
	return &Biz{
		clusterUc: clusterUc,
		appUc:     appUc,
	}
}

func (b *Biz) BizRunners() []transport.Server {
	return []transport.Server{
		b.clusterUc,
		b.appUc,
	}
}

type ContextKey string

func (c ContextKey) String() string {
	return string(c)
}
