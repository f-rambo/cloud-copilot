package pulumi

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/ecs"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/ram"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/resourcemanager"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/vpc"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/spf13/cast"
)

const (
	AlicloudProjectName = "ocean-project"
	AlicloudStackName   = "ocean-stack"
)

const (
	resourceGroupName = "ocean-resource-group"
	roleName          = "ocean-cluster-role"
	csPolicy          = `{"Statement":[{"Action":"sts:AssumeRole","Effect":"Allow","Principal":{"Service":["cs.aliyuncs.com"]}}],"Version":"1"}`
	escPolicy         = `{"Statement":[{"Action":"sts:AssumeRole","Effect":"Allow","Principal":{"Service":["ecs.aliyuncs.com"]}}],"Version":"1"}`
	aliVpcName        = "ocean-vpc"
	aliVswitchName    = "ocean-vswitch"
	ecsSecurityGroup  = "ocean-ecs-security-group"
	bostionHostName   = "ocean-bostion"
	bostionEipName    = "ocean-bostion-eip"
	natGatewayName    = "ocean-nat-gateway"
	natEipName        = "ocean-nat-eip"
	keyPairName       = "ocean-key-pair"
)

type GetInstanceTypesInstanceTypes []ecs.GetInstanceTypesInstanceType

// sort by cpu core and memory size
func (a GetInstanceTypesInstanceTypes) Len() int {
	return len(a)
}

func (a GetInstanceTypesInstanceTypes) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a GetInstanceTypesInstanceTypes) Less(i, j int) bool {
	if a[i].CpuCoreCount == a[j].CpuCoreCount {
		if a[i].MemorySize == a[j].MemorySize {
			return cast.ToInt32(a[i].Gpu.Amount) < cast.ToInt32(a[j].Gpu.Amount)
		}
		return a[i].MemorySize < a[j].MemorySize
	}
	return a[i].CpuCoreCount < a[j].CpuCoreCount
}

type AlicloudCluster struct {
	cluster         *biz.Cluster
	resourceGroupID pulumi.StringInput
	vpcID           pulumi.StringInput
	vSwitchs        []*vpc.Switch
	sgID            pulumi.StringInput
	eipID           pulumi.StringInput
}

func StartAlicloudCluster(cluster *biz.Cluster) *AlicloudCluster {
	return &AlicloudCluster{
		cluster: cluster,
	}
}

func (a *AlicloudCluster) getIntanceTypeFamilies(nodeGroup *biz.NodeGroup) string {
	if nodeGroup == nil {
		return "ecs.g6"
	}
	switch nodeGroup.Type {
	case biz.NodeGroupTypeNormal:
		return "ecs.g6"
	case biz.NodeGroupTypeHighComputation:
		return "ecs.c6"
	case biz.NodeGroupTypeGPUAcceleraterd:
		return "ecs.gn6i"
	case biz.NodeGroupTypeHighMemory:
		return "ecs.r6"
	case biz.NodeGroupTypeLargeHardDisk:
		return "ecs.g6" // 支持挂载大磁盘
	default:
		return "ecs.g6"
	}
}

func (a *AlicloudCluster) StartServers(ctx *pulumi.Context) error {
	err := a.infrastructural(ctx)
	if err != nil {
		return errors.Wrap(err, "alicloud cluster init failed")
	}
	err = a.bostionHost(ctx)
	if err != nil {
		return errors.Wrap(err, "start bostion host failed")
	}
	err = a.nodes(ctx)
	if err != nil {
		return errors.Wrap(err, "start ecs failed")
	}
	err = a.localBalancer(ctx)
	if err != nil {
		return errors.Wrap(err, "start local balancer failed")
	}
	return nil
}

