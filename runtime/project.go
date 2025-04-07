package runtime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
)

const (
	CloudProjectKind = "CloudProject"
)

type ProjectRuntime struct {
	log *log.Helper
}

func NewProjectRuntime(logger log.Logger) biz.ProjectRuntime {
	return &ProjectRuntime{
		log: log.NewHelper(logger),
	}
}

func (uc *ProjectRuntime) Reload(ctx context.Context, project *biz.Project) error {
	obj := NewUnstructured(CloudProjectKind)
	obj.SetName(project.Name)
	obj.SetNamespace(project.Namespace)
	SetSpec(obj, project)
	dynamicClient, err := GetKubeDynamicClient()
	if err != nil {
		return err
	}
	_, err = GetResource(ctx, dynamicClient, obj)
	if err != nil {
		if k8sErr.IsNotFound(err) {
			err = CreateResource(ctx, dynamicClient, obj)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		err = UpdateResource(ctx, dynamicClient, obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (uc *ProjectRuntime) Delete(ctx context.Context, project *biz.Project) error {
	obj := NewUnstructured(CloudProjectKind)
	obj.SetName(project.Name)
	obj.SetNamespace(project.Namespace)
	SetSpec(obj, project)
	dynamicClient, err := GetKubeDynamicClient()
	if err != nil {
		return err
	}
	_, err = GetResource(ctx, dynamicClient, obj)
	if err != nil {
		if k8sErr.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	} else {
		err = DeleteResource(ctx, dynamicClient, obj)
		if err != nil {
			return err
		}
	}
	return nil
}
