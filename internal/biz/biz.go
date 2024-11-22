package biz

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/wire"
	"github.com/pkg/errors"
)

type QueueKey string

func (k QueueKey) String() string {
	return string(k)
}

type UserKey string

func (u UserKey) String() string {
	return string(u)
}

const (
	ClusterQueueKey QueueKey = "cluster-queue-key"
	AppQueueKey     QueueKey = "app-queue-key"
	ServiceQueueKey QueueKey = "service-queue-key"

	TokenKey     UserKey = "token"
	SignType     UserKey = "sign_type"
	UserEmailKey UserKey = "user_email"
)

const (
	ClusterPoolNumber = 10

	AppPoolNumber = 100
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewBiz, NewClusterUseCase, NewAppUsecase, NewServicesUseCase, NewUseUser, NewProjectUseCase)

var ErrClusterNotFound error = errors.New("cluster not found")

type Biz struct {
	clusterUc  *ClusterUsecase
	appUc      *AppUsecase
	servicesUc *ServicesUseCase
	userUc     *UserUseCase
	projectUc  *ProjectUsecase
	conf       *conf.Bootstrap
	log        *log.Helper
}

func NewBiz(clusterUc *ClusterUsecase, appUc *AppUsecase, servicesUc *ServicesUseCase, userUc *UserUseCase, projectUc *ProjectUsecase, conf *conf.Bootstrap, logger log.Logger) *Biz {
	return &Biz{
		clusterUc:  clusterUc,
		appUc:      appUc,
		servicesUc: servicesUc,
		userUc:     userUc,
		projectUc:  projectUc,
		conf:       conf,
		log:        log.NewHelper(logger),
	}
}

func (b *Biz) Initialize(ctx context.Context) error {
	bizIntiFunc := []func(context.Context) error{
		b.appUc.Init,
		b.servicesUc.Init,
		b.userUc.InitAdminUser,
		b.projectUc.Init,
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
