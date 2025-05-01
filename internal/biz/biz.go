package biz

import (
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewClusterUseCase, NewAppUsecase, NewServicesUseCase, NewUseUser, NewProjectUseCase, NewWorkspaceUsecase, NewAgentUsecase)

type ContextKey string

func (c ContextKey) String() string {
	return string(c)
}
