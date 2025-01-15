package biz

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
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

func (uc *ProjectUsecase) Save(ctx context.Context, project *Project) error {
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
