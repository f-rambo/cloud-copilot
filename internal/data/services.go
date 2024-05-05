package data

import (
	"context"
	"encoding/json"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
)

type servicesRepo struct {
	data *Data
	log  *log.Helper
}

func NewServicesRepo(data *Data, logger log.Logger) biz.ServicesRepo {
	return &servicesRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (s *servicesRepo) List(ctx context.Context, serviceParam *biz.Service, page, pageSize int) ([]*biz.Service, int64, error) {
	var itemCount int64 = 0
	services := make([]*biz.Service, 0)
	serviceModel := s.data.db.Model(&biz.Service{})
	if serviceParam.ProjectID != 0 {
		serviceModel = serviceModel.Where("project_id = ?", serviceParam.ProjectID)
	}
	if serviceParam.Name != "" {
		serviceModel = serviceModel.Where("name like ?", "%"+serviceParam.Name+"%")
	}
	err := serviceModel.Count(&itemCount).Error
	if err != nil {
		return nil, 0, err
	}
	if itemCount == 0 {
		return services, 0, nil
	}
	err = serviceModel.Offset((page - 1) * pageSize).Limit(pageSize).Find(&services).Error
	if err != nil {
		return nil, 0, err
	}
	for _, service := range services {
		err = json.Unmarshal(service.PortsJson, &service.Ports)
		if err != nil {
			return nil, 0, err
		}
	}
	return services, itemCount, nil
}

func (s *servicesRepo) Save(ctx context.Context, service *biz.Service) (err error) {
	service.PortsJson, err = json.Marshal(service.Ports)
	if err != nil {
		return err
	}
	return s.data.db.Save(service).Error
}

func (s *servicesRepo) Get(ctx context.Context, id int64) (*biz.Service, error) {
	service := &biz.Service{}
	err := s.data.db.Where("id = ?", id).First(service).Error
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(service.PortsJson, &service.Ports)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (s *servicesRepo) Delete(ctx context.Context, id int64) error {
	return s.data.db.Delete(&biz.Service{}, id).Error
}

func (s *servicesRepo) GetWorkflow(ctx context.Context, id int64) (*biz.Workflow, error) {
	workflow := &biz.Workflow{}
	err := s.data.db.Where("id = ?", id).First(workflow).Error
	if err != nil {
		return nil, err
	}
	return workflow, nil
}

func (s *servicesRepo) SaveWrkflow(ctx context.Context, workflow *biz.Workflow) error {
	return s.data.db.Save(workflow).Error
}

func (s *servicesRepo) DeleteWrkflow(ctx context.Context, id int64) error {
	return s.data.db.Delete(&biz.Workflow{}, id).Error

}
