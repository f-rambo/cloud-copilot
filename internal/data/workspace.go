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
	workspace := &biz.Workspace{}
	err := w.data.db.Model(&biz.Workspace{}).Where("id = ?", id).First(workspace).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	workspaceClusterRelationships := make([]*biz.WorkspaceClusterRelationship, 0)
	err = w.data.db.Model(&biz.WorkspaceClusterRelationship{}).Where("workspace_id = ?", id).
		Find(&workspaceClusterRelationships).Error
	if err != nil {
		return nil, err
	}
	workspace.WorkspaceClusterRelationships = workspaceClusterRelationships
	return workspace, nil
}

func (w *workspaceRepo) Save(ctx context.Context, workspace *biz.Workspace) error {
	return w.data.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(workspace).Error; err != nil {
			return err
		}

		var existingRelationships []*biz.WorkspaceClusterRelationship
		if err := tx.Where("workspace_id = ?", workspace.Id).Find(&existingRelationships).Error; err != nil {
			return err
		}

		existingMap := make(map[int64]*biz.WorkspaceClusterRelationship)
		for _, rel := range existingRelationships {
			existingMap[rel.ClusterId] = rel
		}

		currentMap := make(map[int64]*biz.WorkspaceClusterRelationship)
		for _, rel := range workspace.WorkspaceClusterRelationships {
			rel.WorkspaceId = workspace.Id // 确保workspace_id正确
			currentMap[rel.ClusterId] = rel
		}

		for clusterId, existingRel := range existingMap {
			if _, exists := currentMap[clusterId]; !exists {
				if err := tx.Delete(existingRel).Error; err != nil {
					return err
				}
			}
		}

		for _, currentRel := range workspace.WorkspaceClusterRelationships {
			if existingRel, exists := existingMap[currentRel.ClusterId]; exists {

				currentRel.Id = existingRel.Id
			}
			if err := tx.Save(currentRel).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (w *workspaceRepo) List(ctx context.Context, workspaceName string, page, size int32) ([]*biz.Workspace, int64, error) {
	workspaces := make([]*biz.Workspace, 0)
	db := w.data.db
	if workspaceName != "" {
		db = db.Where("name LIKE ?", "%"+workspaceName+"%")
	}

	var total int64 = 0
	if err := db.Model(&biz.Workspace{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return workspaces, total, nil
	}

	offset := (page - 1) * size
	err := db.Offset(int(offset)).Limit(int(size)).Find(&workspaces).Error
	if err != nil {
		return nil, 0, err
	}

	workspaceIds := make([]int64, 0)
	for _, v := range workspaces {
		workspaceIds = append(workspaceIds, v.Id)
	}
	workspaceClusterRelationships := make([]*biz.WorkspaceClusterRelationship, 0)
	err = db.Model(&biz.WorkspaceClusterRelationship{}).Where("workspace_id in (?)", workspaceIds).
		Find(&workspaceClusterRelationships).Error
	if err != nil {
		return workspaces, total, nil
	}

	// 创建一个map来按workspaceId分组关系数据
	relationshipMap := make(map[int64][]*biz.WorkspaceClusterRelationship)
	for _, relationship := range workspaceClusterRelationships {
		relationshipMap[relationship.WorkspaceId] = append(relationshipMap[relationship.WorkspaceId], relationship)
	}

	// 将关系数据分配到对应的workspace上
	for _, workspace := range workspaces {
		if relationships, exists := relationshipMap[workspace.Id]; exists {
			workspace.WorkspaceClusterRelationships = relationships
		}
	}

	return workspaces, total, nil
}

func (w *workspaceRepo) GetByName(ctx context.Context, name string) (*biz.Workspace, error) {
	workspace := &biz.Workspace{}
	if err := w.data.db.Where("name = ?", name).First(workspace).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return workspace, nil
}

func (w *workspaceRepo) Delete(ctx context.Context, workspace *biz.Workspace) error {
	return w.data.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("workspace_id = ?", workspace.Id).Delete(&biz.WorkspaceClusterRelationship{}).Error; err != nil {
			return err
		}

		if err := tx.Delete(workspace).Error; err != nil {
			return err
		}

		return nil
	})
}
