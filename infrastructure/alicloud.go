package infrastructure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs "github.com/alibabacloud-go/ecs-20140526/v6/client"
	slb "github.com/alibabacloud-go/slb-20140515/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	vpc "github.com/alibabacloud-go/vpc-20160428/v6/client"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

const (
	alicloudDefaultRegion = "cn-hangzhou"

	ALICLOUD_ACCESS_KEY     = "ALICLOUD_ACCESS_KEY"
	ALICLOUD_SECRET_KEY     = "ALICLOUD_SECRET_KEY"
	ALICLOUD_REGION         = "ALICLOUD_REGION"
	ALICLOUD_DEFAULT_REGION = "ALICLOUD_DEFAULT_REGION"
)

type AliCloudUsecase struct {
	c         *conf.Bootstrap
	log       *log.Helper
	vpcClient *vpc.Client
	ecsClient *ecs.Client
	slbClient *slb.Client
}

func NewAliCloudUseCase(c *conf.Bootstrap, logger log.Logger) *AliCloudUsecase {
	return &AliCloudUsecase{
		c:   c,
		log: log.NewHelper(logger),
	}
}

func (a *AliCloudUsecase) Connections(ctx context.Context, accessId, accessKey string, regionParam ...string) (err error) {
	var region string
	if len(regionParam) == 0 {
		region = alicloudDefaultRegion
	} else {
		region = regionParam[0]
	}
	os.Setenv(ALICLOUD_ACCESS_KEY, accessId)
	os.Setenv(ALICLOUD_SECRET_KEY, accessKey)
	os.Setenv(ALICLOUD_REGION, region)
	os.Setenv(ALICLOUD_DEFAULT_REGION, region)
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessId),
		AccessKeySecret: tea.String(accessKey),
		RegionId:        tea.String(region),
	}
	a.vpcClient, err = vpc.NewClient(config)
	if err != nil {
		return errors.Wrap(err, "failed to create vpc client")
	}
	a.ecsClient, err = ecs.NewClient(config)
	if err != nil {
		return errors.Wrap(err, "failed to create ecs client")
	}
	a.slbClient, err = slb.NewClient(config)
	if err != nil {
		return errors.Wrap(err, "failed to create slb client")
	}
	return nil
}

