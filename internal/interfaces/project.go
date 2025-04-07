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

func (p *ProjectInterface) Get(ctx context.Context, projectReq *v1alpha1.ProjectReq) (*v1alpha1.Project, error) {
	if projectReq.Id == 0 {
		return nil, errors.New("project id is required")
	}
	bizProject, err := p.projectUc.Get(ctx, projectReq.Id)
	if err != nil {
		return nil, err
	}
	if bizProject == nil {
		return nil, errors.New("project not found")
	}
	project := p.bizProjectToProject(bizProject)
	user, err := p.userUc.GetUser(ctx, bizProject.UserId)
	if err != nil {
		return nil, err
	}
	project.UserName = user.Name
	return project, nil
}

func (p *ProjectInterface) List(ctx context.Context, projectReq *v1alpha1.ProjectReq) (*v1alpha1.ProjectList, error) {
	if projectReq.ClusterId == 0 {
		return nil, errors.New("cluster id is required")
	}
	bizProjects, err := p.projectUc.List(ctx, projectReq.ClusterId)
	if err != nil {
		return nil, err
	}
	projects := make([]*v1alpha1.Project, 0)
	userIds := make([]int64, 0)
	for _, bizProject := range bizProjects {
		project := p.bizProjectToProject(bizProject)
		projects = append(projects, project)
		userIds = append(userIds, bizProject.UserId)
	}
	userIds = utils.RemoveDuplicatesInt64(userIds)
	users, err := p.userUc.GetUserByBatchID(ctx, userIds)
	if err != nil {
		return nil, err
	}
	for _, v := range users {
		for _, project := range projects {
			if project.UserId == v.Id {
				project.UserName = v.Name
			}
		}
	}
	return &v1alpha1.ProjectList{Projects: projects}, nil
}

func (p *ProjectInterface) Delete(ctx context.Context, projectReq *v1alpha1.ProjectReq) (*common.Msg, error) {
	if projectReq.Id == 0 {
		return nil, errors.New("project id is required")
	}
	err := p.projectUc.Delete(ctx, projectReq.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (p *ProjectInterface) projectTobizProject(project *v1alpha1.Project) *biz.Project {
	return &biz.Project{
		Id:          project.Id,
		Name:        project.Name,
		Description: project.Description,
		ClusterId:   project.ClusterId,
		UserId:      project.UserId,
		WorkspaceId: project.WorkspaceId,
		LimitCpu:    project.LimitCpu,
		LimitGpu:    project.LimitGpu,
		LimitMemory: project.LimitMemory,
		LimitDisk:   project.LimitDisk,
	}
}

func (p *ProjectInterface) bizProjectToProject(bizProject *biz.Project) *v1alpha1.Project {
	return &v1alpha1.Project{
		Id:          bizProject.Id,
		Name:        bizProject.Name,
		Description: bizProject.Description,
		ClusterId:   bizProject.ClusterId,
		UserId:      bizProject.UserId,
		WorkspaceId: bizProject.WorkspaceId,
		LimitCpu:    bizProject.LimitCpu,
		LimitGpu:    bizProject.LimitGpu,
		LimitMemory: bizProject.LimitMemory,
		LimitDisk:   bizProject.LimitDisk,
	}
}
