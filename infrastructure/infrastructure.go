package infrastructure

import (
	"context"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/pkg/errors"
)

var ProviderSet = wire.NewSet(NewInfrastructure, NewBaremetal, NewAwsCloudUseCase, NewAliCloudUseCase)

type Infrastructure struct {
	c         *conf.Bootstrap
	baremetal *Baremetal
	aliCloud  *AliCloudUsecase
	awsCloud  *AwsCloudUsecase
	log       *log.Helper
}

func NewInfrastructure(c *conf.Bootstrap, baremetal *Baremetal, aliCloud *AliCloudUsecase, awsCloud *AwsCloudUsecase, logger log.Logger) biz.ClusterInfrastructure {
	return &Infrastructure{
		c:         c,
		baremetal: baremetal,
		aliCloud:  aliCloud,
		awsCloud:  awsCloud,
		log:       log.NewHelper(logger),
	}
}

func (i *Infrastructure) GetRegions(ctx context.Context, provider biz.ClusterProvider, accessId, accessKey string) ([]*biz.CloudResource, error) {
	if provider == biz.ClusterProvider_Aws {
		err := i.awsCloud.Connections(ctx, accessId, accessKey)
		if err != nil {
			return nil, err
		}
		return i.awsCloud.GetAvailabilityRegions(ctx)
	}
	if provider == biz.ClusterProvider_AliCloud {
		err := i.aliCloud.Connections(ctx, accessId, accessKey)
		if err != nil {
			return nil, err
		}
		return i.aliCloud.GetAvailabilityRegions(ctx)
	}
	return nil, errors.New("Not support")
}

func (i *Infrastructure) GetZones(ctx context.Context, cluster *biz.Cluster) (resrouces []*biz.CloudResource, err error) {
	if !cluster.Provider.IsCloud() {
		return
	}

	if cluster.Provider == biz.ClusterProvider_Aws {
		err = i.awsCloud.Connections(ctx, cluster.AccessId, cluster.AccessKey)
		if err != nil {
			return
		}
		resrouces, err = i.awsCloud.GetAvailabilityZones(ctx, cluster)
		if err != nil {
			return
		}
	}
	if cluster.Provider == biz.ClusterProvider_AliCloud {
		err = i.aliCloud.Connections(ctx, cluster.AccessId, cluster.AccessKey)
		if err != nil {
			return
		}
		resrouces, err = i.aliCloud.GetAvailabilityZones(ctx, cluster)
		if err != nil {
			return
		}
	}
	return
}

