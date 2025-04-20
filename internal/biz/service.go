package biz

import (
	"context"

	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
)

type ServiceEnv int32

const (
	ServiceEnv_UNSPECIFIED      ServiceEnv = 0
	ServiceEnv_SERVICE_NAME     ServiceEnv = 1
	ServiceEnv_VERSION          ServiceEnv = 2
	ServiceEnv_BRANCH           ServiceEnv = 3
	ServiceEnv_TAG              ServiceEnv = 4
	ServiceEnv_COMMIT_ID        ServiceEnv = 5
	ServiceEnv_SERVICE_ID       ServiceEnv = 6
	ServiceEnv_IMAGE            ServiceEnv = 7
	ServiceEnv_GIT_REPO         ServiceEnv = 8
	ServiceEnv_IMAGE_REPO       ServiceEnv = 9
	ServiceEnv_GIT_REPO_NAME    ServiceEnv = 10
	ServiceEnv_IMAGE_REPO_NAME  ServiceEnv = 11
	ServiceEnv_GIT_REPO_TOKEN   ServiceEnv = 12
	ServiceEnv_IMAGE_REPO_TOKEN ServiceEnv = 13
)

// ServiceEnv to string
func (s ServiceEnv) String() string {
	switch s {
	case ServiceEnv_SERVICE_NAME:
		return "SERVICE_NAME"
	case ServiceEnv_VERSION:
		return "VERSION"
	case ServiceEnv_BRANCH:
		return "BRANCH"
	case ServiceEnv_TAG:
		return "TAG"
	case ServiceEnv_COMMIT_ID:
		return "COMMIT_ID"
	case ServiceEnv_SERVICE_ID:
		return "SERVICE_ID"
	case ServiceEnv_IMAGE:
		return "IMAGE"
	case ServiceEnv_GIT_REPO:
		return "GIT_REPO"
	case ServiceEnv_IMAGE_REPO:
		return "IMAGE_REPO"
	case ServiceEnv_GIT_REPO_NAME:
		return "GIT_REPO_NAME"
	case ServiceEnv_IMAGE_REPO_NAME:
		return "IMAGE_REPO_NAME"
	case ServiceEnv_GIT_REPO_TOKEN:
		return "GIT_REPO_TOKEN"
	case ServiceEnv_IMAGE_REPO_TOKEN:
		return "IMAGE_REPO_TOKEN"
	default:
		return ""
	}
}

// ServiceEnv items
func ServiceEnvItems() []ServiceEnv {
	return []ServiceEnv{
		ServiceEnv_SERVICE_NAME,
		ServiceEnv_VERSION,
		ServiceEnv_BRANCH,
		ServiceEnv_TAG,
		ServiceEnv_COMMIT_ID,
		ServiceEnv_SERVICE_ID,
		ServiceEnv_IMAGE,
		ServiceEnv_GIT_REPO,
		ServiceEnv_IMAGE_REPO,
		ServiceEnv_GIT_REPO_NAME,
		ServiceEnv_IMAGE_REPO_NAME,
		ServiceEnv_GIT_REPO_TOKEN,
		ServiceEnv_IMAGE_REPO_TOKEN,
	}
}

type AccessExternal int32

const (
	AccessExternal_UNSPECIFIED AccessExternal = 0
	AccessExternal_True        AccessExternal = 1
	AccessExternal_False       AccessExternal = 2
)

type ServiceStatus int32

const (
	ServiceStatus_UNSPECIFIED ServiceStatus = 0
	ServiceStatus_Starting    ServiceStatus = 1
	ServiceStatus_Running     ServiceStatus = 2
	ServiceStatus_Terminated  ServiceStatus = 3
)

type ContinuousIntegration struct {
	Id              int64         `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Version         string        `json:"version,omitempty" gorm:"column:version;default:'';NOT NULL"`
	Branch          string        `json:"branch,omitempty" gorm:"column:branch;default:'';NOT NULL"`
	CommitId        string        `json:"commit_id,omitempty" gorm:"column:commit_id;default:'';NOT NULL"`
	Tag             string        `json:"tag,omitempty" gorm:"column:tag;default:'';NOT NULL"`
	Status          WorkfloStatus `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	Description     string        `json:"description,omitempty" gorm:"column:description;default:'';NOT NULL"`
	ServiceId       int64         `json:"service_id,omitempty" gorm:"column:service_id;default:0;NOT NULL;index:idx_ci_service_id"`
	UserId          int64         `json:"user_id,omitempty" gorm:"column:user_id;default:0;NOT NULL;index:idx_ci_user_id"`
	WorkflowRuntime string        `json:"workflow_runtime,omitempty" gorm:"column:workflow_runtime;default:'';NOT NULL"`
	Logs            string        `json:"logs,omitempty" gorm:"-"`
}

