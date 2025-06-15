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

	// 保存Service主体
	err = tx.Where("id = ?", service.Id).Save(service).Error
	if err != nil {
		return err
	}

	// 处理Pods
	if len(service.Pods) > 0 {
		// 获取数据库中现有的pods
		existingPods := make([]*biz.Pod, 0)
		err = tx.Where("service_id = ?", service.Id).Find(&existingPods).Error
		if err != nil {
			return err
		}

		// 创建现有pods的映射 (name -> pod)
		existingPodMap := make(map[string]*biz.Pod)
		for _, pod := range existingPods {
			existingPodMap[pod.Name] = pod
		}

		// 创建新pods的映射
		newPodMap := make(map[string]*biz.Pod)
		for _, pod := range service.Pods {
			pod.ServiceId = service.Id
			newPodMap[pod.Name] = pod
		}

		// 删除不在新列表中的pods
		for name, existingPod := range existingPodMap {
			if _, exists := newPodMap[name]; !exists {
				err = tx.Delete(existingPod).Error
				if err != nil {
					return err
				}
			}
		}

		// 保存或更新pods
		for _, pod := range service.Pods {
			if existingPod, exists := existingPodMap[pod.Name]; exists {
				// 更新现有pod
				pod.Id = existingPod.Id
				err = tx.Save(pod).Error
			} else {
				// 创建新pod
				err = tx.Create(pod).Error
			}
			if err != nil {
				return err
			}
		}
	} else {
		// 如果pods数组为空，删除所有现有pods
		err = tx.Where("service_id = ?", service.Id).Delete(&biz.Pod{}).Error
		if err != nil {
			return err
		}
	}

	// 处理Volumes
	if len(service.Volumes) > 0 {
		// 获取数据库中现有的volumes
		existingVolumes := make([]*biz.Volume, 0)
		err = tx.Where("service_id = ?", service.Id).Find(&existingVolumes).Error
		if err != nil {
			return err
		}

		// 创建现有volumes的映射 (name -> volume)
		existingVolumeMap := make(map[string]*biz.Volume)
		for _, volume := range existingVolumes {
			existingVolumeMap[volume.Name] = volume
		}

		// 创建新volumes的映射
		newVolumeMap := make(map[string]*biz.Volume)
		for _, volume := range service.Volumes {
			volume.ServiceId = service.Id
			newVolumeMap[volume.Name] = volume
		}

		// 删除不在新列表中的volumes
		for name, existingVolume := range existingVolumeMap {
			if _, exists := newVolumeMap[name]; !exists {
				err = tx.Delete(existingVolume).Error
				if err != nil {
					return err
				}
			}
		}

		// 保存或更新volumes
		for _, volume := range service.Volumes {
			if existingVolume, exists := existingVolumeMap[volume.Name]; exists {
				// 更新现有volume
				volume.Id = existingVolume.Id
				err = tx.Save(volume).Error
			} else {
				// 创建新volume
				err = tx.Create(volume).Error
			}
			if err != nil {
				return err
			}
		}
	} else {
		// 如果volumes数组为空，删除所有现有volumes
		err = tx.Where("service_id = ?", service.Id).Delete(&biz.Volume{}).Error
		if err != nil {
			return err
		}
	}

	// 处理Ports
	if len(service.Ports) > 0 {
		// 获取数据库中现有的ports
		existingPorts := make([]*biz.Port, 0)
		err = tx.Where("service_id = ?", service.Id).Find(&existingPorts).Error
		if err != nil {
			return err
		}

		// 创建现有ports的映射 (name -> port)
		existingPortMap := make(map[string]*biz.Port)
		for _, port := range existingPorts {
			existingPortMap[port.Name] = port
		}

		// 创建新ports的映射
		newPortMap := make(map[string]*biz.Port)
		for _, port := range service.Ports {
			port.ServiceId = service.Id
			newPortMap[port.Name] = port
		}

		// 删除不在新列表中的ports
		for name, existingPort := range existingPortMap {
			if _, exists := newPortMap[name]; !exists {
				err = tx.Delete(existingPort).Error
				if err != nil {
					return err
				}
			}
		}

		// 保存或更新ports
		for _, port := range service.Ports {
			if existingPort, exists := existingPortMap[port.Name]; exists {
				// 更新现有port
				port.Id = existingPort.Id
				err = tx.Save(port).Error
			} else {
				// 创建新port
				err = tx.Create(port).Error
			}
			if err != nil {
				return err
			}
		}
	} else {
		// 如果ports数组为空，删除所有现有ports
		err = tx.Where("service_id = ?", service.Id).Delete(&biz.Port{}).Error
		if err != nil {
			return err
		}
	}

	err = tx.Commit().Error
	if err != nil {
		return err
	}
	return nil
}

