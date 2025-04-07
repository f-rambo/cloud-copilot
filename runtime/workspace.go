package runtime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
)

const (
	CloudWorkspaceKind = "CloudWorkspace"
)

type WorkspaceRuntime struct {
	log *log.Helper
}

func NewWorkspaceRuntime(logger log.Logger) biz.WorkspaceRuntime {
	return &WorkspaceRuntime{
		log: log.NewHelper(logger),
	}
}

func (uc *WorkspaceRuntime) Reload(ctx context.Context, wk *biz.Workspace) error {
	obj := NewUnstructured(CloudProjectKind)
	obj.SetName(wk.Name)
	obj.SetNamespace(biz.ClusterNamespace_cloudcopilot.String())
	SetSpec(obj, wk)
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

func (uc *WorkspaceRuntime) Delete(ctx context.Context, wk *biz.Workspace) error {
	obj := NewUnstructured(CloudProjectKind)
	obj.SetName(wk.Name)
	obj.SetNamespace(biz.ClusterNamespace_cloudcopilot.String())
	SetSpec(obj, wk)
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
