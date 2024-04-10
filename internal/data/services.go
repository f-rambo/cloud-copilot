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
	err := s.data.db.Model(&biz.Service{}).Where(serviceParam).Count(&itemCount).Error
	if err != nil {
		return nil, 0, err
	}
	if itemCount == 0 {
		return services, 0, nil
	}
	err = s.data.db.Where(serviceParam).Offset((page - 1) * pageSize).Limit(pageSize).Find(&services).Error
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
