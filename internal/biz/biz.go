package biz

import (
	"github.com/google/wire"
)

type QueueKey string

func (k QueueKey) String() string {
	return string(k)
}

const (
	ClusterQueueKey QueueKey = "cluster-queue-key"
	AppQueueKey     QueueKey = "app-queue-key"
	ServiceQueueKey QueueKey = "service-queue-key"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewClusterUseCase, NewAppUsecase, NewServicesUseCase, NewUseUser, NewProjectUseCase)
