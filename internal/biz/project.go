package biz

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

const (
	ProjectKey ContextKey = "project"
)

type ProjectData interface {
	Save(context.Context, *Project) error
	Get(context.Context, int64) (*Project, error)
	GetByName(context.Context, string) (*Project, error)
	List(context.Context, int64) ([]*Project, error)
	ListByIds(context.Context, []int64) ([]*Project, error)
	Delete(context.Context, int64) error
}

type ProjectRuntime interface {
	CreateNamespace(context.Context, string) error
	GetNamespaces(context.Context) (namespaces []string, err error)
}

type ProjectAgent interface {
}

type ProjectUsecase struct {
	projectData    ProjectData
	ProjectRuntime ProjectRuntime
	log            *log.Helper
	conf           *conf.Bootstrap
}

func NewProjectUseCase(projectData ProjectData, ProjectTime ProjectRuntime, logger log.Logger, conf *conf.Bootstrap) *ProjectUsecase {
	return &ProjectUsecase{projectData: projectData, ProjectRuntime: ProjectTime, log: log.NewHelper(logger), conf: conf}
}

func GetProject(ctx context.Context) *Project {
	v, ok := ctx.Value(ProjectKey).(*Project)
	if !ok {
		return nil
	}
	return v
}

func WithProject(ctx context.Context, p *Project) context.Context {
	return context.WithValue(ctx, ProjectKey, p)
}

func (uc *ProjectUsecase) Save(ctx context.Context, project *Project) error {
	workspace := GetWorkspace(ctx)
	if project.LimitCpu > workspace.LimitCpu {
		project.LimitCpu = workspace.LimitCpu
	}
	if project.LimitGpu > workspace.LimitGpu {
		project.LimitGpu = workspace.LimitGpu
	}
	if project.LimitMemory > workspace.LimitMemory {
		project.LimitMemory = workspace.LimitMemory
	}
	if project.LimitDisk > workspace.LimitDisk {
		project.LimitDisk = workspace.LimitDisk
	}
	project.UserId = GetUserInfo(ctx).Id
	return uc.projectData.Save(ctx, project)
}

func (uc *ProjectUsecase) Get(ctx context.Context, id int64) (*Project, error) {
	return uc.projectData.Get(ctx, id)
}

func (uc *ProjectUsecase) List(ctx context.Context, clusterID int64) ([]*Project, error) {
	return uc.projectData.List(ctx, clusterID)
}

func (uc *ProjectUsecase) ListByIds(ctx context.Context, ids []int64) ([]*Project, error) {
	return uc.projectData.ListByIds(ctx, ids)
}

func (uc *ProjectUsecase) Delete(ctx context.Context, id int64) error {
	return uc.projectData.Delete(ctx, id)
}

func (uc *ProjectUsecase) GetByName(ctx context.Context, name string) (*Project, error) {
	return uc.projectData.GetByName(ctx, name)
}
