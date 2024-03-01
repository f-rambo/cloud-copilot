package interfaces

import (
	v1alpha1 "github.com/f-rambo/ocean/api/service/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
)

type ServicesInterface struct {
	v1alpha1.UnimplementedServiceInterfaceServer
	uc *biz.ServicesUseCase
}

func NewServicesInterface(uc *biz.ServicesUseCase) *ServicesInterface {
	return &ServicesInterface{uc: uc}
}
