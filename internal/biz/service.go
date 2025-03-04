package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

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
	GetService(context.Context, *Service) error
	CommitWorkflow(context.Context, *Workflow) error
	GetWorkflow(context.Context, *Workflow) error
	CleanWorkflow(context.Context, *Workflow) error
}

type ServiceAgent interface {
}

type ServicesUseCase struct {
	serviceData    ServicesData
	serviceRuntime ServiceRuntime
	log            *log.Helper
}

func NewServicesUseCase(serviceData ServicesData, serviceRuntime ServiceRuntime, logger log.Logger) *ServicesUseCase {
	return &ServicesUseCase{serviceData: serviceData, serviceRuntime: serviceRuntime, log: log.NewHelper(logger)}
}

func (w *Workflow) GetFirstStep() *WorkflowStep {
	if len(w.WorkflowSteps) == 0 {
		return nil
	}
	var firstStep *WorkflowStep
	for _, step := range w.WorkflowSteps {
		if firstStep == nil || step.Order < firstStep.Order {
			firstStep = step
		}
	}
	return firstStep
}

func (w *Workflow) GetNextStep(s *WorkflowStep) *WorkflowStep {
	if len(w.WorkflowSteps) == 0 {
		return nil
	}
	var nextStep *WorkflowStep
	for _, step := range w.WorkflowSteps {
		if step.Order > s.Order {
			if nextStep == nil || step.Order < nextStep.Order {
				nextStep = step
			}
		}
	}
	return nextStep
}

func (s *WorkflowStep) GetFirstTask() *WorkflowTask {
	if len(s.WorkflowTasks) == 0 {
		return nil
	}
	var firstTask *WorkflowTask
	for _, task := range s.WorkflowTasks {
		if firstTask == nil || task.Order < firstTask.Order {
			firstTask = task
		}
	}
	return firstTask
}

func (s *WorkflowStep) GetNextTask(t *WorkflowTask) *WorkflowTask {
	if len(s.WorkflowTasks) == 0 {
		return nil
	}
	var nextTask *WorkflowTask
	for _, task := range s.WorkflowTasks {
		if task.Order > t.Order {
			if nextTask == nil || task.Order < nextTask.Order {
				nextTask = task
			}
		}
	}
	return nextTask
}

func (w WorkflowStepType) Image() string {
	if w == WorkflowStepType_CodePull {
		return "alpine/git:latest"
	}
	if w == WorkflowStepType_ImageRepoAuth {
		return "docker:latest"
	}
	if w == WorkflowStepType_Build {
		return "moby/buildkit:latest"
	}
	if w == WorkflowStepType_Deploy {
		return "curlimages/curl:latest"
	}
	return ""
}

func (w *Workflow) GetWorkdir() string {
	return "/app"
}

func (w *Workflow) GetWorkdirName() string {
	return "app"
}

func (s *Service) GetDefaultWorkflow(wfType WorkflowType) *Workflow {
	workflow := &Workflow{
		Name:         fmt.Sprintf("%s-%s", s.Name, strings.ToLower(wfType.String())),
		Type:         wfType,
		ServiceId:    s.Id,
		StorageClass: s.StorageClass,
	}
	workflow.Description = "These are the environment variables that can be used\n"
	for _, v := range ServiceEnv_name {
		workflow.Description += fmt.Sprintf("{%s} ", v)
	}
	workflow.Description += fmt.Sprintf("\n\n Example: if %s = git_repo_url 'git clone {%s}', You get 'git clone git_repo_url' like this",
		ServiceEnv_GIT_REPO.String(), ServiceEnv_GIT_REPO.String())
	workflow.WorkflowSteps = make([]*WorkflowStep, 0)
	var order int32 = 1
	var taskCommamd string
	for name, v := range WorkflowStepType_value {
		if WorkflowStepType(v) == WorkflowStepType_Customizable {
			continue
		}
		if wfType == WorkflowType_ContinuousIntegrationType && v > int32(WorkflowStepType_Build) {
			continue
		}
		if wfType == WorkflowType_ContinuousDeploymentType && v < int32(WorkflowStepType_Deploy) {
			continue
		}
		if WorkflowStepType(v) == WorkflowStepType_CodePull {
			taskCommamd = "Git pull code handler... (This is the default and cannot be changed)"
		}
		if WorkflowStepType(v) == WorkflowStepType_Build {
			taskCommamd = "Build image handler... (This is the default and cannot be changed)"
		}
		if WorkflowStepType(v) == WorkflowStepType_Deploy {
			taskCommamd = "Deploy handler... (This is the default and cannot be changed)"
		}
		workflow.WorkflowSteps = append(workflow.WorkflowSteps, &WorkflowStep{
			Name:             name,
			Order:            order,
			Description:      name,
			WorkflowStepType: WorkflowStepType(v),
			Image:            WorkflowStepType(v).Image(),
			WorkflowTasks: []*WorkflowTask{
				{
					Name:        name,
					Order:       order,
					Description: name,
					TaskCommand: taskCommamd,
				},
			},
		})
		order += 1
	}
	return workflow
}

