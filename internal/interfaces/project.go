package interfaces

import (
	"context"
	"errors"

	"github.com/f-rambo/cloud-copilot/api/common"
	"github.com/f-rambo/cloud-copilot/api/project/v1alpha1"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
)

type ProjectInterface struct {
	v1alpha1.UnimplementedProjectServiceServer
	projectUc *biz.ProjectUsecase
	userUc    *biz.UserUseCase
	c         *conf.Bootstrap
	log       *log.Helper
}

func NewProjectInterface(uc *biz.ProjectUsecase, userUc *biz.UserUseCase, c *conf.Bootstrap, logger log.Logger) *ProjectInterface {
	return &ProjectInterface{projectUc: uc, userUc: userUc, c: c, log: log.NewHelper(logger)}
}

func (p *ProjectInterface) GetProject(ctx context.Context, projectId int64) (*biz.Project, error) {
	return p.projectUc.Get(ctx, projectId)
}

func (p *ProjectInterface) Save(ctx context.Context, project *v1alpha1.Project) (*common.Msg, error) {
	if project.Name == "" {
		return nil, errors.New("project name is required")
	}
	if !utils.IsValidKubernetesName(project.Name) {
		return nil, errors.New("project name is invalid")
	}
	if project.Id == 0 {
		projectData, err := p.projectUc.GetByName(ctx, project.Name)
		if err != nil {
			return nil, err
		}
		if projectData != nil && projectData.Id > 0 {
			return nil, errors.New("project name already exists")
		}
	}
	bizProject := p.projectTobizProject(project)
	err := p.projectUc.Save(ctx, bizProject)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (p *ProjectInterface) Get(ctx context.Context, projectReq *v1alpha1.ProjectDetailRequest) (*v1alpha1.Project, error) {
	if projectReq.Id == 0 {
		return nil, errors.New("project id is required")
	}
	bizProject, err := p.projectUc.Get(ctx, int64(projectReq.Id))
	if err != nil {
		return nil, err
	}
	if bizProject == nil {
		return nil, errors.New("project not found")
	}
	project := p.bizProjectToProject(bizProject)
	return project, nil
}

func (p *ProjectInterface) List(ctx context.Context, projectsReq *v1alpha1.ProjectsReqquest) (*v1alpha1.Projects, error) {
	bizProjects, total, err := p.projectUc.List(ctx, projectsReq.Name, projectsReq.Page, projectsReq.Size)
	if err != nil {
		return nil, err
	}
	projects := make([]*v1alpha1.Project, 0)
	for _, bizProject := range bizProjects {
		projects = append(projects, p.bizProjectToProject(bizProject))
	}
	return &v1alpha1.Projects{Projects: projects, Total: total}, nil
}

func (p *ProjectInterface) Delete(ctx context.Context, projectReq *v1alpha1.ProjectDetailRequest) (*common.Msg, error) {
	if projectReq.Id == 0 {
		return nil, errors.New("project id is required")
	}
	err := p.projectUc.Delete(ctx, int64(projectReq.Id))
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (p *ProjectInterface) projectTobizProject(project *v1alpha1.Project) *biz.Project {
	return &biz.Project{
		Id:            int64(project.Id),
		Name:          project.Name,
		Description:   project.Description,
		WorkspaceId:   int64(project.WorkspaceId),
		UserId:        int64(project.UserId),
		ResourceQuota: resourceQuotaInterfaceToBiz(project.ResourceQuota),
	}
}

func (p *ProjectInterface) bizProjectToProject(bizProject *biz.Project) *v1alpha1.Project {
	return &v1alpha1.Project{
		Id:            int32(bizProject.Id),
		Name:          bizProject.Name,
		Description:   bizProject.Description,
		WorkspaceId:   int32(bizProject.WorkspaceId),
		UserId:        int32(bizProject.UserId),
		ResourceQuota: resourceQuotaBizToInterface(bizProject.ResourceQuota),
	}
}
