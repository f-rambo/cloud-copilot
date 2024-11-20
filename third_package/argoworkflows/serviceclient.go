package argoworkflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

type WorkflowRepo struct {
	c   *conf.Bootstrap
	log *log.Helper
}

func NewWorkflowRepo(c *conf.Bootstrap, logger log.Logger) biz.WorkflowRuntime {
	return &WorkflowRepo{
		c:   c,
		log: log.NewHelper(logger),
	}
}

func (r *WorkflowRepo) GenerateCIWorkflow(ctx context.Context, service *biz.Service) (ciWf *biz.Workflow, cdwf *biz.Workflow, err error) {
	ciWorkflowJson, err := GetDefaultWorklfows(ctx, strings.ToLower(service.Business), strings.ToLower(service.Technology), biz.WorkflowTypeCI.String())
	if err != nil {
		return nil, nil, err
	}
	ciWf = &biz.Workflow{
		Name:     fmt.Sprintf("default-%s-%s-%s", service.Business, service.Technology, biz.WorkflowTypeCI.String()),
		Workflow: ciWorkflowJson,
	}
	cdWorkflowJson, err := GetDefaultWorklfows(ctx, strings.ToLower(service.Business), strings.ToLower(service.Technology), biz.WorkflowTypeCD.String())
	if err != nil {
		return nil, nil, err
	}
	cdwf = &biz.Workflow{
		Name:     fmt.Sprintf("default-%s-%s-%s", service.Business, service.Technology, biz.WorkflowTypeCD.String()),
		Workflow: cdWorkflowJson,
	}
	return ciWf, cdwf, nil
}

func (r *WorkflowRepo) Create(ctx context.Context, namespace string, workflow *biz.Workflow) error {
	kubeConf, err := getKubeConfig()
	if err != nil {
		return err
	}
	argoClient, err := newForConfig(kubeConf)
	if err != nil {
		return err
	}
	argoWf := &wfv1.Workflow{}
	err = json.Unmarshal(workflow.Workflow, argoWf)
	if err != nil {
		return err
	}
	_, err = argoClient.Workflows(namespace).Create(ctx, argoWf)
	if err != nil {
		return err
	}
	return nil
}
