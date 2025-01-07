package interfaces

import (
	"context"
	"encoding/json"

	"github.com/f-rambo/cloud-copilot/api/common"
	v1alpha1 "github.com/f-rambo/cloud-copilot/api/service/v1alpha1"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/pkg/errors"
)

type ServicesInterface struct {
	v1alpha1.UnimplementedServiceInterfaceServer
	serviceUc *biz.ServicesUseCase
	projectUc *biz.ProjectUsecase
	userUc    *biz.UserUseCase
}

func NewServicesInterface(serviceUc *biz.ServicesUseCase, projectUc *biz.ProjectUsecase) *ServicesInterface {
	return &ServicesInterface{serviceUc: serviceUc, projectUc: projectUc}
}

func (s *ServicesInterface) List(ctx context.Context, serviceParam *v1alpha1.ServiceRequest) (*v1alpha1.Services, error) {
	services, itemCount, err := s.serviceUc.List(ctx, &biz.Service{
		Name:      serviceParam.Name,
		ProjectId: serviceParam.ProjectId,
	}, int(serviceParam.Page), int(serviceParam.PageSize))
	if err != nil {
		return nil, err
	}
	projectIds := make([]int64, 0)
	for _, v := range services {
		projectIds = append(projectIds, v.ProjectId)
	}
	projects, err := s.projectUc.ListByIds(ctx, projectIds)
	if err != nil {
		return nil, err
	}
	projectMap := make(map[int64]string)
	for _, v := range projects {
		projectMap[v.Id] = v.Name
	}
	interfaceServices := make([]*v1alpha1.Service, 0)
	for _, service := range services {
		interfaceServices = append(interfaceServices, s.bizToInterface(service))
	}
	for _, v := range interfaceServices {
		projectName, ok := projectMap[v.ProjectID]
		if ok {
			v.ProjectName = projectName
		}
	}
	return &v1alpha1.Services{
		Services: interfaceServices,
		Total:    itemCount,
	}, nil
}

func (s *ServicesInterface) Save(ctx context.Context, serviceParam *v1alpha1.Service) (*common.Msg, error) {
	err := s.serviceUc.Save(ctx, s.interfaceToBiz(serviceParam))
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) Get(ctx context.Context, serviceParam *v1alpha1.ServiceRequest) (*v1alpha1.Service, error) {
	service, err := s.serviceUc.Get(ctx, serviceParam.Id)
	if err != nil {
		return nil, err
	}
	return s.bizToInterface(service), nil
}

