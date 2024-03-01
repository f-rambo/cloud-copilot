package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type Project struct {
	ID               int64          `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name             string         `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Namespace        string         `json:"namespace" gorm:"column:namespace; default:''; NOT NULL"`
	State            string         `json:"state" gorm:"column:state; default:''; NOT NULL"`
	Description      string         `json:"description" gorm:"column:description; default:''; NOT NULL"`
	ClusterID        int64          `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
	BusinessTypes    []BusinessType `json:"business_types" gorm:"-"`
	BusinessTypeJson []byte         `json:"business_type_json" gorm:"column:business_type_json; type:json"`
	gorm.Model
}

const (
	ProjectStateInit    = "init"
	ProjectStateRunning = "running"
	ProjectStateStopped = "stopped"
)

type BusinessType struct {
	ID              int64            `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name            string           `json:"name" gorm:"column:name; default:''; NOT NULL"`
	ProjectID       int64            `json:"project_id" gorm:"column:project_id; default:0; NOT NULL"`
	TechnologyTypes []TechnologyType `json:"technology_types" gorm:"-"`
}

type TechnologyType struct {
	ID             int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name           string `json:"name" gorm:"column:name; default:''; NOT NULL"`
	BusinessTypeID int64  `json:"business_type_id" gorm:"column:business_type_id; default:0; NOT NULL"`
}

const (
	BusinessTypeBackend  = 1
	BusinessTypeFrontend = 2
	BusinessTypeBigData  = 3
	BusinessTypeAI       = 4
	BusinessTypeOther    = 5
)

const (
	TechnologyTypeGolang     = 1
	TechnologyTypeJava       = 2
	TechnologyTypePython     = 3
	TechnologyTypeVue        = 4
	TechnologyTypeReact      = 5
	TechnologyTypeOther      = 6
	TechnologyTypeHadoop     = 7
	TechnologyTypeSpark      = 8
	TechnologyTypeTensorflow = 9
	TechnologyTypePytorch    = 10
)

func (p *Project) GetBusinessTypes() {
	p.BusinessTypes = []BusinessType{
		{
			ID:   BusinessTypeBackend,
			Name: "Backend",
			TechnologyTypes: []TechnologyType{
				{
					ID:   TechnologyTypeGolang,
					Name: "golang",
				}, {
					ID:   TechnologyTypeJava,
					Name: "java",
				}, {
					ID:   TechnologyTypePython,
					Name: "python",
				},
			},
		}, {
			ID:   BusinessTypeFrontend,
			Name: "Frontend",
			TechnologyTypes: []TechnologyType{
				{
					ID:   TechnologyTypeVue,
					Name: "vue",
				}, {
					ID:   TechnologyTypeReact,
					Name: "react",
				},
			},
		}, {
			ID:   BusinessTypeBigData,
			Name: "Big Data",
			TechnologyTypes: []TechnologyType{
				{
					ID:   TechnologyTypeHadoop,
					Name: "hadoop",
				}, {
					ID:   TechnologyTypeSpark,
					Name: "spark",
				},
			},
		},
		{
			ID:   BusinessTypeAI,
			Name: "AI",
			TechnologyTypes: []TechnologyType{
				{
					ID:   TechnologyTypeTensorflow,
					Name: "tensorflow",
				}, {
					ID:   TechnologyTypePytorch,
					Name: "pytorch",
				},
			},
		},
		{
			ID:   BusinessTypeOther,
			Name: "Other",
			TechnologyTypes: []TechnologyType{
				{
					ID:   TechnologyTypeOther,
					Name: "other",
				},
			},
		},
	}
}

type ProjectRepo interface {
	Save(context.Context, *Project) error
	Get(context.Context, int64) (*Project, error)
	List(context.Context, int64) ([]*Project, error)
	Delete(context.Context, int64) error
}

type ProjectUsecase struct {
	repo ProjectRepo
	log  *log.Helper
}

func NewProjectUseCase(repo ProjectRepo, logger log.Logger) *ProjectUsecase {
	return &ProjectUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *ProjectUsecase) Save(ctx context.Context, project *Project) error {
	if project.ID == 0 {
		project.State = ProjectStateInit
	}
	return uc.repo.Save(ctx, project)
}

func (uc *ProjectUsecase) Get(ctx context.Context, id int64) (*Project, error) {
	return uc.repo.Get(ctx, id)
}

func (uc *ProjectUsecase) List(ctx context.Context, clusterID int64) ([]*Project, error) {
	return uc.repo.List(ctx, clusterID)
}

func (uc *ProjectUsecase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}
