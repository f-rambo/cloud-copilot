package infrastructure

import (
	"fmt"

	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/cs"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// alicloud managed kubernetes cluster
func (a *AlicloudCluster) Startkubernetes(ctx *pulumi.Context) error {
	if err := a.infrastructural(ctx); err != nil {
		return err
	}
	var nodeInstanceType string
	masterGetInstanceType, err := ecs.GetInstanceTypes(ctx, &ecs.GetInstanceTypesArgs{
		InstanceTypeFamily: pulumi.StringRef("ecs.c7"),
		CpuCoreCount:       pulumi.IntRef(4),
		MemorySize:         pulumi.Float64Ref(8),
	}, nil)
	if err != nil {
		return err
	}
	if len(masterGetInstanceType.InstanceTypes) == 0 {
		return fmt.Errorf("no available instance type found")
	}
	for i, v := range masterGetInstanceType.InstanceTypes {
		ctx.Export(fmt.Sprintf("instanceType-%d", i), pulumi.String(v.Id))
		nodeInstanceType = v.Id
		break
	}

	vSwitchIDs := make(pulumi.StringArray, 0)
	for _, v := range a.vSwitchs {
		vSwitchIDs = append(vSwitchIDs, v.ID())
	}
	// 创建cs kubernetes集群
	cluster, err := cs.NewManagedKubernetes(ctx, "managedKubernetesResource", &cs.ManagedKubernetesArgs{
		Name:             pulumi.String(a.cluster.Name),
		WorkerVswitchIds: vSwitchIDs,
		ClusterSpec:      pulumi.String("ack.pro.small"),
		ServiceCidr:      pulumi.String("172.16.0.0/16"),
		NewNatGateway:    pulumi.Bool(true),
		PodVswitchIds:    vSwitchIDs,
		ProxyMode:        pulumi.String("ipvs"),
		Addons: cs.ManagedKubernetesAddonArray{
			&cs.ManagedKubernetesAddonArgs{
				Name: pulumi.String("terway-eniip"),
			},
			&cs.ManagedKubernetesAddonArgs{
				Name: pulumi.String("csi-plugin"),
			},
			&cs.ManagedKubernetesAddonArgs{
				Name: pulumi.String("csi-provisioner"),
			},
		},
		ResourceGroupId: a.resourceGroupID,
	})
	if err != nil {
		return err
	}

	ctx.Export("clusterName", cluster.Name)
	ctx.Export("clusterId", cluster.ID().ToStringOutput())
	ctx.Export("Connections", cluster.Connections)
	ctx.Export("CertificateAuthority", cluster.CertificateAuthority)

	// 创建nodepool
	nodePool, err := cs.NewNodePool(ctx, "exampleNodePool", &cs.NodePoolArgs{
		NodePoolName:       pulumi.String("pulumi-nodepool-example"),
		ClusterId:          cluster.ID(),
		VswitchIds:         vSwitchIDs,
		SystemDiskCategory: pulumi.String("cloud_essd"),
		SystemDiskSize:     pulumi.Int(120),
		DesiredSize:        pulumi.Int(3),
		InstanceTypes:      pulumi.StringArray{pulumi.String(nodeInstanceType)},
		Management: &cs.NodePoolManagementArgs{
			Enable: pulumi.Bool(false),
		},
	})
	if err != nil {
		return err
	}

	// Export the NodePool ID
	ctx.Export("nodePoolID", nodePool.ID().ToStringOutput())

	return nil
}
