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

func (p *projectRepo) List(ctx context.Context, name string, page, size int32) ([]*biz.Project, int32, error) {
	projects := make([]*biz.Project, 0)
	var total int64
	db := p.data.db

	if name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}

	if err := db.Model(&biz.Project{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return projects, 0, nil
	}

	offset := (page - 1) * size
	if err := db.Offset(int(offset)).Limit(int(size)).Find(&projects).Error; err != nil {
		return nil, 0, err
	}

	return projects, int32(total), nil
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