func (w *Workflow) SettingServiceEnv(ctx context.Context, project *Project, s *Service, ci *ContinuousIntegration) map[string]string {
	user := GetUserInfo(ctx)
	serviceEnv := make(map[string]string)
	for _, val := range ServiceEnv_value {
		switch ServiceEnv(val) {
		case ServiceEnv_SERVICE_NAME:
			serviceEnv[ServiceEnv_SERVICE_NAME.String()] = s.Name
		case ServiceEnv_VERSION:
			serviceEnv[ServiceEnv_VERSION.String()] = cast.ToString(ci.Version)
		case ServiceEnv_BRANCH:
			serviceEnv[ServiceEnv_BRANCH.String()] = ci.Branch
		case ServiceEnv_TAG:
			serviceEnv[ServiceEnv_TAG.String()] = ci.Tag
		case ServiceEnv_COMMIT_ID:
			serviceEnv[ServiceEnv_COMMIT_ID.String()] = ci.CommitId
		case ServiceEnv_SERVICE_ID:
			serviceEnv[ServiceEnv_SERVICE_ID.String()] = cast.ToString(s.Id)
		case ServiceEnv_IMAGE:
			serviceEnv[ServiceEnv_IMAGE.String()] = ci.GetImage(user, s)
		case ServiceEnv_GIT_REPO:
			serviceEnv[ServiceEnv_GIT_REPO.String()] = project.GitRepository
		case ServiceEnv_IMAGE_REPO:
			serviceEnv[ServiceEnv_IMAGE_REPO.String()] = project.ImageRepository
		case ServiceEnv_GIT_REPO_NAME:
			serviceEnv[ServiceEnv_GIT_REPO_NAME.String()] = user.GitrepoName
		case ServiceEnv_IMAGE_REPO_NAME:
			serviceEnv[ServiceEnv_IMAGE_REPO_NAME.String()] = user.ImagerepoName
		case ServiceEnv_GIT_REPO_TOKEN:
			serviceEnv[ServiceEnv_GIT_REPO_TOKEN.String()] = user.GitRepositoryToken
		case ServiceEnv_IMAGE_REPO_TOKEN:
			serviceEnv[ServiceEnv_IMAGE_REPO_TOKEN.String()] = user.ImageRepositoryToken
		}
	}
	w.Env = utils.MapToString(serviceEnv)
	return serviceEnv
}

func (w *Workflow) SettingContinuousIntegration(ctx context.Context, service *Service, ci *ContinuousIntegration) {
	project := GetProject(ctx)
	user := GetUserInfo(ctx)
	serviceEnv := w.SettingServiceEnv(ctx, project, service, ci)
	for _, step := range w.WorkflowSteps {
		for _, task := range step.WorkflowTasks {
			wfType, ok := WorkflowStepType_value[task.Name]
			if !ok {
				task.TaskCommand = utils.DecodeString(task.TaskCommand, serviceEnv)
				continue
			}
			switch WorkflowStepType(wfType) {
			case WorkflowStepType_CodePull:
				project.GitRepository = strings.ReplaceAll(project.GitRepository, "https://", "")
				gitRepoUrl := fmt.Sprintf("https://%s:%s@%s/%s.git", user.GitrepoName, user.GitRepositoryToken, project.GitRepository, service.Name)
				if ci.Branch != "" && ci.CommitId != "" {
					task.TaskCommand = fmt.Sprintf("git clone %s --depth 1 --branch %s /app && cd /app && git checkout %s && ls -la /app",
						gitRepoUrl, ci.Branch, ci.CommitId)
				}
				if ci.Tag != "" {
					task.TaskCommand = fmt.Sprintf("git clone %s --depth 1 /app && cd /app && git checkout tags/%s",
						gitRepoUrl, ci.Tag)
				}
			case WorkflowStepType_ImageRepoAuth:
				task.TaskCommand = fmt.Sprintf("ls -la /app && echo '%s' | docker login -u '%s' --password-stdin %s && cp /root/.docker/config.json /app/config.json",
					user.ImageRepositoryToken, user.ImagerepoName, project.ImageRepository)
			case WorkflowStepType_Build:
				task.TaskCommand = fmt.Sprintf("ls -la /app && mkdir /root/.docker && cp /app/config.json /root/.docker && buildkitd --rootless --addr unix:///run/buildkit/buildkitd.sock & sleep 3 && buildctl build --frontend dockerfile.v0 --local context=/app --local dockerfile=/app --output type=image,name=%s,push=true",
					ci.GetImage(user, service))
			}
		}
	}
}

