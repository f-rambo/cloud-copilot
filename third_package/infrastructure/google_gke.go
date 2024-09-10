package infrastructure

import (
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type GoogleGkeCluster struct {
	cluster *biz.Cluster
}

func GoogleGke(cluster *biz.Cluster) *GoogleGkeCluster {
	return &GoogleGkeCluster{
		cluster: cluster,
	}
}

func (g *GoogleGkeCluster) Start(ctx *pulumi.Context) error {
	return nil
}

func (g *GoogleGkeCluster) Clean(ctx *pulumi.Context) error {
	return nil
}

func (g *GoogleGkeCluster) Import(ctx *pulumi.Context) error {
	return nil
}
