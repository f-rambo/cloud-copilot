package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	WorkspaceKey ContextKey = "workspace"
)

type WorkspaceData interface {
	Get(ctx context.Context, id int64) (*Workspace, error)
	Save(context.Context, *Workspace) error
}

type WorkspaceAgent interface {
}

type WorkspaceUsecase struct {
	workspaceData WorkspaceData
	log           *log.Helper
}

func NewWorkspaceUsecase(workspaceData WorkspaceData, logger log.Logger) *WorkspaceUsecase {
	return &WorkspaceUsecase{log: log.NewHelper(logger)}
}

func GetWorkspace(ctx context.Context) *Workspace {
	v, ok := ctx.Value(WorkspaceKey).(*Workspace)
	if !ok {
		return nil
	}
	return v
}

func WithWorkspace(ctx context.Context, w *Workspace) context.Context {
	return context.WithValue(ctx, WorkspaceKey, w)
}

func (uc *WorkspaceUsecase) Get(ctx context.Context, id int64) (*Workspace, error) {
	return uc.workspaceData.Get(ctx, id)
}

func (uc *WorkspaceUsecase) Save(ctx context.Context, workspace *Workspace) error {
	return uc.workspaceData.Save(ctx, workspace)
}
