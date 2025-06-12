package biz

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
)

const (
	WorkspaceKey ContextKey = "workspace"
)

type AlreadyResource struct {
	Cpu     int32 `json:"cpu,omitempty"`
	Memory  int32 `json:"memory,omitempty"`
	Gpu     int32 `json:"gpu,omitempty"`
	Storage int32 `json:"storage,omitempty"`
}

type ResourceQuotaString string

type ResourceQuota struct {
	CPU     ResourceLimit `json:"cpu"`
	Memory  ResourceLimit `json:"memory"`
	GPU     ResourceLimit `json:"gpu"`
	Storage ResourceLimit `json:"storage"`
	Pods    ResourceLimit `json:"pods"`
}

type ResourceLimit struct {
	Request int32 `json:"request"`
	Limit   int32 `json:"limit"`
}

func (r ResourceQuota) ToString() ResourceQuotaString {
	b, _ := json.Marshal(r)
	return ResourceQuotaString(b)
}

func (rs ResourceQuotaString) ToResourceQuota() ResourceQuota {
	var r ResourceQuota
	_ = json.Unmarshal([]byte(rs), &r)
	return r
}

type WorkspaceStatus int32

const (
	WorkspaceStatus_CREATING WorkspaceStatus = 0
	WorkspaceStatus_ACTIVE   WorkspaceStatus = 1
	WorkspaceStatus_INACTIVE WorkspaceStatus = 2
	WorkspaceStatus_DELETING WorkspaceStatus = 3
)

func (ws WorkspaceStatus) String() string {
	switch ws {
	case WorkspaceStatus_CREATING:
		return "creating"
	case WorkspaceStatus_ACTIVE:
		return "active"
	case WorkspaceStatus_INACTIVE:
		return "inactive"
	case WorkspaceStatus_DELETING:
		return "deleting"
	default:
		return "unknown"
	}
}

type Workspace struct {
	Id                            int64                           `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Title                         string                          `json:"title,omitempty" gorm:"column:title;default:'';NOT NULL"`
	Name                          string                          `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"` // as namespace
	Description                   string                          `json:"description,omitempty" gorm:"column:namespace;default:'';NOT NULL"`
	UserId                        int64                           `json:"user_id,omitempty" gorm:"column:user_id;default:0;NOT NULL"`
	ResourceQuota                 ResourceQuotaString             `json:"resource_quota,omitempty" gorm:"column:resource_quota;default:'';NOT NULL"`
	GitRepository                 string                          `json:"git_repository,omitempty" gorm:"column:git_repository;default:'';NOT NULL"`
	GitRepositoryToken            string                          `json:"git_repository_token,omitempty" gorm:"column:gitrepository_token;default:'';NOT NULL"`
	ImageRepository               string                          `json:"image_repository,omitempty" gorm:"column:image_repository;default:'';NOT NULL"`
	ImageRepositoryToken          string                          `json:"image_repository_token,omitempty" gorm:"column:imagerepository_token;default:'';NOT NULL"`
	Status                        WorkspaceStatus                 `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	WorkspaceClusterRelationships []*WorkspaceClusterRelationship `json:"workspace_cluster_relationships,omitempty" gorm:"-"`
}

func (w *Workspace) GetGitRepoName() string {
	return w.GitRepository[strings.LastIndex(w.GitRepository, "/")+1:]
}

func (w *Workspace) GetImageRepoName() string {
	return w.ImageRepository[strings.LastIndex(w.ImageRepository, "/")+1:]
}

type WorkspaceClusterPermissions int32

const (
	WorkspaceClusterPermissions_READ  WorkspaceClusterPermissions = 0
	WorkspaceClusterPermissions_WRITE WorkspaceClusterPermissions = 1
	WorkspaceClusterPermissions_ADMIN WorkspaceClusterPermissions = 2
)

func (wcp WorkspaceClusterPermissions) String() string {
	switch wcp {
	case WorkspaceClusterPermissions_READ:
		return "read"
	case WorkspaceClusterPermissions_WRITE:
		return "write"
	case WorkspaceClusterPermissions_ADMIN:
		return "admin"
	default:
		return "unknown"
	}
}

func WorkspaceClusterPermissionsFromString(s string) WorkspaceClusterPermissions {
	switch s {
	case "read":
		return WorkspaceClusterPermissions_READ
	case "write":
		return WorkspaceClusterPermissions_WRITE
	case "admin":
		return WorkspaceClusterPermissions_ADMIN
	default:
		return WorkspaceClusterPermissions_READ
	}
}

type WorkspaceClusterRelationship struct {
	Id          int64                       `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	WorkspaceId int64                       `json:"workspace_id,omitempty" gorm:"column:workspace_id;default:0;NOT NULL"`
	ClusterId   int64                       `json:"cluster_id,omitempty" gorm:"column:cluster_id;default:0;NOT NULL"`
	Permissions WorkspaceClusterPermissions `json:"permissions,omitempty" gorm:"column:permissions;default:0;NOT NULL"`
}

type WorkspaceData interface {
	Get(ctx context.Context, id int64) (*Workspace, error)
	Save(context.Context, *Workspace) error
	List(ctx context.Context, workspaceName string, page, size int32) ([]*Workspace, int64, error)
	Delete(context.Context, *Workspace) error
	GetByName(ctx context.Context, name string) (*Workspace, error)
}

type WorkspaceRuntime interface {
	Reload(context.Context, *Workspace) error
	Delete(context.Context, *Workspace) error
}

type WorkspaceUsecase struct {
	workspaceData WorkspaceData
	log           *log.Helper
}

func NewWorkspaceUsecase(workspaceData WorkspaceData, logger log.Logger) *WorkspaceUsecase {
	return &WorkspaceUsecase{
		workspaceData: workspaceData,
		log:           log.NewHelper(logger),
	}
}

func GetWorkspace(ctx context.Context) *Workspace {
	v, ok := ctx.Value(WorkspaceKey).(*Workspace)
	if !ok {
		return nil
	}
	return v
}

func WithWorkspace(ctx context.Context, w *Workspace) context.Context {
	return context.WithValue(ctx, WorkspaceKey, w)
}

func (w *Workspace) GetLabels() map[string]string {
	return map[string]string{
		"workspace": w.Name,
	}
}

func (uc *WorkspaceUsecase) Get(ctx context.Context, id int64) (*Workspace, error) {
	workspace, err := uc.workspaceData.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return workspace, nil
}

func (uc *WorkspaceUsecase) Save(ctx context.Context, workspace *Workspace) error {
	if strings.TrimSpace(workspace.Name) == "" {
		return errors.New("workspace name cannot be empty")
	}
	return uc.workspaceData.Save(ctx, workspace)
}

func (uc *WorkspaceUsecase) List(ctx context.Context, workspaceName string, page, size int32) ([]*Workspace, int64, error) {
	return uc.workspaceData.List(ctx, workspaceName, page, size)
}

func (uc *WorkspaceUsecase) GetByName(ctx context.Context, name string) (*Workspace, error) {
	workspace, err := uc.workspaceData.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return workspace, nil
}

func (uc *WorkspaceUsecase) Delete(ctx context.Context, workspaceId int64) error {
	workspace, err := uc.workspaceData.Get(ctx, workspaceId)
	if err != nil {
		return err
	}
	if workspace == nil || workspace.Id <= 0 {
		return errors.New("workspace not found")
	}
	return uc.workspaceData.Delete(ctx, workspace)
}
