package biz

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type Service struct {
	ID           int            `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name         string         `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Repo         string         `json:"repo,omitempty" gorm:"column:repo; default:''; NOT NULL"`
	Registry     string         `json:"registry" gorm:"column:registry; default:''; NOT NULL"`
	RegistryUser string         `json:"registry_user" gorm:"column:registry_user; default:''; NOT NULL"`
	RegistryPwd  string         `json:"registry_pwd" gorm:"column:registry_pwd; default:''; NOT NULL"`
	CIItems      []CI           `json:"ci_items,omitempty" gorm:"-"`
	Workflow     *wfv1.Workflow `json:"workflow,omitempty" gorm:"column:workflow; type:json"`
	gorm.Model
}

type CI struct {
	ID          int               `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Version     string            `json:"version,omitempty" gorm:"column:version; default:''; NOT NULL"`
	Branch      string            `json:"branch,omitempty" gorm:"column:branch; default:''; NOT NULL"`
	Tag         string            `json:"tag,omitempty" gorm:"column:tag; default:''; NOT NULL"`
	Args        map[string]string `json:"args,omitempty" gorm:"column:args; type:json"`
	Description string            `json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`
	SubmitWfRes *wfv1.Workflow    `json:"submit_wf_res,omitempty" gorm:"column:submit_wf_res; type:json"`
	ServiceID   int               `json:"service_id,omitempty" gorm:"column:service_id; default:0; NOT NULL"`
	gorm.Model
}

type ServicesRepo interface {
	Save(context.Context, *Service) error
	Get(context.Context, int) (*Service, error)
	GetServices(context.Context) ([]*Service, error)
	Delete(context.Context, *Service) error
	SaveCI(context.Context, *CI) error
	CreateWf(context.Context, *Service, *CI) error
	GetCI(context.Context, int) (*CI, error)
	GetWf(context.Context, *Service, *CI) (*wfv1.Workflow, error)
	GetCIs(context.Context) ([]*CI, error)
	Deploy(context.Context, *Service, *CI) error
	GetOceanService(context.Context) (*Service, error)
}

type ServicesUseCase struct {
	repo ServicesRepo
	log  *log.Helper
}

func NewServicesUseCase(repo ServicesRepo, logger log.Logger) *ServicesUseCase {
	return &ServicesUseCase{repo: repo, log: log.NewHelper(logger)}
}

func (s *ServicesUseCase) SaveService(ctx context.Context, service *Service) error {
	return nil
}

func (s *ServicesUseCase) GetService(ctx context.Context, id int) (*Service, error) {
	return nil, nil
}

func (s *ServicesUseCase) GetServices(ctx context.Context) ([]*Service, error) {
	return nil, nil
}

func (s *ServicesUseCase) DeleteService(ctx context.Context, id int) error {
	return nil
}

func (s *ServicesUseCase) SaveCI(ctx context.Context, ci *CI) error {
	return nil
}

func (s *ServicesUseCase) GetCI(ctx context.Context, id int) (*CI, error) {
	return nil, nil
}

func (s *ServicesUseCase) GetCIs(ctx context.Context) ([]*CI, error) {
	return nil, nil
}

func (s *ServicesUseCase) Deploy(ctx context.Context, CIID int) error {
	// 新建app，有Operator维护和创建资源
	return nil
}

func (s *ServicesUseCase) GetOceanService(ctx context.Context) (*Service, error) {
	return s.repo.GetOceanService(ctx)
}