func (a *AlicloudCluster) infrastructural(ctx *pulumi.Context) error {
	// 创建资源组
	res, err := resourcemanager.NewResourceGroup(ctx, resourceGroupName, &resourcemanager.ResourceGroupArgs{
		ResourceGroupName: pulumi.String(resourceGroupName),
		DisplayName:       pulumi.String(resourceGroupName),
	})
	if err != nil {
		return err
	}
	a.resourceGroupID = res.ID()

	// 创建角色/策略
	roleMap := map[string]string{
		"csPolicy":  csPolicy,
		"ecsPolicy": escPolicy,
	}
	for name, rolePolicy := range roleMap {
		_, err := ram.NewRole(ctx, name, &ram.RoleArgs{
			Name:        pulumi.String(name),
			Document:    pulumi.String(rolePolicy),
			Description: pulumi.String("ocean cluster role."),
		})
		if err != nil {
			return err
		}
	}

	// vpc
	network, err := vpc.NewNetwork(ctx, aliVpcName, &vpc.NetworkArgs{
		VpcName:   pulumi.String(aliVpcName),
		CidrBlock: pulumi.String("192.168.0.0/16"),
	})
	if err != nil {
		return err
	}
	a.vpcID = network.ID()
	ctx.Export("vpc_id", a.vpcID)

	// 创建交换机
	foo, err := alicloud.GetZones(ctx, &alicloud.GetZonesArgs{
		AvailableResourceCreation: pulumi.StringRef("VSwitch"),
	}, nil)
	if err != nil {
		return err
	}
	zoneIds := make([]string, 0)
	for i, zone := range foo.Zones {
		zoneIds = append(zoneIds, zone.Id)
		ctx.Export(fmt.Sprintf("zoneId-%d", i), pulumi.String(zone.Id))
	}
	if len(zoneIds) == 0 {
		return fmt.Errorf("no available zone found")
	}

	vSwitchs := make([]*vpc.Switch, 0)
	for i, zoneId := range zoneIds {
		vSwitch, err := vpc.NewSwitch(ctx, fmt.Sprintf("%s-%d", aliVswitchName, i), &vpc.SwitchArgs{
			VswitchName: pulumi.String(fmt.Sprintf("%s-%d", aliVswitchName, i)),
			CidrBlock:   pulumi.String(fmt.Sprintf("192.168.%d.0/24", i)),
			VpcId:       a.vpcID,
			ZoneId:      pulumi.String(zoneId),
		})
		if err != nil {
			return err
		}
		ctx.Export(fmt.Sprintf("vSwitchId-%d", i), vSwitch.ID())
		vSwitchs = append(vSwitchs, vSwitch)
	}
	if len(vSwitchs) == 0 {
		return fmt.Errorf("no available vswitch found")
	}
	a.vSwitchs = vSwitchs

	group, err := ecs.NewSecurityGroup(ctx, ecsSecurityGroup, &ecs.SecurityGroupArgs{
		Name:        pulumi.String(ecsSecurityGroup),
		Description: pulumi.String("ocean ecs security group."),
		VpcId:       a.vpcID,
	})
	if err != nil {
		return err
	}
	a.sgID = group.ID()
	ctx.Export("security_group_id", a.sgID)

	// sg rule: can add more rules
	_, err = ecs.NewSecurityGroupRule(ctx, "allow_all_tcp", &ecs.SecurityGroupRuleArgs{
		Type:            pulumi.String("ingress"),
		IpProtocol:      pulumi.String("tcp"),
		NicType:         pulumi.String("internet"),
		Policy:          pulumi.String("accept"),
		PortRange:       pulumi.String("22/22"),
		Priority:        pulumi.Int(1),
		SecurityGroupId: group.ID(),
		CidrIp:          pulumi.String("0.0.0.0/0"),
	})
	if err != nil {
		return err
	}

	// 公网IP
	eipAddress, err := ecs.NewEipAddress(ctx, natEipName, &ecs.EipAddressArgs{
		AddressName:        pulumi.String(natEipName),
		InternetChargeType: pulumi.String("PayByTraffic"),
	})
	if err != nil {
		return err
	}
	a.eipID = eipAddress.ID()

	ctx.Export("external_ip", eipAddress.IpAddress)

	// 创建nat网关
	var vswitchId pulumi.StringInput
	for _, v := range vSwitchs {
		vswitchId = v.ID()
		break
	}
	natGateway, err := vpc.NewNatGateway(ctx, natGatewayName, &vpc.NatGatewayArgs{
		VpcId:              a.vpcID,
		VswitchId:          vswitchId,
		NatGatewayName:     pulumi.String(natGatewayName),
		InternetChargeType: pulumi.String("PayByTraffic"),
		NatType:            pulumi.String("Enhanced"),
	})
	if err != nil {
		return err
	}

	_, err = ecs.NewEipAssociation(ctx, natGatewayName+"eip-association", &ecs.EipAssociationArgs{
		AllocationId: a.eipID,
		InstanceId:   natGateway.ID(),
	})
	if err != nil {
		return err
	}

	// Import an existing public key to build a alicloud key pair
	key, err := ecs.NewKeyPair(ctx, keyPairName, &ecs.KeyPairArgs{
		KeyName:   pulumi.String(keyPairName),
		PublicKey: pulumi.String(a.cluster.PublicKey),
	})
	if err != nil {
		return err
	}
	ctx.Export("key_pair_name", key.KeyPairName)

	return nil
}

