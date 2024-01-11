package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type Project struct {
	ID          int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Namespace   string `json:"namespace" gorm:"column:namespace; default:''; NOT NULL"`
	ClusterID   int64  `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
	Description string `json:"description" gorm:"column:description; default:''; NOT NULL"`
	gorm.Model
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