func (i *Infrastructure) ManageCloudBasicResource(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.Provider == biz.ClusterProvider_Aws {
		err := i.awsCloud.Connections(ctx, cluster.AccessId, cluster.AccessKey)
		if err != nil {
			return err
		}
		err = i.awsCloud.CreateNetwork(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.awsCloud.ImportKeyPair(ctx, cluster)
		if err != nil {
			return err
		}
	}
	if cluster.Provider == biz.ClusterProvider_AliCloud {
		err := i.aliCloud.Connections(ctx, cluster.AccessId, cluster.AccessKey)
		if err != nil {
			return err
		}
		err = i.aliCloud.CreateNetwork(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.aliCloud.ImportKeyPair(ctx, cluster)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Infrastructure) DeleteCloudBasicResource(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.Provider == biz.ClusterProvider_Aws {
		err := i.awsCloud.Connections(ctx, cluster.AccessId, cluster.AccessKey)
		if err != nil {
			return err
		}
		err = i.awsCloud.DeleteNetwork(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.awsCloud.DeleteKeyPair(ctx, cluster)
		if err != nil {
			return err
		}
	}
	if cluster.Provider == biz.ClusterProvider_AliCloud {
		err := i.aliCloud.Connections(ctx, cluster.AccessId, cluster.AccessKey)
		if err != nil {
			return err
		}
		err = i.aliCloud.DeleteNetwork(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.aliCloud.DeleteKeyPair(ctx, cluster)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Infrastructure) ManageNodeResource(ctx context.Context, cluster *biz.Cluster) error {
	// Slb resource from cloud and metallb
	if cluster.Provider == biz.ClusterProvider_Aws {
		err := i.awsCloud.Connections(ctx, cluster.AccessId, cluster.AccessKey)
		if err != nil {
			return err
		}
		err = i.awsCloud.ManageSecurityGroup(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.awsCloud.ManageInstance(ctx, cluster)
		if err != nil {
			return err
		}
	}
	if cluster.Provider == biz.ClusterProvider_AliCloud {
		err := i.aliCloud.Connections(ctx, cluster.AccessId, cluster.AccessKey)
		if err != nil {
			return err
		}
		err = i.aliCloud.ManageSecurityGroup(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.aliCloud.ManageInstance(ctx, cluster)
		if err != nil {
			return err
		}
	}
	if cluster.Provider == biz.ClusterProvider_BareMetal {
		err := i.baremetal.PreInstall(cluster)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Infrastructure) GetNodesSystemInfo(ctx context.Context, cluster *biz.Cluster) error {
	if !cluster.Provider.IsCloud() {
		return i.baremetal.GetNodesSystemInfo(ctx, cluster)
	}
	return i.GetCloudtNodesSystemInfo(ctx, cluster)
}

func (i *Infrastructure) GetCloudtNodesSystemInfo(ctx context.Context, cluster *biz.Cluster) error {
	for _, nodeGroup := range cluster.NodeGroups {
		isFindNode := false
		for _, node := range cluster.Nodes {
			if node.NodeGroupId != nodeGroup.Id {
				continue
			}
			if node.Status == biz.NodeStatus_NODE_FINDING {
				isFindNode = true
				break
			}
		}
		if !isFindNode {
			continue
		}
		imageId := ""
		systemDiskName := ""
		instanceTypeId := ""
		nodeUser := "root"
		backupInstanceTypeIds := make([]string, 0)
		if cluster.Provider == biz.ClusterProvider_Aws {
			err := i.awsCloud.Connections(ctx, cluster.AccessId, cluster.AccessKey)
			if err != nil {
				return err
			}
			image, err := i.awsCloud.FindImage(ctx, nodeGroup.Arch)
			if err != nil {
				return err
			}
			imageId = aws.ToString(image.ImageId)
			nodeUser = AwsDetermineUsername(aws.ToString(image.Name), aws.ToString(image.Description))
			systemDiskName = aws.ToString(image.RootDeviceName)
			instanceTypes, err := i.awsCloud.FindInstanceType(ctx, FindInstanceTypeParam{
				Os:            nodeGroup.Os,
				CPU:           nodeGroup.Cpu,
				Memory:        nodeGroup.Memory,
				Arch:          nodeGroup.Arch,
				GPU:           nodeGroup.Gpu,
				GPUSpec:       nodeGroup.GpuSpec,
				NodeGroupType: nodeGroup.Type,
			})
			if err != nil {
				return err
			}
			for _, v := range instanceTypes {
				memSize := int32(aws.ToInt64(v.MemoryInfo.SizeInMiB) / 1024)
				if nodeGroup.Memory != memSize {
					nodeGroup.Memory = memSize
				}
				if instanceTypeId == "" {
					instanceTypeId = string(v.InstanceType)
					continue
				}
				backupInstanceTypeIds = append(backupInstanceTypeIds, string(v.InstanceType))
			}
		}
		if cluster.Provider == biz.ClusterProvider_AliCloud {
			err := i.aliCloud.Connections(ctx, cluster.AccessId, cluster.AccessKey)
			if err != nil {
				return err
			}
			image, err := i.aliCloud.FindImage(cluster.Region, nodeGroup.Arch)
			if err != nil {
				return err
			}
			imageId = tea.StringValue(image.ImageId)
			instanceTypes, err := i.aliCloud.FindInstanceType(FindInstanceTypeParam{
				Os:            nodeGroup.Os,
				CPU:           nodeGroup.Cpu,
				Memory:        nodeGroup.Memory,
				Arch:          nodeGroup.Arch,
				GPU:           nodeGroup.Gpu,
				GPUSpec:       nodeGroup.GpuSpec,
				NodeGroupType: nodeGroup.Type,
			})
			if err != nil {
				return err
			}
			for _, v := range instanceTypes {
				if nodeGroup.Memory != int32(tea.Float32Value(v.MemorySize)) {
					nodeGroup.Memory = int32(tea.Float32Value(v.MemorySize))
				}
				if instanceTypeId == "" {
					instanceTypeId = tea.StringValue(v.InstanceTypeId)
					continue
				}
				backupInstanceTypeIds = append(backupInstanceTypeIds, tea.StringValue(v.InstanceTypeId))
			}
		}
		for _, node := range cluster.Nodes {
			if node.NodeGroupId != nodeGroup.Id {
				continue
			}
			node.User = nodeUser
			node.ImageId = imageId
			node.SystemDiskName = systemDiskName
			node.InstanceType = instanceTypeId
			node.BackupInstanceIds = strings.Join(backupInstanceTypeIds, ",")
		}
	}
	return nil
}

func (i *Infrastructure) Install(ctx context.Context, cluster *biz.Cluster) error {
	err := i.baremetal.Install(ctx, cluster)
	if err != nil {
		return err
	}
	err = i.baremetal.ApplyCloudCopilot(ctx, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (i *Infrastructure) UnInstall(_ context.Context, cluster *biz.Cluster) error {
	return i.baremetal.UnInstall(cluster)
}

func (i *Infrastructure) HandlerNodes(_ context.Context, cluster *biz.Cluster) error {
	return i.baremetal.HandlerNodes(cluster)
}
