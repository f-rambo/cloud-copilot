package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/f-rambo/ocean/pkg/argoworkflows"
	"github.com/f-rambo/ocean/pkg/kubeclient"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type Service struct {
	ID           int64   `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name         string  `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	CodeRepo     string  `json:"code_repo,omitempty" gorm:"column:code_repo; default:''; NOT NULL"`   // git repo url
	Replicas     int32   `json:"replicas" gorm:"column:replicas; default:0; NOT NULL"`                // 0 == auto
	CPU          float32 `json:"cpu" gorm:"column:cpu; default:0; NOT NULL"`                          // 0.5=500m
	LimitCpu     float32 `json:"limit_cpu" gorm:"column:limit_cpu; default:0; NOT NULL"`              // 0 == auto
	GPU          float32 `json:"gpu" gorm:"column:gpu; default:0; NOT NULL"`                          // 0 == no gpu
	LimitGPU     float32 `json:"limit_gpu" gorm:"column:limit_gpu; default:0; NOT NULL"`              // GPU !==0 LimitGPU 0 == auto
	Memory       float32 `json:"memory" gorm:"column:memory; default:0; NOT NULL"`                    // 0.5=500m
	LimitMemory  float32 `json:"limit_memory" gorm:"column:limit_memory; default:0; NOT NULL"`        // 0 == auto
	Disk         float32 `json:"disk" gorm:"column:disk; default:0; NOT NULL"`                        // 0.5=500m
	LimitDisk    float32 `json:"limit_disk" gorm:"column:limit_disk; default:0; NOT NULL"`            // 0 == auto
	Business     string  `json:"business,omitempty" gorm:"column:business; default:''; NOT NULL"`     // business name
	Technology   string  `json:"technology,omitempty" gorm:"column:technology; default:''; NOT NULL"` // technology name
	Ports        []Port  `json:"ports" gorm:"-"`
	ProjectID    int64   `json:"project_id,omitempty" gorm:"column:project_id; default:0; NOT NULL"`
	PortsJson    []byte  `json:"ports_json" gorm:"column:ports_json; type:json"`
	CIWorklfowID int64   `json:"ci_workflow_id,omitempty" gorm:"column:ci_workflow_id; default:0; NOT NULL"`
	CDWorklfowID int64   `json:"cd_workflow_id,omitempty" gorm:"column:cd_workflow_id; default:0; NOT NULL"`
	gorm.Model
}

type Port struct {
	ID            int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	IngressPath   string `json:"ingress_path" gorm:"column:ingress_path; default:''; NOT NULL"`    // /api/v1
	Protocol      string `json:"protocol" gorm:"column:protocol; default:''; NOT NULL"`            // TCP, UDP
	ContainerPort int32  `json:"container_port" gorm:"column:container_port; default:0; NOT NULL"` // 80
}

type CI struct {
	ID          int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Version     string `json:"version,omitempty" gorm:"column:version; default:''; NOT NULL"`
	Branch      string `json:"branch,omitempty" gorm:"column:branch; default:''; NOT NULL"`
	Tag         string `json:"tag,omitempty" gorm:"column:tag; default:''; NOT NULL"`
	Description string `json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`
	ServiceID   int64  `json:"service_id,omitempty" gorm:"column:service_id; default:0; NOT NULL"`
	UserID      int64  `json:"user_id,omitempty" gorm:"column:user_id; default:0; NOT NULL"`
	Logs        string `json:"logs" gorm:"-"`
	gorm.Model
}

type CD struct {
	ID        int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	CIID      int64  `json:"ci_id" gorm:"column:ci_id; default:0; NOT NULL"`
	ServiceID int64  `json:"service_id" gorm:"column:service_id; default:0; NOT NULL"`
	UserID    int64  `json:"user_id" gorm:"column:user_id; default:0; NOT NULL"`
	Logs      string `json:"logs" gorm:"-"`
	gorm.Model
}