func (a *AliCloudUsecase) GetAvailabilityRegions(ctx context.Context) ([]*biz.CloudResource, error) {
	res, err := a.ecsClient.DescribeRegions(&ecs.DescribeRegionsRequest{
		AcceptLanguage:     tea.String("zh-CN"),
		InstanceChargeType: tea.String("PostPaid"),
		ResourceType:       tea.String("instance"),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to describe regions")
	}
	cloudResources := make([]*biz.CloudResource, 0)
	for _, v := range res.Body.Regions.Region {
		if tea.StringValue(v.Status) != "available" {
			continue
		}
		cloudResources = append(cloudResources, &biz.CloudResource{
			Type:  biz.ResourceType_REGION,
			RefId: tea.StringValue(v.RegionId),
			Name:  tea.StringValue(v.LocalName),
			Value: tea.StringValue(v.RegionEndpoint),
		})
	}
	return cloudResources, nil
}

func (a *AliCloudUsecase) GetAvailabilityZones(ctx context.Context, cluster *biz.Cluster) ([]*biz.CloudResource, error) {
	zonesRes, err := a.ecsClient.DescribeZones(&ecs.DescribeZonesRequest{
		AcceptLanguage:     tea.String("en-US"),
		RegionId:           tea.String(cluster.Region),
		InstanceChargeType: tea.String("PostPaid"),
		SpotStrategy:       tea.String("NoSpot"),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to describe zones")
	}
	if len(zonesRes.Body.Zones.Zone) == 0 {
		return nil, errors.New("no availability zones found")
	}
	zones := make([]*ecs.DescribeZonesResponseBodyZonesZone, 0)
	for _, zone := range zonesRes.Body.Zones.Zone {
		if tea.StringValue(zone.ZoneType) != "AvailabilityZone" {
			continue
		}
		zoneResourceType := tea.StringSliceValue(zone.AvailableResourceCreation.ResourceTypes)

		if !slices.Contains(zoneResourceType, "VSwitch") {
			continue
		}
		if !slices.Contains(zoneResourceType, "IoOptimized") {
			continue
		}
		if !slices.Contains(zoneResourceType, "Instance") {
			continue
		}
		if !slices.Contains(zoneResourceType, "Disk") {
			continue
		}
		if !slices.Contains(zoneResourceType, "DedicatedHost") {
			continue
		}
		zones = append(zones, zone)

	}
	gatewayAvailableZones, err := a.vpcClient.ListEnhanhcedNatGatewayAvailableZones(&vpc.ListEnhanhcedNatGatewayAvailableZonesRequest{
		RegionId:       tea.String(cluster.Region),
		AcceptLanguage: tea.String("en-US"),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list nat gateway available zones")
	}
	clusterResouces := make([]*biz.CloudResource, 0)
	for _, zone := range zones {
		gatewayOk := false
		for _, gatewayZone := range gatewayAvailableZones.Body.Zones {
			if tea.StringValue(gatewayZone.ZoneId) == tea.StringValue(zone.ZoneId) {
				gatewayOk = true
				break
			}
		}
		if gatewayOk {
			clusterResouces = append(clusterResouces, &biz.CloudResource{
				RefId: tea.StringValue(zone.ZoneId),
				Name:  tea.StringValue(zone.LocalName),
				Type:  biz.ResourceType_AVAILABILITY_ZONES,
				Value: os.Getenv(ALICLOUD_REGION),
			})
		}
	}
	return clusterResouces, nil
}

func (a *AliCloudUsecase) CreateNetwork(ctx context.Context, cluster *biz.Cluster) error {
	fs := []func(context.Context, *biz.Cluster) error{
		a.createVPC,
		a.createSubnets,
		a.createEips,
		a.createNatGateways,
		a.createRouteTables,
	}
	for _, f := range fs {
		if err := f(ctx, cluster); err != nil {
			return err
		}
	}
	return nil
}

func (a *AliCloudUsecase) ImportKeyPair(ctx context.Context, cluster *biz.Cluster) error {
	keyPairName := cluster.GetkeyPairName()
	if cluster.GetCloudResourceByName(biz.ResourceType_KEY_PAIR, keyPairName) != nil {
		a.log.Infof("key pair %s already exists", keyPairName)
		return nil
	}

	// List existing key pairs
	var pageNumber int32 = 1
	for {
		keyPairs, err := a.ecsClient.DescribeKeyPairs(&ecs.DescribeKeyPairsRequest{
			RegionId:    tea.String(cluster.Region),
			PageNumber:  tea.Int32(pageNumber),
			PageSize:    tea.Int32(50),
			KeyPairName: tea.String(keyPairName),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe key pairs")
		}

		for _, kp := range keyPairs.Body.KeyPairs.KeyPair {
			if tea.StringValue(kp.KeyPairName) == keyPairName {
				if cluster.GetCloudResourceByRefID(biz.ResourceType_KEY_PAIR, tea.StringValue(kp.KeyPairName)) != nil {
					continue
				}
				cluster.AddCloudResource(&biz.CloudResource{
					Name:  tea.StringValue(kp.KeyPairName),
					RefId: tea.StringValue(kp.KeyPairName),
					Type:  biz.ResourceType_KEY_PAIR,
				})
				a.log.Infof("key pair %s already exists", keyPairName)
				return nil
			}
		}

		if len(keyPairs.Body.KeyPairs.KeyPair) < 50 {
			break
		}
		pageNumber++
	}

	// Import key pair
	importReq := &ecs.ImportKeyPairRequest{
		RegionId:      tea.String(cluster.Region),
		KeyPairName:   tea.String(keyPairName),
		PublicKeyBody: tea.String(cluster.PublicKey),
	}

	importRes, err := a.ecsClient.ImportKeyPair(importReq)
	if err != nil {
		return errors.Wrap(err, "failed to import key pair")
	}

	// Add tags to key pair
	err = a.createEcsTag(cluster.Region, tea.StringValue(importRes.Body.KeyPairName), "keypair", map[biz.ResourceTypeKeyValue]any{biz.ResourceTypeKeyValue_NAME: keyPairName})
	if err != nil {
		return errors.Wrap(err, "failed to tag key pair")
	}

	// Add to cluster resources
	cluster.AddCloudResource(&biz.CloudResource{
		Name:  keyPairName,
		RefId: tea.StringValue(importRes.Body.KeyPairName),
		Type:  biz.ResourceType_KEY_PAIR,
		Tags:  cluster.EncodeTags(map[biz.ResourceTypeKeyValue]any{biz.ResourceTypeKeyValue_NAME: keyPairName}),
	})

	a.log.Infof("key pair %s imported successfully", keyPairName)
	return nil
}

func (a *AliCloudUsecase) DeleteKeyPair(ctx context.Context, cluster *biz.Cluster) error {
	// Get key pair from cluster resources
	keyPairName := cluster.GetkeyPairName()
	keyPair := cluster.GetCloudResourceByName(biz.ResourceType_KEY_PAIR, keyPairName)
	if keyPair == nil {
		a.log.Infof("key pair %s not found", keyPairName)
		return nil
	}

	res, err := a.ecsClient.DescribeKeyPairs(&ecs.DescribeKeyPairsRequest{
		RegionId:    tea.String(cluster.Region),
		KeyPairName: tea.String(keyPairName),
		PageNumber:  tea.Int32(1),
		PageSize:    tea.Int32(1),
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe key pairs")
	}
	if tea.Int32Value(res.Body.TotalCount) == 0 {
		cluster.DeleteCloudResource(biz.ResourceType_KEY_PAIR)
		a.log.Infof("key pair %s deleted successfully", keyPairName)
		return nil
	}

	// Delete key pair
	_, err = a.ecsClient.DeleteKeyPairs(&ecs.DeleteKeyPairsRequest{
		RegionId:     tea.String(cluster.Region),
		KeyPairNames: tea.String(fmt.Sprintf("[\"%s\"]", keyPairName)),
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete key pair")
	}

	// Remove from cluster resources
	cluster.DeleteCloudResource(biz.ResourceType_KEY_PAIR)
	a.log.Infof("key pair %s deleted successfully", keyPairName)
	return nil
}

func (a *AliCloudUsecase) checkingInstanceInventory(regionId, zoneId, instanceTypeId string) (bool, error) {
	res, err := a.ecsClient.DescribeAvailableResource(&ecs.DescribeAvailableResourceRequest{
		RegionId:            tea.String(regionId),
		ZoneId:              tea.String(zoneId),
		InstanceChargeType:  tea.String("PostPaid"),
		InstanceType:        tea.String(instanceTypeId),
		DestinationResource: tea.String("InstanceType"),
		IoOptimized:         tea.String("optimized"),
		SystemDiskCategory:  tea.String("cloud_ssd"),
		DataDiskCategory:    tea.String("cloud_ssd"),
		NetworkCategory:     tea.String("vpc"),
		ResourceType:        tea.String("instance"),
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to describe available resource")
	}
	if res.Body.AvailableZones == nil || len(res.Body.AvailableZones.AvailableZone) == 0 {
		return false, nil
	}
	for _, v := range res.Body.AvailableZones.AvailableZone {
		if tea.StringValue(v.Status) == "Available" && tea.StringValue(v.StatusCategory) == "WithStock" {
			for _, vv := range v.AvailableResources.AvailableResource {
				if tea.StringValue(vv.Type) != "InstanceType" {
					continue
				}
				for _, vvv := range vv.SupportedResources.SupportedResource {
					if tea.StringValue(vvv.Value) == instanceTypeId && tea.StringValue(v.Status) == "Available" && tea.StringValue(v.StatusCategory) == "WithStock" {
						return true, nil
					}
				}
			}
		}
	}
	return true, nil
}

func (a *AliCloudUsecase) ManageInstance(ctx context.Context, cluster *biz.Cluster) error {
	vpc := cluster.GetSingleCloudResource(biz.ResourceType_VPC)
	if vpc == nil {
		return errors.New("vpc not found")
	}
	sg := cluster.GetSingleCloudResource(biz.ResourceType_SECURITY_GROUP)
	if sg == nil {
		return errors.New("security group not found")
	}
	keyPair := cluster.GetSingleCloudResource(biz.ResourceType_KEY_PAIR)
	if keyPair == nil {
		return errors.New("key pair not found")
	}
	// find all instances in the current vpc
	instances := make([]*ecs.DescribeInstancesResponseBodyInstancesInstance, 0)
	pageNumber := 1
	for {
		instancesRes, err := a.ecsClient.DescribeInstances(&ecs.DescribeInstancesRequest{
			RegionId:   tea.String(cluster.Region),
			VpcId:      tea.String(vpc.RefId),
			PageNumber: tea.Int32(int32(pageNumber)),
			PageSize:   tea.Int32(50),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe instances")
		}
		instances = append(instances, instancesRes.Body.Instances.Instance...)
		if len(instancesRes.Body.Instances.Instance) < 50 {
			break
		}
		pageNumber++
	}
	// clear history nodes
	for _, node := range cluster.Nodes {
		nodeExits := false
		for _, instance := range instances {
			if node.InstanceId == tea.StringValue(instance.InstanceId) {
				nodeExits = true
				break
			}
		}
		if !nodeExits && (node.Status == biz.NodeStatus_NODE_RUNNING || node.Status == biz.NodeStatus_NODE_PENDING) {
			node.InstanceId = ""
		}
	}

	// handler need delete instances
	needDeleteInstanceIDs := make([]string, 0)
	for _, node := range cluster.Nodes {
		if node.Status == biz.NodeStatus_NODE_DELETING && node.InstanceId != "" {
			needDeleteInstanceIDs = append(needDeleteInstanceIDs, node.InstanceId)
		}
	}
	deleteInstanceIDs := make([]string, 0)
	for _, instance := range instances {
		if slices.Contains(needDeleteInstanceIDs, tea.StringValue(instance.InstanceId)) {
			deleteInstanceIDs = append(deleteInstanceIDs, tea.StringValue(instance.InstanceId))
		}
	}
	if len(deleteInstanceIDs) > 0 {
		_, err := a.ecsClient.DeleteInstances(&ecs.DeleteInstancesRequest{
			RegionId:   tea.String(cluster.Region),
			InstanceId: tea.StringSlice(deleteInstanceIDs),
			Force:      tea.Bool(true),
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete instances")
		}
	}

	// Create instances
	for _, nodeGroup := range cluster.NodeGroups {
		for index, node := range cluster.Nodes {
			if node.Status != biz.NodeStatus_NODE_CREATING || node.NodeGroupId != nodeGroup.Id || node.InstanceId != "" {
				continue
			}
			privateSubnet := cluster.DistributeNodePrivateSubnets(index)
			if privateSubnet == nil {
				return errors.New("no private subnet found")
			}
			privateSubnetTagsMap := cluster.DecodeTags(privateSubnet.Tags)
			zoneId := cast.ToString(privateSubnetTagsMap[biz.ResourceTypeKeyValue_ZONE_ID])
			// check instance inventory
			ok, err := a.checkingInstanceInventory(cluster.Region, zoneId, node.InstanceType)
			if err != nil {
				return err
			}
			if !ok {
				for _, instanceId := range strings.Split(node.BackupInstanceIds, ",") {
					ok, err = a.checkingInstanceInventory(cluster.Region, zoneId, instanceId)
					if err != nil {
						return err
					}
					if ok {
						node.InstanceId = instanceId
						break
					}
				}
			}
			if !ok {
				node.ErrorType = biz.NodeErrorType_INFRASTRUCTURE_ERROR
				node.ErrorMessage = "INSUFFICIENT INVENTORY"
				continue
			}
			createInstanceRequest := &ecs.CreateInstanceRequest{
				InstanceChargeType: tea.String("PostPaid"),
				RegionId:           tea.String(cluster.Region),
				KeyPairName:        tea.String(keyPair.Name),
				SecurityGroupId:    tea.String(sg.RefId),
				ImageId:            tea.String(node.ImageId),
				InstanceType:       tea.String(node.InstanceType),
				VSwitchId:          tea.String(privateSubnet.RefId),
				SystemDisk: &ecs.CreateInstanceRequestSystemDisk{
					Category: tea.String("cloud_ssd"),
					Size:     tea.Int32(node.SystemDiskSize),
				},
			}
			if cluster.Status == biz.ClusterStatus_STARTING && node.Role == biz.NodeRole_MASTER {
				realInstallShell, realInstallShellErr := getRealInstallShell(a.c.Infrastructure.Shell, cluster)
				if realInstallShellErr != nil {
					return realInstallShellErr
				}
				installShell, readShellErr := os.ReadFile(realInstallShell)
				if readShellErr != nil {
					return readShellErr
				}
				createInstanceRequest.UserData = tea.String(base64.StdEncoding.EncodeToString(installShell))
			}
			createInstanceRes, err := a.ecsClient.CreateInstance(createInstanceRequest)
			if err != nil {
				node.ErrorType = biz.NodeErrorType_INFRASTRUCTURE_ERROR
				node.ErrorMessage = "CREATE FAILURE"
				return errors.Wrap(err, "failed to create instance")
			}
			node.InstanceId = tea.StringValue(createInstanceRes.Body.InstanceId)
			if nodeGroup.NodePrice < tea.Float32Value(createInstanceRes.Body.TradePrice) {
				nodeGroup.NodePrice = tea.Float32Value(createInstanceRes.Body.TradePrice)
			}
			a.log.Infof("instance %s creating", tea.StringValue(createInstanceRes.Body.InstanceId))
		}
	}

	// wait instance status to be running
	needWatiInstanceIds := make([]string, 0)
	for _, node := range cluster.Nodes {
		if node.Status == biz.NodeStatus_NODE_CREATING && node.InstanceId != "" {
			needWatiInstanceIds = append(needWatiInstanceIds, node.InstanceId)
		}
	}
	timeOutNumber := 0
	instanceCount := len(needWatiInstanceIds)
	finishNumber := 0
	instanceMap := make(map[string]bool)
	for {
		if finishNumber >= instanceCount {
			break
		}
		if timeOutNumber > TimeOutCountNumber*instanceCount {
			break
		}
		time.Sleep(time.Second * TimeOutSecond)
		timeOutNumber++
		var pageNumber int32 = 1
		instanceStatusData := make([]*ecs.DescribeInstanceStatusResponseBodyInstanceStatusesInstanceStatus, 0)
		for {
			instanceStatus, err := a.ecsClient.DescribeInstanceStatus(&ecs.DescribeInstanceStatusRequest{
				RegionId:   tea.String(cluster.Region),
				InstanceId: tea.StringSlice(needWatiInstanceIds),
				PageNumber: tea.Int32(pageNumber),
				PageSize:   tea.Int32(50),
			})
			if err != nil {
				return errors.Wrap(err, "failed to describe instance status")
			}
			instanceStatusData = append(instanceStatusData, instanceStatus.Body.InstanceStatuses.InstanceStatus...)
			if len(instanceStatus.Body.InstanceStatuses.InstanceStatus) < 50 {
				break
			}
			pageNumber += 1
		}

		for _, instanceStatus := range instanceStatusData {
			if tea.StringValue(instanceStatus.Status) == "Stopped" {
				_, err := a.ecsClient.StartInstance(&ecs.StartInstanceRequest{
					InstanceId: instanceStatus.InstanceId,
				})
				if err != nil {
					return errors.Wrap(err, "failed to start instance")
				}
				a.log.Infof("instance %s starting", tea.StringValue(instanceStatus.InstanceId))
				time.Sleep(time.Second)
			}
			if tea.StringValue(instanceStatus.Status) == "Running" {
				finishNumber += 1
				instanceMap[tea.StringValue(instanceStatus.InstanceId)] = true
				a.log.Infof("instance %s created successfully", tea.StringValue(instanceStatus.InstanceId))
			}
		}
	}

	for _, node := range cluster.Nodes {
		if node.Status != biz.NodeStatus_NODE_CREATING || node.InstanceId == "" {
			continue
		}
		if _, ok := instanceMap[node.InstanceId]; !ok {
			node.ErrorType = biz.NodeErrorType_INFRASTRUCTURE_ERROR
			node.ErrorMessage = "START TIMEOUT"
			continue
		}
		netWorkInterface, err := a.ecsClient.DescribeNetworkInterfaces(&ecs.DescribeNetworkInterfacesRequest{
			RegionId:   tea.String(cluster.Region),
			InstanceId: tea.String(node.InstanceId),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe instance attribute")
		}
		if len(netWorkInterface.Body.NetworkInterfaceSets.NetworkInterfaceSet) == 0 {
			return errors.New("network interface not found")
		}
		node.Ip = tea.StringValue(netWorkInterface.Body.NetworkInterfaceSets.NetworkInterfaceSet[0].PrivateIpAddress)
		node.User = "root"
		time.Sleep(time.Second)
	}
	return nil
}

func (a *AliCloudUsecase) DeleteNetwork(ctx context.Context, cluster *biz.Cluster) error {
	// Delete SLB
	for _, v := range cluster.GetCloudResource(biz.ResourceType_LOAD_BALANCER) {
		res, err := a.slbClient.DescribeLoadBalancers(&slb.DescribeLoadBalancersRequest{
			RegionId:       tea.String(cluster.Region),
			LoadBalancerId: tea.String(v.RefId),
			PageNumber:     tea.Int32(1),
			PageSize:       tea.Int32(1),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe load balancers")
		}
		if tea.Int32Value(res.Body.TotalCount) == 0 {
			cluster.DeleteCloudResourceByID(biz.ResourceType_LOAD_BALANCER, v.Id)
			continue
		}
		_, err = a.slbClient.DeleteLoadBalancer(&slb.DeleteLoadBalancerRequest{
			RegionId:       tea.String(cluster.Region),
			LoadBalancerId: tea.String(v.RefId),
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete SLB")
		}
		cluster.DeleteCloudResourceByID(biz.ResourceType_LOAD_BALANCER, v.Id)
	}
	// delete sg
	for _, sg := range cluster.GetCloudResource(biz.ResourceType_SECURITY_GROUP) {
		res, err := a.ecsClient.DescribeSecurityGroups(&ecs.DescribeSecurityGroupsRequest{
			RegionId:        tea.String(cluster.Region),
			SecurityGroupId: tea.String(sg.RefId),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe security group")
		}
		if tea.Int32Value(res.Body.TotalCount) == 0 {
			cluster.DeleteCloudResourceByID(biz.ResourceType_SECURITY_GROUP, sg.Id)
			continue
		}
		_, err = a.ecsClient.DeleteSecurityGroup(&ecs.DeleteSecurityGroupRequest{
			RegionId:        tea.String(cluster.Region),
			SecurityGroupId: tea.String(sg.RefId),
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete security group")
		}
		cluster.DeleteCloudResourceByID(biz.ResourceType_SECURITY_GROUP, sg.Id)
	}
	// delete eip
	eipIds := make([]string, 0)
	for _, eip := range cluster.GetCloudResource(biz.ResourceType_ELASTIC_IP) {
		res, err := a.vpcClient.DescribeEipAddresses(&vpc.DescribeEipAddressesRequest{
			RegionId:     tea.String(cluster.Region),
			AllocationId: tea.String(eip.RefId),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe eip")
		}
		if tea.Int32Value(res.Body.TotalCount) == 0 {
			cluster.DeleteCloudResourceByID(biz.ResourceType_ELASTIC_IP, eip.Id)
			continue
		}

		for _, eipAddress := range res.Body.EipAddresses.EipAddress {
			_, err = a.vpcClient.UnassociateEipAddress(&vpc.UnassociateEipAddressRequest{
				RegionId:     tea.String(cluster.Region),
				AllocationId: tea.String(eip.RefId),
				InstanceId:   eipAddress.InstanceId,
				Force:        tea.Bool(true),
			})
			if err != nil {
				a.log.Warnf("failed to disassociate EIP %s: %v", eip.RefId, err)
			}
		}
		eipIds = append(eipIds, eip.RefId)
	}
	// wait eip status to be available
	timeOutNumber := 0
	eipsOk := false
	for {
		if timeOutNumber > TimeOutCountNumber || eipsOk {
			break
		}
		time.Sleep(time.Second * TimeOutSecond)
		timeOutNumber++
		res, err := a.vpcClient.DescribeEipAddresses(&vpc.DescribeEipAddressesRequest{
			RegionId:     tea.String(cluster.Region),
			Status:       tea.String("Available"),
			AllocationId: tea.String(strings.Join(eipIds, ",")),
			PageNumber:   tea.Int32(1),
			PageSize:     tea.Int32(50),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe nat gateway")
		}
		if tea.Int32Value(res.Body.TotalCount) == int32(len(eipIds)) {
			eipsOk = true
			break
		}
	}
	if !eipsOk {
		return errors.New("eip delete failed")
	}

	// Release EIP
	for _, eipid := range eipIds {
		_, err := a.vpcClient.ReleaseEipAddress(&vpc.ReleaseEipAddressRequest{
			RegionId:     tea.String(cluster.Region),
			AllocationId: tea.String(eipid),
		})
		if err != nil {
			a.log.Warnf("failed to release EIP %s: %v", eipid, err)
		}
	}
	cluster.DeleteCloudResource(biz.ResourceType_ELASTIC_IP)
	time.Sleep(time.Second * TimeOutSecond)

	// Delete NAT Gateways
	for _, nat := range cluster.GetCloudResource(biz.ResourceType_NAT_GATEWAY) {
		// Delete NAT Gateway
		_, err := a.vpcClient.DeleteNatGateway(&vpc.DeleteNatGatewayRequest{
			RegionId:     tea.String(cluster.Region),
			NatGatewayId: tea.String(nat.RefId),
			Force:        tea.Bool(true),
		})
		if err != nil {
			a.log.Warnf("failed to delete NAT Gateway %s: %v", nat.RefId, err)
		}
		time.Sleep(time.Second * 5 * TimeOutSecond)
		cluster.DeleteCloudResourceByID(biz.ResourceType_NAT_GATEWAY, nat.Id)
	}

	// Delete Route Tables
	for _, rt := range cluster.GetCloudResource(biz.ResourceType_ROUTE_TABLE) {
		routeTables, err := a.vpcClient.DescribeRouteTableList(&vpc.DescribeRouteTableListRequest{
			RegionId:     tea.String(cluster.Region),
			RouteTableId: tea.String(rt.RefId),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe route table")
		}
		for _, v := range routeTables.Body.RouterTableList.RouterTableListType {
			if v.VSwitchIds == nil {
				continue
			}
			for _, vs := range v.VSwitchIds.VSwitchId {
				_, err = a.vpcClient.UnassociateRouteTable(&vpc.UnassociateRouteTableRequest{
					RegionId:     tea.String(cluster.Region),
					RouteTableId: tea.String(rt.RefId),
					VSwitchId:    vs,
				})
				if err != nil {
					return errors.Wrap(err, "failed to unassociate route table")
				}
			}
		}
		_, err = a.vpcClient.DeleteRouteTable(&vpc.DeleteRouteTableRequest{
			RegionId:     tea.String(cluster.Region),
			RouteTableId: tea.String(rt.RefId),
		})
		if err != nil {
			a.log.Warnf("failed to delete route table %s: %v", rt.RefId, err)
		}
		cluster.DeleteCloudResourceByID(biz.ResourceType_ROUTE_TABLE, rt.Id)
	}

	// Delete VSwitches (Subnets)
	vswitches := cluster.GetCloudResource(biz.ResourceType_SUBNET)
	for _, vsw := range vswitches {
		_, err := a.vpcClient.DeleteVSwitch(&vpc.DeleteVSwitchRequest{
			RegionId:  tea.String(cluster.Region),
			VSwitchId: tea.String(vsw.RefId),
		})
		if err != nil {
			a.log.Warnf("failed to delete VSwitch %s: %v", vsw.RefId, err)
		}
		time.Sleep(time.Second)
		cluster.DeleteCloudResourceByID(biz.ResourceType_SUBNET, vsw.Id)
	}

	// Delete VPC
	vpcRes := cluster.GetSingleCloudResource(biz.ResourceType_VPC)
	if vpcRes != nil {
		_, err := a.vpcClient.DeleteVpc(&vpc.DeleteVpcRequest{
			RegionId: tea.String(cluster.Region),
			VpcId:    tea.String(vpcRes.RefId),
		})
		if err != nil {
			a.log.Warnf("failed to delete VPC %s: %v", vpcRes.RefId, err)
		}
		cluster.DeleteCloudResource(biz.ResourceType_VPC)
	}
	return nil
}

func (a *AliCloudUsecase) createVPC(ctx context.Context, cluster *biz.Cluster) error {
	vpcs := make([]*vpc.DescribeVpcsResponseBodyVpcsVpc, 0)
	pageNumber := 1
	for {
		vpcsRes, err := a.vpcClient.DescribeVpcs(&vpc.DescribeVpcsRequest{
			RegionId:   tea.String(cluster.Region),
			PageNumber: tea.Int32(int32(pageNumber)),
			PageSize:   tea.Int32(50),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe VPCs")
		}
		vpcs = append(vpcs, vpcsRes.Body.Vpcs.Vpc...)
		if len(vpcsRes.Body.Vpcs.Vpc) < 50 {
			break
		}
		pageNumber++
	}
	for _, vpc := range vpcs {
		if v := cluster.GetCloudResourceByRefID(biz.ResourceType_VPC, tea.StringValue(vpc.VpcId)); v != nil {
			a.log.Info("vpc already exists ", "vpc ", v.Name)
			return nil
		}
	}
	if len(cluster.GetCloudResource(biz.ResourceType_VPC)) > 0 {
		cluster.DeleteCloudResource(biz.ResourceType_VPC)
	}

	vpcName := cluster.GetVpcName()
	vpcTags := cluster.GetTags()
	vpcTags[biz.ResourceTypeKeyValue_NAME] = vpcName
	for _, vpc := range vpcs {
		if len(cluster.GetCloudResource(biz.ResourceType_VPC)) > 0 {
			return nil
		}
		if tea.StringValue(vpc.CidrBlock) != cluster.VpcCidr {
			continue
		}
		a.createVpcTags(cluster.Region, tea.StringValue(vpc.VpcId), "VPC", vpcTags)
		cluster.AddCloudResource(&biz.CloudResource{
			RefId: tea.StringValue(vpc.VpcId),
			Name:  vpcName,
			Type:  biz.ResourceType_VPC,
			Tags:  cluster.EncodeTags(vpcTags),
		})
		a.log.Infof("vpc %s already exists", vpcName)
	}
	if len(cluster.GetCloudResource(biz.ResourceType_VPC)) > 0 {
		return nil
	}
	vpcResponce, err := a.vpcClient.CreateVpc(&vpc.CreateVpcRequest{
		VpcName:   tea.String(cluster.Name + "-vpc"),
		RegionId:  tea.String(cluster.Region),
		CidrBlock: tea.String(cluster.VpcCidr),
	})
	err = a.handlerError(err)
	if err != nil {
		return err
	}
	// wait vpc status to be available
	timeOutNumber := 0
	vpcOk := false
	for {
		if timeOutNumber > TimeOutCountNumber || vpcOk {
			break
		}
		time.Sleep(time.Second * TimeOutSecond)
		timeOutNumber++
		res, err := a.vpcClient.DescribeVpcs(&vpc.DescribeVpcsRequest{
			RegionId:   tea.String(cluster.Region),
			VpcId:      vpcResponce.Body.VpcId,
			PageNumber: tea.Int32(1),
			PageSize:   tea.Int32(10),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe nat gateway")
		}
		for _, v := range res.Body.Vpcs.Vpc {
			if tea.StringValue(v.Status) == "Available" {
				vpcOk = true
				break
			}
		}
	}
	if !vpcOk {
		return errors.New("vpc not available")
	}
	cluster.AddCloudResource(&biz.CloudResource{
		RefId: tea.StringValue(vpcResponce.Body.VpcId),
		Name:  vpcName,
		Type:  biz.ResourceType_VPC,
	})
	a.log.Infof("vpc %s created", vpcName)
	return nil
}

func (a *AliCloudUsecase) createSubnets(ctx context.Context, cluster *biz.Cluster) error {
	vpcRes := cluster.GetSingleCloudResource(biz.ResourceType_VPC)
	if vpcRes == nil {
		return errors.New("vpc not found")
	}
	subnets := make([]*vpc.DescribeVSwitchesResponseBodyVSwitchesVSwitch, 0)
	pageNumber := 1
	for {
		existingSubnetRes, err := a.vpcClient.DescribeVSwitches(&vpc.DescribeVSwitchesRequest{
			VpcId:      tea.String(vpcRes.RefId),
			PageNumber: tea.Int32(int32(pageNumber)),
			PageSize:   tea.Int32(50),
		})
		if err != nil || tea.Int32Value(existingSubnetRes.StatusCode) != http.StatusOK {
			return err
		}
		subnets = append(subnets, existingSubnetRes.Body.VSwitches.VSwitch...)
		if len(existingSubnetRes.Body.VSwitches.VSwitch) < 50 {
			break
		}
		pageNumber++
	}

	// clear history subnet
	for _, subnetCloudResource := range cluster.GetCloudResource(biz.ResourceType_SUBNET) {
		subnetCloudResourceExits := false
		for _, subnet := range subnets {
			if subnetCloudResource.RefId == tea.StringValue(subnet.VSwitchId) {
				subnetCloudResourceExits = true
				break
			}
		}
		if !subnetCloudResourceExits {
			cluster.DeleteCloudResourceByRefID(biz.ResourceType_SUBNET, subnetCloudResource.RefId)
		}
	}

	// One subnet for one available zone
	subnetExitsCidrs := make([]string, 0)
	zoneSubnets := make(map[string]*vpc.DescribeVSwitchesResponseBodyVSwitchesVSwitch)
	for _, subnet := range subnets {
		if subnet.CidrBlock == nil || subnet.VSwitchId == nil {
			continue
		}
		subnetExitsCidrs = append(subnetExitsCidrs, tea.StringValue(subnet.CidrBlock))
		if subnet.ZoneId == nil {
			continue
		}
		if _, ok := zoneSubnets[tea.StringValue(subnet.ZoneId)]; ok {
			continue
		}
		zoneSubnets[tea.StringValue(subnet.ZoneId)] = subnet
	}
	for zoneId, subnet := range zoneSubnets {
		if cluster.GetCloudResourceByRefID(biz.ResourceType_SUBNET, tea.StringValue(subnet.VSwitchId)) != nil {
			a.log.Infof("subnet %s already exists", tea.StringValue(subnet.VSwitchId))
			continue
		}
		name := cluster.GetSubnetName(zoneId)
		tags := cluster.GetTags()
		tags[biz.ResourceTypeKeyValue_ZONE_ID] = zoneId
		tags[biz.ResourceTypeKeyValue_ACCESS] = biz.ResourceTypeKeyValue_ACCESS_PRIVATE
		tags[biz.ResourceTypeKeyValue_NAME] = name
		a.createVpcTags(cluster.Region, tea.StringValue(subnet.VSwitchId), "VSWITCH", tags)
		cluster.AddCloudResource(&biz.CloudResource{
			Name:  name,
			RefId: tea.StringValue(subnet.VSwitchId),
			Tags:  cluster.EncodeTags(tags),
			Type:  biz.ResourceType_SUBNET,
			Value: tea.StringValue(subnet.CidrBlock),
		})
		a.log.Infof("subnet %s already exists", name)
	}

	// Create subnets
	for _, zone := range cluster.GetCloudResource(biz.ResourceType_AVAILABILITY_ZONES) {
		name := cluster.GetSubnetName(zone.RefId)
		tags := cluster.GetTags()
		tags[biz.ResourceTypeKeyValue_NAME] = name
		tags[biz.ResourceTypeKeyValue_ACCESS] = biz.ResourceTypeKeyValue_ACCESS_PRIVATE
		tags[biz.ResourceTypeKeyValue_ZONE_ID] = zone.RefId
		if cluster.GetCloudResourceByTags(biz.ResourceType_SUBNET, map[biz.ResourceTypeKeyValue]any{biz.ResourceTypeKeyValue_NAME: name}) != nil {
			continue
		}
		cidr, err := utils.GenerateSubnet(cluster.VpcCidr, subnetExitsCidrs)
		if err != nil {
			return err
		}
		subnetExitsCidrs = append(subnetExitsCidrs, cidr)
		privateSubnetTags := make([]*vpc.CreateVSwitchRequestTag, 0)
		for k, v := range tags {
			privateSubnetTags = append(privateSubnetTags, &vpc.CreateVSwitchRequestTag{
				Key:   tea.String(k.String()),
				Value: tea.String(cast.ToString(v)),
			})
		}
		subnetOutput, err := a.vpcClient.CreateVSwitch(&vpc.CreateVSwitchRequest{
			VSwitchName: tea.String(name),
			RegionId:    tea.String(cluster.Region),
			VpcId:       tea.String(vpcRes.RefId),
			CidrBlock:   tea.String(cidr),
			ZoneId:      tea.String(zone.RefId),
			Tag:         privateSubnetTags,
		})
		if err != nil {
			return errors.Wrap(err, "failed to create private subnet")
		}
		cluster.AddCloudResource(&biz.CloudResource{
			Name:         name,
			RefId:        tea.StringValue(subnetOutput.Body.VSwitchId),
			AssociatedId: vpcRes.RefId,
			Tags:         cluster.EncodeTags(tags),
			Type:         biz.ResourceType_SUBNET,
			Value:        cidr,
		})
		a.log.Infof("private subnet %s created", name)
		time.Sleep(time.Second * TimeOutSecond)
	}
	return nil
}

func (a *AliCloudUsecase) createEips(_ context.Context, cluster *biz.Cluster) error {
	// Get Elastic IP
	eips := make([]*vpc.DescribeEipAddressesResponseBodyEipAddressesEipAddress, 0)
	var pageNumber int32 = 1
	for {
		eipRes, err := a.vpcClient.DescribeEipAddresses(&vpc.DescribeEipAddressesRequest{
			RegionId:   tea.String(cluster.Region),
			PageNumber: tea.Int32(pageNumber),
			PageSize:   tea.Int32(50),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe eip addresses")
		}
		if eipRes.Body.EipAddresses == nil {
			break
		}
		eips = append(eips, eipRes.Body.EipAddresses.EipAddress...)
		if len(eipRes.Body.EipAddresses.EipAddress) < 50 {
			break
		}
		pageNumber++
	}

	for _, eipResource := range cluster.GetCloudResource(biz.ResourceType_ELASTIC_IP) {
		eipResourceExits := false
		for _, eip := range eips {
			if eipResource.RefId == tea.StringValue(eip.AllocationId) {
				eipResourceExits = true
				break
			}
		}
		if !eipResourceExits {
			cluster.DeleteCloudResourceByRefID(biz.ResourceType_ELASTIC_IP, eipResource.RefId)
		}
	}

	// one zone one eip for nat gateway
	eipIds := make([]string, 0)
	for _, az := range cluster.GetCloudResource(biz.ResourceType_AVAILABILITY_ZONES) {
		name := cluster.GetEipName(az.RefId)
		tags := cluster.GetTags()
		tags[biz.ResourceTypeKeyValue_ZONE_ID] = az.RefId
		tags[biz.ResourceTypeKeyValue_NAME] = name
		for _, eip := range eips {
			if tea.StringValue(eip.InstanceId) != "" {
				continue
			}
			if cluster.GetCloudResourceByRefID(biz.ResourceType_ELASTIC_IP, tea.StringValue(eip.AllocationId)) != nil {
				a.log.Infof("eip %s already exists", tea.StringValue(eip.AllocationId))
				continue
			}
			if cluster.GetCloudResourceByTags(biz.ResourceType_ELASTIC_IP, map[biz.ResourceTypeKeyValue]any{biz.ResourceTypeKeyValue_ZONE_ID: az.RefId}) != nil {
				break
			}
			cluster.AddCloudResource(&biz.CloudResource{
				Name:  name,
				RefId: tea.StringValue(eip.AllocationId),
				Type:  biz.ResourceType_ELASTIC_IP,
				Value: tea.StringValue(eip.IpAddress),
				Tags:  cluster.EncodeTags(tags),
			})
			eipIds = append(eipIds, tea.StringValue(eip.AllocationId))
			a.log.Infof("elastic ip %s already exists", tea.StringValue(eip.IpAddress))
			break
		}
		if cluster.GetCloudResourceByTags(biz.ResourceType_ELASTIC_IP, map[biz.ResourceTypeKeyValue]any{biz.ResourceTypeKeyValue_ZONE_ID: az.RefId}) != nil {
			continue
		}
		// Allocate new EIP
		eipRes, err := a.vpcClient.AllocateEipAddress(&vpc.AllocateEipAddressRequest{
			RegionId:           tea.String(cluster.Region),
			Bandwidth:          tea.String("5"),
			InternetChargeType: tea.String("PayByTraffic"),
		})
		if err != nil {
			return errors.Wrap(err, "failed to allocate eip address")
		}
		eipIds = append(eipIds, tea.StringValue(eipRes.Body.AllocationId))
		// Add tags to EIP
		err = a.createVpcTags(cluster.Region, tea.StringValue(eipRes.Body.AllocationId), "EIP", tags)
		if err != nil {
			return errors.Wrap(err, "failed to tag eip")
		}
		cluster.AddCloudResource(&biz.CloudResource{
			Name:  name,
			RefId: tea.StringValue(eipRes.Body.AllocationId),
			Type:  biz.ResourceType_ELASTIC_IP,
			Value: tea.StringValue(eipRes.Body.EipAddress),
			Tags:  cluster.EncodeTags(tags),
		})
		a.log.Infof("elastic ip %s allocated for zone %s", tea.StringValue(eipRes.Body.EipAddress), az.RefId)
	}
	// wait eip status to be available
	if len(eipIds) == 0 {
		return nil
	}
	timeOutNumber := 0
	eipsOk := false
	for {
		if timeOutNumber > TimeOutCountNumber || eipsOk {
			break
		}
		time.Sleep(time.Second * TimeOutSecond)
		timeOutNumber++
		res, err := a.vpcClient.DescribeEipAddresses(&vpc.DescribeEipAddressesRequest{
			RegionId:     tea.String(cluster.Region),
			Status:       tea.String("Available"),
			AllocationId: tea.String(strings.Join(eipIds, ",")),
			PageNumber:   tea.Int32(1),
			PageSize:     tea.Int32(100),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe nat gateway")
		}
		if tea.Int32Value(res.Body.TotalCount) == int32(len(eipIds)) {
			eipsOk = true
			break
		}
	}
	if !eipsOk {
		return errors.New("eips not ready")
	}
	return nil
}

func (a *AliCloudUsecase) createNatGateways(ctx context.Context, cluster *biz.Cluster) error {
	vpcRes := cluster.GetSingleCloudResource(biz.ResourceType_VPC)
	if vpcRes == nil {
		return errors.New("vpc not found")
	}
	existingNatGateways := make([]*vpc.DescribeNatGatewaysResponseBodyNatGatewaysNatGateway, 0)
	var pageNumber int32 = 1
	for {
		existingNatGatewayRes, err := a.vpcClient.DescribeNatGateways(&vpc.DescribeNatGatewaysRequest{
			VpcId:       tea.String(vpcRes.RefId),
			Status:      tea.String("Available"),
			RegionId:    tea.String(cluster.Region),
			PageNumber:  tea.Int32(pageNumber),
			PageSize:    tea.Int32(50),
			NetworkType: tea.String("internet"),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe nat gateways")
		}
		existingNatGateways = append(existingNatGateways, existingNatGatewayRes.Body.NatGateways.NatGateway...)
		if len(existingNatGatewayRes.Body.NatGateways.NatGateway) < 50 {
			break
		}
		pageNumber++
	}

	for _, nategatway := range cluster.GetCloudResource(biz.ResourceType_NAT_GATEWAY) {
		nategatewayExits := false
		for _, natGateway := range existingNatGateways {
			if tea.StringValue(natGateway.NatGatewayId) == nategatway.RefId {
				nategatewayExits = true
				break
			}
		}
		if !nategatewayExits {
			cluster.DeleteCloudResourceByRefID(biz.ResourceType_NAT_GATEWAY, nategatway.RefId)
		}
	}

	for _, natGateway := range existingNatGateways {
		if cluster.GetCloudResourceByRefID(biz.ResourceType_NAT_GATEWAY, tea.StringValue(natGateway.NatGatewayId)) != nil {
			a.log.Infof("nat gateway %s already exists", tea.StringValue(natGateway.NatGatewayId))
			continue
		}
		if natGateway.NatGatewayPrivateInfo == nil || natGateway.NatGatewayPrivateInfo.VswitchId == nil {
			continue
		}
		subnetCloudResource := cluster.GetCloudResourceByRefID(biz.ResourceType_SUBNET, tea.StringValue(natGateway.NatGatewayPrivateInfo.VswitchId))
		if subnetCloudResource == nil {
			continue
		}
		subnetCloudResourceMapTags := cluster.DecodeTags(subnetCloudResource.Tags)
		if val, ok := subnetCloudResourceMapTags[biz.ResourceTypeKeyValue_ACCESS]; !ok || cast.ToInt32(val) != int32(biz.ResourceTypeKeyValue_ACCESS_PRIVATE) {
			continue
		}
		eipBindOk := false
		eipId := ""
		for _, eip := range natGateway.IpLists.IpList {
			eipId = tea.StringValue(eip.AllocationId)
			if eipId != "" && cluster.GetCloudResourceByRefID(biz.ResourceType_ELASTIC_IP, eipId) != nil {
				eipBindOk = true
				break
			}
		}
		if !eipBindOk {
			continue
		}
		tags := cluster.GetTags()
		name := cluster.GetNatgatewayName(cast.ToString(subnetCloudResourceMapTags[biz.ResourceTypeKeyValue_ZONE_ID]))
		tags[biz.ResourceTypeKeyValue_NAME] = name
		tags[biz.ResourceTypeKeyValue_ZONE_ID] = subnetCloudResourceMapTags[biz.ResourceTypeKeyValue_ZONE_ID]
		tags[biz.ResourceTypeKeyValue_ACCESS] = biz.ResourceTypeKeyValue_ACCESS_PRIVATE
		cluster.AddCloudResource(&biz.CloudResource{
			Name:         name,
			RefId:        tea.StringValue(natGateway.NatGatewayId),
			Tags:         cluster.EncodeTags(tags),
			Type:         biz.ResourceType_NAT_GATEWAY,
			AssociatedId: subnetCloudResource.RefId,
			Value:        eipId,
		})
		a.log.Infof("nat gateway %s already exists", tea.StringValue(natGateway.Name))
	}

	// create NAT Gateways for each AZ
	for _, az := range cluster.GetCloudResource(biz.ResourceType_AVAILABILITY_ZONES) {
		natgatewayResource := cluster.GetCloudResourceByTagsSingle(biz.ResourceType_NAT_GATEWAY, map[biz.ResourceTypeKeyValue]any{biz.ResourceTypeKeyValue_ZONE_ID: az.RefId})
		// value is the eip id
		if natgatewayResource != nil && natgatewayResource.Value != "" {
			continue
		}
		// Get private subnet for the AZ
		privateSubnet := cluster.GetCloudResourceByTagsSingle(biz.ResourceType_SUBNET, map[biz.ResourceTypeKeyValue]any{
			biz.ResourceTypeKeyValue_ACCESS:  biz.ResourceTypeKeyValue_ACCESS_PRIVATE,
			biz.ResourceTypeKeyValue_ZONE_ID: az.RefId,
		})
		if privateSubnet == nil {
			return errors.New("no private subnet found for AZ " + az.RefId)
		}
		// Get Elastic IP
		eip := cluster.GetCloudResourceByTagsSingle(biz.ResourceType_ELASTIC_IP, map[biz.ResourceTypeKeyValue]any{biz.ResourceTypeKeyValue_ZONE_ID: az.RefId})
		if eip == nil {
			return errors.New("no eip found for AZ " + az.RefId)
		}
		if natgatewayResource != nil && natgatewayResource.Value == "" {
			// Associate EIP with NAT Gateway
			_, err := a.vpcClient.AssociateEipAddress(&vpc.AssociateEipAddressRequest{
				RegionId:     tea.String(cluster.Region),
				AllocationId: tea.String(eip.RefId),
				InstanceId:   tea.String(natgatewayResource.RefId),
				InstanceType: tea.String("Nat"),
			})
			if err != nil {
				return errors.Wrap(err, "failed to associate eip with nat gateway")
			}
			natgatewayResource.Value = eip.RefId
			continue
		}

		// Create NAT Gateway
		natGatewayName := cluster.GetNatgatewayName(az.RefId)
		natRes, err := a.vpcClient.CreateNatGateway(&vpc.CreateNatGatewayRequest{
			RegionId:           tea.String(cluster.Region),
			VpcId:              tea.String(vpcRes.RefId),
			VSwitchId:          tea.String(privateSubnet.RefId),
			NatType:            tea.String("Enhanced"),
			NetworkType:        tea.String("internet"),
			Name:               tea.String(natGatewayName),
			InternetChargeType: tea.String("PayByLcu"),
		})
		if err != nil {
			return errors.Wrap(err, "failed to create nat gateway")
		}
		a.log.Infof("nat gateway %s createing", tea.StringValue(natRes.Body.NatGatewayId))
		// wait nategateway status to be available
		timeOutNumber := 0
		natgatewayOk := false
		for {
			if timeOutNumber > TimeOutCountNumber || natgatewayOk {
				break
			}
			time.Sleep(time.Second * 3 * TimeOutSecond)
			timeOutNumber++
			res, DescribeNatGatewaysErr := a.vpcClient.DescribeNatGateways(&vpc.DescribeNatGatewaysRequest{
				RegionId:     tea.String(cluster.Region),
				VpcId:        tea.String(vpcRes.RefId),
				NatGatewayId: natRes.Body.NatGatewayId,
				Name:         tea.String(natGatewayName),
				PageNumber:   tea.Int32(1),
				PageSize:     tea.Int32(10),
			})
			if DescribeNatGatewaysErr != nil {
				return errors.Wrap(DescribeNatGatewaysErr, "failed to describe nat gateway")
			}
			for _, v := range res.Body.NatGateways.NatGateway {
				if tea.StringValue(v.Status) == "Available" {
					natgatewayOk = true
					break
				}
			}
		}
		if !natgatewayOk {
			return errors.New("nat gateway " + tea.StringValue(natRes.Body.NatGatewayId) + " creation failed")
		}
		a.log.Infof("nat gateway %s created", tea.StringValue(natRes.Body.NatGatewayId))
		// Associate EIP with NAT Gateway
		_, err = a.vpcClient.AssociateEipAddress(&vpc.AssociateEipAddressRequest{
			RegionId:     tea.String(cluster.Region),
			AllocationId: tea.String(eip.RefId),
			InstanceId:   natRes.Body.NatGatewayId,
			InstanceType: tea.String("Nat"),
		})
		if err != nil {
			return errors.Wrap(err, "failed to associate eip with nat gateway")
		}
		// wait eip bind to natgateway
		time.Sleep(time.Second * 3 * TimeOutSecond)
		snatTableId := ""
		for _, v := range natRes.Body.SnatTableIds.SnatTableId {
			if tea.StringValue(v) != "" {
				snatTableId = tea.StringValue(v)
				break
			}
		}
		// Create natgateway SNAT
		_, err = a.vpcClient.CreateSnatEntry(&vpc.CreateSnatEntryRequest{
			RegionId:        tea.String(cluster.Region),
			SourceVSwitchId: tea.String(privateSubnet.RefId),
			SnatIp:          tea.String(eip.Value),
			SnatEntryName:   tea.String(natGatewayName + "-snat"),
			SnatTableId:     tea.String(snatTableId),
		})
		if err != nil {
			return errors.Wrap(err, "failed to create nat gateway snat")
		}
		cluster.AddCloudResource(&biz.CloudResource{
			Name:         natGatewayName,
			RefId:        tea.StringValue(natRes.Body.NatGatewayId),
			Type:         biz.ResourceType_NAT_GATEWAY,
			AssociatedId: privateSubnet.RefId,
			Value:        eip.RefId,
			Tags: cluster.EncodeTags(map[biz.ResourceTypeKeyValue]any{
				biz.ResourceTypeKeyValue_NAME:    natGatewayName,
				biz.ResourceTypeKeyValue_ACCESS:  biz.ResourceTypeKeyValue_ACCESS_PUBLIC,
				biz.ResourceTypeKeyValue_ZONE_ID: az.RefId,
			}),
		})
	}

	return nil
}

func (a *AliCloudUsecase) createRouteTables(ctx context.Context, cluster *biz.Cluster) error {
	vpcRes := cluster.GetSingleCloudResource(biz.ResourceType_VPC)
	if vpcRes == nil {
		return errors.New("vpc not found")
	}
	// List existing route tables
	var pageNumber int32 = 1
	existingRouteTables := make([]*vpc.DescribeRouteTableListResponseBodyRouterTableListRouterTableListType, 0)
	for {
		routeTablesRes, err := a.vpcClient.DescribeRouteTableList(&vpc.DescribeRouteTableListRequest{
			RegionId:   tea.String(cluster.Region),
			PageNumber: tea.Int32(pageNumber),
			PageSize:   tea.Int32(50),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe route tables")
		}
		existingRouteTables = append(existingRouteTables, routeTablesRes.Body.RouterTableList.RouterTableListType...)
		if len(routeTablesRes.Body.RouterTableList.RouterTableListType) < 50 {
			break
		}
		pageNumber++
	}

	// clear history route table
	for _, routeTableCloudResource := range cluster.GetCloudResource(biz.ResourceType_ROUTE_TABLE) {
		routeTableCloudResourceExits := false
		for _, routeTable := range existingRouteTables {
			if routeTableCloudResource.RefId == tea.StringValue(routeTable.RouteTableId) {
				routeTableCloudResourceExits = true
				break
			}
		}
		if !routeTableCloudResourceExits {
			cluster.DeleteCloudResourceByRefID(biz.ResourceType_ROUTE_TABLE, routeTableCloudResource.RefId)
		}
	}

	// Create private route tables (one per AZ)
	for _, az := range cluster.GetCloudResource(biz.ResourceType_AVAILABILITY_ZONES) {
		if cluster.GetCloudResourceByTags(biz.ResourceType_ROUTE_TABLE, map[biz.ResourceTypeKeyValue]any{biz.ResourceTypeKeyValue_ZONE_ID: az.RefId}) != nil {
			continue
		}
		routeTableName := cluster.GetRouteTableName(az.RefId)
		tags := cluster.GetTags()
		tags[biz.ResourceTypeKeyValue_NAME] = routeTableName
		tags[biz.ResourceTypeKeyValue_ACCESS] = biz.ResourceTypeKeyValue_ACCESS_PRIVATE
		tags[biz.ResourceTypeKeyValue_ZONE_ID] = az.RefId
		// Create private route table
		privateRouteTableRes, err := a.vpcClient.CreateRouteTable(&vpc.CreateRouteTableRequest{
			RegionId:       tea.String(cluster.Region),
			VpcId:          tea.String(vpcRes.RefId),
			RouteTableName: tea.String(routeTableName),
			AssociateType:  tea.String("VSwitch"),
		})
		if err != nil {
			return errors.Wrap(err, "failed to create private route table for AZ "+az.RefId)
		}
		cluster.AddCloudResource(&biz.CloudResource{
			Name:  routeTableName,
			RefId: tea.StringValue(privateRouteTableRes.Body.RouteTableId),
			Tags:  cluster.EncodeTags(tags),
			Type:  biz.ResourceType_ROUTE_TABLE,
		})
		a.log.Infof("private route table %s createing for AZ %s", tea.StringValue(privateRouteTableRes.Body.RouteTableId), az.RefId)
		// wait nategateway status to be available
		timeOutNumber := 0
		routeTableOk := false
		for {
			if timeOutNumber > TimeOutCountNumber || routeTableOk {
				break
			}
			time.Sleep(time.Second * TimeOutSecond)
			timeOutNumber++
			res, err := a.vpcClient.DescribeRouteTableList(&vpc.DescribeRouteTableListRequest{
				RegionId:     tea.String(cluster.Region),
				VpcId:        tea.String(vpcRes.RefId),
				RouteTableId: privateRouteTableRes.Body.RouteTableId,
				PageNumber:   tea.Int32(1),
				PageSize:     tea.Int32(10),
			})
			if err != nil {
				return errors.Wrap(err, "failed to describe nat gateway")
			}
			for _, v := range res.Body.RouterTableList.RouterTableListType {
				if tea.StringValue(v.Status) == "Available" {
					routeTableOk = true
					break
				}
			}
		}
		if !routeTableOk {
			return errors.New("route table create timeout")
		}
		a.log.Infof("private route table %s created for AZ %s", tea.StringValue(privateRouteTableRes.Body.RouteTableId), az.RefId)
	}

	routeTables := cluster.GetCloudResourceByTags(biz.ResourceType_ROUTE_TABLE, map[biz.ResourceTypeKeyValue]any{
		biz.ResourceTypeKeyValue_ACCESS: biz.ResourceTypeKeyValue_ACCESS_PRIVATE,
	})

	// Associate private subnets with private route table
	for _, routeTable := range routeTables {
		subnetIds := make([]string, 0)
		for _, v := range existingRouteTables {
			if tea.StringValue(v.RouteTableId) == routeTable.RefId {
				subnetIds = tea.StringSliceValue(v.VSwitchIds.VSwitchId)
			}
		}
		routeTableTags := cluster.DecodeTags(routeTable.Tags)
		privateSubnet := cluster.GetCloudResourceByTagsSingle(biz.ResourceType_SUBNET, map[biz.ResourceTypeKeyValue]any{
			biz.ResourceTypeKeyValue_ACCESS:  biz.ResourceTypeKeyValue_ACCESS_PRIVATE,
			biz.ResourceTypeKeyValue_ZONE_ID: routeTableTags[biz.ResourceTypeKeyValue_ZONE_ID],
		})
		if slices.Contains(subnetIds, privateSubnet.RefId) {
			continue
		}
		_, err := a.vpcClient.AssociateRouteTable(&vpc.AssociateRouteTableRequest{
			RegionId:     tea.String(cluster.Region),
			RouteTableId: tea.String(routeTable.RefId),
			VSwitchId:    tea.String(privateSubnet.RefId),
		})
		if err != nil {
			return errors.Wrap(err, "failed to associate private subnet")
		}

		// wait
		time.Sleep(time.Second * 3 * TimeOutSecond)
	}

	// Add route to NAT Gateway in private route table
	for _, routeTable := range routeTables {
		routeEntryRes, err := a.vpcClient.DescribeRouteEntryList(&vpc.DescribeRouteEntryListRequest{
			RegionId:     tea.String(cluster.Region),
			RouteTableId: tea.String(routeTable.RefId),
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe route entry list")
		}
		natgatewayIds := make([]string, 0)
		for _, routeEntry := range routeEntryRes.Body.RouteEntrys.RouteEntry {
			if tea.StringValue(routeEntry.Type) != "Custom" || routeEntry.NextHops == nil {
				continue
			}
			for _, nextHop := range routeEntry.NextHops.NextHop {
				natgatewayIds = append(natgatewayIds, tea.StringValue(nextHop.NextHopId))
			}
		}
		routeTableTags := cluster.DecodeTags(routeTable.Tags)
		natGateway := cluster.GetCloudResourceByTagsSingle(biz.ResourceType_NAT_GATEWAY, map[biz.ResourceTypeKeyValue]any{
			biz.ResourceTypeKeyValue_ZONE_ID: routeTableTags[biz.ResourceTypeKeyValue_ZONE_ID],
		})
		if natGateway == nil {
			return errors.New("nat gateway not found in route table tags")
		}
		if slices.Contains(natgatewayIds, natGateway.RefId) {
			continue
		}
		res, err := a.vpcClient.CreateRouteEntry(&vpc.CreateRouteEntryRequest{
			RegionId:             tea.String(cluster.Region),
			RouteTableId:         tea.String(routeTable.RefId),
			DestinationCidrBlock: tea.String("0.0.0.0/0"),
			NextHopType:          tea.String("NatGateway"),
			NextHopId:            tea.String(natGateway.RefId),
		})
		if err != nil {
			return errors.Wrap(err, "failed to add route to NAT Gateway")
		}
		routeTable.AssociatedId = natGateway.RefId
		routeTable.Value = tea.StringValue(res.Body.RouteEntryId)

		// wait
		time.Sleep(time.Second * 3 * TimeOutSecond)
	}
	return nil
}

func (a *AliCloudUsecase) ManageSecurityGroup(ctx context.Context, cluster *biz.Cluster) error {
	vpcRes := cluster.GetSingleCloudResource(biz.ResourceType_VPC)
	if vpcRes == nil {
		return errors.New("vpc not found")
	}
	sgName := cluster.GetSecurityGroupName()
	securityGroupsRes, err := a.ecsClient.DescribeSecurityGroups(&ecs.DescribeSecurityGroupsRequest{
		RegionId:          tea.String(cluster.Region),
		VpcId:             tea.String(vpcRes.RefId),
		SecurityGroupName: tea.String(sgName),
		PageNumber:        tea.Int32(1),
		PageSize:          tea.Int32(1),
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe security groups")
	}
	if len(securityGroupsRes.Body.SecurityGroups.SecurityGroup) == 0 && cluster.GetSingleCloudResource(biz.ResourceType_SECURITY_GROUP) != nil {
		cluster.DeleteCloudResource(biz.ResourceType_SECURITY_GROUP)
	}

	// Process existing security groups
	for _, securityGroup := range securityGroupsRes.Body.SecurityGroups.SecurityGroup {
		if cluster.GetCloudResourceByRefID(biz.ResourceType_SECURITY_GROUP, tea.StringValue(securityGroup.SecurityGroupId)) != nil {
			a.log.Infof("security group %s already exists", tea.StringValue(securityGroup.SecurityGroupId))
			continue
		}
		tags := cluster.GetTags()
		tags[biz.ResourceTypeKeyValue_NAME] = tea.StringValue(securityGroup.SecurityGroupName)
		cluster.AddCloudResource(&biz.CloudResource{
			Name:         tea.StringValue(securityGroup.SecurityGroupName),
			RefId:        tea.StringValue(securityGroup.SecurityGroupId),
			Tags:         cluster.EncodeTags(tags),
			AssociatedId: vpcRes.RefId,
			Type:         biz.ResourceType_SECURITY_GROUP,
		})
		a.log.Infof("security group %s already exists", tea.StringValue(securityGroup.SecurityGroupId))
	}

	sgCloudResource := cluster.GetCloudResourceByName(biz.ResourceType_SECURITY_GROUP, sgName)
	if sgCloudResource == nil {
		// Create security group
		tags := cluster.GetTags()
		tags[biz.ResourceTypeKeyValue_NAME] = sgName
		createSGReq := &ecs.CreateSecurityGroupRequest{
			RegionId:          tea.String(cluster.Region),
			VpcId:             tea.String(vpcRes.RefId),
			SecurityGroupName: tea.String(sgName),
			SecurityGroupType: tea.String("normal"),
			Description:       tea.String(sgName),
		}
		sgRes, CreateSecurityGroupErr := a.ecsClient.CreateSecurityGroup(createSGReq)
		if CreateSecurityGroupErr != nil {
			return errors.Wrap(CreateSecurityGroupErr, "failed to create security group")
		}
		sgCloudResource = &biz.CloudResource{
			Name:         sgName,
			RefId:        tea.StringValue(sgRes.Body.SecurityGroupId),
			Tags:         cluster.EncodeTags(tags),
			Type:         biz.ResourceType_SECURITY_GROUP,
			AssociatedId: vpcRes.RefId,
		}
		cluster.AddCloudResource(sgCloudResource)
		a.log.Infof("security group %s created", tea.StringValue(sgRes.Body.SecurityGroupId))
	}

	// Add security group rules
	sgRuleRes, err := a.ecsClient.DescribeSecurityGroupAttribute(&ecs.DescribeSecurityGroupAttributeRequest{
		RegionId:        tea.String(cluster.Region),
		SecurityGroupId: tea.String(sgCloudResource.RefId),
		MaxResults:      tea.Int32(1000),
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe security group attribute")
	}
	// clear not exits rules
	needClearRuleIds := make([]string, 0)
	exitsRules := make([]string, 0)
	for _, sgRule := range sgRuleRes.Body.Permissions.Permission {
		exits := false
		for _, clusterSgRule := range cluster.IngressControllerRules {
			clusterSgRuelVals := strings.Join([]string{
				clusterSgRule.Protocol, clusterSgRule.IpCidr,
				fmt.Sprintf("%d/%d", clusterSgRule.StartPort, clusterSgRule.EndPort)},
				"-")
			sgRuleVals := strings.Join([]string{
				tea.StringValue(sgRule.IpProtocol), tea.StringValue(sgRule.SourceCidrIp),
				tea.StringValue(sgRule.PortRange)},
				"-")
			if clusterSgRuelVals == sgRuleVals {
				exits = true
				exitsRules = append(exitsRules, sgRuleVals)
				break
			}
		}
		if !exits {
			needClearRuleIds = append(needClearRuleIds, tea.StringValue(sgRule.SecurityGroupRuleId))
		}
	}
	if len(needClearRuleIds) != 0 {
		_, err = a.ecsClient.RevokeSecurityGroup(&ecs.RevokeSecurityGroupRequest{
			RegionId:            tea.String(cluster.Region),
			SecurityGroupId:     tea.String(sgCloudResource.RefId),
			SecurityGroupRuleId: tea.StringSlice(needClearRuleIds),
		})
		if err != nil {
			return errors.Wrap(err, "failed to clear security group rule")
		}
		time.Sleep(time.Second)
	}

	for _, sgRule := range cluster.IngressControllerRules {
		sgRuelVals := strings.Join([]string{
			sgRule.Protocol, sgRule.IpCidr,
			fmt.Sprintf("%d/%d", sgRule.StartPort, sgRule.EndPort)},
			"-")

		if slices.Contains(exitsRules, sgRuelVals) {
			continue
		}
		_, err = a.ecsClient.AuthorizeSecurityGroup(&ecs.AuthorizeSecurityGroupRequest{
			RegionId:        tea.String(cluster.Region),
			SecurityGroupId: tea.String(sgCloudResource.RefId),
			IpProtocol:      tea.String(strings.ToUpper(sgRule.Protocol)),
			PortRange:       tea.String(fmt.Sprintf("%d/%d", sgRule.StartPort, sgRule.EndPort)),
			SourceCidrIp:    tea.String(sgRule.IpCidr),
			Description:     tea.String(fmt.Sprintf("Allow %s access", sgRule.Protocol)),
		})
		if err != nil {
			return errors.Wrap(err, "failed to add security group rule")
		}
		time.Sleep(time.Second)
	}
	return nil
}

func (a *AliCloudUsecase) ManageSLB(_ context.Context, cluster *biz.Cluster) error {
	vpcRes := cluster.GetSingleCloudResource(biz.ResourceType_VPC)
	if vpcRes == nil {
		return errors.New("vpc not found")
	}
	// Check if SLB already exists
	slbName := cluster.GetLoadBalancerName()
	loadBalancers, err := a.slbClient.DescribeLoadBalancers(&slb.DescribeLoadBalancersRequest{
		LoadBalancerName: tea.String(slbName),
		RegionId:         tea.String(cluster.Region),
		VpcId:            tea.String(vpcRes.RefId),
		PageNumber:       tea.Int32(1),
		PageSize:         tea.Int32(1),
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe load balancers")
	}
	for _, lb := range loadBalancers.Body.LoadBalancers.LoadBalancer {
		if cluster.GetCloudResourceByRefID(biz.ResourceType_LOAD_BALANCER, tea.StringValue(lb.LoadBalancerId)) != nil {
			a.log.Infof("slb %s already exists", tea.StringValue(lb.LoadBalancerId))
			continue
		}
		cluster.AddCloudResource(&biz.CloudResource{
			Name:  tea.StringValue(lb.LoadBalancerName),
			RefId: tea.StringValue(lb.LoadBalancerId),
			Type:  biz.ResourceType_LOAD_BALANCER,
			Value: tea.StringValue(lb.Address),
		})
		a.log.Infof("slb %s already exists", tea.StringValue(lb.LoadBalancerName))
	}
	if len(cluster.GetCloudResource(biz.ResourceType_LOAD_BALANCER)) == 0 {
		// Create SLB
		slbRes, CreateLoadBalancerErr := a.slbClient.CreateLoadBalancer(&slb.CreateLoadBalancerRequest{
			RegionId:           tea.String(cluster.Region),
			VpcId:              tea.String(vpcRes.RefId),
			LoadBalancerName:   tea.String(slbName),
			PayType:            tea.String("PayOnDemand"),
			AddressType:        tea.String("internet"),
			InternetChargeType: tea.String("paybytraffic"),
			InstanceChargeType: tea.String("PayByCLCU"),
		})
		if CreateLoadBalancerErr != nil {
			return errors.Wrap(CreateLoadBalancerErr, "failed to create SLB")
		}

		a.log.Infof("slb %s created", tea.StringValue(slbRes.Body.LoadBalancerName))
		cluster.AddCloudResource(&biz.CloudResource{
			Name:  slbName,
			RefId: tea.StringValue(slbRes.Body.LoadBalancerId),
			Type:  biz.ResourceType_LOAD_BALANCER,
			Value: tea.StringValue(slbRes.Body.Address),
		})
	}
	slbCloudResource := cluster.GetSingleCloudResource(biz.ResourceType_LOAD_BALANCER)
	if slbCloudResource == nil {
		return errors.New("slb not found")
	}

	// handler listener
	masterNodes := make([]*biz.Node, 0)
	for _, node := range cluster.Nodes {
		if node.Role != biz.NodeRole_MASTER || node.InstanceId == "" {
			continue
		}
		masterNodes = append(masterNodes, node)
	}
	vServerNames := make([]string, 0)
	vServerNameProtMap := make(map[string]int32)
	vServerBackendServerMap := make(map[string][]map[string]string)
	if len(masterNodes) > 0 && len(cluster.IngressControllerRules) > 0 {
		for _, v := range cluster.IngressControllerRules {
			if v.Access != biz.IngressControllerRuleAccess_PUBLIC {
				continue
			}
			for port := v.StartPort; port <= v.EndPort; port++ {
				backendServerMaps := make([]map[string]string, 0)
				instanceids := make([]string, 0)
				for _, masterNode := range masterNodes {
					instanceids = append(instanceids, masterNode.InstanceId)
					backendServerMaps = append(backendServerMaps, map[string]string{
						"ServerId":    masterNode.InstanceId,
						"Weight":      "100",
						"Type":        "ecs",
						"Port":        fmt.Sprintf("%d", port),
						"Description": fmt.Sprintf("%s-%s", masterNode.Name, masterNode.InstanceId),
					})
				}
				instanceidStr := utils.Md5(strings.Join(instanceids, ","))
				vServerName := fmt.Sprintf("%s-%d", instanceidStr, port)
				vServerNames = append(vServerNames, vServerName)
				vServerBackendServerMap[vServerName] = backendServerMaps
				vServerNameProtMap[vServerName] = port
			}
		}
	}

	res, err := a.slbClient.DescribeVServerGroups(&slb.DescribeVServerGroupsRequest{
		RegionId:        tea.String(cluster.Region),
		LoadBalancerId:  tea.String(slbCloudResource.RefId),
		IncludeListener: tea.Bool(true),
		// IncludeRule:     tea.Bool(true),
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe vserver groups")
	}
	// clear not exits vserver group
	for _, vserverGroup := range res.Body.VServerGroups.VServerGroup {
		if slices.Contains(vServerNames, tea.StringValue(vserverGroup.VServerGroupName)) {
			continue
		}
		// delete listener
		for _, listener := range vserverGroup.AssociatedObjects.Listeners.Listener {
			_, err := a.slbClient.DeleteLoadBalancerListener(&slb.DeleteLoadBalancerListenerRequest{
				RegionId:         tea.String(cluster.Region),
				LoadBalancerId:   tea.String(slbCloudResource.RefId),
				ListenerPort:     listener.Port,
				ListenerProtocol: listener.Protocol,
			})
			if err != nil {
				return errors.Wrap(err, "failed to delete load balancer listener")
			}
			time.Sleep(time.Second)
		}
		_, err := a.slbClient.DeleteVServerGroup(&slb.DeleteVServerGroupRequest{
			RegionId:       tea.String(cluster.Region),
			VServerGroupId: vserverGroup.VServerGroupId,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete vserver group")
		}
		time.Sleep(time.Second)
	}
	// if not exits vserver group, create it
	for _, vServerName := range vServerNames {
		exits := false
		for _, vserverGroup := range res.Body.VServerGroups.VServerGroup {
			if tea.StringValue(vserverGroup.VServerGroupName) == vServerName {
				exits = true
				break
			}
		}
		if exits {
			continue
		}
		backendServerMaps := vServerBackendServerMap[vServerName]
		backendServerJson, err := json.Marshal(backendServerMaps)
		if err != nil {
			return errors.Wrap(err, "failed to marshal backend server maps")
		}
		vserverGroupRes, err := a.slbClient.CreateVServerGroup(&slb.CreateVServerGroupRequest{
			RegionId:         tea.String(cluster.Region),
			LoadBalancerId:   tea.String(slbCloudResource.RefId),
			VServerGroupName: tea.String(vServerName),
			BackendServers:   tea.String(string(backendServerJson)),
		})
		if err != nil {
			return errors.Wrap(err, "failed to create vserver group")
		}
		a.log.Infof("vserver group %s created", vServerName)
		time.Sleep(time.Second)
		port := vServerNameProtMap[vServerName]
		_, err = a.slbClient.CreateLoadBalancerTCPListener(&slb.CreateLoadBalancerTCPListenerRequest{
			RegionId:       tea.String(cluster.Region),
			LoadBalancerId: tea.String(slbCloudResource.RefId),
			ListenerPort:   tea.Int32(port),
			VServerGroupId: vserverGroupRes.Body.VServerGroupId,
			Scheduler:      tea.String("wrr"),
			Description:    tea.String(vServerName),
		})
		if err != nil {
			return errors.Wrap(err, "failed to create load balancer tcp listener")
		}
		time.Sleep(time.Second * TimeOutSecond)
		_, err = a.slbClient.StartLoadBalancerListener(&slb.StartLoadBalancerListenerRequest{
			RegionId:       tea.String(cluster.Region),
			LoadBalancerId: tea.String(slbCloudResource.RefId),
			ListenerPort:   tea.Int32(port),
		})
		if err != nil {
			return errors.Wrap(err, "failed to start load balancer listener")
		}
	}
	return nil
}

func (a *AliCloudUsecase) FindImage(regionId string, arch biz.NodeArchType) (*ecs.DescribeImagesResponseBodyImagesImage, error) {
	archStr, ok := NodeArchToMagecloudType[arch]
	if !ok {
		return nil, errors.New("unsupported arch")
	}
	pageNumber := 1
	for {
		images, err := a.ecsClient.DescribeImages(&ecs.DescribeImagesRequest{
			RegionId:        tea.String(regionId),
			Status:          tea.String("Available"),
			OSType:          tea.String("Linux"),
			ImageOwnerAlias: tea.String("system"),
			Architecture:    tea.String(archStr),
			ActionType:      tea.String("CreateEcs"),
			PageNumber:      tea.Int32(int32(pageNumber)),
			PageSize:        tea.Int32(100),
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to describe images")
		}
		if images.Body.Images == nil || tea.Int32Value(images.Body.TotalCount) == 0 {
			return nil, errors.New("no images found")
		}
		for _, v := range images.Body.Images.Image {
			if strings.ToLower(tea.StringValue(v.Platform)) == "ubuntu" {
				return v, nil
			}
		}
		if len(images.Body.Images.Image) < 100 {
			break
		}
		pageNumber++
	}
	return nil, errors.New("failed to find image")
}

// GenerateInstanceSize generates the instance size based on CPU
func aliGenerateInstanceSize(cpu int32) string {
	if cpu == 1 {
		return "small"
	}
	if cpu > 1 && cpu <= 2 {
		return "large"
	}
	if cpu > 2 && cpu <= 4 {
		return "xlarge"
	}
	size := ""
	if cpu >= 4 {
		if cpu%4 != 0 {
			cpu = (cpu/4 + 1) * 4
			fmt.Println(cpu)
		}
		size = fmt.Sprintf("%dxlarge", cpu/4)
	}
	return size
}

// https://help.aliyun.com/zh/ecs/user-guide/overview-of-instance-families?spm=a2c4g.11186623.help-menu-25365.d_4_1_1.30491fda4fxXnm&scm=20140722.H_25378._.OR_help-T_cn-DAS-zh-V_1#c8i
func aliGetInstanceIds(nodeGroupType biz.NodeGroupType, ecsSize string) []string {
	instanceTypeIds := make([]string, 0)
	if nodeGroupType == biz.NodeGroupType_NORMAL {
		instanceTypeIds = []string{
			fmt.Sprintf("ecs.g8i.%s", ecsSize),
			fmt.Sprintf("ecs.g7.%s", ecsSize),
			fmt.Sprintf("ecs.g6e.%s", ecsSize),
			fmt.Sprintf("ecs.g6.%s", ecsSize),
			fmt.Sprintf("ecs.g8a.%s", ecsSize),
			fmt.Sprintf("ecs.g8ae.%s", ecsSize),
			fmt.Sprintf("ecs.g7a.%s", ecsSize),
			fmt.Sprintf("ecs.g6a.%s", ecsSize),
			fmt.Sprintf("ecs.g8y.%s", ecsSize), // arm64
		}
	}
	if nodeGroupType == biz.NodeGroupType_HIGH_COMPUTATION {
		instanceTypeIds = []string{
			fmt.Sprintf("ecs.c8i.%s", ecsSize),
			fmt.Sprintf("ecs.c7.%s", ecsSize),
			fmt.Sprintf("ecs.c6e.%s", ecsSize),
			fmt.Sprintf("ecs.c6.%s", ecsSize),
			fmt.Sprintf("ecs.c6.%s", ecsSize),
			fmt.Sprintf("ecs.c8a.%s", ecsSize),
			fmt.Sprintf("ecs.c8ae.%s", ecsSize),
			fmt.Sprintf("ecs.c7a.%s", ecsSize),
			fmt.Sprintf("ecs.c6a.%s", ecsSize),
			fmt.Sprintf("ecs.c8y.%s", ecsSize), // arm64
		}
	}
	if nodeGroupType == biz.NodeGroupType_HIGH_MEMORY {
		instanceTypeIds = []string{
			fmt.Sprintf("ecs.r8i.%s", ecsSize),
			fmt.Sprintf("ecs.r7p.%s", ecsSize),
			fmt.Sprintf("ecs.r7.%s", ecsSize),
			fmt.Sprintf("ecs.r6e.%s", ecsSize),
			fmt.Sprintf("ecs.r6.%s", ecsSize),
			fmt.Sprintf("ecs.r8a.%s", ecsSize),
			fmt.Sprintf("ecs.r8ae.%s", ecsSize),
			fmt.Sprintf("ecs.r7a.%s", ecsSize),
			fmt.Sprintf("ecs.r6a.%s", ecsSize),
			fmt.Sprintf("ecs.r8y.%s", ecsSize), // arm64
		}
	}
	if nodeGroupType == biz.NodeGroupType_LARGE_HARD_DISK {
		// Big local disk
		instanceTypeIds = []string{
			fmt.Sprintf("ecs.d3s.%s", ecsSize),
			fmt.Sprintf("ecs.d3c.%s", ecsSize),
			fmt.Sprintf("ecs.d2c.%s", ecsSize),
			fmt.Sprintf("ecs.d2s.%s", ecsSize),
			fmt.Sprintf("ecs.d1ne.%s", ecsSize),
		}
	}
	if nodeGroupType == biz.NodeGroupType_LOAD_DISK {
		// Samll local disk
		instanceTypeIds = []string{
			fmt.Sprintf("ecs.i2.%s", ecsSize),
			fmt.Sprintf("ecs.i2g.%s", ecsSize),
			fmt.Sprintf("ecs.i2ne.%s", ecsSize),
			fmt.Sprintf("ecs.i2gne.%s", ecsSize),
			fmt.Sprintf("ecs.i3g.%s", ecsSize),
			fmt.Sprintf("ecs.i3.%s", ecsSize),
			fmt.Sprintf("ecs.i4.%s", ecsSize),
			fmt.Sprintf("ecs.i4r.%s", ecsSize),
			fmt.Sprintf("ecs.i4g.%s", ecsSize),
			fmt.Sprintf("ecs.i4p.%s", ecsSize),
		}
	}
	if nodeGroupType == biz.NodeGroupType_GPU_ACCELERATERD {
		instanceTypeIds = []string{
			fmt.Sprintf("ecs.gn8v.%s", ecsSize),
			fmt.Sprintf("ecs.gn8is.%s", ecsSize),
			fmt.Sprintf("ecs.gn7e.%s", ecsSize),
			fmt.Sprintf("ecs.gn7i.%s", ecsSize),
			fmt.Sprintf("ecs.gn7s.%s", ecsSize),
			fmt.Sprintf("ecs.gn7.%s", ecsSize),
			fmt.Sprintf("ecs.gn7r.%s", ecsSize),
			fmt.Sprintf("ecs.gn6i.%s", ecsSize),
			fmt.Sprintf("ecs.gn6e.%s", ecsSize),
			fmt.Sprintf("ecs.gn6v.%s", ecsSize),
		}
	}
	return instanceTypeIds
}

func (a *AliCloudUsecase) FindInstanceType(param FindInstanceTypeParam) ([]*ecs.DescribeInstanceTypesResponseBodyInstanceTypesInstanceType, error) {
	ecsSize := aliGenerateInstanceSize(param.CPU)
	instanceTypeIds := aliGetInstanceIds(param.NodeGroupType, ecsSize)
	instanceTypes := make([]*ecs.DescribeInstanceTypesResponseBodyInstanceTypesInstanceType, 0)
	nexttoken := ""
	for {
		instancesReq := &ecs.DescribeInstanceTypesRequest{
			InstanceTypes:       tea.StringSlice(instanceTypeIds),
			CpuArchitecture:     tea.String(NodeArchToCloudType[param.Arch]),
			MinimumCpuCoreCount: tea.Int32(param.CPU),
			MaximumCpuCoreCount: tea.Int32(param.CPU),
			NextToken:           tea.String(nexttoken),
			MaxResults:          tea.Int64(10),
		}
		if param.GPU > 0 {
			instancesReq.MinimumGPUAmount = tea.Int32(param.GPU)
			instancesReq.MaximumGPUAmount = tea.Int32(param.GPU)
			if param.GPUSpec.String() != "" {
				instancesReq.GPUSpec = tea.String(NodeGPUSpecToCloudSpec[param.GPUSpec])
			}
		}
		instancesRes, err := a.ecsClient.DescribeInstanceTypes(instancesReq)
		if err != nil {
			return nil, errors.Wrap(err, "failed to describe instance types")
		}
		instanceTypes = append(instanceTypes, instancesRes.Body.InstanceTypes.InstanceType...)
		if tea.StringValue(instancesRes.Body.NextToken) == "" {
			break
		}
		nexttoken = tea.StringValue(instancesRes.Body.NextToken)
	}
	return instanceTypes, nil
}

func (a *AliCloudUsecase) createVpcTags(regionID, resourceID, resourceType string, tags map[biz.ResourceTypeKeyValue]any) error {
	vpcTags := make([]*vpc.TagResourcesRequestTag, 0)
	for key, value := range tags {
		vpcTags = append(vpcTags, &vpc.TagResourcesRequestTag{
			Key:   tea.String(key.String()),
			Value: tea.String(cast.ToString(value)),
		})
	}
	_, err := a.vpcClient.TagResources(&vpc.TagResourcesRequest{
		RegionId:     tea.String(regionID),
		ResourceType: tea.String(resourceType),
		ResourceId:   tea.StringSlice([]string{resourceID}),
		Tag:          vpcTags,
	})
	if err != nil {
		return errors.Wrap(err, "failed to tag vpc")
	}
	return nil
}

func (a *AliCloudUsecase) createEcsTag(regionID, resourceID, resourceType string, tags map[biz.ResourceTypeKeyValue]any) error {
	ecsTags := make([]*ecs.TagResourcesRequestTag, 0)
	for key, value := range tags {
		ecsTags = append(ecsTags, &ecs.TagResourcesRequestTag{
			Key:   tea.String(key.String()),
			Value: tea.String(cast.ToString(value)),
		})
	}
	_, err := a.ecsClient.TagResources(&ecs.TagResourcesRequest{
		RegionId:     tea.String(regionID),
		ResourceType: tea.String(resourceType),
		ResourceId:   tea.StringSlice([]string{resourceID}),
		Tag:          ecsTags,
	})
	if err != nil {
		return errors.Wrap(err, "failed to tag ecs")
	}
	return nil
}

func (a *AliCloudUsecase) handlerError(err error) error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*tea.SDKError); ok && e.Code != nil && tea.StringValue(e.Code) == "DryRunOperation" {
		return nil
	}
	return err
}
