package biz

import (
	"context"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
)

type ServiceEnv string

const (
	ServiceEnv_SERVICE_NAME     ServiceEnv = "SERVICE_NAME"
	ServiceEnv_VERSION          ServiceEnv = "VERSION"
	ServiceEnv_BRANCH           ServiceEnv = "BRANCH"
	ServiceEnv_TAG              ServiceEnv = "TAG"
	ServiceEnv_COMMIT_ID        ServiceEnv = "COMMIT_ID"
	ServiceEnv_SERVICE_ID       ServiceEnv = "SERVICE_ID"
	ServiceEnv_IMAGE            ServiceEnv = "IMAGE"
	ServiceEnv_GIT_REPO         ServiceEnv = "GIT_REPO"
	ServiceEnv_IMAGE_REPO       ServiceEnv = "IMAGE_REPO"
	ServiceEnv_GIT_REPO_NAME    ServiceEnv = "GIT_REPO_NAME"
	ServiceEnv_IMAGE_REPO_NAME  ServiceEnv = "IMAGE_REPO_NAME"
	ServiceEnv_GIT_REPO_TOKEN   ServiceEnv = "GIT_REPO_TOKEN"
	ServiceEnv_IMAGE_REPO_TOKEN ServiceEnv = "IMAGE_REPO_TOKEN"
)

func (s ServiceEnv) String() string {
	return string(s)
}

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

type ContinuousIntegrationStatus int32

const (
	ContinuousIntegrationStatus_UNSPECIFIED ContinuousIntegrationStatus = 0
	ContinuousIntegrationStatus_PENDING     ContinuousIntegrationStatus = 1
	ContinuousIntegrationStatus_RUNNING     ContinuousIntegrationStatus = 2
	ContinuousIntegrationStatus_SUCCESS     ContinuousIntegrationStatus = 3
	ContinuousIntegrationStatus_FAILED      ContinuousIntegrationStatus = 4
)

func (cis ContinuousIntegrationStatus) String() string {
	switch cis {
	case ContinuousIntegrationStatus_PENDING:
		return "PENDING"
	case ContinuousIntegrationStatus_RUNNING:
		return "RUNNING"
	case ContinuousIntegrationStatus_SUCCESS:
		return "SUCCESS"
	case ContinuousIntegrationStatus_FAILED:
		return "FAILED"
	default:
		return "UNSPECIFIED"
	}
}

func ContinuousIntegrationStatusFromString(s string) ContinuousIntegrationStatus {
	switch s {
	case "PENDING":
		return ContinuousIntegrationStatus_PENDING
	case "RUNNING":
		return ContinuousIntegrationStatus_RUNNING
	case "SUCCESS":
		return ContinuousIntegrationStatus_SUCCESS
	case "FAILED":
		return ContinuousIntegrationStatus_FAILED
	default:
		return ContinuousIntegrationStatus_UNSPECIFIED
	}
}

type ContinuousDeploymentStatus int32

const (
	ContinuousDeploymentStatus_UNSPECIFIED ContinuousDeploymentStatus = 0
	ContinuousDeploymentStatus_PENDING     ContinuousDeploymentStatus = 1
	ContinuousDeploymentStatus_RUNNING     ContinuousDeploymentStatus = 2
	ContinuousDeploymentStatus_SUCCESS     ContinuousDeploymentStatus = 3
	ContinuousDeploymentStatus_FAILED      ContinuousDeploymentStatus = 4
)

func (cds ContinuousDeploymentStatus) String() string {
	switch cds {
	case ContinuousDeploymentStatus_PENDING:
		return "PENDING"
	case ContinuousDeploymentStatus_RUNNING:
		return "RUNNING"
	case ContinuousDeploymentStatus_SUCCESS:
		return "SUCCESS"
	case ContinuousDeploymentStatus_FAILED:
		return "FAILED"
	default:
		return "UNSPECIFIED"
	}
}

