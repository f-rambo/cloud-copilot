package biz

import (
	"context"
	"strings"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

const (
	ProjectKey ContextKey = "project"
)

type Project struct {
	Id            int64               `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name          string              `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Description   string              `json:"description,omitempty" gorm:"column:namespace;default:'';NOT NULL"`
	UserId        int64               `json:"user_id,omitempty" gorm:"column:user_id;default:0;NOT NULL"`
	WorkspaceId   int64               `json:"workspace_id,omitempty" gorm:"column:workspace_id;default:0;NOT NULL"`
	ResourceQuota ResourceQuotaString `json:"resource_quota,omitempty" gorm:"column:resource_quota;default:'';NOT NULL"`
}

type ProjectData interface {
	Save(context.Context, *Project) error
	Get(context.Context, int64) (*Project, error)
	List(ctx context.Context, name string, page, size int32) ([]*Project, int32, error)
	Delete(context.Context, int64) error
	GetByName(context.Context, string) (*Project, error)
}

type ProjectRuntime interface {
	Reload(context.Context, *Project) error
	Delete(context.Context, *Project) error
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

func (p *Project) GetLabels() map[string]string {
	return map[string]string{
		"project": p.Name,
	}
}

func (uc *ProjectUsecase) Save(ctx context.Context, project *Project) error {
	return uc.projectData.Save(ctx, project)
}

func (uc *ProjectUsecase) Get(ctx context.Context, id int64) (*Project, error) {
	return uc.projectData.Get(ctx, id)
}

func (uc *ProjectUsecase) List(ctx context.Context, name string, page, size int32) ([]*Project, int32, error) {
	return uc.projectData.List(ctx, strings.TrimSpace(name), page, size)
}

func (uc *ProjectUsecase) Delete(ctx context.Context, id int64) error {
	return uc.projectData.Delete(ctx, id)
}

func (uc *ProjectUsecase) GetByName(ctx context.Context, name string) (*Project, error) {
	return uc.projectData.GetByName(ctx, name)
}
