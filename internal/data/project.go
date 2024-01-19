package data

import (
	"context"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

type projectRepo struct {
	data       *Data
	log        *log.Helper
	confServer *conf.Server
}

func NewProjectRepo(data *Data, logger log.Logger, confServer *conf.Server) (biz.ProjectRepo, error) {
	projectRepo := &projectRepo{
		data:       data,
		log:        log.NewHelper(logger),
		confServer: confServer,
	}
	return projectRepo, projectRepo.init()
}

func (p *projectRepo) init() error {
	var count int64 = 0
	err := p.data.db.Model(&biz.Project{}).Where("name = ?", p.confServer.GetName()).Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	project := &biz.Project{
		Name:        p.confServer.GetName(),
		Namespace:   p.confServer.GetName(),
		ClusterID:   1,
		Description: "defualt project",
	}
	return p.data.db.Model(&biz.Project{}).Create(project).Error
}

func (p *projectRepo) Save(ctx context.Context, project *biz.Project) error {
	return p.data.db.Save(project).Error
}

func (p *projectRepo) Get(ctx context.Context, id int64) (*biz.Project, error) {
	project := &biz.Project{}
	err := p.data.db.Where("id = ?", id).First(project).Error
	if err != nil {
		return nil, err
	}
	return project, nil
}

func (p *projectRepo) List(ctx context.Context, clusterID int64) ([]*biz.Project, error) {
	projects := make([]*biz.Project, 0)
	err := p.data.db.Where("cluster_id = ?", clusterID).Find(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (p *projectRepo) Delete(ctx context.Context, id int64) error {
	return p.data.db.Where("id = ?", id).Delete(&biz.Project{}).Error
}
