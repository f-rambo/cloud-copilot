package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	projectApi "github.com/f-rambo/cloud-copilot/internal/repository/clusterruntime/api/project"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ClusterRuntimeProject struct {
	conf *conf.Bootstrap
	log  *log.Helper
}

func NewClusterRuntimeProject(conf *conf.Bootstrap, logger log.Logger) biz.ProjectRuntime {
	return &ClusterRuntimeProject{
		conf: conf,
		log:  log.NewHelper(logger),
	}
}

func (c *ClusterRuntimeProject) CreateNamespace(ctx context.Context, namespace string) error {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	_, err = projectApi.NewProjectInterfaceClient(grpconn.Conn).CreateNamespace(ctx, &projectApi.CreateNamespaceReq{
		Namespace: namespace,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *ClusterRuntimeProject) GetNamespaces(ctx context.Context) (namespaces []string, err error) {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return nil, err
	}
	defer grpconn.Close()
	res, err := projectApi.NewProjectInterfaceClient(grpconn.Conn).GetNamespaces(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	namespaces = res.Namespaces
	return nil, nil
}
