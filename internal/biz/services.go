package biz

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type Service struct {
	ID           int    `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name         string `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	NameSpace    string `json:"namespace,omitempty" gorm:"column:namespace; default:''; NOT NULL"`
	Repo         string `json:"repo,omitempty" gorm:"column:repo; default:''; NOT NULL"`
	Registry     string `json:"registry" gorm:"column:registry; default:''; NOT NULL"`
	RegistryUser string `json:"registry_user" gorm:"column:registry_user; default:''; NOT NULL"`
	RegistryPwd  string `json:"registry_pwd" gorm:"column:registry_pwd; default:''; NOT NULL"`
	Workflow     string `json:"workflow" gorm:"column:workflow; type:text"`
	CIItems      []*CI  `json:"ci_items,omitempty" gorm:"-"`
	Replicas     int32  `json:"replicas" gorm:"column:replicas; default:0; NOT NULL"`
	CPU          string `json:"cpu" gorm:"column:cpu; default:''; NOT NULL"`
	LimitCpu     string `json:"limit_cpu" gorm:"column:limit_cpu; default:''; NOT NULL"`
	Memory       string `json:"memory" gorm:"column:memory; default:''; NOT NULL"`
	LimitMemory  string `json:"limit_memory" gorm:"column:limit_memory; default:''; NOT NULL"`
	Config       string `json:"config" gorm:"column:config; default:''; NOT NULL"`
	Secret       string `json:"secret" gorm:"column:secret; default:''; NOT NULL"`
	Ports        []Port `json:"ports" gorm:"-"`
	gorm.Model
}

type Port struct {
	ID            int    `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	IngressPath   string `json:"ingress_path" gorm:"column:ingress_path; default:''; NOT NULL"`
	ContainerPort int32  `json:"container_port" gorm:"column:container_port; default:0; NOT NULL"`
}

type CI struct {
	ID           int    `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Version      string `json:"version,omitempty" gorm:"column:version; default:''; NOT NULL"`
	Branch       string `json:"branch,omitempty" gorm:"column:branch; default:''; NOT NULL"`
	Tag          string `json:"tag,omitempty" gorm:"column:tag; default:''; NOT NULL"`
	Args         string `json:"args,omitempty" gorm:"column:args; type:json"`
	Description  string `json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`
	WorkflowName string `json:"workflow_name,omitempty" gorm:"column:workflow_name; default:''; NOT NULL"`
	ServiceID    int    `json:"service_id,omitempty" gorm:"column:service_id; default:0; NOT NULL"`
	gorm.Model
}

func (c *CI) SetServiceID(id int) {
	c.ServiceID = id
}

func (c *CI) SetWorkflowName(name string) {
	c.WorkflowName = name
}

type ServicesRepo interface {
	Save(context.Context, *Service) error
	Get(context.Context, int) (*Service, error)
	GetServices(context.Context) ([]*Service, error)
	Delete(context.Context, *Service) error
	SaveCI(context.Context, *CI) error
	GetCI(context.Context, int) (*CI, error)
	GetCIs(context.Context, int) ([]*CI, error)
	DeleteCI(ctx context.Context, ci *CI) error
	Deploy(context.Context, *Service, *CI) error
	UnDeploy(context.Context, *Service) error
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
	services, err := s.GetServices(ctx)
	if err != nil {
		return err
	}
	for _, s := range services {
		if s.Name == service.Name && s.NameSpace == service.NameSpace && service.ID == 0 {
			service.ID = s.ID
		}
	}
	return s.repo.Save(ctx, service)
}

func (s *ServicesUseCase) GetService(ctx context.Context, id int) (*Service, error) {
	return s.repo.Get(ctx, id)
}

func (s *ServicesUseCase) GetServices(ctx context.Context) ([]*Service, error) {
	// 不包含ci，只有service列表
	return s.repo.GetServices(ctx)
}

func (s *ServicesUseCase) DeleteService(ctx context.Context, id int) error {
	service, err := s.GetService(ctx, id)
	if err != nil || service == nil {
		return err
	}
	// 需要把相关ci删除掉
	return s.repo.Delete(ctx, service)
}

func (s *ServicesUseCase) SaveCI(ctx context.Context, ci *CI) error {
	if ci.ServiceID == 0 {
		return errors.New("service id is empty")
	}
	service, err := s.GetService(ctx, ci.ServiceID)
	if err != nil {
		return err
	}
	if service == nil {
		return errors.New("service not found")
	}
	return s.repo.SaveCI(ctx, ci)
}

func (s *ServicesUseCase) GetCI(ctx context.Context, id int) (*CI, error) {
	return s.repo.GetCI(ctx, id)
}

func (s *ServicesUseCase) GetCIs(ctx context.Context, serviceID int) ([]*CI, error) {
	return s.repo.GetCIs(ctx, serviceID)
}

func (s *ServicesUseCase) DeleteCI(ctx context.Context, CIID int) error {
	ci, err := s.GetCI(ctx, CIID)
	if err != nil {
		return err
	}
	return s.repo.DeleteCI(ctx, ci)
}

func (s *ServicesUseCase) Deploy(ctx context.Context, CIID int) error {
	// 新建app，有Operator维护和创建资源
	ci, err := s.repo.GetCI(ctx, CIID)
	if err != nil {
		return err
	}
	service, err := s.GetService(ctx, ci.ServiceID)
	if err != nil {
		return err
	}
	return s.repo.Deploy(ctx, service, ci)
}

func (s *ServicesUseCase) UnDeploy(ctx context.Context, serviceID int) error {
	service, err := s.GetService(ctx, serviceID)
	if err != nil {
		return err
	}
	return s.repo.UnDeploy(ctx, service)
}

func (s *ServicesUseCase) GetOceanService(ctx context.Context) (*Service, error) {
	return s.repo.GetOceanService(ctx)
}
