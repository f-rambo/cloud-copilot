package data

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/utils"
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

func (s *servicesRepo) k8s() error {
	if s.data.k8sClient != nil {
		return nil
	}
	return s.data.newKubernetes()
}

func (s *servicesRepo) Save(ctx context.Context, svc *biz.Service) error {
	return nil
}

func (s *servicesRepo) Get(ctx context.Context, id int) (*biz.Service, error) {
	return nil, nil
}

func (s *servicesRepo) GetServices(ctx context.Context) ([]*biz.Service, error) {
	// gorm get services
	services := make([]*biz.Service, 0)
	err := s.data.db.Find(&services).Error
	if err != nil {
		return nil, err
	}
	return services, nil
}

func (s *servicesRepo) Delete(ctx context.Context, svc *biz.Service) error {
	return nil
}

func (s *servicesRepo) SaveCI(ctx context.Context, ci *biz.CI) error {
	return nil
}

func (s *servicesRepo) CreateWf(ctx context.Context, svc *biz.Service, ci *biz.CI) error {
	return nil
}

func (s *servicesRepo) GetCI(ctx context.Context, id int) (*biz.CI, error) {
	return nil, nil
}

func (s *servicesRepo) GetWf(ctx context.Context, svc *biz.Service, ci *biz.CI) (*wfv1.Workflow, error) {
	return nil, nil
}

func (s *servicesRepo) GetCIs(ctx context.Context) ([]*biz.CI, error) {
	return nil, nil
}

func (s *servicesRepo) Deploy(ctx context.Context, svc *biz.Service, ci *biz.CI) error {
	return nil
}

func (s *servicesRepo) GetOceanService(ctx context.Context) (*biz.Service, error) {
	wf, err := utils.GetDefaultWorkflow()
	if err != nil {
		return nil, err
	}
	service := &biz.Service{
		Name:         "ocean",
		Repo:         "https://github.com/f-rambo/ocean.git",
		Registry:     "https://docker.io/frambos",
		RegistryUser: "",
		RegistryPwd:  "",
	}
	ci := biz.CI{
		Version:     "0.0.1",
		Branch:      "master",
		Tag:         "latest",
		Args:        map[string]string{},
		Description: "example",
		ServiceID:   service.ID,
	}
	s.wfAssign(ctx, wf, service, &ci)
	service.Workflow = wf
	service.CIItems = append(service.CIItems, ci)
	return service, nil
}

func (s *servicesRepo) wfAssign(ctx context.Context, wf *wfv1.Workflow, svc *biz.Service, ci *biz.CI) {
	for i, param := range wf.Spec.Arguments.Parameters {
		switch param.Name {
		case "name":
			wf.Spec.Arguments.Parameters[i].Value = wfv1.AnyStringPtr(svc.Name)
		case "repo":
			wf.Spec.Arguments.Parameters[i].Value = wfv1.AnyStringPtr(svc.Repo)
		case "registry":
			wf.Spec.Arguments.Parameters[i].Value = wfv1.AnyStringPtr(svc.Registry)
		case "registry_user":
			wf.Spec.Arguments.Parameters[i].Value = wfv1.AnyStringPtr(svc.RegistryUser)
		case "registry_pwd":
			wf.Spec.Arguments.Parameters[i].Value = wfv1.AnyStringPtr(svc.RegistryPwd)
		case "version":
			wf.Spec.Arguments.Parameters[i].Value = wfv1.AnyStringPtr(ci.Version)
		case "branch":
			wf.Spec.Arguments.Parameters[i].Value = wfv1.AnyStringPtr(ci.Branch)
		case "tag":
			wf.Spec.Arguments.Parameters[i].Value = wfv1.AnyStringPtr(ci.Tag)
		default:
			if val, ok := ci.Args[param.Name]; ok {
				wf.Spec.Arguments.Parameters[i].Value = wfv1.AnyStringPtr(val)
			}
		}
	}
}