type Workflow struct {
	ID       int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name     string `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Workflow []byte `json:"workflow" gorm:"column:workflow; type:json"`
}

type ServicesRepo interface {
	List(ctx context.Context, serviceParam *Service, page, pageSize int) ([]*Service, int64, error)
	Save(ctx context.Context, service *Service) error
	Get(ctx context.Context, id int64) (*Service, error)
	Delete(ctx context.Context, id int64) error
	GetWorkflow(ctx context.Context, id int64) (*Workflow, error)
	SaveWrkflow(ctx context.Context, workflow *Workflow) error
	DeleteWrkflow(ctx context.Context, id int64) error
}

type ServicesUseCase struct {
	repo ServicesRepo
	log  *log.Helper
}

const (
	ci = "ci"
	cd = "cd"
)

func NewServicesUseCase(repo ServicesRepo, logger log.Logger) *ServicesUseCase {
	return &ServicesUseCase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *ServicesUseCase) List(ctx context.Context, serviceParam *Service, page, pageSize int) ([]*Service, int64, error) {
	return uc.repo.List(ctx, serviceParam, page, pageSize)
}

func (uc *ServicesUseCase) Save(ctx context.Context, service *Service) error {
	return uc.repo.Save(ctx, service)
}

func (uc *ServicesUseCase) Get(ctx context.Context, id int64) (*Service, error) {
	return uc.repo.Get(ctx, id)
}

func (uc *ServicesUseCase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *ServicesUseCase) GetWorkflow(ctx context.Context, id int64, args string) (*Workflow, error) {
	service, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if args == "ci" && service.CIWorklfowID != 0 {
		return uc.repo.GetWorkflow(ctx, service.CIWorklfowID)
	}
	if args == "cd" && service.CDWorklfowID != 0 {
		return uc.repo.GetWorkflow(ctx, service.CDWorklfowID)
	}
	argoWorkflowJson, err := argoworkflows.GetDefaultWorklfows(ctx, strings.ToLower(service.Business), strings.ToLower(service.Technology), args)
	if err != nil {
		return nil, err
	}
	return &Workflow{
		Name:     fmt.Sprintf("default-%s-%s-%s", service.Business, service.Technology, args),
		Workflow: argoWorkflowJson,
	}, nil
}

func (uc *ServicesUseCase) SaveWorkflow(ctx context.Context, serviceId int64, wfType string, wf *Workflow) error {
	service, err := uc.Get(ctx, serviceId)
	if err != nil {
		return err
	}
	if wfType == ci && service.CIWorklfowID != 0 {
		wf.ID = service.CIWorklfowID
	}
	if wfType == cd && service.CDWorklfowID != 0 {
		wf.ID = service.CDWorklfowID
	}
	err = uc.repo.SaveWrkflow(ctx, wf)
	if err != nil {
		return err
	}
	if wfType == ci {
		service.CIWorklfowID = wf.ID
	}
	if wfType == cd {
		service.CDWorklfowID = wf.ID
	}
	return uc.repo.Save(ctx, service)
}

func (uc *ServicesUseCase) CommitWorklfow(ctx context.Context, project *Project, service *Service, wfType string, workflowsId int64) error {
	if wfType == ci && service.CIWorklfowID != workflowsId {
		return fmt.Errorf("ci workflow not match")
	}
	if wfType == cd && service.CDWorklfowID != workflowsId {
		return fmt.Errorf("cd workflow not match")
	}
	wf, err := uc.repo.GetWorkflow(ctx, workflowsId)
	if err != nil {
		return err
	}
	kubeConf, err := kubeclient.GetKubeConfig()
	if err != nil {
		return err
	}
	argoClient, err := argoworkflows.NewForConfig(kubeConf)
	if err != nil {
		return err
	}
	argoWf := &wfv1.Workflow{}
	err = json.Unmarshal(wf.Workflow, argoWf)
	if err != nil {
		return err
	}
	_, err = argoClient.Workflows(project.Namespace).Create(ctx, argoWf)
	if err != nil {
		return err
	}
	return nil
}