func (a *AlicloudCluster) Clear(ctx *pulumi.Context) error {
	// 清理资源
	return nil
}

func (a *AlicloudCluster) bostionHost(ctx *pulumi.Context) error {
	// https://help.aliyun.com/zh/ecs/user-guide/overview-of-instance-families?spm=a2c4g.11186623.0.0.717dd156Lt8LI2
	// NVIDIA Tesla V100 GPUs. NVIDIA A100 Tensor Core GPUs. NVIDIA T4 GPUs.
	// 实例类型选择改为 low midd high 3种 是否包含GPU
	// 2 core 4G 经济型
	bostionHostInstanceType, err := ecs.GetInstanceTypes(ctx, &ecs.GetInstanceTypesArgs{
		InstanceTypeFamily: pulumi.StringRef("ecs.t6"),
	}, nil)
	if err != nil {
		return err
	}
	sort.Sort(GetInstanceTypesInstanceTypes(bostionHostInstanceType.InstanceTypes))
	var nodeInstanceType string
	for _, v := range bostionHostInstanceType.InstanceTypes {
		if v.CpuCoreCount >= int(a.cluster.BostionHost.CPU) && v.MemorySize >= float64(a.cluster.BostionHost.Memory) {
			nodeInstanceType = v.Id
			break
		}
	}

	images, err := ecs.GetImages(ctx, &ecs.GetImagesArgs{
		NameRegex: pulumi.StringRef("^ubuntu_22_04_x64*"),
		Owners:    pulumi.StringRef("system"),
	}, nil)
	if err != nil {
		return err
	}
	if len(images.Images) == 0 {
		return fmt.Errorf("no available image found")
	}

	var vswitchId pulumi.StringInput
	var zoneId pulumi.StringInput
	for _, v := range a.vSwitchs {
		vswitchId = v.ID()
		zoneId = v.ZoneId
		break
	}
	instance, err := ecs.NewInstance(ctx, bostionHostName, &ecs.InstanceArgs{
		InstanceName:            pulumi.String(bostionHostName),
		AvailabilityZone:        zoneId,
		SecurityGroups:          pulumi.StringArray{a.sgID},
		InstanceType:            pulumi.String(nodeInstanceType),
		ImageId:                 pulumi.String(images.Images[0].Id),
		VswitchId:               vswitchId,
		InternetMaxBandwidthOut: pulumi.Int(20), // 出网带宽
		SystemDiskCategory:      pulumi.String("cloudEssd"),
		SystemDiskName:          pulumi.String("bostion_host_system_disk"),
		SystemDiskSize:          pulumi.Int(20),
		SystemDiskDescription:   pulumi.String("bostion host system disk"),
		KeyName:                 pulumi.String(keyPairName),
		HostName:                pulumi.String(bostionHostName),
		ResourceGroupId:         a.resourceGroupID,
		Tags: pulumi.StringMap{
			"Name": pulumi.String(bostionHostName),
		},
	})
	if err != nil {
		return err
	}

	_, err = ecs.NewEipAssociation(ctx, bostionEipName+"eip-association", &ecs.EipAssociationArgs{
		AllocationId: a.eipID,
		InstanceId:   instance.ID(),
	})
	if err != nil {
		return err
	}

	ctx.Export("bostion_private_ip", instance.PrivateIp)
	ctx.Export("bostion_public_ip", instance.PublicIp)
	ctx.Export("bostion_id", instance.ID())
	ctx.Export("bostion_hostname", instance.HostName)

	return nil
}

