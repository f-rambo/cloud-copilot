package biz

import "github.com/go-kratos/kratos/v2/log"

type WorkspaceAgent interface {
}

type WorkspaceUsecase struct {
	log *log.Helper
}

func NewWorkspaceUsecase(logger log.Logger) *WorkspaceUsecase {
	return &WorkspaceUsecase{log: log.NewHelper(logger)}
}
