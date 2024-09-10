package infrastructure

import (
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type GoogleGcpCluster struct {
	cluster *biz.Cluster
}

func GoogleGcp(cluster *biz.Cluster) *GoogleGcpCluster {
	return &GoogleGcpCluster{
		cluster: cluster,
	}
}

func (g *GoogleGcpCluster) Start(ctx *pulumi.Context) error {
	return nil
}

func (g *GoogleGcpCluster) Clean(ctx *pulumi.Context) error {
	return nil
}

func (g *GoogleGcpCluster) Import(ctx *pulumi.Context) error {
	return nil
}
