package runtime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
)

const (
	CloudWorkflowKind = "CloudWorkflow"
)

type WorkflowRuntime struct {
	log *log.Helper
}

func NewWorkflowRuntime(logger log.Logger) biz.WorkflowRuntime {
	return &WorkflowRuntime{
		log: log.NewHelper(logger),
	}
}

func (w *WorkflowRuntime) CommitWorkflow(ctx context.Context, workflow *biz.Workflow) error {
	obj := NewUnstructuredWithGenerateName(CloudWorkflowKind, workflow.Name)
	obj.SetNamespace(workflow.Namespace)
	obj.SetLabels(biz.LablesToMap(workflow.Lables))
	SetSpec(obj, workflow)
	dynamicClient, err := GetKubeDynamicClient()
	if err != nil {
		return err
	}
	err = CreateResource(ctx, dynamicClient, obj)
	if err != nil {
		return err
	}
	workflow.Name = obj.GetName()
	return nil
}

func (w *WorkflowRuntime) GetWorkflowStatus(ctx context.Context, workflow *biz.Workflow) error {
	obj := NewUnstructured(CloudWorkflowKind)
	obj.SetName(workflow.Name)
	obj.SetNamespace(workflow.Namespace)
	dynamicClient, err := GetKubeDynamicClient()
	if err != nil {
		return err
	}
	obj, err = GetResource(ctx, dynamicClient, obj)
	if err != nil {
		return err
	}
	err = GetSpec(obj, workflow)
	if err != nil {
		return err
	}
	return nil
}

func (w *WorkflowRuntime) CleanWorkflow(ctx context.Context, workflow *biz.Workflow) error {
	obj := NewUnstructured(CloudWorkflowKind)
	obj.SetName(workflow.Name)
	obj.SetNamespace(workflow.Namespace)
	dynamicClient, err := GetKubeDynamicClient()
	if err != nil {
		return err
	}
	_, err = GetResource(ctx, dynamicClient, obj)
	if err != nil {
		if k8sErr.IsNotFound(err) {
			return nil
		}
		return err
	}
	err = DeleteResource(ctx, dynamicClient, obj)
	if err != nil {
		return err
	}
	return nil
}
