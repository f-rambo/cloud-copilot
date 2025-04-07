package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// todo config & Add case...

type Agent interface {
	Get(context.Context)
}

type AgentUsecase struct {
	log *log.Helper
}

func NewAgentUsecase(logger log.Logger) *AgentUsecase {
	return &AgentUsecase{
		log: log.NewHelper(logger),
	}
}
