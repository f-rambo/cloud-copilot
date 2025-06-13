package data

import (
	"context"
	"strings"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type servicesRepo struct {
	data *Data
	log  *log.Helper
}

func NewServicesRepo(data *Data, logger log.Logger) biz.ServicesData {
	return &servicesRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (s *servicesRepo) Save(ctx context.Context, service *biz.Service) (err error) {
	tx := s.data.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
	}()
	err = tx.Where("id = ?", service.Id).Save(service).Error
	if err != nil {
		return err
	}
	err = s.savePods(ctx, tx, service)
	if err != nil {
		return err
	}
	err = s.saveVolume(ctx, tx, service)
	if err != nil {
		return err
	}
	err = s.savePort(ctx, tx, service)
	if err != nil {
		return err
	}
	err = tx.Commit().Error
	if err != nil {
		return err
	}
	return nil
}

// save pods
func (s *servicesRepo) savePods(_ context.Context, tx *gorm.DB, service *biz.Service) (err error) {
	// Get existing pods
	existingPods := make([]*biz.Pod, 0)
	if err = tx.Where("service_id =?", service.Id).Find(&existingPods).Error; err != nil {
		return err
	}
	// Track pods to keep by ID
	podsToKeep := make(map[int64]bool)
	// Save or update pods
	for _, pod := range service.Pods {
		pod.ServiceId = service.Id
		if pod.Id > 0 {
			// Update existing pod
			if err = tx.Where("id =?", pod.Id).Save(pod).Error; err != nil {
				return err
			}
		} else {
			// Create new pod
			if err = tx.Create(pod).Error; err != nil {
				return err
			}
		}
	}
	// Delete pods that are no longer needed
	for _, existingPod := range existingPods {
		if !podsToKeep[existingPod.Id] {
			if err = tx.Delete(&biz.Pod{}, existingPod.Id).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// save Volume
func (s *servicesRepo) saveVolume(_ context.Context, tx *gorm.DB, service *biz.Service) (err error) {
	// Get existing volumes
	existingVolumes := make([]*biz.Volume, 0)
	if err = tx.Where("service_id = ?", service.Id).Find(&existingVolumes).Error; err != nil {
		return err
	}

	// Track volumes to keep by ID
	volumesToKeep := make(map[int64]bool)

	// Save or update volumes
	for _, volume := range service.Volumes {
		volume.ServiceId = service.Id
		if volume.Id > 0 {
			// Update existing volume
			if err = tx.Where("id = ?", volume.Id).Save(volume).Error; err != nil {
				return err
			}
			volumesToKeep[volume.Id] = true
		} else {
			// Create new volume
			if err = tx.Create(volume).Error; err != nil {
				return err
			}
		}
	}

	// Delete volumes that are no longer needed
	for _, existingVolume := range existingVolumes {
		if !volumesToKeep[existingVolume.Id] {
			if err = tx.Delete(&biz.Volume{}, existingVolume.Id).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// save Port
func (s *servicesRepo) savePort(_ context.Context, tx *gorm.DB, service *biz.Service) (err error) {
	// Get existing ports
	existingPorts := make([]*biz.Port, 0)
	if err = tx.Where("service_id =?", service.Id).Find(&existingPorts).Error; err != nil {
		return err
	}
	// Track ports to keep by ID
	portsToKeep := make(map[int64]bool)
	// Save or update ports
	for _, port := range service.Ports {
		port.ServiceId = service.Id
		if port.Id > 0 {
			// Update existing port
			if err = tx.Where("id =?", port.Id).Save(port).Error; err != nil {
				return err
			}
		} else {
			// Create new port
			if err = tx.Create(port).Error; err != nil {
				return err
			}
		}
	}
	// Delete ports that are no longer needed
	for _, existingPort := range existingPorts {
		if !portsToKeep[existingPort.Id] {
			if err = tx.Delete(&biz.Port{}, existingPort.Id).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *servicesRepo) Get(ctx context.Context, id int64) (*biz.Service, error) {
	service := &biz.Service{
		Volumes: make([]*biz.Volume, 0),
		Ports:   make([]*biz.Port, 0),
	}
	err := s.data.db.Where("id = ?", id).First(service).Error
	if err != nil {
		return nil, err
	}
	// Get volumes
	if err = s.data.db.Where("service_id =?", id).Find(&service.Volumes).Error; err != nil {
		return nil, err
	}
	// Get ports
	if err = s.data.db.Where("service_id =?", id).Find(&service.Ports).Error; err != nil {
		return nil, err
	}
	return service, nil
}

func (s *servicesRepo) List(ctx context.Context, projectId int64, serviceName string, page, pageSize int32) ([]*biz.Service, int64, error) {
	var itemCount int64 = 0
	services := make([]*biz.Service, 0)
	serviceModel := s.data.db.Model(&biz.Service{})
	if projectId != 0 {
		serviceModel = serviceModel.Where("project_id = ?", projectId)
	}
	if serviceName != "" {
		serviceModel = serviceModel.Where("name like ?", "%"+serviceName+"%")
	}
	err := serviceModel.Count(&itemCount).Error
	if err != nil {
		return nil, 0, err
	}
	if itemCount == 0 {
		return services, 0, nil
	}
	err = serviceModel.Offset((int(page) - 1) * int(pageSize)).Limit(int(pageSize)).Find(&services).Error
	if err != nil {
		return nil, 0, err
	}
	return services, itemCount, nil
}

func (s *servicesRepo) Delete(ctx context.Context, id int64) error {
	return s.data.db.Delete(&biz.Service{}, id).Error
}

func (s *servicesRepo) GetByName(ctx context.Context, projectId int64, name string) (*biz.Service, error) {
	service := &biz.Service{}
	err := s.data.db.Where("project_id = ? and name = ?", projectId, name).First(service).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return service, nil
}

func (s *servicesRepo) SaveWorkflow(ctx context.Context, workflow *biz.Workflow) (err error) {
	tx := s.data.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
	}()

	// Save workflow
	if err = tx.Where("id = ?", workflow.Id).Save(workflow).Error; err != nil {
		return err
	}

	// Get existing steps and tasks
	existingSteps := make([]*biz.WorkflowStep, 0)
	existingTasks := make([]*biz.WorkflowTask, 0)
	if err = tx.Where("workflow_id = ?", workflow.Id).Find(&existingSteps).Error; err != nil {
		return err
	}
	if err = tx.Where("workflow_id = ?", workflow.Id).Find(&existingTasks).Error; err != nil {
		return err
	}

	// Track IDs to keep
	keepStepIDs := make(map[int64]bool)
	keepTaskIDs := make(map[int64]bool)

	// Save or update steps and tasks
	for _, step := range workflow.WorkflowSteps {
		step.WorkflowId = workflow.Id
		if err = tx.Where("id = ?", step.Id).Save(step).Error; err != nil {
			return err
		}
		keepStepIDs[step.Id] = true

		for _, task := range step.WorkflowTasks {
			task.WorkflowId = workflow.Id
			task.StepId = step.Id
			if err = tx.Where("id = ?", task.Id).Save(task).Error; err != nil {
				return err
			}
			keepTaskIDs[task.Id] = true
		}
	}

	// Delete only steps and tasks that are no longer needed
	for _, step := range existingSteps {
		if !keepStepIDs[step.Id] {
			if err = tx.Delete(&biz.WorkflowStep{}, step.Id).Error; err != nil {
				return err
			}
		}
	}

	for _, task := range existingTasks {
		if !keepTaskIDs[task.Id] {
			if err = tx.Delete(&biz.WorkflowTask{}, task.Id).Error; err != nil {
				return err
			}
		}
	}

	return tx.Commit().Error
}

func (s *servicesRepo) GetWorkflowByServiceId(ctx context.Context, serviceId int64) ([]*biz.Workflow, error) {
	workflows := make([]*biz.Workflow, 0)
	err := s.data.db.Where("service_id =?", serviceId).Find(&workflows).Error
	if err != nil {
		return nil, err
	}
	for _, workflow := range workflows {
		workflow.WorkflowSteps = make([]*biz.WorkflowStep, 0)
		err = s.data.db.Where("workflow_id =?", workflow.Id).Find(&workflow.WorkflowSteps).Error
		if err != nil {
			return nil, err
		}
		for _, step := range workflow.WorkflowSteps {
			step.WorkflowTasks = make([]*biz.WorkflowTask, 0)
			err = s.data.db.Where("step_id =?", step.Id).Find(&step.WorkflowTasks).Error
			if err != nil {
				return nil, err
			}
		}
	}
	return workflows, nil
}

func (s *servicesRepo) SaveContinuousIntegration(ctx context.Context, ci *biz.ContinuousIntegration) error {
	return s.data.db.Where("id = ?", ci.Id).Save(ci).Error
}

func (s *servicesRepo) GetContinuousIntegration(ctx context.Context, ciId int64) (*biz.ContinuousIntegration, error) {
	ci := &biz.ContinuousIntegration{}
	err := s.data.db.Where("id =?", ciId).First(ci).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return ci, nil
}

func (s *servicesRepo) DeleteContinuousIntegration(ctx context.Context, ciId int64) error {
	return s.data.db.Delete(&biz.ContinuousIntegration{}, ciId).Error
}

func (s *servicesRepo) GetContinuousIntegrations(ctx context.Context, serviceId int64, page, pageSize int32) ([]*biz.ContinuousIntegration, int64, error) {
	var itemCount int64 = 0
	cis := make([]*biz.ContinuousIntegration, 0)
	query := s.data.db.Model(&biz.ContinuousIntegration{}).Where("service_id = ?", serviceId)
	if err := query.Count(&itemCount).Error; err != nil {
		return nil, 0, err
	}
	if itemCount == 0 {
		return cis, 0, nil
	}
	err := query.Offset(int((page - 1) * pageSize)).
		Order("id desc").
		Limit(int(pageSize)).
		Find(&cis).Error
	if err != nil {
		return nil, 0, err
	}
	return cis, itemCount, nil
}

func (s *servicesRepo) SaveContinuousDeployment(ctx context.Context, cd *biz.ContinuousDeployment) error {
	return s.data.db.Where("id =?", cd.Id).Save(cd).Error
}

func (s *servicesRepo) GetContinuousDeployment(ctx context.Context, cdId int64) (*biz.ContinuousDeployment, error) {
	cd := &biz.ContinuousDeployment{}
	err := s.data.db.Where("id =?", cdId).First(cd).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return cd, nil
}

func (s *servicesRepo) DeleteContinuousDeployment(ctx context.Context, cdId int64) error {
	return s.data.db.Delete(&biz.ContinuousDeployment{}, cdId).Error
}

func (s *servicesRepo) GetContinuousDeployments(ctx context.Context, serviceId int64, page, pageSize int32) ([]*biz.ContinuousDeployment, int64, error) {
	var itemCount int64 = 0
	cds := make([]*biz.ContinuousDeployment, 0)
	query := s.data.db.Model(&biz.ContinuousDeployment{}).Where("service_id =?", serviceId)
	if err := query.Count(&itemCount).Error; err != nil {
		return nil, 0, err
	}
	if itemCount == 0 {
		return cds, 0, nil
	}
	err := query.Offset(int((page - 1) * pageSize)).
		Order("id desc").
		Limit(int(pageSize)).
		Find(&cds).Error
	if err != nil {
		return nil, 0, err
	}
	return cds, itemCount, nil
}

func (s *servicesRepo) GetContinuousIntegrationLog(ctx context.Context, ci *biz.ContinuousIntegration, page, pageSize int) (biz.LogResponse, error) {
	logResponse := biz.LogResponse{}
	pods := strings.Split(ci.Pods, ",")
	if len(pods) == 0 || s.data.esClient == nil {
		return logResponse, nil
	}
	podLogRes, err := s.data.esClient.SearchByKeyword(
		ctx,
		s.data.esClient.GetIndexPatternName(PodLogIndexName),
		map[string][]string{
			PodKeyWord: pods,
		},
		page,
		pageSize,
	)
	if err != nil {
		return logResponse, err
	}
	logResponse.Total = podLogRes.Total
	logResponse.Log = podLogRes.Data
	return logResponse, nil
}

// ContinuousDeployment log
func (s *servicesRepo) GetContinuousDeploymentLog(ctx context.Context, cd *biz.ContinuousDeployment, page, pageSize int) (biz.LogResponse, error) {
	logResponse := biz.LogResponse{}
	pods := strings.Split(cd.Pods, ",")
	if len(pods) == 0 || s.data.esClient == nil {
		return logResponse, nil
	}
	podLogRes, err := s.data.esClient.SearchByKeyword(
		ctx,
		s.data.esClient.GetIndexPatternName(PodLogIndexName),
		map[string][]string{
			PodKeyWord: pods,
		},
		page,
		pageSize,
	)
	if err != nil {
		return logResponse, err
	}
	logResponse.Total = podLogRes.Total
	logResponse.Log = podLogRes.Data
	return logResponse, nil
}

// Service log
func (s *servicesRepo) GetServicePodLog(ctx context.Context, service *biz.Service, page, pageSize int) (biz.LogResponse, error) {
	logResponse := biz.LogResponse{}
	if len(service.Pods) == 0 || s.data.esClient == nil {
		return logResponse, nil
	}
	podNames := make([]string, 0)
	for _, pod := range service.Pods {
		podNames = append(podNames, pod.Name)
	}
	podLogRes, err := s.data.esClient.SearchByKeyword(
		ctx,
		s.data.esClient.GetIndexPatternName(PodLogIndexName),
		map[string][]string{
			PodKeyWord: podNames,
		},
		page,
		pageSize,
	)
	if err != nil {
		return logResponse, err
	}
	logResponse.Total = podLogRes.Total
	logResponse.Log = podLogRes.Data
	return logResponse, nil
}

func (s *servicesRepo) GetServiceLog(ctx context.Context, service *biz.Service, page, pageSize int) (biz.LogResponse, error) {
	logResponse := biz.LogResponse{}
	if s.data.esClient == nil {
		return logResponse, nil
	}
	podLogRes, err := s.data.esClient.SearchByKeyword(
		ctx,
		s.data.esClient.GetIndexPatternName(ServiceLogIndexName),
		map[string][]string{
			ServiceIdKeyWord: {cast.ToString(service.Id)},
		},
		page,
		pageSize,
	)
	if err != nil {
		return logResponse, err
	}
	logResponse.Total = podLogRes.Total
	logResponse.Log = podLogRes.Data
	return logResponse, nil
}