func ContinuousDeploymentStatusFromString(s string) ContinuousDeploymentStatus {
	switch s {
	case "PENDING":
		return ContinuousDeploymentStatus_PENDING
	case "RUNNING":
		return ContinuousDeploymentStatus_RUNNING
	case "SUCCESS":
		return ContinuousDeploymentStatus_SUCCESS
	case "FAILED":
		return ContinuousDeploymentStatus_FAILED
	default:
		return ContinuousDeploymentStatus_UNSPECIFIED
	}
}

type ServiceStatus int32

const (
	ServiceStatus_UNSPECIFIED ServiceStatus = 0
	ServiceStatus_Starting    ServiceStatus = 1
	ServiceStatus_Running     ServiceStatus = 2
	ServiceStatus_Terminated  ServiceStatus = 3
)

func (ss ServiceStatus) String() string {
	switch ss {
	case ServiceStatus_Starting:
		return "STARTING"
	case ServiceStatus_Running:
		return "RUNNING"
	case ServiceStatus_Terminated:
		return "TERMINATED"
	default:
		return "UNSPECIFIED"
	}
}

func ServiceStatusFromString(s string) ServiceStatus {
	switch s {
	case "STARTING":
		return ServiceStatus_Starting
	case "RUNNING":
		return ServiceStatus_Running
	case "TERMINATED":
		return ServiceStatus_Terminated
	default:
		return ServiceStatus_UNSPECIFIED
	}
}

type PodStatus int32

const (
	PodStatus_UNSPECIFIED PodStatus = 0
	PodStatus_PENDING     PodStatus = 1
	PodStatus_RUNNING     PodStatus = 2
	PodStatus_SUCCEEDED   PodStatus = 3
	PodStatus_FAILED      PodStatus = 4
)

func (ps PodStatus) String() string {
	switch ps {
	case PodStatus_PENDING:
		return "PENDING"
	case PodStatus_RUNNING:
		return "RUNNING"
	case PodStatus_SUCCEEDED:
		return "SUCCEEDED"
	case PodStatus_FAILED:
		return "FAILED"
	default:
		return "UNSPECIFIED"
	}
}

func PodStatusFromString(s string) PodStatus {
	switch s {
	case "PENDING":
		return PodStatus_PENDING
	case "RUNNING":
		return PodStatus_RUNNING
	case "SUCCEEDED":
		return PodStatus_SUCCEEDED
	case "FAILED":
		return PodStatus_FAILED
	default:
		return PodStatus_UNSPECIFIED
	}
}

type TimeRange string

const (
	TimeRangeHalfHour  TimeRange = "30m"
	TimeRangeOneHour   TimeRange = "1h"
	TimeRangeOneDay    TimeRange = "24h"
	TimeRangeThreeDays TimeRange = "72h"
)

type MetricPoints []MetricPoint

type MetricsResult struct {
	CPU        MetricPoints `json:"cpu"`
	Memory     MetricPoints `json:"memory"`
	Disk       MetricPoints `json:"disk"`
	NetworkIn  MetricPoints `json:"network_in"`
	NetworkOut MetricPoints `json:"network_out"`
	GPU        MetricPoints `json:"gpu,omitempty"`
	GPUMem     MetricPoints `json:"gpu_mem,omitempty"`
	QPS        MetricPoints `json:"qps"`
}

type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}
type Protocol string

const (
	ProtocolTCP Protocol = "TCP"
	ProtocolUDP Protocol = "UDP"
)

func (p Protocol) String() string {
	return string(p)
}