func (s *ServicesInterface) Delete(ctx context.Context, serviceParam *v1alpha1.ServiceRequest) (*common.Msg, error) {
	if serviceParam.Id == 0 {
		return nil, errors.New("id is required")
	}

	err := s.serviceUc.Delete(ctx, serviceParam.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

// GetDefaultWorkflow
func (s *ServicesInterface) GetWorkflow(ctx context.Context, serviceParam *v1alpha1.ServiceRequest) (*v1alpha1.Worklfow, error) {
	if serviceParam.Id == 0 {
		return nil, errors.New("id is required")
	}
	workflow, err := s.serviceUc.GetWorkflow(ctx, serviceParam.Id, biz.WorkflowType(serviceParam.WfType))
	if err != nil {
		return nil, err
	}
	wfData := &v1alpha1.Worklfow{
		ID:          workflow.Id,
		Name:        workflow.Name,
		Steps:       make([]*v1alpha1.WfStep, 0),
		Templates:   make([]*v1alpha1.WfTemplate, 0),
		Description: "",
	}
	// argoWf := wfv1.Workflow{}
	// err = json.Unmarshal([]byte(workflow.Workflow), &argoWf)
	// if err != nil {
	// 	return nil, err
	// }
	// wfTasks := make(map[string][]*v1alpha1.WfTask) // stepname -> wftasks
	// for _, template := range argoWf.Spec.Templates {
	// 	if template.Container != nil && template.Container.Image != "" {
	// 		wfData.Templates = append(wfData.Templates, &v1alpha1.WfTemplate{
	// 			Name:    template.Name,
	// 			Image:   template.Container.Image,
	// 			Command: template.Container.Command,
	// 			Args:    template.Container.Args,
	// 		})
	// 	}
	// 	if template.Script != nil && template.Script.Image != "" {
	// 		wfData.Templates = append(wfData.Templates, &v1alpha1.WfTemplate{
	// 			Name:     template.Name,
	// 			Image:    template.Script.Image,
	// 			Command:  template.Script.Command,
	// 			Source:   template.Script.Source,
	// 			IsScript: true,
	// 		})
	// 	}
	// 	if template.DAG != nil && len(template.DAG.Tasks) > 0 {
	// 		for _, task := range template.DAG.Tasks {
	// 			if task.Template == "" {
	// 				continue
	// 			}
	// 			taskVal := &v1alpha1.WfTask{
	// 				Name:         task.Name,
	// 				TemplateName: task.Template,
	// 			}
	// 			if strings.Contains(task.Name, "default-") {
	// 				taskVal.Name = strings.Replace(task.Name, "default-", "", 1)
	// 				taskVal.Default = true
	// 			}
	// 			if _, ok := wfTasks[template.Name]; !ok {
	// 				wfTasks[template.Name] = make([]*v1alpha1.WfTask, 0)
	// 			}
	// 			wfTasks[template.Name] = append(wfTasks[template.Name], taskVal)
	// 		}
	// 	}
	// }
	// parallelSteps := make([]wfv1.ParallelSteps, 0)
	// for _, template := range argoWf.Spec.Templates {
	// 	if len(template.Steps) > 0 {
	// 		parallelSteps = append(parallelSteps, template.Steps...)
	// 	}
	// }
	// for _, steps := range parallelSteps {
	// 	for _, step := range steps.Steps {
	// 		wfStep := &v1alpha1.WfStep{
	// 			Name: step.Name,
	// 		}
	// 		if strings.Contains(step.Name, "default-") {
	// 			wfStep.Name = strings.Replace(step.Name, "default-", "", 1)
	// 			wfStep.Default = true
	// 		}
	// 		if wftasks, ok := wfTasks[step.Template]; ok {
	// 			wfStep.Tasks = wftasks
	// 		}
	// 		wfData.Steps = append(wfData.Steps, wfStep)
	// 	}
	// }
	return wfData, nil
}

func (s *ServicesInterface) SaveWorkflow(ctx context.Context, request *v1alpha1.ServiceRequest) (*common.Msg, error) {
	if request.Id == 0 {
		return nil, errors.New("service id is required")
	}
	if request.Workflow == nil || len(request.Workflow.Steps) == 0 || len(request.Workflow.Templates) == 0 {
		return nil, errors.New("workflow is required")
	}
	wfData, err := json.Marshal(request.Workflow)
	if err != nil {
		return nil, err
	}
	wf := &biz.Workflow{
		Name:     request.Workflow.Name,
		Workflow: string(wfData),
	}
	err = s.serviceUc.SaveWorkflow(ctx, request.Id, biz.WorkflowType(request.WfType), wf)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) CommitWorklfow(ctx context.Context, request *v1alpha1.ServiceRequest) (*common.Msg, error) {
	if request.Id == 0 {
		return nil, errors.New("service id is required")
	}
	if request.WorkflowId == 0 {
		return nil, errors.New("workflow id is required")
	}
	if request.WfType == 0 {
		return nil, errors.New("workflow type is required")
	}
	service, err := s.serviceUc.Get(ctx, request.Id)
	if err != nil {
		return nil, err
	}
	project, err := s.projectUc.Get(ctx, service.ProjectId)
	if err != nil {
		return nil, err
	}
	err = s.serviceUc.CommitWorklfow(ctx, project, service, biz.WorkflowType(request.WfType), request.WorkflowId)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) GetServiceCis(ctx context.Context, request *v1alpha1.CIsRequest) (*v1alpha1.CIsResult, error) {
	cis, total, err := s.serviceUc.GetServiceCis(ctx, request.ServiceID, request.Page, request.PageSize)
	if err != nil {
		return nil, err
	}
	cisResult := &v1alpha1.CIsResult{
		CIs:   make([]*v1alpha1.CI, 0),
		Total: total,
	}
	userIds := make([]int64, 0)
	for _, v := range cis {
		userIds = append(userIds, v.UserId)
	}
	users, err := s.userUc.GetUserByBatchID(ctx, userIds)
	if err != nil {
		return nil, err
	}
	userMap := make(map[int64]string)
	for _, v := range users {
		userMap[v.Id] = v.Name
	}
	for _, ci := range cis {
		data := s.bizCiTointerface(ci)
		userName, ok := userMap[data.UserId]
		if ok {
			data.Username = userName
		}
		cisResult.CIs = append(cisResult.CIs, data)
	}
	return cisResult, nil
}

func (s *ServicesInterface) bizToInterface(service *biz.Service) *v1alpha1.Service {
	servicesInterface := &v1alpha1.Service{
		ID:          service.Id,
		Name:        service.Name,
		CodeRepo:    service.CodeRepo,
		Replicas:    service.Replicas,
		CPU:         service.Cpu,
		LimitCpu:    service.LimitCpu,
		GPU:         service.Gpu,
		LimitGPU:    service.LimitGpu,
		Memory:      service.Memory,
		LimitMemory: service.LimitMemory,
		Disk:        service.Disk,
		LimitDisk:   service.LimitDisk,
		ProjectID:   service.ProjectId,
		Business:    service.Business,
		Technology:  service.Technology,
		Ports:       make([]*v1alpha1.Port, 0),
	}
	for _, port := range service.Ports {
		port := &v1alpha1.Port{
			ID:            port.Id,
			IngressPath:   port.IngressPath,
			Protocol:      port.Protocol,
			ContainerPort: port.ContainerPort,
		}
		servicesInterface.Ports = append(servicesInterface.Ports, port)
	}
	return servicesInterface
}

func (s *ServicesInterface) interfaceToBiz(service *v1alpha1.Service) *biz.Service {
	ports := make([]*biz.Port, 0)
	for _, port := range service.Ports {
		ports = append(ports, &biz.Port{
			Id:            port.ID,
			IngressPath:   port.IngressPath,
			Protocol:      port.Protocol,
			ContainerPort: port.ContainerPort,
		})
	}
	return &biz.Service{
		Id:          service.ID,
		Name:        service.Name,
		CodeRepo:    service.CodeRepo,
		Replicas:    service.Replicas,
		Cpu:         service.CPU,
		LimitCpu:    service.LimitCpu,
		Gpu:         service.GPU,
		LimitGpu:    service.LimitGPU,
		Memory:      service.Memory,
		LimitMemory: service.LimitMemory,
		Disk:        service.Disk,
		LimitDisk:   service.LimitDisk,
		ProjectId:   service.ProjectID,
		Business:    service.Business,
		Technology:  service.Technology,
		Ports:       ports,
	}
}

func (s *ServicesInterface) bizCiTointerface(ci *biz.CI) *v1alpha1.CI {
	ciInterface := &v1alpha1.CI{
		ID:          ci.Id,
		Version:     ci.Version,
		Branch:      ci.Branch,
		Tag:         ci.Tag,
		Status:      ci.Status.String(),
		Description: ci.Description,
		ServiceID:   ci.ServiceId,
		UserId:      ci.UserId,
		Logs:        ci.Logs,
		CreatedAt:   ci.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	return ciInterface
}
