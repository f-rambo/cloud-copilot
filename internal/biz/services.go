package biz

import (
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

type ServicesRepo interface {
}

type ServicesUseCase struct {
	repo ServicesRepo
	log  *log.Helper
}

func NewServicesUseCase(repo ServicesRepo, logger log.Logger) *ServicesUseCase {
	return &ServicesUseCase{repo: repo, log: log.NewHelper(logger)}
}