type ContinuousDeployment struct {
	Id               int64             `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	CiId             int64             `json:"ci_id,omitempty" gorm:"column:ci_id;default:0;NOT NULL;index:idx_ci_id"`
	ServiceId        int64             `json:"service_id,omitempty" gorm:"column:service_id;default:0;NOT NULL;index:idx_cd_service_id"`
	UserId           int64             `json:"user_id,omitempty" gorm:"column:user_id;default:0;NOT NULL"`
	Status           WorkfloStatus     `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	Image            string            `json:"image,omitempty" gorm:"column:image;default:'';NOT NULL"`
	WorkflowRuntime  string            `json:"workflow_runtime,omitempty" gorm:"column:workflow_runtime;default:'';NOT NULL"`
	Config           map[string]string `json:"config,omitempty" gorm:"-"` // key: filename, value: content
	CanaryDeployment *CanaryDeployment `json:"canary_deployment,omitempty" gorm:"-"`
	IsAccessExternal AccessExternal    `json:"is_access_external,omitempty" gorm:"column:is_access_external;default:0;NOT NULL"`
	Logs             string            `json:"logs,omitempty" gorm:"-"`
}

type CanaryDeployment struct {
	CdId       int64             `json:"cd_id,omitempty" gorm:"column:cd_id;default:0;NOT NULL;index:idx_cd_id"`
	Image      string            `json:"image,omitempty" gorm:"column:image;default:'';NOT NULL"`
	Replicas   int32             `json:"replicas,omitempty" gorm:"column:replicas;default:0;NOT NULL"`
	Config     map[string]string `json:"config,omitempty" gorm:"-"` // key: filename, value: content
	TrafficPct int32             `json:"traffic_pct,omitempty" gorm:"column:traffic_pct;default:0;NOT NULL"`
}

type Port struct {
	Id            int64  `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name          string `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	IngressPath   string `json:"ingress_path,omitempty" gorm:"column:ingress_path;default:'';NOT NULL"`
	Protocol      string `json:"protocol,omitempty" gorm:"column:protocol;default:'';NOT NULL"`
	ContainerPort int32  `json:"container_port,omitempty" gorm:"column:container_port;default:0;NOT NULL"`
	ServiceId     int64  `json:"service_id,omitempty" gorm:"column:service_id;default:0;NOT NULL;index:idx_port_service_id"`
}

type Volume struct {
	Id           int64  `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name         string `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	MountPath    string `json:"mount_path,omitempty" gorm:"column:mount_path;default:'';NOT NULL"`
	Storage      int32  `json:"storage,omitempty" gorm:"column:storage;default:0;NOT NULL"`
	StorageClass string `json:"storage_class,omitempty" gorm:"column:storage_class;default:'';NOT NULL"`
	ServiceId    int64  `json:"service_id,omitempty" gorm:"column:service_id;default:0;NOT NULL;index:idx_volume_service_id"`
}

