package interfaces

import (
	"context"
	"strings"

	"github.com/f-rambo/cloud-copilot/api/common"
	"github.com/f-rambo/cloud-copilot/api/workspace/v1alpha1"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/utils"
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
	if workspaceParam.Name == "" {
		return nil, errors.New("workspace name is empty")
	}
	if !utils.IsValidKubernetesName(workspaceParam.Name) {
		return nil, errors.New("workspace name is invalid")
	}
	if workspaceParam.Id == 0 {
		workspaceData, err := w.workspaceUc.GetByName(ctx, workspaceParam.Name)
		if err != nil {
			return nil, err
		}
		if workspaceData != nil && workspaceData.Id > 0 {
			return nil, errors.New("workspace name already exists")
		}
	}
	workspace := w.workspaceToBiz(workspaceParam)
	err := w.workspaceUc.Save(ctx, workspace)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (w *WorkspaceInterface) Get(ctx context.Context, workspaceId *v1alpha1.WorkspaceDetailParam) (*v1alpha1.Workspace, error) {
	if workspaceId.Id == 0 {
		return nil, errors.New("workspaceId is empty")
	}
	workspace, err := w.workspaceUc.Get(ctx, int64(workspaceId.Id))
	if err != nil {
		return nil, err
	}
	return w.bizToWorkspace(workspace), nil
}

func (w *WorkspaceInterface) List(ctx context.Context, workspaceParam *v1alpha1.WorkspaceListParam) (*v1alpha1.WorkspaceList, error) {
	workspaceName := strings.TrimSpace(workspaceParam.WorkspaceName)
	workspaceList, total, err := w.workspaceUc.List(ctx, workspaceName, workspaceParam.Page, workspaceParam.Size)
	if err != nil {
		return nil, err
	}
	workspaceListRes := &v1alpha1.WorkspaceList{Total: int32(total), Items: make([]*v1alpha1.Workspace, 0)}
	for _, workspace := range workspaceList {
		workspaceListRes.Items = append(workspaceListRes.Items, w.bizToWorkspace(workspace))
	}
	return workspaceListRes, nil
}

func (w *WorkspaceInterface) Delete(ctx context.Context, workspaceId *v1alpha1.WorkspaceDetailParam) (*common.Msg, error) {
	if workspaceId.Id == 0 {
		return nil, errors.New("workspaceId is empty")
	}
	err := w.workspaceUc.Delete(ctx, int64(workspaceId.Id))
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (w *WorkspaceInterface) bizToWorkspace(workspace *biz.Workspace) *v1alpha1.Workspace {
	if workspace == nil {
		return nil
	}
	resourceQuota := workspace.ResourceQuota.ToResourceQuota()
	res := &v1alpha1.Workspace{
		Id:              int32(workspace.Id),
		Name:            workspace.Name,
		Description:     workspace.Description,
		UserId:          int32(workspace.UserId),
		Status:          workspace.Status.String(),
		GitRepository:   workspace.GitRepository,
		ImageRepository: workspace.ImageRepository,
		ResourceQuota: &v1alpha1.ResourceQuota{
			Cpu: &v1alpha1.ResourceLimit{
				Limit:   resourceQuota.CPU.Limit,
				Request: resourceQuota.CPU.Request,
			},
			Memory: &v1alpha1.ResourceLimit{
				Limit:   resourceQuota.Memory.Limit,
				Request: resourceQuota.Memory.Request,
			},
			Gpu: &v1alpha1.ResourceLimit{
				Limit:   resourceQuota.GPU.Limit,
				Request: resourceQuota.GPU.Request,
			},
			Storage: &v1alpha1.ResourceLimit{
				Limit:   resourceQuota.Storage.Limit,
				Request: resourceQuota.Storage.Request,
			},
			Pods: &v1alpha1.ResourceLimit{
				Limit:   resourceQuota.Pods.Limit,
				Request: resourceQuota.Pods.Request,
			},
		},
		ClusterRelationships: make([]*v1alpha1.WorkspaceClusterRelationship, 0),
	}
	for _, clusterRelationship := range workspace.WorkspaceClusterRelationships {
		res.ClusterRelationships = append(res.ClusterRelationships, &v1alpha1.WorkspaceClusterRelationship{
			Id:          int32(clusterRelationship.Id),
			WorkspaceId: int32(clusterRelationship.WorkspaceId),
			ClusterId:   int32(clusterRelationship.ClusterId),
			Permissions: clusterRelationship.Permissions.String(),
		})
	}
	return res
}

func (w *WorkspaceInterface) workspaceToBiz(workspace *v1alpha1.Workspace) *biz.Workspace {
	resourceQuota := biz.ResourceQuota{
		CPU: biz.ResourceLimit{
			Limit:   workspace.ResourceQuota.Cpu.Limit,
			Request: workspace.ResourceQuota.Cpu.Request,
		},
		Memory: biz.ResourceLimit{
			Limit:   workspace.ResourceQuota.Memory.Limit,
			Request: workspace.ResourceQuota.Memory.Request,
		},
		GPU: biz.ResourceLimit{
			Limit:   workspace.ResourceQuota.Gpu.Limit,
			Request: workspace.ResourceQuota.Gpu.Request,
		},
		Storage: biz.ResourceLimit{
			Limit:   workspace.ResourceQuota.Storage.Limit,
			Request: workspace.ResourceQuota.Storage.Request,
		},
		Pods: biz.ResourceLimit{
			Limit:   workspace.ResourceQuota.Pods.Limit,
			Request: workspace.ResourceQuota.Pods.Request,
		},
	}
	clusterRelationship := make([]*biz.WorkspaceClusterRelationship, 0)
	for _, cluster := range workspace.ClusterRelationships {
		clusterRelationship = append(clusterRelationship, &biz.WorkspaceClusterRelationship{
			Id:          int64(cluster.Id),
			WorkspaceId: int64(cluster.WorkspaceId),
			ClusterId:   int64(cluster.ClusterId),
			Permissions: biz.WorkspaceClusterPermissionsFromString(cluster.Permissions),
		})
	}
	return &biz.Workspace{
		Id:                            int64(workspace.Id),
		Name:                          workspace.Name,
		Description:                   workspace.Description,
		UserId:                        int64(workspace.UserId),
		GitRepository:                 workspace.GitRepository,
		ImageRepository:               workspace.ImageRepository,
		ResourceQuota:                 resourceQuota.ToString(),
		WorkspaceClusterRelationships: clusterRelationship,
	}
}
