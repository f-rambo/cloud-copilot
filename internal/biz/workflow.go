package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/spf13/cast"
)

type Workflows []*Workflow

type WorkflowTask struct {
	Id          int64          `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	WorkflowId  int64          `json:"workflow_id,omitempty" gorm:"column:workflow_id;default:0;NOT NULL;index:idx_task_workflow_id"`
	StepId      int64          `json:"step_id,omitempty" gorm:"column:step_id;default:0;NOT NULL"`
	Name        string         `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Order       int32          `json:"order,omitempty" gorm:"column:order;default:0;NOT NULL"`
	TaskCommand string         `json:"task_command,omitempty" gorm:"column:task_command;default:'';NOT NULL"`
	Description string         `json:"description,omitempty" gorm:"column:description;default:'';NOT NULL"`
	Status      WorkflowStatus `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	Log         string         `json:"log,omitempty" gorm:"-"`
}

type WorkflowStep struct {
	Id               int64            `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	WorkflowId       int64            `json:"workflow_id,omitempty" gorm:"column:workflow_id;default:0;NOT NULL;index:idx_step_workflow_id"`
	Order            int32            `json:"order,omitempty" gorm:"column:order;default:0;NOT NULL"`
	Name             string           `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Description      string           `json:"description,omitempty" gorm:"column:description;default:'';NOT NULL"`
	Image            string           `json:"image,omitempty" gorm:"column:image;default:'';NOT NULL"`
	WorkflowStepType WorkflowStepType `json:"workflow_step_type,omitempty" gorm:"column:workflow_step_type;default:0;NOT NULL"`
	WorkflowTasks    []*WorkflowTask  `json:"workflow_tasks,omitempty" gorm:"-"`
}

type Workflow struct {
	Id            int64           `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name          string          `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Namespace     string          `json:"namespace,omitempty" gorm:"column:namespace;default:'';NOT NULL"`
	Lables        string          `json:"lables,omitempty" gorm:"column:lables;default:'';NOT NULL"`
	Env           string          `json:"env,omitempty" gorm:"column:env;default:'';NOT NULL"`
	StorageClass  string          `json:"storage_class,omitempty" gorm:"column:storage_class;default:'';NOT NULL"`
	Type          WorkflowType    `json:"type,omitempty" gorm:"column:type;default:0;NOT NULL"`
	Description   string          `json:"description,omitempty" gorm:"column:description;default:'';NOT NULL"`
	ServiceId     int64           `json:"service_id,omitempty" gorm:"column:service_id;default:0;NOT NULL;index:idx_wf_service_id"`
	WorkflowSteps []*WorkflowStep `json:"workflow_steps,omitempty" gorm:"-"`
}

type WorkflowRuntime interface {
	CommitWorkflow(context.Context, *Workflow) error
	GetWorkflowStatus(context.Context, *Workflow) error
	CleanWorkflow(context.Context, *Workflow) error
}

type WorkflowType int32

const (
	WorkflowType_UNSPECIFIED               WorkflowType = 0
	WorkflowType_ContinuousIntegrationType WorkflowType = 1
	WorkflowType_ContinuousDeploymentType  WorkflowType = 2
)

// WorkflowType to string
func (w WorkflowType) String() string {
	switch w {
	case WorkflowType_ContinuousIntegrationType:
		return "ContinuousIntegrationType"
	case WorkflowType_ContinuousDeploymentType:
		return "ContinuousDeploymentType"
	default:
		return ""
	}
}

// WorkflowType items
func WorkflowTypeItems() []WorkflowType {
	return []WorkflowType{
		WorkflowType_ContinuousIntegrationType,
		WorkflowType_ContinuousDeploymentType,
	}
}

// WorkflowType find by WorkflowType
func WorkflowTypeFindByString(w string) WorkflowType {
	for _, v := range WorkflowTypeItems() {
		if v.String() == w {
			return v
		}
	}
	return WorkflowType_UNSPECIFIED
}

type WorkflowStatus int32

const (
	WorkflowStatus_UNSPECIFIED WorkflowStatus = 0
	WorkflowStatus_Pending     WorkflowStatus = 1
	WorkflowStatus_Success     WorkflowStatus = 2
	WorkflowStatus_Failure     WorkflowStatus = 3
)

type WorkflowStepType int32

const (
	WorkflowStepType_Customizable  WorkflowStepType = 0
	WorkflowStepType_CodePull      WorkflowStepType = 1
	WorkflowStepType_ImageRepoAuth WorkflowStepType = 2
	WorkflowStepType_Build         WorkflowStepType = 3
	WorkflowStepType_Deploy        WorkflowStepType = 4
)

// WorkflowStepType to string
func (w WorkflowStepType) String() string {
	switch w {
	case WorkflowStepType_Customizable:
		return "Customizable"
	case WorkflowStepType_CodePull:
		return "CodePull"
	case WorkflowStepType_ImageRepoAuth:
		return "ImageRepoAuth"
	case WorkflowStepType_Build:
		return "Build"
	case WorkflowStepType_Deploy:
		return "Deploy"
	default:
		return ""
	}
}

// WorkflowStepType string to WorkflowStepType
func WorkflowStepTypeStringToEnum(s string) WorkflowStepType {
	for _, v := range WorkflowStepTypeItems() {
		if v.String() == s {
			return v
		}
	}
	return WorkflowStepType_Customizable
}

// WorkflowStepType items
func WorkflowStepTypeItems() []WorkflowStepType {
	return []WorkflowStepType{
		WorkflowStepType_Customizable,
		WorkflowStepType_CodePull,
		WorkflowStepType_ImageRepoAuth,
		WorkflowStepType_Build,
		WorkflowStepType_Deploy,
	}
}

func (ws Workflows) GetWorkflowByType(wfType WorkflowType) *Workflow {
	for _, v := range ws {
		if v.Type == wfType {
			return v
		}
	}
	return nil
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

// Get all tasks by order step task
func (w *Workflow) GetAllTasksByOrder() []*WorkflowTask {
	steps := make([]*WorkflowStep, len(w.WorkflowSteps))
	copy(steps, w.WorkflowSteps)
	sort.Slice(steps, func(i, j int) bool {
		return steps[i].Order < steps[j].Order
	})

	var allTasks []*WorkflowTask
	for _, step := range steps {
		tasks := make([]*WorkflowTask, len(step.WorkflowTasks))
		copy(tasks, step.WorkflowTasks)
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].Order < tasks[j].Order
		})
		allTasks = append(allTasks, tasks...)
	}
	return allTasks
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

func (w *Workflow) GetStorageName() string {
	return w.Name + "-storage"
}

func (w *Workflow) GetWorkdir() string {
	return "/app"
}

func (w *Workflow) GetWorkdirName() string {
	return "app"
}

func (w *Workflow) GetTask(taskName string) *WorkflowTask {
	for _, step := range w.WorkflowSteps {
		for _, task := range step.WorkflowTasks {
			if task.Name == taskName {
				return task
			}
		}
	}
	return nil
}

func (s *Service) GetDefaultWorkflow(wfType WorkflowType) *Workflow {
	workflow := &Workflow{
		Name:         fmt.Sprintf("%s-%s", s.Name, strings.ToLower(wfType.String())),
		Type:         wfType,
		ServiceId:    s.Id,
		StorageClass: s.StorageClass,
	}
	workflow.Description = "These are the environment variables that can be used\n"
	for _, v := range ServiceEnvItems() {
		workflow.Description += fmt.Sprintf("{%s} ", v.String())
	}
	workflow.Description += fmt.Sprintf("\n\n Example: if %s = git_repo_url 'git clone {%s}', You get 'git clone git_repo_url' like this",
		ServiceEnv_GIT_REPO.String(), ServiceEnv_GIT_REPO.String())
	workflow.WorkflowSteps = make([]*WorkflowStep, 0)
	var order int32 = 1
	var taskCommamd string
	for _, v := range WorkflowStepTypeItems() {
		if WorkflowStepType(v) == WorkflowStepType_Customizable {
			continue
		}
		if wfType == WorkflowType_ContinuousIntegrationType && v > WorkflowStepType_Build {
			continue
		}
		if wfType == WorkflowType_ContinuousDeploymentType && v < WorkflowStepType_Deploy {
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
			Name:             v.String(),
			Order:            order,
			Description:      v.String(),
			WorkflowStepType: WorkflowStepType(v),
			Image:            WorkflowStepType(v).Image(),
			WorkflowTasks: []*WorkflowTask{
				{
					Name:        v.String(),
					Order:       order,
					Description: v.String(),
					TaskCommand: taskCommamd,
				},
			},
		})
		order += 1
	}
	return workflow
}

func (w *Workflow) SettingServiceEnv(ctx context.Context, workspace *Workspace, service *Service, ci *ContinuousIntegration) map[string]string {
	serviceEnv := make(map[string]string)
	for _, val := range ServiceEnvItems() {
		switch ServiceEnv(val) {
		case ServiceEnv_SERVICE_NAME:
			serviceEnv[ServiceEnv_SERVICE_NAME.String()] = service.Name
		case ServiceEnv_VERSION:
			serviceEnv[ServiceEnv_VERSION.String()] = cast.ToString(ci.Version)
		case ServiceEnv_BRANCH:
			serviceEnv[ServiceEnv_BRANCH.String()] = ci.Branch
		case ServiceEnv_TAG:
			serviceEnv[ServiceEnv_TAG.String()] = ci.Tag
		case ServiceEnv_COMMIT_ID:
			serviceEnv[ServiceEnv_COMMIT_ID.String()] = ci.CommitId
		case ServiceEnv_SERVICE_ID:
			serviceEnv[ServiceEnv_SERVICE_ID.String()] = cast.ToString(service.Id)
		case ServiceEnv_IMAGE:
			serviceEnv[ServiceEnv_IMAGE.String()] = ci.GetImage(workspace, service)
		case ServiceEnv_GIT_REPO:
			serviceEnv[ServiceEnv_GIT_REPO.String()] = workspace.GitRepository
		case ServiceEnv_IMAGE_REPO:
			serviceEnv[ServiceEnv_IMAGE_REPO.String()] = workspace.ImageRepository
		case ServiceEnv_GIT_REPO_NAME:
			serviceEnv[ServiceEnv_GIT_REPO_NAME.String()] = workspace.GetGitRepoName()
		case ServiceEnv_IMAGE_REPO_NAME:
			serviceEnv[ServiceEnv_IMAGE_REPO_NAME.String()] = workspace.GetImageRepoName()
		case ServiceEnv_GIT_REPO_TOKEN:
			serviceEnv[ServiceEnv_GIT_REPO_TOKEN.String()] = workspace.GitRepositoryToken
		case ServiceEnv_IMAGE_REPO_TOKEN:
			serviceEnv[ServiceEnv_IMAGE_REPO_TOKEN.String()] = workspace.ImageRepositoryToken
		}
	}
	w.Env = utils.MapToString(serviceEnv)
	return serviceEnv
}

func (w *Workflow) SettingContinuousIntegration(ctx context.Context, service *Service, ci *ContinuousIntegration) {
	workspace := GetWorkspace(ctx)
	serviceEnv := w.SettingServiceEnv(ctx, workspace, service, ci)
	for _, step := range w.WorkflowSteps {
		for _, task := range step.WorkflowTasks {
			wfType := WorkflowStepTypeStringToEnum(task.Name)
			if wfType == WorkflowStepType_Customizable {
				task.TaskCommand = utils.DecodeString(task.TaskCommand, serviceEnv)
				continue
			}
			switch WorkflowStepType(wfType) {
			case WorkflowStepType_CodePull:
				workspace.GitRepository = strings.ReplaceAll(workspace.GitRepository, "https://", "")
				gitRepoUrl := fmt.Sprintf("https://%s:%s@%s/%s.git", workspace.GetGitRepoName(), workspace.GitRepositoryToken, workspace.GitRepository, service.Name)
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
					workspace.ImageRepositoryToken, workspace.GetImageRepoName(), workspace.ImageRepository)
			case WorkflowStepType_Build:
				task.TaskCommand = fmt.Sprintf("ls -la /app && mkdir /root/.docker && cp /app/config.json /root/.docker && buildkitd --rootless --addr unix:///run/buildkit/buildkitd.sock & sleep 3 && buildctl build --frontend dockerfile.v0 --local context=/app --local dockerfile=/app --output type=image,name=%s,push=true",
					ci.GetImage(workspace, service))
			}
		}
	}
}

func (w *Workflow) SettingContinuousDeployment(ctx context.Context, service *Service, ci *ContinuousIntegration, cd *ContinuousDeployment) error {
	cluster := GetCluster(ctx)
	workspace := GetWorkspace(ctx)
	serviceEnv := w.SettingServiceEnv(ctx, workspace, service, ci)
	for _, step := range w.WorkflowSteps {
		for _, task := range step.WorkflowTasks {
			wfType := WorkflowStepTypeStringToEnum(task.Name)
			if wfType == WorkflowStepType_Customizable {
				task.TaskCommand = utils.DecodeString(task.TaskCommand, serviceEnv)
				continue
			}
			if WorkflowStepType_Deploy == WorkflowStepType(wfType) {
				task.TaskCommand = fmt.Sprintf("curl -X POST -H 'Content-Type: application/json' -d '{\"id\":\"%d\",\"ci_id\":\"%d\",\"cd_id\":\"%d\"}' http://%s/api/v1alpha1/service/apply",
					service.Id, ci.Id, cd.Id, cluster.Name)
			}
		}
	}
	return nil
}

func (ci *ContinuousIntegration) GetImage(w *Workspace, s *Service) string {
	return fmt.Sprintf("%s/%s:%s", w.GetImageRepoName(), s.Name, ci.Version)
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