func (w *Workflow) SettingContinuousDeployment(ctx context.Context, service *Service, ci *ContinuousIntegration, cd *ContinuousDeployment) error {
	cluster := GetCluster(ctx)
	project := GetProject(ctx)
	serviceEnv := w.SettingServiceEnv(ctx, project, service, ci)
	for _, step := range w.WorkflowSteps {
		for _, task := range step.WorkflowTasks {
			wfType, ok := WorkflowStepType_value[task.Name]
			if !ok {
				task.TaskCommand = utils.DecodeString(task.TaskCommand, serviceEnv)
				continue
			}
			if WorkflowStepType_Deploy == WorkflowStepType(wfType) {
				task.TaskCommand = fmt.Sprintf("curl -X POST -H 'Content-Type: application/json' -H 'Authorization: Bearer %s' -d '{\"id\":\"%d\",\"ci_id\":\"%d\",\"cd_id\":\"%d\"}' http://%s/api/v1alpha1/service/apply",
					cluster.ServiceAccessToken, service.Id, ci.Id, cd.Id, cluster.Name)
			}
		}
	}
	return nil
}

func (ci *ContinuousIntegration) GetImage(u *User, s *Service) string {
	return fmt.Sprintf("%s/%s:%s", u.ImagerepoName, s.Name, ci.Version)
}

func (ci *ContinuousIntegration) SetWorkflow(wf *Workflow) error {
	jsonData, err := json.Marshal(wf)
	if err != nil {
		return err
	}
	ci.WorkflowRuntime = string(jsonData)
	return nil
}

func (ci *ContinuousIntegration) GetWorkflow() (*Workflow, error) {
	if ci.WorkflowRuntime == "" {
		return nil, nil
	}
	var workflow Workflow
	err := json.Unmarshal([]byte(ci.WorkflowRuntime), &workflow)
	if err != nil {
		return nil, err
	}
	return &workflow, nil
}

func (cd *ContinuousDeployment) SetWorkflow(wf *Workflow) error {
	jsonData, err := json.Marshal(wf)
	if err != nil {
		return err
	}
	cd.WorkflowRuntime = string(jsonData)
	return nil
}

func (cd *ContinuousDeployment) GetWorkflow() (*Workflow, error) {
	if cd.WorkflowRuntime == "" {
		return nil, nil
	}
	var workflow Workflow
	err := json.Unmarshal([]byte(cd.WorkflowRuntime), &workflow)
	if err != nil {
		return nil, err
	}
	return &workflow, nil
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
	err = uc.serviceRuntime.GetService(ctx, service)
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

type Workflows []*Workflow

func (ws Workflows) GetWorkflowByType(wfType WorkflowType) *Workflow {
	for _, v := range ws {
		if v.Type == wfType {
			return v
		}
	}
	return nil
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
	err = uc.serviceRuntime.CleanWorkflow(ctx, workflow)
	if err != nil {
		return err
	}
	workflow.SettingContinuousIntegration(ctx, service, ci)
	err = uc.serviceRuntime.CommitWorkflow(ctx, workflow)
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
	err = uc.serviceRuntime.GetWorkflow(ctx, workflow)
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
	err = uc.serviceRuntime.CleanWorkflow(ctx, workflow)
	if err != nil {
		return err
	}
	workflow.SettingContinuousDeployment(ctx, service, ci, cd)
	err = uc.serviceRuntime.CommitWorkflow(ctx, workflow)
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
	err = uc.serviceRuntime.GetWorkflow(ctx, workflow)
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
