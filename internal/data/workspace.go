package data

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
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