type Service struct {
	Id            int64         `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name          string        `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Namespace     string        `json:"namespace,omitempty" gorm:"column:namespace;default:'';NOT NULL"`
	Lables        string        `json:"lables,omitempty" gorm:"column:lables;default:'';NOT NULL"`
	Replicas      int32         `json:"replicas,omitempty" gorm:"column:replicas;default:0;NOT NULL"`
	RequestCpu    int32         `json:"request_cpu,omitempty" gorm:"column:request_cpu;default:0;NOT NULL"`
	LimitCpu      int32         `json:"limit_cpu,omitempty" gorm:"column:limit_cpu;default:0;NOT NULL"`
	RequestGpu    int32         `json:"request_gpu,omitempty" gorm:"column:request_gpu;default:0;NOT NULL"`
	LimitGpu      int32         `json:"limit_gpu,omitempty" gorm:"column:limit_gpu;default:0;NOT NULL"`
	RequestMemory int32         `json:"request_memory,omitempty" gorm:"column:request_memory;default:0;NOT NULL"`
	LimitMemory   int32         `json:"limit_memory,omitempty" gorm:"column:limit_memory;default:0;NOT NULL"`
	Volumes       []*Volume     `json:"volumes,omitempty" gorm:"-"`
	Gateway       string        `json:"gateway,omitempty" gorm:"column:gateway;default:'';NOT NULL"`
	Ports         []*Port       `json:"ports,omitempty" gorm:"-"`
	StorageClass  string        `json:"storage_class,omitempty" gorm:"column:storage_class;default:'';NOT NULL"`
	ProjectId     int64         `json:"project_id,omitempty" gorm:"column:project_id;default:0;NOT NULL;index:idx_project_id"`
	WorkspaceId   int64         `json:"workspace_id,omitempty" gorm:"column:workspace_id;default:0;NOT NULL;index:idx_workspace_id"`
	ClusterId     int64         `json:"cluster_id,omitempty" gorm:"column:cluster_id;default:0;NOT NULL;index:idx_cluster_id"`
	UserId        int64         `json:"user_id,omitempty" gorm:"column:user_id;default:0;NOT NULL;index:idx_service_user_id"`
	Status        ServiceStatus `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	Description   string        `json:"description,omitempty" gorm:"column:description;default:'';NOT NULL"`
	Log           string        `json:"log,omitempty" gorm:"-"`
}

type ServicesData interface {
	Save(ctx context.Context, service *Service) error
	Get(ctx context.Context, id int64) (*Service, error)
	List(ctx context.Context, projectId int64, serviceName string, page, pageSize int32) ([]*Service, int64, error)
	Delete(ctx context.Context, id int64) error
	GetServiceResourceByProject(ctx context.Context, projectId int64, alreadyResource *AlreadyResource) error
	GetByName(ctx context.Context, projectId int64, name string) (*Service, error)
	SaveWorkflow(ctx context.Context, workflow *Workflow) error
	GetWorkflowByServiceId(ctx context.Context, serviceId int64) ([]*Workflow, error)
	SaveContinuousIntegration(context.Context, *ContinuousIntegration) error
	GetContinuousIntegration(context.Context, int64) (*ContinuousIntegration, error)
	DeleteContinuousIntegration(context.Context, int64) error
	GetContinuousIntegrations(ctx context.Context, serviceId int64, page, pageSize int32) ([]*ContinuousIntegration, int64, error)
	SaveContinuousDeployment(context.Context, *ContinuousDeployment) error
	GetContinuousDeployment(context.Context, int64) (*ContinuousDeployment, error)
	DeleteContinuousDeployment(context.Context, int64) error
	GetContinuousDeployments(ctx context.Context, serviceId int64, page, pageSize int32) ([]*ContinuousDeployment, int64, error)
}

type ServiceRuntime interface {
	ApplyService(context.Context, *Service, *ContinuousDeployment) error
	GetServiceStatus(context.Context, *Service) error
}

type ServicesUseCase struct {
	serviceData     ServicesData
	serviceRuntime  ServiceRuntime
	workflowRuntime WorkflowRuntime
	log             *log.Helper
}

func NewServicesUseCase(serviceData ServicesData, serviceRuntime ServiceRuntime, wfRuntime WorkflowRuntime, logger log.Logger) *ServicesUseCase {
	return &ServicesUseCase{serviceData: serviceData, serviceRuntime: serviceRuntime, workflowRuntime: wfRuntime, log: log.NewHelper(logger)}
}

func (s *Service) GetLabels() map[string]string {
	serviceLables := utils.LabelsToMap(s.Lables)
	serviceLables["service"] = s.Name
	return serviceLables
}

func (uc *ServicesUseCase) Save(ctx context.Context, service *Service) error {
	if service.Id == 0 {
		serviceData, err := uc.serviceData.GetByName(ctx, service.ProjectId, service.Name)
		if err != nil {
			return err
		}
		if serviceData.Id > 0 {
			return errors.New("service name already exists")
		}
	}
	return uc.serviceData.Save(ctx, service)
}

func (uc *ServicesUseCase) Get(ctx context.Context, id int64) (*Service, error) {
	service, err := uc.serviceData.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	err = uc.serviceRuntime.GetServiceStatus(ctx, service)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (uc *ServicesUseCase) List(ctx context.Context, projectId int64, serviceName string, page, pageSize int32) ([]*Service, int64, error) {
	return uc.serviceData.List(ctx, projectId, serviceName, page, pageSize)
}

func (uc *ServicesUseCase) Delete(ctx context.Context, id int64) error {
	return uc.serviceData.Delete(ctx, id)
}

func (uc *ServicesUseCase) GetServiceResourceByProject(ctx context.Context, projectId int64) (*AlreadyResource, error) {
	alreadyResource := &AlreadyResource{}
	err := uc.serviceData.GetServiceResourceByProject(ctx, projectId, alreadyResource)
	if err != nil {
		return nil, err
	}
	return alreadyResource, nil
}

func (uc *ServicesUseCase) GetDefaultWorkflow(ctx context.Context, serviceId int64, wfType WorkflowType) (*Workflow, error) {
	service, err := uc.serviceData.Get(ctx, serviceId)
	if err != nil {
		return nil, err
	}
	wf := service.GetDefaultWorkflow(wfType)
	return wf, nil
}

func (uc *ServicesUseCase) SaveWorkflow(ctx context.Context, serviceId int64, wf *Workflow) error {
	service, err := uc.Get(ctx, serviceId)
	if err != nil {
		return err
	}
	if service.Id == 0 {
		return errors.New("service not found")
	}
	if wf.ServiceId == 0 {
		wf.ServiceId = serviceId
	}
	if wf.Id == 0 {
		workflows, err := uc.serviceData.GetWorkflowByServiceId(ctx, serviceId)
		if err != nil {
			return err
		}
		for _, v := range workflows {
			if v.Type == wf.Type {
				return errors.New("workflow already exists")
			}
		}
	}
	return uc.serviceData.SaveWorkflow(ctx, wf)
}

func (uc *ServicesUseCase) GetWorkflow(ctx context.Context, serviceId int64, wfType WorkflowType) (*Workflow, error) {
	workflows, err := uc.serviceData.GetWorkflowByServiceId(ctx, serviceId)
	if err != nil {
		return nil, err
	}
	for _, v := range workflows {
		if v.Type == wfType {
			return v, nil
		}
	}
	return nil, errors.New("workflow not found")
}

func (uc *ServicesUseCase) CreateContinuousIntegration(ctx context.Context, ci *ContinuousIntegration) error {
	service, err := uc.Get(ctx, ci.ServiceId)
	if err != nil {
		return err
	}
	var workflows Workflows
	workflows, err = uc.serviceData.GetWorkflowByServiceId(ctx, service.Id)
	if err != nil {
		return err
	}
	workflow := workflows.GetWorkflowByType(WorkflowType_ContinuousIntegrationType)
	if workflow == nil {
		return errors.New("workflow not found")
	}
	err = uc.workflowRuntime.CleanWorkflow(ctx, workflow)
	if err != nil {
		return err
	}
	workflow.SettingContinuousIntegration(ctx, service, ci)
	err = uc.workflowRuntime.CommitWorkflow(ctx, workflow)
	if err != nil {
		return err
	}
	err = ci.SetWorkflow(workflow)
	if err != nil {
		return err
	}
	ci.Status = WorkfloStatus_Pending
	return uc.serviceData.SaveContinuousIntegration(ctx, ci)
}

func (uc *ServicesUseCase) GetContinuousIntegration(ctx context.Context, ciId int64) (*ContinuousIntegration, *Workflow, error) {
	ci, err := uc.serviceData.GetContinuousIntegration(ctx, ciId)
	if err != nil {
		return nil, nil, err
	}
	workflow, err := ci.GetWorkflow()
	if err != nil {
		return nil, nil, err
	}
	err = uc.workflowRuntime.GetWorkflowStatus(ctx, workflow)
	if err != nil {
		return nil, nil, err
	}
	return ci, workflow, nil
}

func (uc *ServicesUseCase) UpdateContinuousIntegration(ctx context.Context, ciId int64) error {
	ci, workflow, err := uc.GetContinuousIntegration(ctx, ciId)
	if err != nil {
		return err
	}
	err = ci.SetWorkflow(workflow)
	if err != nil {
		return err
	}
	defaultStatus := WorkfloStatus_Pending
	taskPendingNumber := 0
	for _, step := range workflow.WorkflowSteps {
		for _, task := range step.WorkflowTasks {
			if task.Status == WorkfloStatus_Failure {
				defaultStatus = WorkfloStatus_Failure
				break
			}
			if task.Status == WorkfloStatus_Pending {
				taskPendingNumber++
			}
		}
	}
	if defaultStatus == WorkfloStatus_Failure {
		ci.Status = WorkfloStatus_Failure
	}
	if defaultStatus != WorkfloStatus_Failure && taskPendingNumber == 0 {
		ci.Status = WorkfloStatus_Success
	}
	return uc.serviceData.SaveContinuousIntegration(ctx, ci)
}

func (uc *ServicesUseCase) GetContinuousIntegrations(ctx context.Context, serviceId int64, page, pageSize int32) ([]*ContinuousIntegration, int64, error) {
	return uc.serviceData.GetContinuousIntegrations(ctx, serviceId, page, pageSize)
}

func (uc *ServicesUseCase) DeleteContinuousIntegration(ctx context.Context, ciId int64) error {
	return uc.serviceData.DeleteContinuousIntegration(ctx, ciId)
}

func (uc *ServicesUseCase) CreateContinuousDeployment(ctx context.Context, cd *ContinuousDeployment) error {
	user := GetUserInfo(ctx)
	service, err := uc.Get(ctx, cd.ServiceId)
	if err != nil {
		return err
	}
	ci, err := uc.serviceData.GetContinuousIntegration(ctx, cd.CiId)
	if err != nil {
		return err
	}
	cd.Image = ci.GetImage(user, service)
	var workflows Workflows
	workflows, err = uc.serviceData.GetWorkflowByServiceId(ctx, service.Id)
	if err != nil {
		return err
	}
	workflow := workflows.GetWorkflowByType(WorkflowType_ContinuousDeploymentType)
	if workflow == nil {
		return errors.New("workflow not found")
	}
	err = uc.workflowRuntime.CleanWorkflow(ctx, workflow)
	if err != nil {
		return err
	}
	workflow.SettingContinuousDeployment(ctx, service, ci, cd)
	err = uc.workflowRuntime.CommitWorkflow(ctx, workflow)
	if err != nil {
		return err
	}
	err = cd.SetWorkflow(workflow)
	if err != nil {
		return err
	}
	cd.Status = WorkfloStatus_Pending
	return uc.serviceData.SaveContinuousDeployment(ctx, cd)
}

func (uc *ServicesUseCase) GetContinuousDeployment(ctx context.Context, cdId int64) (*ContinuousDeployment, *Workflow, error) {
	cd, err := uc.serviceData.GetContinuousDeployment(ctx, cdId)
	if err != nil {
		return nil, nil, err
	}
	workflow, err := cd.GetWorkflow()
	if err != nil {
		return nil, nil, err
	}
	err = uc.workflowRuntime.GetWorkflowStatus(ctx, workflow)
	if err != nil {
		return nil, nil, err
	}
	return cd, workflow, nil
}

func (uc *ServicesUseCase) UpdateContinuousDeployment(ctx context.Context, cdId int64) error {
	cd, workflow, err := uc.GetContinuousDeployment(ctx, cdId)
	if err != nil {
		return err
	}
	err = cd.SetWorkflow(workflow)
	if err != nil {
		return err
	}
	defaultStatus := WorkfloStatus_Pending
	taskPendingNumber := 0
	for _, step := range workflow.WorkflowSteps {
		for _, task := range step.WorkflowTasks {
			if task.Status == WorkfloStatus_Failure {
				defaultStatus = WorkfloStatus_Failure
				break
			}
			if task.Status == WorkfloStatus_Pending {
				taskPendingNumber++
			}
		}
	}
	if defaultStatus == WorkfloStatus_Failure {
		cd.Status = WorkfloStatus_Failure
	}
	if defaultStatus != WorkfloStatus_Failure && taskPendingNumber == 0 {
		cd.Status = WorkfloStatus_Success
	}
	return uc.serviceData.SaveContinuousDeployment(ctx, cd)
}

func (uc *ServicesUseCase) GetContinuousDeployments(ctx context.Context, serviceId int64, page, pageSize int32) ([]*ContinuousDeployment, int64, error) {
	return uc.serviceData.GetContinuousDeployments(ctx, serviceId, page, pageSize)
}

func (uc *ServicesUseCase) DeleteContinuousDeployment(ctx context.Context, cdId int64) error {
	return uc.serviceData.DeleteContinuousDeployment(ctx, cdId)
}

func (uc *ServicesUseCase) ApplyService(ctx context.Context, serviceId, ciId, cdId int64) error {
	service, err := uc.Get(ctx, serviceId)
	if err != nil {
		return err
	}
	cd, err := uc.serviceData.GetContinuousDeployment(ctx, cdId)
	if err != nil {
		return err
	}
	service.Status = ServiceStatus_Starting
	err = uc.serviceRuntime.ApplyService(ctx, service, cd)
	if err != nil {
		return err
	}
	return nil
}
