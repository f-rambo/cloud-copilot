package biz

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewBiz, NewClusterUseCase, NewAppUsecase, NewServicesUseCase, NewUseUser, NewProjectUseCase)

type Biz struct {
}

func NewBiz(logger log.Logger) (*Biz, error) {
	return &Biz{}, nil
}
