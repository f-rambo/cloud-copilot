package infrastructure

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/f-rambo/ocean/internal/biz"
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
	lb              *alb.LoadBalancer
}

func Alicloud(cluster *biz.Cluster) *AlicloudCluster {
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

func (a *AlicloudCluster) Start(ctx *pulumi.Context) error {
	err := a.infrastructural(ctx)
	if err != nil {
		return errors.Wrap(err, "alicloud cluster init failed")
	}
	err = a.setImageByNodeGroups(ctx)
	if err != nil {
		return errors.Wrap(err, "set image by node groups failed")
	}
	err = a.setInstanceTypeByNodeGroups(ctx)
	if err != nil {
		return errors.Wrap(err, "set instance type by node groups failed")
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
	// create resource group
	res, err := resourcemanager.NewResourceGroup(ctx, alicloudResourceGroupName, &resourcemanager.ResourceGroupArgs{
		ResourceGroupName: pulumi.String(alicloudResourceGroupName),
		DisplayName:       pulumi.String(alicloudResourceGroupName),
	})
	if err != nil {
		return err
	}
	a.resourceGroupID = res.ID()

	// 创建角色/策略
	roleMap := map[string]string{
		"csPolicy":  alicloudCsPolicy,
		"ecsPolicy": alicloudEscPolicy,
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

	// create vpc
	network, err := vpc.NewNetwork(ctx, alicloudVpcName, &vpc.NetworkArgs{
		VpcName:   pulumi.String(alicloudVpcName),
		CidrBlock: pulumi.String("192.168.0.0/16"),
	})
	if err != nil {
		return err
	}
	a.vpcID = network.ID()

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

	vSwitchs := make([]*vpc.Switch, 0)
	for i, zoneId := range zoneIds {
		vSwitch, err := vpc.NewSwitch(ctx, fmt.Sprintf("%s-%d", alicloudVswitchName, i), &vpc.SwitchArgs{
			VswitchName: pulumi.String(fmt.Sprintf("%s-%d", alicloudVswitchName, i)),
			CidrBlock:   pulumi.String(fmt.Sprintf("192.168.%d.0/24", i)),
			VpcId:       a.vpcID,
			ZoneId:      pulumi.String(zoneId),
		})
		if err != nil {
			return err
		}
		vSwitchs = append(vSwitchs, vSwitch)
	}
	if len(vSwitchs) == 0 {
		return fmt.Errorf("no available vswitch found")
	}
	a.vSwitchs = vSwitchs

	// create security group
	group, err := ecs.NewSecurityGroup(ctx, alicloudEcsSecurityGroup, &ecs.SecurityGroupArgs{
		Name:        pulumi.String(alicloudEcsSecurityGroup),
		Description: pulumi.String("ocean ecs security group."),
		VpcId:       a.vpcID,
	})
	if err != nil {
		return err
	}
	a.sgID = group.ID()

	// sg rule: can add more rules
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

	// create eip
	eipAddress, err := ecs.NewEipAddress(ctx, alicloudNatEipName, &ecs.EipAddressArgs{
		AddressName:        pulumi.String(alicloudNatEipName),
		InternetChargeType: pulumi.String("PayByTraffic"),
	})
	if err != nil {
		return err
	}
	a.eipID = eipAddress.ID()

	// create nat gateway
	var vswitchId pulumi.StringInput
	for _, v := range vSwitchs {
		vswitchId = v.ID()
		break
	}
	natGateway, err := vpc.NewNatGateway(ctx, alicloudNatGatewayName, &vpc.NatGatewayArgs{
		VpcId:              a.vpcID,
		VswitchId:          vswitchId,
		NatGatewayName:     pulumi.String(alicloudNatGatewayName),
		InternetChargeType: pulumi.String("PayByTraffic"),
		NatType:            pulumi.String("Enhanced"),
	})
	if err != nil {
		return err
	}

	_, err = ecs.NewEipAssociation(ctx, alicloudNatGatewayEipAssociation, &ecs.EipAssociationArgs{
		AllocationId: a.eipID,
		InstanceId:   natGateway.ID(),
	})
	if err != nil {
		return err
	}

	// Import an existing public key to build a alicloud key pair
	_, err = ecs.NewKeyPair(ctx, alicloudKeyPairName, &ecs.KeyPairArgs{
		KeyName:   pulumi.String(alicloudKeyPairName),
		PublicKey: pulumi.String(a.cluster.PublicKey),
	})
	if err != nil {
		return err
	}

	return nil
}

func (a *AlicloudCluster) setImageByNodeGroups(ctx *pulumi.Context) error {
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

func (a *AlicloudCluster) setInstanceTypeByNodeGroups(ctx *pulumi.Context) error {
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

func (a *AlicloudCluster) nodes(ctx *pulumi.Context) error {
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
		instanceArgs := &ecs.InstanceArgs{
			HostName:                pulumi.String(node.Name),
			InstanceName:            pulumi.String(node.Name),
			AvailabilityZone:        vswitch.ZoneId,
			VswitchId:               vswitch.ID(),
			SecurityGroups:          pulumi.StringArray{a.sgID},
			InstanceType:            pulumi.String(nodeGroup.InstanceType),
			ImageId:                 pulumi.String(nodeGroup.Image),
			InternetMaxBandwidthOut: pulumi.Int(node.InternetMaxBandwidthOut),
			SystemDiskCategory:      pulumi.String("cloudEssd"),
			SystemDiskName:          pulumi.String(fmt.Sprintf("system_disk_%s", node.Name)),
			SystemDiskSize:          pulumi.Int(node.SystemDisk),
			KeyName:                 pulumi.String(alicloudKeyPairName),
			ResourceGroupId:         a.resourceGroupID,
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
		instance, err := ecs.NewInstance(ctx, node.Name, instanceArgs)
		if err != nil {
			return err
		}
		if node.Role == biz.NodeRoleMaster && !selectedBostionHost {
			selectedBostionHost = true
			// bind eip to instance
			_, err = ecs.NewEipAssociation(ctx, alicloudNatGatewayEipAssociation, &ecs.EipAssociationArgs{
				AllocationId: a.eipID,
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

func (a *AlicloudCluster) localBalancer(ctx *pulumi.Context) (err error) {
	zoneMappings := make(alb.LoadBalancerZoneMappingArray, 0)
	for _, v := range a.vSwitchs {
		zoneMappings = append(zoneMappings, &alb.LoadBalancerZoneMappingArgs{
			VswitchId: v.ID(),
			ZoneId:    v.ZoneId,
		})
	}
	a.lb, err = alb.NewLoadBalancer(ctx, alicloudSlbName, &alb.LoadBalancerArgs{
		LoadBalancerName: pulumi.String(alicloudSlbName),
		VpcId:            a.vpcID,
		AddressType:      pulumi.String("internet"),
		AddressIpVersion: pulumi.String("ipv4"),
		ZoneMappings:     zoneMappings,
	})
	if err != nil {
		return err
	}

	// Create Load Balancer Listener http
	_, err = alb.NewListener(ctx, alicloudSlbListenerName, &alb.ListenerArgs{
		LoadBalancerId:   a.lb.ID(),
		ListenerPort:     pulumi.Int(80),
		ListenerProtocol: pulumi.String("HTTP"),
	})
	if err != nil {
		return err
	}

	// Create Load Balancer Listener https
	_, err = alb.NewListener(ctx, alicloudSlbListenerName, &alb.ListenerArgs{
		LoadBalancerId:   a.lb.ID(),
		ListenerPort:     pulumi.Int(443),
		ListenerProtocol: pulumi.String("HTTPS"),
	})
	if err != nil {
		return err
	}

	// Load Balancer Listener k8s apiserver
	_, err = alb.NewListener(ctx, alicloudSlbListenerName, &alb.ListenerArgs{
		LoadBalancerId:   a.lb.ID(),
		ListenerPort:     pulumi.Int(6443),
		ListenerProtocol: pulumi.String("HTTPS"),
	})
	if err != nil {
		return err
	}

	return nil
}

func (a *AlicloudCluster) Import(ctx *pulumi.Context) error {
	instances, err := ecs.GetInstances(ctx, &ecs.GetInstancesArgs{
		Status: pulumi.StringRef("Running"),
	})
	if err != nil {
		return err
	}
	var vpcId, resourceGroupID, sgIDs, eipID string
	instanceTypes := make(map[string]struct{})
	for _, node := range a.cluster.Nodes {
		for _, instance := range instances.Instances {
			if node.InternalIP == instance.PrivateIp {
				node.InstanceID = instance.Id
				node.SubnetId = instance.VswitchId
				node.Zone = instance.AvailabilityZone
				node.ExternalIP = instance.PublicIp
				vpcId = instance.VpcId
				resourceGroupID = instance.ResourceGroupId
				sgIDs = strings.Join(instance.SecurityGroups, ",")
				if instance.Eip != "" {
					eipID = instance.Eip
				}
				instanceTypes[instance.InstanceType] = struct{}{}
				for _, v := range instance.DiskDeviceMappings {
					if v.Type == "system disk" {
						node.SystemDisk += int32(v.Size)
					}
					if v.Type == "data disk" {
						node.DataDisk += int32(v.Size)
					}
				}
				break
			}
		}
	}
	a.cluster.VpcID = vpcId
	a.cluster.ResourceGroupID = resourceGroupID
	a.cluster.SecurityGroupIDs = sgIDs
	a.cluster.ApiServerAddress = eipID
	nodeGroups := make([]*biz.NodeGroup, 0)
	for instanceType := range instanceTypes {
		nodeGroup := &biz.NodeGroup{}
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
		nodeGroups = append(nodeGroups, nodeGroup)
	}
	a.cluster.NodeGroups = nodeGroups
	return nil
}

func (a *AlicloudCluster) Clean(ctx *pulumi.Context) error {
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
