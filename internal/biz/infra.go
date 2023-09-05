package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// infrastructure entity
type Infra struct {
	ScriptPath       string `yaml:"script_path"`
	KubesprayVersion string `yaml:"kubespray_version"`
	KubesprayPath    string `yaml:"kubespray_path"`
	KubesprayPkgTag  string `yaml:"kubespary_package_tag"`
}

type GetInfraRepo interface {
	GetInfra(context.Context) (*Infra, error)
	SaveInfra(context.Context, *Infra) error
}

type InfraUsecase struct {
	repo GetInfraRepo
	log  *log.Helper
}

func NewInfraUseCase(repo GetInfraRepo, logger log.Logger) *InfraUsecase {
	return &InfraUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (i *InfraUsecase) GetInfra(ctx context.Context) (*Infra, error) {
	return i.repo.GetInfra(ctx)
}

func (i *InfraUsecase) SaveInfra(ctx context.Context, infra *Infra) error {
	return i.repo.SaveInfra(ctx, infra)
}
