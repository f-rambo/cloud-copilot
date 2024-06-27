package pulumiapi

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/cs"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/ecs"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/ram"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/resourcemanager"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/vpc"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
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

type AlicloudCluster struct {
	clusterArgs     AlicloudClusterArgs
	resourceGroupID pulumi.StringInput
	vpcID           pulumi.StringInput
	vSwitchs        []*vpc.Switch
	sgID            pulumi.StringInput
	eipID           pulumi.StringInput
}

type AlicloudClusterArgs struct {
	Name      string // cluster name *required*
	PublicKey string // public key for ssh login *required*
	Nodes     []AlicloudNodeArgs
}

type AlicloudNodeArgs struct {
	Name                    string            // node name *required*
	InstanceType            string            // instance type
	CPU                     int               // cpu cores *required*
	Memory                  float64           // memory in GB *required*
	GPU                     int               // gpu cores
	GpuSpec                 string            // gpu spec
	OSImage                 string            // os image
	InternetMaxBandwidthOut int               // internet max bandwidth out *required*
	SystemDisk              int               // system disk size in GB *required*
	DataDisk                int               // data disk size in GB
	Labels                  map[string]string // labels for node selector
	NodeInitScript          string            // node init script : user data for ecs instance
}

func StartAlicloudCluster(clusterArgs AlicloudClusterArgs) *AlicloudCluster {
	return &AlicloudCluster{
		clusterArgs: clusterArgs,
	}
}

func (a *AlicloudCluster) init(ctx *pulumi.Context) error {
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
		PublicKey: pulumi.String(a.clusterArgs.PublicKey),
	})
	if err != nil {
		return err
	}
	ctx.Export("key_pair_name", key.KeyPairName)

	return nil
}

func (a *AlicloudCluster) StartServers(ctx *pulumi.Context) error {
	err := a.init(ctx)
	if err != nil {
		return errors.Wrap(err, "alicloud cluster init failed")
	}
	err = a.bostionHost(ctx)
	if err != nil {
		return errors.Wrap(err, "start bostion host failed")
	}
	err = a.servers(ctx)
	if err != nil {
		return errors.Wrap(err, "start ecs failed")
	}
	err = a.localBalancer(ctx)
	if err != nil {
		return errors.Wrap(err, "start local balancer failed")
	}
	return nil
}

func (a *AlicloudCluster) Clear(ctx *pulumi.Context) error {
	// 清理资源
	return nil
}

func (a *AlicloudCluster) bostionHost(ctx *pulumi.Context) error {
	// 2 core 4G 经济型
	masterGetInstanceType, err := ecs.GetInstanceTypes(ctx, &ecs.GetInstanceTypesArgs{
		InstanceTypeFamily: pulumi.StringRef("ecs.e"),
		CpuCoreCount:       pulumi.IntRef(2),
		MemorySize:         pulumi.Float64Ref(4),
	}, nil)
	if err != nil {
		return err
	}
	if len(masterGetInstanceType.InstanceTypes) == 0 {
		return fmt.Errorf("no available instance type found")
	}
	var nodeInstanceType string
	for _, v := range masterGetInstanceType.InstanceTypes {
		nodeInstanceType = v.Id
		break
	}

	images, err := ecs.GetImages(ctx, &ecs.GetImagesArgs{
		NameRegex: pulumi.StringRef("^ubuntu_22_[0-9]+_x64"),
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

func (a *AlicloudCluster) servers(ctx *pulumi.Context) error {
	for nodeIndex, node := range a.clusterArgs.Nodes {
		if node.InstanceType == "" {
			instanceArgs := &ecs.GetInstanceTypesArgs{}
			if node.CPU != 0 {
				instanceArgs.CpuCoreCount = pulumi.IntRef(node.CPU)
			}
			if node.Memory != 0 {
				instanceArgs.MemorySize = pulumi.Float64Ref(node.Memory)
			}
			if node.GPU != 0 {
				instanceArgs.GpuAmount = pulumi.IntRef(node.GPU)
			}
			if node.GpuSpec != "" {
				instanceArgs.GpuSpec = pulumi.StringRef(node.GpuSpec)
			}
			nodeInstanceTypes, err := ecs.GetInstanceTypes(ctx, instanceArgs, nil)
			if err != nil {
				return err
			}
			if len(nodeInstanceTypes.InstanceTypes) == 0 {
				return fmt.Errorf("no available instance type found")
			}
			for _, v := range nodeInstanceTypes.InstanceTypes {
				node.InstanceType = v.Id
			}
		}
		if node.OSImage == "" {
			images, err := ecs.GetImages(ctx, &ecs.GetImagesArgs{
				NameRegex: pulumi.StringRef("^ubuntu_22_[0-9]+_x64"),
				Owners:    pulumi.StringRef("system"),
			}, nil)
			if err != nil {
				return err
			}
			if len(images.Images) == 0 {
				return fmt.Errorf("no available image found")
			}
			for _, v := range images.Images {
				node.OSImage = v.Id
			}
		}

		vswitch := a.distributeNodeVswitches(nodeIndex)
		instanceArgs := &ecs.InstanceArgs{
			HostName:                pulumi.String(node.Name),
			InstanceName:            pulumi.String(node.Name),
			AvailabilityZone:        vswitch.ZoneId,
			VswitchId:               vswitch.ID(),
			SecurityGroups:          pulumi.StringArray{a.sgID},
			InstanceType:            pulumi.String(node.InstanceType),
			ImageId:                 pulumi.String(node.OSImage),
			InternetMaxBandwidthOut: pulumi.Int(node.InternetMaxBandwidthOut), // 出网带宽
			SystemDiskCategory:      pulumi.String("cloudEssd"),
			SystemDiskName:          pulumi.String(fmt.Sprintf("system_disk_%s", node.Name)),
			SystemDiskSize:          pulumi.Int(node.SystemDisk),
			KeyName:                 pulumi.String(keyPairName),
			ResourceGroupId:         a.resourceGroupID,
		}
		if node.NodeInitScript != "" {
			instanceArgs.UserData = pulumi.String(node.NodeInitScript) // 节点初始化脚本
		}
		if node.Labels != nil {
			tags := make(pulumi.StringMap)
			for k, v := range node.Labels {
				tags[k] = pulumi.String(v)
			}
			instanceArgs.Tags = tags
		}
		if node.DataDisk != 0 {
			instanceArgs.DataDisks = ecs.InstanceDataDiskArray{
				&ecs.InstanceDataDiskArgs{
					Size:     pulumi.Int(node.DataDisk),
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
	return nil
}

func (a *AlicloudCluster) localBalancer(ctx *pulumi.Context) error {
	return nil
}

// alicloud managed kubernetes cluster
func (a *AlicloudCluster) Startkubernetes(ctx *pulumi.Context) error {
	if err := a.init(ctx); err != nil {
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
		Name:             pulumi.String(a.clusterArgs.Name),
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

func (a *AlicloudCluster) distributeNodeVswitches(nodeIndex int) *vpc.Switch {
	nodeSize := len(a.clusterArgs.Nodes)
	vSwitchSize := len(a.vSwitchs)
	if nodeSize <= vSwitchSize {
		return a.vSwitchs[nodeIndex%vSwitchSize]
	}
	interval := nodeSize / vSwitchSize
	return a.vSwitchs[(nodeIndex/interval)%vSwitchSize]
}
