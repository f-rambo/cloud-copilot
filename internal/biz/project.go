package biz

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type Project struct {
	ID           int64      `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name         string     `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Namespace    string     `json:"namespace" gorm:"column:namespace; default:''; NOT NULL"`
	State        string     `json:"state" gorm:"column:state; default:''; NOT NULL"`
	Description  string     `json:"description" gorm:"column:description; default:''; NOT NULL"`
	ClusterID    int64      `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
	Business     []Business `json:"business" gorm:"-"`
	BusinessJson []byte     `json:"business_json" gorm:"column:business_json; type:json"`
	gorm.Model
}

const (
	ProjectStateInit    = "init"
	ProjectStateRunning = "running"
	ProjectStateStopped = "stopped"
)

const (
	BackendBusiness  = "backend"
	FrontendBusiness = "frontend"
	BigDataBusiness  = "bigdata"
	MLBusiness       = "ml"
)

const (
	GolangTechnology = "golang"
	PythonTechnology = "python"
	JavaTechnology   = "java"
	NodejsTechnology = "nodejs"
)

type Business struct {
	Name        string       `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Technologys []Technology `json:"technologys" gorm:"-"`
}

type Technology struct {
	Name string `json:"name" gorm:"column:name; default:''; NOT NULL"`
}

type ProjectData interface {
	Save(context.Context, *Project) error
	Get(context.Context, int64) (*Project, error)
	GetByName(context.Context, string) (*Project, error)
	List(context.Context, int64) ([]*Project, error)
	ListByIds(context.Context, []int64) ([]*Project, error)
	Delete(context.Context, int64) error
}

type PorjectRuntime interface {
	CreateNamespace(context.Context, string) error
	GetNamespaces(context.Context) (namespaces []string, err error)
}

type ProjectUsecase struct {
	projectData    ProjectData
	ProjectRuntime PorjectRuntime
	log            *log.Helper
	conf           *conf.Bootstrap
}

func NewProjectUseCase(projectData ProjectData, ProjectTime PorjectRuntime, logger log.Logger, conf *conf.Bootstrap) *ProjectUsecase {
	return &ProjectUsecase{projectData: projectData, ProjectRuntime: ProjectTime, log: log.NewHelper(logger), conf: conf}
}

// project init
func (uc *ProjectUsecase) Init(ctx context.Context) error {
	project, err := uc.projectData.GetByName(ctx, uc.conf.Server.Name)
	if err != nil {
		return err
	}
	if project == nil {
		project := &Project{Name: uc.conf.Server.Name, State: ProjectStateInit}
		err = uc.Save(ctx, project)
		if err != nil {
			return err
		}
	}
	return nil
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
