package biz

import (
	"context"

	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
)

const (
	WorkspaceKey ContextKey = "workspace"
)

type AlreadyResource struct {
	Cpu     int32 `json:"cpu,omitempty"`
	Memory  int32 `json:"memory,omitempty"`
	Gpu     int32 `json:"gpu,omitempty"`
	Storage int32 `json:"storage,omitempty"`
}

type Workspace struct {
	Id              int64  `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name            string `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Description     string `json:"description,omitempty" gorm:"column:namespace;default:'';NOT NULL"`
	ClusterId       int64  `json:"cluster_id,omitempty" gorm:"column:cluster_id;default:0;NOT NULL"`
	UserId          int64  `json:"user_id,omitempty" gorm:"column:user_id;default:0;NOT NULL"`
	CpuRate         int32  `json:"cpu_rate,omitempty" gorm:"column:cpu_rate;default:0;NOT NULL"`
	GpuRate         int32  `json:"gpu_rate,omitempty" gorm:"column:gpu_rate;default:0;NOT NULL"`
	MemoryRate      int32  `json:"memory_rate,omitempty" gorm:"column:memory_rate;default:0;NOT NULL"`
	DiskRate        int32  `json:"disk_rate,omitempty" gorm:"column:disk_rate;default:0;NOT NULL"`
	LimitCpu        int32  `json:"limit_cpu,omitempty" gorm:"column:limit_cpu;default:0;NOT NULL"`
	LimitGpu        int32  `json:"limit_gpu,omitempty" gorm:"column:limit_gpu;default:0;NOT NULL"`
	LimitMemory     int32  `json:"limit_memory,omitempty" gorm:"column:limit_memory;default:0;NOT NULL"`
	LimitDisk       int32  `json:"limit_disk,omitempty" gorm:"column:limit_disk;default:0;NOT NULL"`
	GitRepository   string `json:"git_repository,omitempty" gorm:"column:git_repository;default:'';NOT NULL"`
	ImageRepository string `json:"image_repository,omitempty" gorm:"column:image_repository;default:'';NOT NULL"`
}

type WorkspaceData interface {
	Get(ctx context.Context, id int64) (*Workspace, error)
	Save(context.Context, *Workspace) error
	List(ctx context.Context, clusterId int64, workspaceName string) ([]*Workspace, error)
	GetByName(ctx context.Context, name string) (*Workspace, error)
}

type WorkspaceRuntime interface {
	Reload(context.Context, *Workspace) error
	Delete(context.Context, *Workspace) error
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
	v.GetCpuCount(ctx)
	v.GetGpuCount(ctx)
	v.GetMemoryCount(ctx)
	v.GetDiskSize(ctx)
	return v
}

func WithWorkspace(ctx context.Context, w *Workspace) context.Context {
	return context.WithValue(ctx, WorkspaceKey, w)
}

func (w *Workspace) GetLabels() map[string]string {
	return map[string]string{
		"workspace": w.Name,
	}
}

func (w *Workspace) GetCpuCount(ctx context.Context) {
	cluster := GetCluster(ctx)
	clusterCpuCount := cluster.GetCpuCount()
	if w.CpuRate > 0 && clusterCpuCount > 0 {
		w.LimitCpu = utils.CalculatePercentageInt32(w.CpuRate, clusterCpuCount)
	}
}

func (w *Workspace) GetGpuCount(ctx context.Context) {
	cluster := GetCluster(ctx)
	clusterGpuCount := cluster.GetGpuCount()
	if w.GpuRate > 0 && clusterGpuCount > 0 {
		w.LimitGpu = utils.CalculatePercentageInt32(w.GpuRate, clusterGpuCount)
	}
}

func (w *Workspace) GetMemoryCount(ctx context.Context) {
	cluster := GetCluster(ctx)
	clusterMemoryCount := cluster.GetMemoryCount()
	if w.MemoryRate > 0 && clusterMemoryCount > 0 {
		w.LimitMemory = utils.CalculatePercentageInt32(w.MemoryRate, clusterMemoryCount)
	}
}

func (w *Workspace) GetDiskSize(ctx context.Context) {
	cluster := GetCluster(ctx)
	clusterDiskSize := cluster.GetDiskSizeCount()
	if w.DiskRate > 0 && clusterDiskSize > 0 {
		w.LimitDisk = utils.CalculatePercentageInt32(w.DiskRate, clusterDiskSize)
	}
}

func (uc *WorkspaceUsecase) Get(ctx context.Context, id int64) (*Workspace, error) {
	workspace, err := uc.workspaceData.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	workspace.GetCpuCount(ctx)
	workspace.GetGpuCount(ctx)
	workspace.GetMemoryCount(ctx)
	workspace.GetDiskSize(ctx)
	return workspace, nil
}

func (uc *WorkspaceUsecase) Save(ctx context.Context, workspace *Workspace) error {
	if workspace.CpuRate > 100 {
		workspace.CpuRate = 100
	}
	if workspace.GpuRate > 100 {
		workspace.GpuRate = 100
	}
	if workspace.MemoryRate > 100 {
		workspace.MemoryRate = 100
	}
	if workspace.DiskRate > 100 {
		workspace.DiskRate = 100
	}
	return uc.workspaceData.Save(ctx, workspace)
}

func (uc *WorkspaceUsecase) List(ctx context.Context, clusterId int64, workspaceName string) ([]*Workspace, error) {
	workspaces, err := uc.workspaceData.List(ctx, clusterId, workspaceName)
	if err != nil {
		return nil, err
	}
	for _, workspace := range workspaces {
		workspace.GetCpuCount(ctx)
		workspace.GetGpuCount(ctx)
		workspace.GetMemoryCount(ctx)
		workspace.GetDiskSize(ctx)
	}
	return workspaces, nil
}

func (uc *WorkspaceUsecase) GetByName(ctx context.Context, name string) (*Workspace, error) {
	workspace, err := uc.workspaceData.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return workspace, nil
}
