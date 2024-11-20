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
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProjectInterface struct {
	v1alpha1.UnimplementedProjectServiceServer
	projectUc *biz.ProjectUsecase
	c         *conf.Bootstrap
	log       *log.Helper
}

func NewProjectInterface(uc *biz.ProjectUsecase, c *conf.Bootstrap, logger log.Logger) *ProjectInterface {
	return &ProjectInterface{projectUc: uc, c: c, log: log.NewHelper(logger)}
}

func (p *ProjectInterface) Ping(ctx context.Context, _ *emptypb.Empty) (*common.Msg, error) {
	return common.Response(), nil
}

func (p *ProjectInterface) Save(ctx context.Context, project *v1alpha1.Project) (*common.Msg, error) {
	if project.Name == "" {
		return nil, errors.New("project name is required")
	}
	if len(project.Business) == 0 {
		return nil, errors.New("business technology is required")
	}
	for _, v := range project.Business {
		if len(v.Technologys) == 0 {
			return nil, errors.New("technology type is required")
		}
	}
	notProjectNames := []string{"admin", "system", "public", "default", "kube", "kubernetes", "kube-public", "kube-system", "cloud-copilot"}
	if utils.Contains(notProjectNames, project.Name) {
		return nil, errors.New("project name is not allowed")
	}
	bizProject, err := p.projectTobizProject(project)
	if err != nil {
		return nil, err
	}
	err = p.projectUc.Save(ctx, bizProject)
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
	bizProjects, err := p.projectUc.List(ctx, projectReq.ClusterId)
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
	businessTechnologyMap := make(map[string][]string)
	for _, v := range bizProject.Business {
		technologyTypeArr := make([]string, 0)
		for _, v2 := range v.Technologys {
			technologyTypeArr = append(technologyTypeArr, v2.Name)
		}
		businessTechnologyMap[v.Name] = technologyTypeArr
	}
	businessTechnology := ""
	for name, technologyTypes := range businessTechnologyMap {
		businessTechnology += name + ":"
		for _, technologyType := range technologyTypes {
			businessTechnology += technologyType + ","
		}
		businessTechnology = businessTechnology[:len(businessTechnology)-1]
		businessTechnology += ";\n"
	}
	project.Id = bizProject.ID
	project.BusinessTechnology = businessTechnology
	return project, nil
}
