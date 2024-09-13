package infrastructure

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/utils"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/alb"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/ecs"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/ram"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/resourcemanager"
	"github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/vpc"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/spf13/cast"
)

const (
	AlicloudProjectName = "ocean-alicloud-project"
	AlicloudStackName   = "ocean-alicloud-stack"
)

const (
	alicloudResourceGroupName        = "ocean-resource-group"
	alicloudRoleName                 = "ocean-cluster-role"
	alicloudCsPolicy                 = `{"Statement":[{"Action":"sts:AssumeRole","Effect":"Allow","Principal":{"Service":["cs.aliyuncs.com"]}}],"Version":"1"}`
	alicloudEscPolicy                = `{"Statement":[{"Action":"sts:AssumeRole","Effect":"Allow","Principal":{"Service":["ecs.aliyuncs.com"]}}],"Version":"1"}`
	alicloudVpcName                  = "ocean-vpc"
	alicloudVswitchName              = "ocean-vswitch"
	alicloudEcsSecurityGroup         = "ocean-ecs-security-group"
	alicloudEcsSecurityGroupRuleName = "ocean-ecs-security-group-rule"
	alicloudBostionHostName          = "ocean-bostion"
	alicloudBostionEipName           = "ocean-bostion-eip"
	alicloudNatGatewayName           = "ocean-nat-gateway"
	alicloudNatGatewayEipAssociation = "ocean-nat-gateway-eip-association"
	alicloudNatEipName               = "ocean-nat-eip"
	alicloudKeyPairName              = "ocean-key-pair"
	alicloudSlbName                  = "ocean-slb"
	alicloudSlbListenerName          = "ocean-slb-listener"
	alicloudVpcCidrBlock             = "192.168.0.0/16"
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

func (a *Alicloud) getIntanceTypeFamilies(nodeGroup *biz.NodeGroup) string {
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

func (a *Alicloud) Start(ctx *pulumi.Context) error {
	err := a.getClusterInfoByInstance(ctx)
	if err != nil {
		return err
	}
	err = a.getLocalBalancer(ctx)
	if err != nil {
		return err
	}
	err = a.getNatGateway(ctx)
	if err != nil {
		return err
	}
	err = a.infrastructural(ctx)
	if err != nil {
		return errors.Wrap(err, "alicloud cluster init failed")
	}
	err = a.createNodes(ctx)
	if err != nil {
		return errors.Wrap(err, "start ecs failed")
	}
	return nil
}

func (a *Alicloud) infrastructural(ctx *pulumi.Context) error {
	err := a.createResourceGroup(ctx)
	if err != nil {
		return err
	}

	err = a.createRolesAndPolicies(ctx)
	if err != nil {
		return err
	}

	err = a.createVPC(ctx)
	if err != nil {
		return err
	}

	err = a.createVSwitches(ctx)
	if err != nil {
		return err
	}

	err = a.createSecurityGroup(ctx)
	if err != nil {
		return err
	}

	err = a.createEIP(ctx)
	if err != nil {
		return err
	}

	err = a.createNATGateway(ctx)
	if err != nil {
		return err
	}

	err = a.createKeyPair(ctx)
	if err != nil {
		return err
	}
	err = a.localBalancer(ctx)
	if err != nil {
		return errors.Wrap(err, "start local balancer failed")
	}
	err = a.setImageByNodeGroups(ctx)
	if err != nil {
		return errors.Wrap(err, "set image by node groups failed")
	}
	err = a.setInstanceTypeByNodeGroups(ctx)
	if err != nil {
		return errors.Wrap(err, "set instance type by node groups failed")
	}
	return nil
}

func (a *Alicloud) createResourceGroup(ctx *pulumi.Context) (err error) {
	resourceGroupArgs := &resourcemanager.ResourceGroupArgs{
		ResourceGroupName: pulumi.String(alicloudResourceGroupName),
		DisplayName:       pulumi.String(alicloudResourceGroupName),
	}
	if a.cluster.ResourceGroupID != "" {
		a.resourceGroup, err = resourcemanager.NewResourceGroup(ctx, alicloudResourceGroupName, resourceGroupArgs,
			pulumi.Import(pulumi.ID(a.cluster.ResourceGroupID)))
		if err != nil {
			return err
		}
		return nil
	}
	a.resourceGroup, err = resourcemanager.NewResourceGroup(ctx, alicloudResourceGroupName, resourceGroupArgs)
	if err != nil {
		return err
	}
	return nil
}

func (a *Alicloud) createRolesAndPolicies(ctx *pulumi.Context) (err error) {
	roleMap := map[string]string{
		"csPolicy":  alicloudCsPolicy,
		"ecsPolicy": alicloudEscPolicy,
	}
	for name, rolePolicy := range roleMap {
		a.role, err = ram.NewRole(ctx, name, &ram.RoleArgs{
			Name:        pulumi.String(name),
			Document:    pulumi.String(rolePolicy),
			Description: pulumi.String("ocean cluster role."),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Alicloud) createVPC(ctx *pulumi.Context) (err error) {
	cidrBlock := alicloudVpcCidrBlock
	if a.cluster.VpcCidr != "" {
		cidrBlock = a.cluster.VpcCidr
	}
	vpcArgs := &vpc.NetworkArgs{
		VpcName:   pulumi.String(alicloudVpcName),
		CidrBlock: pulumi.String(cidrBlock),
	}
	if a.cluster.VpcID != "" {
		vpcNetwork, err := vpc.GetNetwork(ctx, alicloudVpcName, pulumi.ID(a.cluster.VpcID), nil)
		if err != nil {
			return err
		}
		vpcArgs.CidrBlock = vpcNetwork.CidrBlock
		a.vpcNetWork, err = vpc.NewNetwork(ctx, alicloudVpcName, vpcArgs, pulumi.Import(pulumi.ID(a.cluster.VpcID)))
		if err != nil {
			return err
		}
		return nil
	}
	a.vpcNetWork, err = vpc.NewNetwork(ctx, alicloudVpcName, vpcArgs)
	if err != nil {
		return err
	}
	return nil
}

func (a *Alicloud) createVSwitches(ctx *pulumi.Context) error {
	// import vswitch
	var subnetIds []string
	for _, node := range a.cluster.Nodes {
		subnetIds = append(subnetIds, node.SubnetId)
	}
	subnetIds = utils.RemoveDuplicateString(subnetIds)
	for i, subnetId := range subnetIds {
		vswitchName := fmt.Sprintf("%s-%d", alicloudVswitchName, i)
		vswitch, err := vpc.GetSwitch(ctx, vswitchName, pulumi.ID(subnetId), nil)
		if err != nil {
			return err
		}
		a.vSwitchs = append(a.vSwitchs, vswitch)
		_, err = vpc.NewSwitch(ctx, vswitchName, &vpc.SwitchArgs{
			VswitchName: vswitch.VswitchName,
			CidrBlock:   vswitch.CidrBlock,
			VpcId:       a.vpcNetWork.ID(),
			ZoneId:      vswitch.ZoneId,
		}, pulumi.Import(pulumi.ID(subnetId)))
		if err != nil {
			return err
		}
	}
	if len(a.vSwitchs) > 0 {
		return nil
	}
	// create vswitch
	zones, err := alicloud.GetZones(ctx, &alicloud.GetZonesArgs{
		AvailableResourceCreation: pulumi.StringRef("VSwitch"),
	}, nil)
	if err != nil {
		return err
	}
	zoneIds := make([]string, 0)
	for _, zone := range zones.Zones {
		zoneIds = append(zoneIds, zone.Id)
	}
	if len(zoneIds) == 0 {
		return fmt.Errorf("no available zone found")
	}

	for i, zoneId := range zoneIds {
		vSwitch, err := vpc.NewSwitch(ctx, fmt.Sprintf("%s-%d", alicloudVswitchName, i), &vpc.SwitchArgs{
			VswitchName: pulumi.String(fmt.Sprintf("%s-%d", alicloudVswitchName, i)),
			CidrBlock:   pulumi.String(fmt.Sprintf("192.168.%d.0/24", i)),
			VpcId:       a.vpcNetWork.ID(),
			ZoneId:      pulumi.String(zoneId),
		})
		if err != nil {
			return err
		}
		a.vSwitchs = append(a.vSwitchs, vSwitch)
	}
	if len(a.vSwitchs) == 0 {
		return fmt.Errorf("no available vswitch found")
	}
	return nil
}

func (a *Alicloud) createSecurityGroup(ctx *pulumi.Context) (err error) {
	// import security group
	sgIDs := strings.Split(a.cluster.SecurityGroupIDs, ",")
	for i, sgID := range sgIDs {
		sgName := fmt.Sprintf("%s-%d", alicloudEcsSecurityGroup, i)
		sg, err := ecs.GetSecurityGroup(ctx, sgName, pulumi.ID(sgID), nil)
		if err != nil {
			return err
		}
		a.sgs = append(a.sgs, sg)
		_, err = ecs.NewSecurityGroup(ctx, sgName, &ecs.SecurityGroupArgs{
			Name:        sg.Name,
			Description: sg.Description,
			VpcId:       a.vpcNetWork.ID(),
		}, pulumi.Import(pulumi.ID(sgID)))
		if err != nil {
			return err
		}
	}
	// create security group
	group, err := ecs.NewSecurityGroup(ctx, alicloudEcsSecurityGroup, &ecs.SecurityGroupArgs{
		Name:        pulumi.String(alicloudEcsSecurityGroup),
		Description: pulumi.String("ocean ecs security group."),
		VpcId:       a.vpcNetWork.ID(),
	})
	if err != nil {
		return err
	}
	a.sgs = append(a.sgs, group)

	_, err = ecs.NewSecurityGroupRule(ctx, alicloudEcsSecurityGroupRuleName, &ecs.SecurityGroupRuleArgs{
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
	sgPulumiIDs := make(pulumi.StringArray, 0)
	for _, sg := range a.sgs {
		sgPulumiIDs = append(sgPulumiIDs, sg.ID())
	}
	ctx.Export(getSecurityGroupIDs(), sgPulumiIDs)
	return nil
}

func (a *Alicloud) createEIP(ctx *pulumi.Context) error {
	if a.cluster.EipID != "" {
		eip, err := ecs.GetEipAddress(ctx, alicloudNatEipName, pulumi.ID(a.cluster.EipID), nil)
		if err != nil {
			return err
		}
		_, err = ecs.NewEipAddress(ctx, alicloudNatEipName, &ecs.EipAddressArgs{
			AddressName:        eip.AddressName,
			InternetChargeType: eip.InternetChargeType,
		}, pulumi.Import(pulumi.ID(a.cluster.EipID)))
		if err != nil {
			return err
		}
		a.eipAddress = eip
		return nil
	}
	eipAddress, err := ecs.NewEipAddress(ctx, alicloudNatEipName, &ecs.EipAddressArgs{
		AddressName:        pulumi.String(alicloudNatEipName),
		InternetChargeType: pulumi.String("PayByTraffic"),
	})
	if err != nil {
		return err
	}
	a.eipAddress = eipAddress
	return nil
}

func (a *Alicloud) createNATGateway(ctx *pulumi.Context) (err error) {
	if a.cluster.NatGatewayID != "" {
		a.natGateway, err = vpc.GetNatGateway(ctx, alicloudNatGatewayName, pulumi.ID(a.cluster.NatGatewayID), nil)
		if err != nil {
			return err
		}
		_, err = vpc.NewNatGateway(ctx, alicloudNatGatewayName, &vpc.NatGatewayArgs{
			VpcId:              a.vpcNetWork.ID(),
			VswitchId:          a.natGateway.VswitchId,
			NatGatewayName:     a.natGateway.NatGatewayName,
			InternetChargeType: a.natGateway.InternetChargeType,
			NatType:            a.natGateway.NatType,
		}, pulumi.Import(pulumi.ID(a.cluster.NatGatewayID)))
		if err != nil {
			return err
		}
	} else {
		var vswitchId pulumi.StringInput
		for _, v := range a.vSwitchs {
			vswitchId = v.ID()
			break
		}
		a.natGateway, err = vpc.NewNatGateway(ctx, alicloudNatGatewayName, &vpc.NatGatewayArgs{
			VpcId:              a.vpcNetWork.ID(),
			VswitchId:          vswitchId,
			NatGatewayName:     pulumi.String(alicloudNatGatewayName),
			InternetChargeType: pulumi.String("PayByTraffic"),
			NatType:            pulumi.String("Enhanced"),
		})
		if err != nil {
			return err
		}
	}
	_, err = ecs.NewEipAssociation(ctx, alicloudNatGatewayEipAssociation, &ecs.EipAssociationArgs{
		AllocationId: a.eipAddress.ID(),
		InstanceId:   a.natGateway.ID(),
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *Alicloud) createKeyPair(ctx *pulumi.Context) (err error) {
	if a.cluster.KeyPair != "" {
		a.keyPair, err = ecs.GetKeyPair(ctx, alicloudKeyPairName, pulumi.ID(a.cluster.KeyPair), nil)
		if err != nil {
			return err
		}
		_, err = ecs.NewKeyPair(ctx, alicloudKeyPairName, &ecs.KeyPairArgs{
			KeyPairName: a.keyPair.KeyPairName,
			PublicKey:   a.keyPair.PublicKey,
		}, pulumi.Import(pulumi.ID(a.cluster.KeyPair)))
		if err != nil {
			return err
		}
		return nil
	}
	a.keyPair, err = ecs.NewKeyPair(ctx, alicloudKeyPairName, &ecs.KeyPairArgs{
		KeyPairName: pulumi.String(alicloudKeyPairName),
		PublicKey:   pulumi.String(a.cluster.PublicKey),
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *Alicloud) setImageByNodeGroups(ctx *pulumi.Context) error {
	images, err := ecs.GetImages(ctx, &ecs.GetImagesArgs{
		NameRegex: pulumi.StringRef("^ubuntu_22_04_x64*"),
		Owners:    pulumi.StringRef("system"),
	}, nil)
	if err != nil {
		return err
	}
	imageID := ""
	for _, image := range images.Images {
		imageID = image.Id
		break
	}
	if imageID == "" {
		return fmt.Errorf("no available image found")
	}
	for _, nodeGroup := range a.cluster.NodeGroups {
		nodeGroup.Image = imageID
	}
	return nil
}

func (a *Alicloud) setInstanceTypeByNodeGroups(ctx *pulumi.Context) error {
	for _, nodeGroup := range a.cluster.NodeGroups {
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
	}
	return nil
}

func (a *Alicloud) createNodes(ctx *pulumi.Context) (err error) {
	selectedBostionHost := false
	for nodeIndex, node := range a.cluster.Nodes {
		nodeGroup := a.cluster.GetNodeGroup(node.NodeGroupID)
		if nodeGroup == nil {
			return fmt.Errorf("node group not found")
		}
		if nodeGroup.Image == "" {
			return fmt.Errorf("image not found")
		}
		if nodeGroup.InstanceType == "" {
			return fmt.Errorf("instance type not found")
		}
		vswitch := a.distributeNodeVswitches(nodeIndex)
		sgIDs := make(pulumi.StringArray, 0)
		for _, sg := range a.sgs {
			sgIDs = append(sgIDs, sg.ID())
		}
		instanceArgs := &ecs.InstanceArgs{
			HostName:                pulumi.String(node.Name),
			InstanceName:            pulumi.String(node.Name),
			AvailabilityZone:        vswitch.ZoneId,
			VswitchId:               vswitch.ID(),
			SecurityGroups:          sgIDs,
			InstanceType:            pulumi.String(nodeGroup.InstanceType),
			ImageId:                 pulumi.String(nodeGroup.Image),
			InternetMaxBandwidthOut: pulumi.Int(node.InternetMaxBandwidthOut),
			SystemDiskCategory:      pulumi.String("cloudEssd"),
			SystemDiskName:          pulumi.String(fmt.Sprintf("system_disk_%s", node.Name)),
			SystemDiskSize:          pulumi.Int(node.SystemDisk),
			KeyName:                 pulumi.String(alicloudKeyPairName),
			ResourceGroupId:         a.resourceGroup.ID(),
			RoleName:                a.role.Name,
		}
		if nodeGroup.NodeInitScript != "" {
			instanceArgs.UserData = pulumi.String(nodeGroup.NodeInitScript)
		}
		if node.Labels != "" {
			lableMap := make(map[string]string)
			err := json.Unmarshal([]byte(node.Labels), &lableMap)
			if err != nil {
				return err
			}
			tags := make(pulumi.StringMap)
			for k, v := range lableMap {
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
		var instance *ecs.Instance
		if node.InstanceID != "" {
			instance, err = ecs.GetInstance(ctx, node.Name, pulumi.ID(node.InstanceID), nil)
			if err != nil {
				return err
			}
			_, err = ecs.NewInstance(ctx, node.Name, &ecs.InstanceArgs{
				InstanceName:            instance.InstanceName,
				HostName:                instance.HostName,
				AvailabilityZone:        instance.AvailabilityZone,
				VswitchId:               instance.VswitchId,
				SecurityGroups:          instance.SecurityGroups,
				InstanceType:            instance.InstanceType,
				ImageId:                 instance.ImageId,
				InternetMaxBandwidthOut: instance.InternetMaxBandwidthOut,
				SystemDiskCategory:      instance.SystemDiskCategory,
				SystemDiskName:          instance.SystemDiskName,
				SystemDiskSize:          instance.SystemDiskSize,
				KeyName:                 instance.KeyName,
				ResourceGroupId:         instance.ResourceGroupId,
				RoleName:                instance.RoleName,
				UserData:                instance.UserData,
				Tags:                    instance.Tags,
				DataDisks:               instance.DataDisks,
			}, pulumi.Import(pulumi.ID(node.InstanceID)))
			if err != nil {
				return err
			}
		} else {
			instance, err = ecs.NewInstance(ctx, node.Name, instanceArgs)
			if err != nil {
				return err
			}
		}
		if node.Role == biz.NodeRoleMaster && !selectedBostionHost {
			selectedBostionHost = true
			// bind eip to instance
			_, err = ecs.NewEipAssociation(ctx, alicloudNatGatewayEipAssociation, &ecs.EipAssociationArgs{
				AllocationId: a.eipAddress.ID(),
				InstanceId:   instance.ID(),
			})
			if err != nil {
				return err
			}
			ctx.Export(getBostionHostInstanceID(), instance.ID())
		}
		ctx.Export(getIntanceIDKey(node.Name), instance.ID())
		ctx.Export(getIntanceUser(node.Name), pulumi.String("root"))
		ctx.Export(getIntanceInternalIPKey(node.Name), instance.PrivateIp)
		ctx.Export(getIntancePublicIPKey(node.Name), instance.PublicIp)
	}
	return nil
}

func (a *Alicloud) localBalancer(ctx *pulumi.Context) (err error) {
	if a.cluster.LoadBalancerID != "" {
		a.lb, err = alb.GetLoadBalancer(ctx, alicloudSlbName, pulumi.ID(a.cluster.LoadBalancerID), nil)
		if err != nil {
			return err
		}
		_, err = alb.NewLoadBalancer(ctx, alicloudSlbName, &alb.LoadBalancerArgs{
			LoadBalancerName: a.lb.LoadBalancerName,
			VpcId:            a.vpcNetWork.ID(),
			AddressType:      a.lb.AddressType,
			AddressIpVersion: a.lb.AddressIpVersion,
			ZoneMappings:     a.lb.ZoneMappings,
		}, pulumi.Import(pulumi.ID(a.cluster.LoadBalancerID)))
		if err != nil {
			return err
		}
		listeners, err := alb.GetListeners(ctx, &alb.GetListenersArgs{
			LoadBalancerIds: []string{a.cluster.LoadBalancerID},
		}, nil)
		if err != nil {
			return err
		}
		for _, v := range listeners.Listeners {
			_, err = alb.NewListener(ctx, fmt.Sprintf("listener_%d", v.ListenerPort), &alb.ListenerArgs{
				LoadBalancerId:   a.lb.ID(),
				ListenerPort:     pulumi.Int(v.ListenerPort),
				ListenerProtocol: pulumi.String(v.ListenerProtocol),
			}, pulumi.Import(pulumi.ID(v.ListenerId)))
			if err != nil {
				return err
			}
		}
	} else {
		zoneMappings := make(alb.LoadBalancerZoneMappingArray, 0)
		for _, v := range a.vSwitchs {
			zoneMappings = append(zoneMappings, &alb.LoadBalancerZoneMappingArgs{
				VswitchId: v.ID(),
				ZoneId:    v.ZoneId,
			})
		}
		a.lb, err = alb.NewLoadBalancer(ctx, alicloudSlbName, &alb.LoadBalancerArgs{
			LoadBalancerName: pulumi.String(alicloudSlbName),
			VpcId:            a.vpcNetWork.ID(),
			AddressType:      pulumi.String("internet"),
			AddressIpVersion: pulumi.String("ipv4"),
			ZoneMappings:     zoneMappings,
		})
		if err != nil {
			return err
		}
	}

	// Create Load Balancer Listener http
	_, err = alb.NewListener(ctx, fmt.Sprintf("listener_%d", 80), &alb.ListenerArgs{
		LoadBalancerId:   a.lb.ID(),
		ListenerPort:     pulumi.Int(80),
		ListenerProtocol: pulumi.String("HTTP"),
	})
	if err != nil {
		return err
	}

	// Create Load Balancer Listener https
	_, err = alb.NewListener(ctx, fmt.Sprintf("listener_%d", 443), &alb.ListenerArgs{
		LoadBalancerId:   a.lb.ID(),
		ListenerPort:     pulumi.Int(443),
		ListenerProtocol: pulumi.String("HTTPS"),
	})
	if err != nil {
		return err
	}

	// Load Balancer Listener k8s apiserver
	_, err = alb.NewListener(ctx, fmt.Sprintf("listener_%d", 6443), &alb.ListenerArgs{
		LoadBalancerId:   a.lb.ID(),
		ListenerPort:     pulumi.Int(6443),
		ListenerProtocol: pulumi.String("HTTPS"),
	})
	if err != nil {
		return err
	}
	ctx.Export(getLoadBalancerID(), a.lb.ID())
	return nil
}

func (a *Alicloud) getClusterInfoByInstance(ctx *pulumi.Context) error {
	// get instances
	instances, err := ecs.GetInstances(ctx, &ecs.GetInstancesArgs{
		Status: pulumi.StringRef("Running"),
	})
	if err != nil {
		return err
	}
	instanceTypes := make(map[string][]int64)
	for _, node := range a.cluster.Nodes {
		instance, err := a.getInstanceByNode(instances, node)
		if err != nil {
			return err
		}
		node.InstanceID = instance.Id
		node.SubnetId = instance.VswitchId
		node.Zone = instance.AvailabilityZone
		node.ExternalIP = instance.PublicIp
		a.cluster.VpcID = instance.VpcId
		a.cluster.ResourceGroupID = instance.ResourceGroupId
		a.cluster.SecurityGroupIDs = strings.Join(instance.SecurityGroups, ",")
		a.cluster.KeyPair = instance.KeyName
		a.cluster.Region = instance.RegionId
		if instance.Eip != "" {
			a.cluster.EipID = instance.Eip
		}
		for _, v := range instance.DiskDeviceMappings {
			if v.Type == "system disk" {
				node.SystemDisk += int32(v.Size)
			}
			if v.Type == "data disk" {
				node.DataDisk += int32(v.Size)
			}
		}
		instanceTypes[instance.InstanceType] = append(instanceTypes[instance.InstanceType], node.ID)
	}
	nodeGroups := make([]*biz.NodeGroup, 0)
	for instanceType := range instanceTypes {
		nodeGroup := a.cluster.NewNodeGroup()
		for _, ng := range a.cluster.NodeGroups {
			if ng.InstanceType == instanceType {
				nodeGroup = ng
				nodeGroup.InstanceType = instanceType
				break
			}
		}
		instanceTypes, err := ecs.GetInstanceTypes(ctx, &ecs.GetInstanceTypesArgs{
			InstanceType: pulumi.StringRef(instanceType),
		})
		if err != nil {
			return err
		}
		for _, v := range instanceTypes.InstanceTypes {
			if v.Gpu.Amount != "" {
				nodeGroup.GPU = cast.ToInt32(v.Gpu.Amount)
				nodeGroup.Type = biz.NodeGroupTypeGPUAcceleraterd
			} else {
				nodeGroup.Type = biz.NodeGroupTypeNormal
			}
			nodeGroup.CPU = int32(v.CpuCoreCount)
			nodeGroup.Memory = v.MemorySize
		}
		nodeGroup.Name = a.cluster.GenerateNodeGroupName(nodeGroup)
		nodeGroups = append(nodeGroups, nodeGroup)
	}
	a.cluster.NodeGroups = nodeGroups
	// Assign nodegroupID
	for _, nodeGroup := range a.cluster.NodeGroups {
		for _, nodeID := range instanceTypes[nodeGroup.InstanceType] {
			for _, node := range a.cluster.Nodes {
				if node.ID == nodeID {
					node.NodeGroupID = nodeGroup.ID
					break
				}
			}
		}
	}
	return nil
}

// get instance by node
func (a *Alicloud) getInstanceByNode(instances *ecs.GetInstancesResult, node *biz.Node) (ecs.GetInstancesInstance, error) {
	for _, instance := range instances.Instances {
		if node.InternalIP != instance.PrivateIp {
			continue
		}
		return instance, nil
	}
	return ecs.GetInstancesInstance{}, fmt.Errorf("instance not found")
}

// get local balancer
func (a *Alicloud) getLocalBalancer(ctx *pulumi.Context) error {
	lb, err := alb.GetLoadBalancers(ctx, &alb.GetLoadBalancersArgs{
		VpcId:       pulumi.StringRef(a.cluster.VpcID),
		Status:      pulumi.StringRef("Active"),
		AddressType: pulumi.StringRef("internet"),
	})
	if err != nil {
		return err
	}
	if len(lb.Balancers) == 0 {
		return fmt.Errorf("load balancer not found")
	}
	for _, v := range lb.Balancers {
		a.cluster.LoadBalancerID = v.Id
		return nil
	}
	return fmt.Errorf("load balancer not found")
}

// get nat gateway
func (a *Alicloud) getNatGateway(ctx *pulumi.Context) error {
	natGateway, err := vpc.GetNatGateways(ctx, &vpc.GetNatGatewaysArgs{
		VpcId: pulumi.StringRef(a.cluster.VpcID),
	})
	if err != nil {
		return err
	}
	if len(natGateway.Gateways) == 0 {
		return fmt.Errorf("nat gateway not found")
	}
	a.cluster.NatGatewayID = natGateway.Gateways[0].Id
	return nil
}

func (a *Alicloud) distributeNodeVswitches(nodeIndex int) *vpc.Switch {
	nodeSize := len(a.cluster.Nodes)
	vSwitchSize := len(a.vSwitchs)
	if nodeSize <= vSwitchSize {
		return a.vSwitchs[nodeIndex%vSwitchSize]
	}
	interval := nodeSize / vSwitchSize
	return a.vSwitchs[(nodeIndex/interval)%vSwitchSize]
}