func (a *AlicloudCluster) nodes(ctx *pulumi.Context) error {
	images, err := ecs.GetImages(ctx, &ecs.GetImagesArgs{
		NameRegex: pulumi.StringRef("^ubuntu_22_04_x64*"),
		Owners:    pulumi.StringRef("system"),
	}, nil)
	if err != nil {
		return err
	}
	if len(images.Images) == 0 {
		return fmt.Errorf("no available image found")
	}
	imageID := images.Images[0].Id
	for _, nodeGroup := range a.cluster.NodeGroups {
		nodeGroup.OSImage = imageID
		instanceTypeFamilies := a.getIntanceTypeFamilies(nodeGroup)
		nodeInstanceTypes, err := ecs.GetInstanceTypes(ctx, &ecs.GetInstanceTypesArgs{
			InstanceTypeFamily: pulumi.StringRef(instanceTypeFamilies),
		}, nil)
		if err != nil {
			return err
		}
		sort.Sort(GetInstanceTypesInstanceTypes(nodeInstanceTypes.InstanceTypes))
		for _, v := range nodeInstanceTypes.InstanceTypes {
			if v.MemorySize == 0 {
				continue
			}
			if v.CpuCoreCount >= int(nodeGroup.CPU) && v.MemorySize >= float64(nodeGroup.Memory) {
				nodeGroup.InstanceType = v.Id
			}
			if nodeGroup.InstanceType == "" {
				continue
			}
			if nodeGroup.GPU == 0 {
				break
			}
			if cast.ToInt32(v.Gpu.Amount) >= nodeGroup.GPU {
				break
			}
		}
		if nodeGroup.InstanceType == "" {
			return fmt.Errorf("no available instance type found")
		}

		for nodeIndex, node := range a.cluster.Nodes {
			vswitch := a.distributeNodeVswitches(nodeIndex)
			instanceArgs := &ecs.InstanceArgs{
				HostName:                pulumi.String(node.Name),
				InstanceName:            pulumi.String(node.Name),
				AvailabilityZone:        vswitch.ZoneId,
				VswitchId:               vswitch.ID(),
				SecurityGroups:          pulumi.StringArray{a.sgID},
				InstanceType:            pulumi.String(nodeGroup.InstanceType),
				ImageId:                 pulumi.String(imageID),
				InternetMaxBandwidthOut: pulumi.Int(nodeGroup.InternetMaxBandwidthOut), // 出网带宽
				SystemDiskCategory:      pulumi.String("cloudEssd"),
				SystemDiskName:          pulumi.String(fmt.Sprintf("system_disk_%s", node.Name)),
				SystemDiskSize:          pulumi.Int(nodeGroup.SystemDisk),
				KeyName:                 pulumi.String(keyPairName),
				ResourceGroupId:         a.resourceGroupID,
			}
			if nodeGroup.NodeInitScript != "" {
				instanceArgs.UserData = pulumi.String(nodeGroup.NodeInitScript) // 节点初始化脚本
			}
			if node.Labels != "" {
				lableMap := make(map[string]string)
				err = json.Unmarshal([]byte(node.Labels), &lableMap)
				if err != nil {
					return err
				}
				tags := make(pulumi.StringMap)
				for k, v := range lableMap {
					tags[k] = pulumi.String(v)
				}
				instanceArgs.Tags = tags
			}
			if nodeGroup.DataDisk != 0 {
				instanceArgs.DataDisks = ecs.InstanceDataDiskArray{
					&ecs.InstanceDataDiskArgs{
						Size:     pulumi.Int(nodeGroup.DataDisk),
						Category: pulumi.String("cloudEssd"),
						Name:     pulumi.String(fmt.Sprintf("data_disk_%s", node.Name)),
						Device:   pulumi.String(fmt.Sprintf("/dev/vdb%s", node.Name)),
					},
				}
			}
			instance, err := ecs.NewInstance(ctx, node.Name, instanceArgs)
			if err != nil {
				return err
			}
			ctx.Export(fmt.Sprintf("node-%s-id", node.Name), instance.ID())
			ctx.Export(fmt.Sprintf("node-%s-user", node.Name), pulumi.String("root"))
			ctx.Export(fmt.Sprintf("node-%s-internal-ip", node.Name), instance.PrivateIp)
			ctx.Export(fmt.Sprintf("node-%s-public-ip", node.Name), instance.PublicIp)
		}
	}
	return nil
}

func (a *AlicloudCluster) localBalancer(ctx *pulumi.Context) error {
	ctx.Export("local_balancer_id", pulumi.String("a"))
	return nil
}

func (a *AlicloudCluster) distributeNodeVswitches(nodeIndex int) *vpc.Switch {
	nodeSize := len(a.cluster.Nodes)
	vSwitchSize := len(a.vSwitchs)
	if nodeSize <= vSwitchSize {
		return a.vSwitchs[nodeIndex%vSwitchSize]
	}
	interval := nodeSize / vSwitchSize
	return a.vSwitchs[(nodeIndex/interval)%vSwitchSize]
}
