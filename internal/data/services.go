package data

import (
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
