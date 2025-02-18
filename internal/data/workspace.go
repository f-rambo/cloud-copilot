package data

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type workspaceRepo struct {
	data *Data
	log  *log.Helper
}

func NewWorkspaceRepo(data *Data, logger log.Logger) biz.WorkspaceData {
	return &workspaceRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (w *workspaceRepo) Get(ctx context.Context, id int64) (*biz.Workspace, error) {
	var workspace biz.Workspace
	if err := w.data.db.First(&workspace, id).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

func (w *workspaceRepo) Save(ctx context.Context, workspace *biz.Workspace) error {
	return w.data.db.Save(workspace).Error
}

func (w *workspaceRepo) List(ctx context.Context, clusterId int64, workspaceName string) ([]*biz.Workspace, error) {
	workspaces := make([]*biz.Workspace, 0)
	db := w.data.db
	if clusterId > 0 {
		db = db.Where("cluster_id = ?", clusterId)
	}
	if workspaceName != "" {
		db = db.Where("name LIKE ?", "%"+workspaceName+"%")
	}
	err := db.Find(&workspaces).Error
	if err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (w *workspaceRepo) GetByName(ctx context.Context, name string) (*biz.Workspace, error) {
	workspace := &biz.Workspace{}
	if err := w.data.db.Where("name = ?", name).First(workspace).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return workspace, nil
}