type Service struct {
	Id            int64               `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name          string              `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Status        ServiceStatus       `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	Lables        string              `json:"lables,omitempty" gorm:"column:lables;default:'';NOT NULL"`
	ResourceQuota ResourceQuotaString `json:"resource_quota,omitempty" gorm:"column:resource_quota;default:'';NOT NULL"`
	Pods          []*Pod              `json:"pods,omitempty" gorm:"-"`
	Volumes       []*Volume           `json:"volumes,omitempty" gorm:"-"`
	Ports         []*Port             `json:"ports,omitempty" gorm:"-"`
	Description   string              `json:"description,omitempty" gorm:"column:description;default:'';NOT NULL"`
	UserId        int64               `json:"user_id,omitempty" gorm:"column:user_id;default:0;NOT NULL"`
	ProjectId     int64               `json:"project_id,omitempty" gorm:"column:project_id;default:0;NOT NULL;index:idx_project_id"`
	WorkspaceId   int64               `json:"workspace_id,omitempty" gorm:"column:workspace_id;default:0;NOT NULL"`
	ClusterId     int64               `json:"cluster_id,omitempty" gorm:"column:cluster_id;default:0;NOT NULL"`
}

type ContinuousIntegration struct {
	Id              int64                       `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Version         string                      `json:"version,omitempty" gorm:"column:version;default:'';NOT NULL"`
	Branch          string                      `json:"branch,omitempty" gorm:"column:branch;default:'';NOT NULL"`
	CommitId        string                      `json:"commit_id,omitempty" gorm:"column:commit_id;default:'';NOT NULL"`
	Tag             string                      `json:"tag,omitempty" gorm:"column:tag;default:'';NOT NULL"`
	Status          ContinuousIntegrationStatus `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	Description     string                      `json:"description,omitempty" gorm:"column:description;default:'';NOT NULL"`
	ServiceId       int64                       `json:"service_id,omitempty" gorm:"column:service_id;default:0;NOT NULL;index:idx_ci_service_id"`
	UserId          int64                       `json:"user_id,omitempty" gorm:"column:user_id;default:0;NOT NULL;index:idx_ci_user_id"`
	WorkflowRuntime string                      `json:"workflow_runtime,omitempty" gorm:"column:workflow_runtime;default:'';NOT NULL"`
	Pods            string                      `json:"pods,omitempty" gorm:"column:pods;default:'';NOT NULL"`
}

type ContinuousDeployment struct {
	Id               int64                      `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	CiId             int64                      `json:"ci_id,omitempty" gorm:"column:ci_id;default:0;NOT NULL;index:idx_ci_id"`
	ServiceId        int64                      `json:"service_id,omitempty" gorm:"column:service_id;default:0;NOT NULL;index:idx_cd_service_id"`
	UserId           int64                      `json:"user_id,omitempty" gorm:"column:user_id;default:0;NOT NULL"`
	Status           ContinuousDeploymentStatus `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	Image            string                     `json:"image,omitempty" gorm:"column:image;default:'';NOT NULL"`
	WorkflowRuntime  string                     `json:"workflow_runtime,omitempty" gorm:"column:workflow_runtime;default:'';NOT NULL"`
	Config           map[string]string          `json:"config,omitempty" gorm:"-"` // key: filename, value: content
	CanaryDeployment *CanaryDeployment          `json:"canary_deployment,omitempty" gorm:"-"`
	IsAccessExternal AccessExternal             `json:"is_access_external,omitempty" gorm:"column:is_access_external;default:0;NOT NULL"`
	Pods             string                     `json:"pods,omitempty" gorm:"column:pods;default:'';NOT NULL"`
}

type CanaryDeployment struct {
	CdId       int64             `json:"cd_id,omitempty" gorm:"column:cd_id;default:0;NOT NULL;index:idx_cd_id"`
	Image      string            `json:"image,omitempty" gorm:"column:image;default:'';NOT NULL"`
	Replicas   int32             `json:"replicas,omitempty" gorm:"column:replicas;default:0;NOT NULL"`
	Config     map[string]string `json:"config,omitempty" gorm:"-"` // key: filename, value: content
	TrafficPct int32             `json:"traffic_pct,omitempty" gorm:"column:traffic_pct;default:0;NOT NULL"`
}

type Port struct {
	Id            int64    `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name          string   `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Path          string   `json:"path,omitempty" gorm:"column:path;default:'';NOT NULL"`
	Protocol      Protocol `json:"protocol,omitempty" gorm:"column:protocol;default:'';NOT NULL"`
	ContainerPort int32    `json:"container_port,omitempty" gorm:"column:container_port;default:0;NOT NULL"`
	ServiceId     int64    `json:"service_id,omitempty" gorm:"column:service_id;default:0;NOT NULL;index:idx_port_service_id"`
}

type Volume struct {
	Id           int64  `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name         string `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	MountPath    string `json:"mount_path,omitempty" gorm:"column:mount_path;default:'';NOT NULL"`
	Storage      int32  `json:"storage,omitempty" gorm:"column:storage;default:0;NOT NULL"`
	StorageClass string `json:"storage_class,omitempty" gorm:"column:storage_class;default:'';NOT NULL"`
	ServiceId    int64  `json:"service_id,omitempty" gorm:"column:service_id;default:0;NOT NULL;index:idx_volume_service_id"`
}

type Pod struct {
	Id        int64     `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name      string    `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	NodeName  string    `json:"node_name,omitempty" gorm:"column:node_name;default:'';NOT NULL"`
	Status    PodStatus `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	ServiceId int64     `json:"service_id,omitempty" gorm:"column:service_id;default:0;NOT NULL;index:idx_pod_service_id"`
}

