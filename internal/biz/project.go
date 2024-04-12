package biz

import (
	"context"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/pkg/kubeclient"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Project struct {
	ID           int64      `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name         string     `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Namespace    string     `json:"namespace" gorm:"column:namespace; default:''; NOT NULL"`
	State        string     `json:"state" gorm:"column:state; default:''; NOT NULL"`
	Description  string     `json:"description" gorm:"column:description; default:''; NOT NULL"`
	ClusterID    int64      `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
	Business     []Business `json:"business" gorm:"-"`
	BusinessJson []byte     `json:"business_json" gorm:"column:business_json; type:json"`
	gorm.Model
}

const (
	ProjectStateInit    = "init"
	ProjectStateRunning = "running"
	ProjectStateStopped = "stopped"
)

type Business struct {
	Name        string       `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Technologys []Technology `json:"technologys" gorm:"-"`
}

type Technology struct {
	Name string `json:"name" gorm:"column:name; default:''; NOT NULL"`
}

func (p *Project) initPorject() {
	p.State = ProjectStateInit
	p.Namespace = p.Name
}

type ProjectRepo interface {
	Save(context.Context, *Project) error
	Get(context.Context, int64) (*Project, error)
	List(context.Context, int64) ([]*Project, error)
	ListByIds(context.Context, []int64) ([]*Project, error)
	Delete(context.Context, int64) error
}

type ProjectUsecase struct {
	repo ProjectRepo
	log  *log.Helper
	c    *conf.Bootstrap
}

func NewProjectUseCase(repo ProjectRepo, logger log.Logger, c *conf.Bootstrap) *ProjectUsecase {
	return &ProjectUsecase{repo: repo, log: log.NewHelper(logger), c: c}
}

func (uc *ProjectUsecase) Save(ctx context.Context, projectParam *Project) error {
	if projectParam.ID == 0 {
		// check name exists
		projects, err := uc.List(ctx, projectParam.ClusterID)
		if err != nil {
			return err
		}
		for _, v := range projects {
			if v.Name == projectParam.Name {
				return errors.New("name exists")
			}
		}
		projectParam.initPorject()
		kubeClientSet, err := kubeclient.GetKubeClientSet()
		if err != nil {
			return err
		}
		namespaceOk, err := kubeclient.NamespaceExists(ctx, kubeClientSet, projectParam.Namespace)
		if err != nil {
			return err
		}
		if namespaceOk {
			return errors.New("namespace exists")
		}
		return uc.repo.Save(ctx, projectParam)
	}
	project, err := uc.Get(ctx, projectParam.ID)
	if err != nil {
		return err
	}
	projectParam.Namespace = project.Namespace
	projectParam.State = project.State
	projectParam.ClusterID = project.ClusterID
	return uc.repo.Save(ctx, projectParam)
}

func (uc *ProjectUsecase) Get(ctx context.Context, id int64) (*Project, error) {
	return uc.repo.Get(ctx, id)
}

func (uc *ProjectUsecase) List(ctx context.Context, clusterID int64) ([]*Project, error) {
	return uc.repo.List(ctx, clusterID)
}

func (uc *ProjectUsecase) ListByIds(ctx context.Context, ids []int64) ([]*Project, error) {
	return uc.repo.ListByIds(ctx, ids)
}

func (uc *ProjectUsecase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *ProjectUsecase) Enable(ctx context.Context, project *Project, cluster *Cluster, baseAppInstallation func(context.Context, *Cluster, *Project) error) error {
	// crate namespace
	kubeClientSet, err := kubeclient.GetKubeClientSet()
	if err != nil {
		return err
	}
	namespaceOk, err := kubeclient.NamespaceExists(ctx, kubeClientSet, project.Namespace)
	if err != nil {
		return err
	}
	if !namespaceOk {
		err = kubeclient.CreateNamespace(ctx, kubeClientSet, project.Namespace)
		if err != nil {
			return err
		}
	}
	// todo create service account
	// todo create role
	// todo create rolebinding
	err = baseAppInstallation(ctx, cluster, project)
	if err != nil {
		return err
	}
	project.State = ProjectStateRunning
	return uc.Save(ctx, project)
}
