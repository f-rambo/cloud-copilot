package infrastructure

import (
	"context"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/f-rambo/cloud-copilot/infrastructure/alicloud"
	"github.com/f-rambo/cloud-copilot/infrastructure/awscloud"
	"github.com/f-rambo/cloud-copilot/infrastructure/baremetal"
	"github.com/f-rambo/cloud-copilot/infrastructure/common"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
)

type Infrastructure struct {
	baremetal *baremetal.Baremetal
	aliCloud  *alicloud.AliCloudUsecase
	awsCloud  *awscloud.AwsCloudUsecase
	log       *log.Helper
}

func NewInfrastructure(logger log.Logger) biz.ClusterInfrastructure {
	return &Infrastructure{
		log: log.NewHelper(logger),
	}
}

func (i *Infrastructure) GetRegions(ctx context.Context, cluster *biz.Cluster) error {
	if !cluster.Provider.IsCloud() {
		return nil
	}
	if cluster.Provider == biz.ClusterProvider_Aws {
		err := i.awsCloud.Connections(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.awsCloud.GetAvailabilityRegions(ctx, cluster)
		if err != nil {
			return err
		}
	}
	if cluster.Provider == biz.ClusterProvider_AliCloud {
		err := i.aliCloud.Connections(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.aliCloud.GetAvailabilityRegions(ctx, cluster)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Infrastructure) GetZones(ctx context.Context, cluster *biz.Cluster) error {
	if !cluster.Provider.IsCloud() {
		return nil
	}
	if cluster.Provider == biz.ClusterProvider_Aws {
		err := i.awsCloud.Connections(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.awsCloud.GetAvailabilityZones(ctx, cluster)
		if err != nil {
			return err
		}
	}
	if cluster.Provider == biz.ClusterProvider_AliCloud {
		err := i.aliCloud.Connections(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.aliCloud.GetAvailabilityZones(ctx, cluster)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Infrastructure) CreateCloudBasicResource(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.Provider == biz.ClusterProvider_Aws {
		err := i.awsCloud.Connections(ctx, cluster)
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
		err := i.aliCloud.Connections(ctx, cluster)
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
		err := i.awsCloud.Connections(ctx, cluster)
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
		err := i.aliCloud.Connections(ctx, cluster)
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
	if cluster.Provider == biz.ClusterProvider_Aws {
		err := i.awsCloud.Connections(ctx, cluster)
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
		err = i.awsCloud.ManageSLB(ctx, cluster)
		if err != nil {
			return err
		}
	}
	if cluster.Provider == biz.ClusterProvider_AliCloud {
		err := i.aliCloud.Connections(ctx, cluster)
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
		err = i.aliCloud.ManageSLB(ctx, cluster)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Infrastructure) GetNodesSystemInfo(ctx context.Context, cluster *biz.Cluster) error {
	if !cluster.Provider.IsCloud() {
		err := i.baremetal.GetNodesSystemInfo(ctx, cluster)
		if err != nil {
			return err
		}
		return nil
	}
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
			err := i.awsCloud.Connections(ctx, cluster)
			if err != nil {
				return err
			}
			image, err := i.awsCloud.FindImage(ctx, nodeGroup.Arch)
			if err != nil {
				return err
			}
			imageId = aws.ToString(image.ImageId)
			nodeUser = awscloud.DetermineUsername(aws.ToString(image.Name), aws.ToString(image.Description))
			systemDiskName = aws.ToString(image.RootDeviceName)
			instanceTypes, err := i.awsCloud.FindInstanceType(ctx, common.FindInstanceTypeParam{
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
			err := i.aliCloud.Connections(ctx, cluster)
			if err != nil {
				return err
			}
			image, err := i.aliCloud.FindImage(cluster.Region, nodeGroup.Arch)
			if err != nil {
				return err
			}
			imageId = tea.StringValue(image.ImageId)
			instanceTypes, err := i.aliCloud.FindInstanceType(common.FindInstanceTypeParam{
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
	err := i.openSsh(ctx, cluster)
	if err != nil {
		return err
	}
	err = i.baremetal.Install(ctx, cluster)
	if err != nil {
		return err
	}
	err = i.baremetal.ApplyServices(ctx, cluster)
	if err != nil {
		return err
	}
	err = i.closeSsh(ctx, cluster)
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

func (i *Infrastructure) openSsh(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.Provider == biz.ClusterProvider_AliCloud {
		err := i.aliCloud.Connections(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.aliCloud.OpenSSh(ctx, cluster)
		if err != nil {
			return err
		}
	}
	if cluster.Provider == biz.ClusterProvider_Aws {
		err := i.awsCloud.Connections(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.awsCloud.OpenSSh(ctx, cluster)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Infrastructure) closeSsh(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.Provider == biz.ClusterProvider_AliCloud {
		err := i.aliCloud.Connections(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.aliCloud.CloseSSh(ctx, cluster)
		if err != nil {
			return err
		}
	}
	if cluster.Provider == biz.ClusterProvider_Aws {
		err := i.awsCloud.Connections(ctx, cluster)
		if err != nil {
			return err
		}
		err = i.awsCloud.CloseSSh(ctx, cluster)
		if err != nil {
			return err
		}
	}
	return nil
}
