package data

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
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

func (s *servicesRepo) Save(ctx context.Context, service *biz.Service) error {
	return s.data.db.Where("id = ?", service.Id).Save(service).Error
}

func (s *servicesRepo) Get(ctx context.Context, id int64) (*biz.Service, error) {
	service := &biz.Service{}
	err := s.data.db.Where("id = ?", id).First(service).Error
	if err != nil {
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

func (s *servicesRepo) GetServiceResourceByProject(ctx context.Context, projectId int64, alreadyResource *biz.AlreadyResource) error {
	return s.data.db.Model(&biz.Service{}).
		Select("SUM(replicas * limit_cpu) as cpu, SUM(replicas * limit_memory) as memory, SUM(replicas * limit_gpu) as gpu, SUM(replicas * storage) as storage").
		Where("project_id = ?", projectId).
		Scan(alreadyResource).Error
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
