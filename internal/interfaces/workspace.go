package interfaces

import (
	"context"

	"github.com/f-rambo/cloud-copilot/api/common"
	"github.com/f-rambo/cloud-copilot/api/workspace/v1alpha1"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
)

type WorkspaceInterface struct {
	v1alpha1.UnimplementedWorkspaceInterfaceServer
	workspaceUc *biz.WorkspaceUsecase
	log         *log.Helper
}

func NewWorkspaceInterface(workspaceUc *biz.WorkspaceUsecase, logger log.Logger) *WorkspaceInterface {
	return &WorkspaceInterface{
		workspaceUc: workspaceUc,
		log:         log.NewHelper(logger),
	}
}

func (w *WorkspaceInterface) GetWorkspace(ctx context.Context, id int64) (*biz.Workspace, error) {
	return w.workspaceUc.Get(ctx, id)
}

func (w *WorkspaceInterface) Save(ctx context.Context, workspaceParam *v1alpha1.Workspace) (*common.Msg, error) {
	return common.Response(), nil
}

func (w *WorkspaceInterface) Get(ctx context.Context, workspaceId *v1alpha1.WorkspaceParam) (*v1alpha1.Workspace, error) {
	if workspaceId.Id == 0 {
		return nil, errors.New("workspaceId is empty")
	}
	workspace, err := w.workspaceUc.Get(ctx, workspaceId.Id)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Workspace{
		Id:          workspace.Id,
		Name:        workspace.Name,
		Description: workspace.Description,
		ClusterId:   workspace.ClusterId,
		CpuRate:     workspace.CpuRate,
		GpuRate:     workspace.GpuRate,
		MemoryRate:  workspace.MemoryRate,
		DiskRate:    workspace.DiskRate,
	}, nil
}