type Trace struct {
	Id              int64  `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	FromServiceId   int64  `json:"from_service_id,omitempty" gorm:"column:from_service_id;default:0;NOT NULL;index:idx_from_service_id"`
	FromServiceName string `json:"from_service_name,omitempty" gorm:"column:from_service_name;default:'';NOT NULL;index:idx_from_service_id"`
	ToServiceId     int64  `json:"to_service_id,omitempty" gorm:"column:to_service_id;default:0;NOT NULL"`
	ToServiceName   string `json:"to_service_name,omitempty" gorm:"column:to_service_name;default:'';NOT NULL"`
	FromLabel       string `json:"from_label,omitempty" gorm:"column:from_label;default:'';NOT NULL"`
	ToLabel         string `json:"to_label,omitempty" gorm:"column:to_label;default:'';NOT NULL"`
	NodeName        string `json:"node_name,omitempty" gorm:"column:node_name;default:'';NOT NULL"`
	RequestCount    int64  `json:"request_count,omitempty" gorm:"column:request_count;default:0;NOT NULL"`
	LastRequestTime string `json:"last_request_time,omitempty" gorm:"column:last_request_time;default:'';NOT NULL"`
}

type LogResponse struct {
	Log   []map[string]any `json:"log"`
	Total int              `json:"total"`
}

type ServicesData interface {
	Save(ctx context.Context, service *Service) error
	Get(ctx context.Context, id int64) (*Service, error)
	List(ctx context.Context, projectId int64, serviceName string, page, pageSize int32) ([]*Service, int64, error)
	Delete(ctx context.Context, id int64) error
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
	GetContinuousIntegrationLog(ctx context.Context, ci *ContinuousIntegration, page, pageSize int) (LogResponse, error)
	GetContinuousDeploymentLog(ctx context.Context, cd *ContinuousDeployment, page, pageSize int) (LogResponse, error)
	GetServicePodLog(ctx context.Context, service *Service, page, pageSize int) (LogResponse, error)
	GetServiceLog(ctx context.Context, service *Service, page, pageSize int) (LogResponse, error)
}

type ServiceRuntime interface {
	ApplyService(context.Context, *Service, *ContinuousDeployment) error
	GetServiceStatus(context.Context, *Service) error
	DeleteService(context.Context, *Service) error
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

func (m MetricPoints) GetFirstValue() float64 {
	if len(m) == 0 {
		return 0
	}
	return m[0].Value
}

func (tr TimeRange) CalculateMetricPointsStep() time.Duration {
	switch tr {
	case TimeRangeHalfHour:
		return time.Minute
	case TimeRangeOneHour:
		return 2 * time.Minute
	case TimeRangeOneDay:
		return 30 * time.Minute
	case TimeRangeThreeDays:
		return time.Hour
	default:
		return 5 * time.Minute
	}
}

func (tr TimeRange) MustParseDuration() time.Duration {
	d, _ := time.ParseDuration(string(tr))
	return d
}

func (l LogResponse) GetPageCount(pageSize int) int {
	if pageSize == 0 {
		pageSize = 10
	}
	return int(math.Ceil(float64(l.Total) / float64(pageSize)))
}

func (s *Service) GetLabels() map[string]string {
	return LablesToMap(s.Lables)
}

func (s *Service) GetWorkspaceNameByLable() string {
	lables := s.GetLabels()
	if workspaceName, ok := lables[WorkspaceName]; ok {
		return workspaceName
	}
	return ""
}

func (s *Service) AddLeble(ls string) {
	serviceLables := s.GetLabels()
	lableMap := LablesToMap(ls)
	for k, v := range lableMap {
		if !slices.Contains(getBaseLableKeys(), k) {
			serviceLables[k] = v
		}
	}
}

func (s *Service) SetBaseLables(ctx context.Context) {
	serviceLables := LablesToMap(s.Lables)
	serviceLables[ServiceName] = s.Name
	workspace := GetWorkspace(ctx)
	if workspace != nil {
		serviceLables[WorkspaceId] = strconv.FormatInt(workspace.Id, 10)
		serviceLables[WorkspaceName] = workspace.Name
		s.WorkspaceId = workspace.Id
	}
	project := GetProject(ctx)
	if project != nil {
		serviceLables[ProjectId] = strconv.FormatInt(project.Id, 10)
		serviceLables[ProjectName] = project.Name
		s.ProjectId = project.Id
	}
	s.Lables = mapToLables(serviceLables)
}

func (uc *ServicesUseCase) Save(ctx context.Context, service *Service) error {
	customLables := service.Lables
	service.SetBaseLables(ctx)
	service.AddLeble(customLables)
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

func (uc *ServicesUseCase) GetServiceByName(ctx context.Context, projectId int64, serviceName string) (*Service, error) {
	return uc.serviceData.GetByName(ctx, projectId, serviceName)
}

func (uc *ServicesUseCase) List(ctx context.Context, projectId int64, serviceName string, page, pageSize int32) ([]*Service, int64, error) {
	return uc.serviceData.List(ctx, projectId, serviceName, page, pageSize)
}

func (uc *ServicesUseCase) Delete(ctx context.Context, id int64) error {
	service, err := uc.serviceData.Get(ctx, id)
	if err != nil {
		return err
	}
	if service == nil || service.Id == 0 {
		return nil
	}
	err = uc.serviceRuntime.DeleteService(ctx, service)
	if err != nil {
		return err
	}
	return uc.serviceData.Delete(ctx, id)
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
	ci.Status = ContinuousIntegrationStatus_PENDING
	return uc.serviceData.SaveContinuousIntegration(ctx, ci)
}

func (uc *ServicesUseCase) GetContinuousIntegration(ctx context.Context, ciId int64) (*ContinuousIntegration, error) {
	ci, err := uc.serviceData.GetContinuousIntegration(ctx, ciId)
	if err != nil {
		return nil, err
	}
	workflow, err := ci.GetWorkflow()
	if err != nil {
		return nil, err
	}
	err = uc.workflowRuntime.GetWorkflowStatus(ctx, workflow)
	if err != nil {
		return nil, err
	}
	ci.SetWorkflow(workflow)
	return ci, nil
}

func (uc *ServicesUseCase) UpdateContinuousIntegrationStatusByWorkflowRuntime(ctx context.Context, ci *ContinuousIntegration) error {
	workflow, err := ci.GetWorkflow()
	if err != nil {
		return err
	}
	defaultStatus := WorkflowStatus_Pending
	taskPendingNumber := 0
	for _, step := range workflow.WorkflowSteps {
		for _, task := range step.WorkflowTasks {
			if task.Status == WorkflowStatus_Failure {
				defaultStatus = WorkflowStatus_Failure
				break
			}
			if task.Status == WorkflowStatus_Pending {
				taskPendingNumber++
			}
		}
	}
	if defaultStatus == WorkflowStatus_Failure {
		ci.Status = ContinuousIntegrationStatus_FAILED
	}
	if defaultStatus != WorkflowStatus_Failure && taskPendingNumber == 0 {
		ci.Status = ContinuousIntegrationStatus_SUCCESS
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
	workspace := GetWorkspace(ctx)
	service, err := uc.Get(ctx, cd.ServiceId)
	if err != nil {
		return err
	}
	ci, err := uc.serviceData.GetContinuousIntegration(ctx, cd.CiId)
	if err != nil {
		return err
	}
	cd.Image = ci.GetImage(workspace, service)
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
	cd.Status = ContinuousDeploymentStatus_PENDING
	return uc.serviceData.SaveContinuousDeployment(ctx, cd)
}

func (uc *ServicesUseCase) GetContinuousDeployment(ctx context.Context, cdId int64) (*ContinuousDeployment, error) {
	cd, err := uc.serviceData.GetContinuousDeployment(ctx, cdId)
	if err != nil {
		return nil, err
	}
	workflow, err := cd.GetWorkflow()
	if err != nil {
		return nil, err
	}
	err = uc.workflowRuntime.GetWorkflowStatus(ctx, workflow)
	if err != nil {
		return nil, err
	}
	cd.SetWorkflow(workflow)
	return cd, nil
}

func (uc *ServicesUseCase) UpdateContinuousDeployment(ctx context.Context, cd *ContinuousDeployment) error {
	workflow, err := cd.GetWorkflow()
	if err != nil {
		return err
	}
	defaultStatus := WorkflowStatus_Pending
	taskPendingNumber := 0
	for _, step := range workflow.WorkflowSteps {
		for _, task := range step.WorkflowTasks {
			if task.Status == WorkflowStatus_Failure {
				defaultStatus = WorkflowStatus_Failure
				break
			}
			if task.Status == WorkflowStatus_Pending {
				taskPendingNumber++
			}
		}
	}
	if defaultStatus == WorkflowStatus_Failure {
		cd.Status = ContinuousDeploymentStatus_FAILED
	}
	if defaultStatus != WorkflowStatus_Failure && taskPendingNumber == 0 {
		cd.Status = ContinuousDeploymentStatus_SUCCESS
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

func (uc *ServicesUseCase) GetContinuousIntegrationLog(ctx context.Context, ciId int64, page, pageSize int) (LogResponse, error) {
	ci, err := uc.serviceData.GetContinuousIntegration(ctx, ciId)
	if err != nil {
		return LogResponse{}, err
	}
	return uc.serviceData.GetContinuousIntegrationLog(ctx, ci, page, pageSize)
}

func (uc *ServicesUseCase) GetContinuousDeploymentLog(ctx context.Context, cdId int64, page, pageSize int) (LogResponse, error) {
	cd, err := uc.serviceData.GetContinuousDeployment(ctx, cdId)
	if err != nil {
		return LogResponse{}, err
	}
	return uc.serviceData.GetContinuousDeploymentLog(ctx, cd, page, pageSize)
}

func (uc *ServicesUseCase) GetServicePodLog(ctx context.Context, serviceId int64, page, pageSize int) (LogResponse, error) {
	service, err := uc.Get(ctx, serviceId)
	if err != nil {
		return LogResponse{}, err
	}
	return uc.serviceData.GetServicePodLog(ctx, service, page, pageSize)
}

func (uc *ServicesUseCase) GetServiceLog(ctx context.Context, serviceId int64, page, pageSize int) (LogResponse, error) {
	service, err := uc.Get(ctx, serviceId)
	if err != nil {
		return LogResponse{}, err
	}
	return uc.serviceData.GetServiceLog(ctx, service, page, pageSize)
}
