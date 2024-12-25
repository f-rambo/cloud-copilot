package biz

import "github.com/go-kratos/kratos/v2/log"

// config
// Add case...

type AgentUsecase struct {
	log *log.Helper
}

func NewAgentUsecase(logger log.Logger) *AgentUsecase {
	return &AgentUsecase{
		log: log.NewHelper(logger),
	}
}
