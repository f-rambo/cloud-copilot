package data

import (
	"context"
	"encoding/json"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

type projectRepo struct {
	data       *Data
	log        *log.Helper
	confServer *conf.Server
}

func NewProjectRepo(data *Data, c *conf.Bootstrap, logger log.Logger) biz.ProjectRepo {
	cserver := c.GetOceanServer()
	return &projectRepo{
		data:       data,
		log:        log.NewHelper(logger),
		confServer: &cserver,
	}
}

func (p *projectRepo) Save(ctx context.Context, project *biz.Project) (err error) {
	if len(project.Business) > 0 {
		project.BusinessJson, err = json.Marshal(project.Business)
		if err != nil {
			return err
		}
	}
	return p.data.db.Save(project).Error
}

func (p *projectRepo) Get(ctx context.Context, id int64) (*biz.Project, error) {
	project := &biz.Project{}
	err := p.data.db.Where("id = ?", id).First(project).Error
	if err != nil {
		return nil, err
	}
	if len(project.BusinessJson) > 0 {
		err = json.Unmarshal(project.BusinessJson, &project.Business)
		if err != nil {
			return nil, err
		}
	}
	return project, nil
}

func (p *projectRepo) List(ctx context.Context, clusterID int64) ([]*biz.Project, error) {
	projects := make([]*biz.Project, 0)
	err := p.data.db.Where("cluster_id = ?", clusterID).Find(&projects).Error
	if err != nil {
		return nil, err
	}
	for _, project := range projects {
		if len(project.BusinessJson) > 0 {
			err = json.Unmarshal(project.BusinessJson, &project.Business)
			if err != nil {
				return nil, err
			}
		}
	}
	return projects, nil
}

func (p *projectRepo) ListByIds(ctx context.Context, ids []int64) ([]*biz.Project, error) {
	projects := make([]*biz.Project, 0)
	err := p.data.db.Where("id in (?)", ids).Find(&projects).Error
	if err != nil {
		return nil, err
	}
	for _, project := range projects {
		if len(project.BusinessJson) > 0 {
			err = json.Unmarshal(project.BusinessJson, &project.Business)
			if err != nil {
				return nil, err
			}
		}
	}
	return projects, nil
}

func (p *projectRepo) Delete(ctx context.Context, id int64) error {
	return p.data.db.Where("id = ?", id).Delete(&biz.Project{}).Error
}
