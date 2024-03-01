package interfaces

import (
	"context"
	"errors"

	"github.com/f-rambo/ocean/api/project/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProjectInterface struct {
	v1alpha1.UnimplementedProjectServiceServer
	uc  *biz.ProjectUsecase
	log *log.Helper
}

func NewProjectInterface(uc *biz.ProjectUsecase, logger log.Logger) *ProjectInterface {
	return &ProjectInterface{uc: uc, log: log.NewHelper(logger)}
}

func (p *ProjectInterface) Ping(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.Msg, error) {
	return &v1alpha1.Msg{}, nil
}

func (p *ProjectInterface) Save(ctx context.Context, project *v1alpha1.Project) (*v1alpha1.Msg, error) {
	if project.Name == "" {
		return nil, errors.New("project name is required")
	}
	bizProject, err := p.projectTobizProject(project)
	if err != nil {
		return nil, err
	}
	err = p.uc.Save(ctx, bizProject)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

func (p *ProjectInterface) Get(ctx context.Context, projectReq *v1alpha1.ProjectReq) (*v1alpha1.Project, error) {
	if projectReq.Id == 0 {
		return nil, errors.New("project id is required")
	}
	bizProject, err := p.uc.Get(ctx, projectReq.Id)
	if err != nil {
		return nil, err
	}
	if bizProject == nil {
		return nil, errors.New("project not found")
	}
	project, err := p.bizProjectToProject(bizProject)
	if err != nil {
		return nil, err
	}
	return project, nil
}

func (p *ProjectInterface) List(ctx context.Context, projectReq *v1alpha1.ProjectReq) (*v1alpha1.ProjectList, error) {
	if projectReq.ClusterId == 0 {
		return nil, errors.New("cluster id is required")
	}
	bizProjects, err := p.uc.List(ctx, projectReq.ClusterId)
	if err != nil {
		return nil, err
	}
	projects := make([]*v1alpha1.Project, 0)
	for _, bizProject := range bizProjects {
		project, err := p.bizProjectToProject(bizProject)
		if err != nil {
			return nil, err

		}
		projects = append(projects, project)
	}
	return &v1alpha1.ProjectList{Projects: projects}, nil
}

func (p *ProjectInterface) Delete(ctx context.Context, projectReq *v1alpha1.ProjectReq) (*v1alpha1.Msg, error) {
	if projectReq.Id == 0 {
		return nil, errors.New("project id is required")
	}
	err := p.uc.Delete(ctx, projectReq.Id)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

func (p *ProjectInterface) projectTobizProject(project *v1alpha1.Project) (*biz.Project, error) {
	bizProject := &biz.Project{}
	err := utils.StructTransform(project, bizProject)
	if err != nil {
		return nil, err
	}
	bizProject.ID = project.Id
	return bizProject, nil
}

func (p *ProjectInterface) bizProjectToProject(bizProject *biz.Project) (*v1alpha1.Project, error) {
	project := &v1alpha1.Project{}
	err := utils.StructTransform(bizProject, project)
	if err != nil {
		return nil, err
	}
	project.Id = bizProject.ID
	return project, nil
}
