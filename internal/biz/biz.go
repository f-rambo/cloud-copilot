package biz

import (
	"github.com/google/wire"
)

// common context key
type ContextKey string

func (c ContextKey) String() string {
	return string(c)
}

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewClusterUseCase, NewAppUsecase, NewServicesUseCase, NewUseUser, NewProjectUseCase, NewWorkspaceUsecase)
