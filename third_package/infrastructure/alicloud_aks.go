package infrastructure

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/cs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	alicloudKubernetesClusterName = "alicloud-kubernetes-cluster"
	alicloudNodePoolName          = "alicloud-node-pool"
)

func (a *Alicloud) StartAks(ctx *pulumi.Context) (err error) {
	err = a.infrastructural(ctx)
	if err != nil {
		return errors.Wrap(err, "infrastructural failed")
	}

	err = a.getManagedKubernetes(ctx)
	if err != nil {
		return errors.Wrap(err, "getManagedKubernetes failed")
	}
	vSwitchIDs := make(pulumi.StringArray, 0)
	for _, v := range a.vSwitchs {
		vSwitchIDs = append(vSwitchIDs, v.ID())
	}
	var cluster *cs.ManagedKubernetes
	instanceChargeType := "PostPaid"
	if a.cluster.CloudID != "" {
		clusters, err := cs.GetManagedKubernetesClusters(ctx, &cs.GetManagedKubernetesClustersArgs{
			Ids: []string{a.cluster.CloudID},
		})
		if err != nil {
			return errors.Wrap(err, "getManagedKubernetesClusters failed")
		}
		if len(clusters.Clusters) == 0 {
			return fmt.Errorf("cluster not found")
		}
		clusterRes := clusters.Clusters[0]
		for _, nodegroup := range a.cluster.NodeGroups {
			nodegroup.Image = clusterRes.ImageId
			if len(clusterRes.WorkerInstanceTypes) > 0 {
				nodegroup.InstanceType = clusterRes.WorkerInstanceTypes[0]
			}
			if clusterRes.WorkerInstanceChargeType != "" {
				instanceChargeType = clusterRes.WorkerInstanceChargeType
			}
		}
		cluster, err = cs.NewManagedKubernetes(ctx, alicloudKubernetesClusterName, &cs.ManagedKubernetesArgs{
			Name:             pulumi.String(clusterRes.Name),
			WorkerVswitchIds: vSwitchIDs,
		}, pulumi.Import(pulumi.ID(clusterRes.Id)))
		if err != nil {
			return err
		}
	} else {
		// create cluster
		cluster, err = cs.NewManagedKubernetes(ctx, alicloudKubernetesClusterName, &cs.ManagedKubernetesArgs{
			Name:               pulumi.String(a.cluster.Name),
			Version:            pulumi.String(fmt.Sprintf("%s-aliyun.1", a.cluster.Version)),
			WorkerVswitchIds:   vSwitchIDs,
			ClusterSpec:        pulumi.String("ack.pro.small"),
			ServiceCidr:        pulumi.String(a.cluster.VpcCidr),
			NewNatGateway:      pulumi.Bool(true),
			PodVswitchIds:      vSwitchIDs,
			LoadBalancerSpec:   pulumi.String("slb.s1.small"),
			ProxyMode:          pulumi.String("ipvs"),
			SlbInternetEnabled: pulumi.Bool(true),
			EnableRrsa:         pulumi.Bool(true),
			Addons: cs.ManagedKubernetesAddonArray{
				&cs.ManagedKubernetesAddonArgs{
					Name:    pulumi.String("terway-eniip"),
					Version: pulumi.String("3.1.0-aliyun.1"),
				},
				&cs.ManagedKubernetesAddonArgs{
					Name:    pulumi.String("csi-plugin"),
					Version: pulumi.String("1.22.0-aliyun.1"),
				},
				&cs.ManagedKubernetesAddonArgs{
					Name:    pulumi.String("csi-provisioner"),
					Version: pulumi.String("1.22.0-aliyun.1"),
				},
			},
			ResourceGroupId: a.resourceGroup.ID(),
		})
		if err != nil {
			return err
		}
	}

	// 需要倒入nodepool的话，手动填写nodepool的cloudID
	for _, nodeGroup := range a.cluster.NodeGroups {
		nodepoolArgs := &cs.NodePoolArgs{
			NodePoolName:       pulumi.String(nodeGroup.Name),
			ClusterId:          cluster.ID(),
			VswitchIds:         vSwitchIDs,
			ImageId:            pulumi.String(nodeGroup.Image),
			InstanceTypes:      pulumi.StringArray{pulumi.String(nodeGroup.InstanceType)},
			InstanceChargeType: pulumi.String(instanceChargeType),
			RuntimeName:        pulumi.String("containerd"),
			RuntimeVersion:     pulumi.String("1.6.28"),
			DesiredSize:        pulumi.Int(nodeGroup.TargetSize),
			KeyName:            pulumi.String(alicloudKeyPairName),
			SystemDiskCategory: pulumi.String("cloud_efficiency"),
			SystemDiskSize:     pulumi.Int(nodeGroup.SystemDisk),
		}
		if nodeGroup.DataDisk > 0 {
			nodepoolArgs.DataDisks = &cs.NodePoolDataDiskArray{
				&cs.NodePoolDataDiskArgs{
					Category: pulumi.String("cloud_essd"),
					Size:     pulumi.Int(nodeGroup.DataDisk),
				},
			}
		}
		nodepool, err := cs.NewNodePool(ctx, fmt.Sprintf("%s-%s", alicloudNodePoolName, nodeGroup.Name), nodepoolArgs)
		if err != nil {
			return err
		}
		ctx.Export(getCloudNodeGroupID(nodeGroup.Name), nodepool.ID())
	}
	ctx.Export(getClusterCloudID(), cluster.ID())
	ctx.Export(getConnections(), cluster.Connections)
	ctx.Export(getCertificateAuthority(), cluster.CertificateAuthority)

	return nil
}

func (a *Alicloud) getManagedKubernetes(ctx *pulumi.Context) error {
	cluster, err := cs.GetManagedKubernetesClusters(ctx, &cs.GetManagedKubernetesClustersArgs{
		NameRegex: pulumi.StringRef(a.cluster.Name),
	})
	if err != nil {
		return err
	}
	if len(cluster.Clusters) == 0 {
		return fmt.Errorf("cluster not found")
	}
	a.cluster.CloudID = cluster.Clusters[0].Id
	return nil
}