func (s *servicesRepo) Get(ctx context.Context, id int64) (*biz.Service, error) {
	service := &biz.Service{
		Volumes: make([]*biz.Volume, 0),
		Ports:   make([]*biz.Port, 0),
		Pods:    make([]*biz.Pod, 0),
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
	// Get pods
	if err = s.data.db.Where("service_id =?", id).Find(&service.Pods).Error; err != nil {
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
	serviceIds := make([]int64, 0)
	for _, service := range services {
		serviceIds = append(serviceIds, service.Id)
	}
	// get ports by serviceids
	ports := make([]*biz.Port, 0)
	if err = s.data.db.Where("service_id IN (?)", serviceIds).Find(&ports).Error; err != nil {
		return nil, 0, err
	}
	// get pods by serviceids
	pods := make([]*biz.Pod, 0)
	if err = s.data.db.Where("service_id IN (?)", serviceIds).Find(&pods).Error; err != nil {
		return nil, 0, err
	}
	// get volumes by serviceids
	volumes := make([]*biz.Volume, 0)
	if err = s.data.db.Where("service_id IN (?)", serviceIds).Find(&volumes).Error; err != nil {
		return nil, 0, err
	}
	// get ports by serviceids
	portMap := make(map[int64][]*biz.Port)
	for _, port := range ports {
		portMap[port.ServiceId] = append(portMap[port.ServiceId], port)
	}
	// get pods by serviceids
	podMap := make(map[int64][]*biz.Pod)
	for _, pod := range pods {
		podMap[pod.ServiceId] = append(podMap[pod.ServiceId], pod)
	}
	// get volumes by serviceids
	volumeMap := make(map[int64][]*biz.Volume)
	for _, volume := range volumes {
		volumeMap[volume.ServiceId] = append(volumeMap[volume.ServiceId], volume)
	}
	// assign ports and pods to services
	for _, service := range services {
		service.Ports = portMap[service.Id]
		service.Pods = podMap[service.Id]
		service.Volumes = volumeMap[service.Id]
	}
	return services, itemCount, nil
}

func (s *servicesRepo) Delete(ctx context.Context, id int64) error {
	tx := s.data.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var workflowIDs []int64
	if err := tx.Model(&biz.Workflow{}).Where("service_id = ?", id).Pluck("id", &workflowIDs).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(workflowIDs) > 0 {
		if err := tx.Where("workflow_id IN ?", workflowIDs).Delete(&biz.WorkflowTask{}).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Where("workflow_id IN ?", workflowIDs).Delete(&biz.WorkflowStep{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Where("service_id = ?", id).Delete(&biz.Workflow{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("service_id = ?", id).Delete(&biz.ContinuousDeployment{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("service_id = ?", id).Delete(&biz.ContinuousIntegration{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("service_id = ?", id).Delete(&biz.Pod{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("service_id = ?", id).Delete(&biz.Volume{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("service_id = ?", id).Delete(&biz.Port{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&biz.Service{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit().Error
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
