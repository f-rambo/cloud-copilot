package data

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type projectRepo struct {
	data       *Data
	log        *log.Helper
	confServer *conf.Server
}

func NewProjectRepo(data *Data, c *conf.Bootstrap, logger log.Logger) biz.ProjectData {
	return &projectRepo{
		data:       data,
		log:        log.NewHelper(logger),
		confServer: c.Server,
	}
}

func (p *projectRepo) Save(ctx context.Context, project *biz.Project) (err error) {
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

// GetByName get project by name
func (p *projectRepo) GetByName(ctx context.Context, name string) (*biz.Project, error) {
	project := &biz.Project{}
	err := p.data.db.Where("name = ?", name).First(project).Error
	if err != nil && err != gorm.ErrRecordNotFound {
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

func (p *projectRepo) ListByIds(ctx context.Context, ids []int64) ([]*biz.Project, error) {
	projects := make([]*biz.Project, 0)
	err := p.data.db.Where("id in (?)", ids).Find(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (p *projectRepo) Delete(ctx context.Context, id int64) error {
	return p.data.db.Where("id = ?", id).Delete(&biz.Project{}).Error
}
