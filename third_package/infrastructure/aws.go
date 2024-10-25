package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elasticloadbalancingv2Types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
)

type AwsCloud struct {
	cluster     *biz.Cluster
	ec2Client   *ec2.Client
	elbv2Client *elasticloadbalancingv2.Client
	log         *log.Helper
	conf        *conf.Bootstrap
}

const (
	awsDefaultRegion = "us-east-1"
	AwsTagKeyName    = "Name"
	AwsTagKeyType    = "Type"
	AwsTagKeyZone    = "Zone"
	AwsTagKeyBind    = "Bind"
	AwsTagKeyVpc     = "Vpc"

	AwsResourcePublic        = "Public"
	AwsResourcePrivate       = "Private"
	AwsResourceBind          = "true"
	AwsReosurceUnBind        = "false"
	AwsReousrceBostionHostSG = "bostionHost"
	AwsResourceHttpSG        = "http"
)

const (
	TimeoutPerInstance = 5 * time.Minute
	AwsNotFound        = "NotFound"
)

func NewAwsCloud(ctx context.Context, cluster *biz.Cluster, conf *conf.Bootstrap, log *log.Helper) (*AwsCloud, error) {
	if cluster.Region == "" {
		cluster.Region = awsDefaultRegion
	}
	os.Setenv("AWS_REGION", cluster.Region)
	os.Setenv("AWS_DEFAULT_REGION", cluster.Region)
	os.Setenv("AWS_ACCESS_KEY_ID", cluster.AccessID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", cluster.AccessKey)
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cluster.Region))
	if err != nil {
		return nil, err
	}
	return &AwsCloud{
		cluster:     cluster,
		ec2Client:   ec2.NewFromConfig(cfg),
		elbv2Client: elasticloadbalancingv2.NewFromConfig(cfg),
		conf:        conf,
		log:         log,
	}, nil
}

// Get availability zones
func (a *AwsCloud) GetAvailabilityZones(ctx context.Context) error {
	a.cluster.DeleteCloudResource(biz.ResourceTypeAvailabilityZones)
	result, err := a.ec2Client.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"available"},
			},
			{
				Name:   aws.String("region-name"),
				Values: []string{a.cluster.Region},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe regions")
	}
	if len(result.AvailabilityZones) == 0 {
		return errors.New("no availability zones found")
	}
	for _, az := range result.AvailabilityZones {
		a.cluster.AddCloudResource(biz.ResourceTypeAvailabilityZones, &biz.CloudResource{
			Name:  aws.ToString(az.ZoneName),
			ID:    aws.ToString(az.ZoneId),
			Type:  biz.ResourceTypeAvailabilityZones,
			Value: aws.ToString(az.RegionName),
		})
	}
	return nil
}

// create network(vpc, subnet, internet gateway,nat gateway, route table, security group)
func (a *AwsCloud) CreateNetwork(ctx context.Context) error {
	// Step 1: Check and Create VPC
	err := a.createVPC(ctx)
	if err != nil {
		return err
	}

	// Step 3: Check and Create subnets
	err = a.createSubnets(ctx)
	if err != nil {
		return err
	}

	// Step 4: Check and Create Internet Gateway
	err = a.createInternetGateway(ctx)
	if err != nil {
		return err
	}

	// Step 5: Check and Create NAT Gateways
	err = a.createNATGateways(ctx)
	if err != nil {
		return err
	}

	// Step 6: Check and Create route tables
	err = a.createRouteTables(ctx)
	if err != nil {
		return err
	}

	// Step 7: Check and Create security group
	err = a.createSecurityGroup(ctx)
	if err != nil {
		return err
	}

	err = a.createS3Endpoint(ctx)
	if err != nil {
		return err
	}

	err = a.createSLB(ctx)
	if err != nil {
		return err
	}

	return nil
}

// delete network(vpc, subnet, internet gateway, nat gateway, route table, security group)
func (a *AwsCloud) DeleteNetwork(ctx context.Context) error {
	// Delete vpc s3 endpoints
	for _, endpoint := range a.cluster.GetCloudResource(biz.ResourceTypeVpcEndpointS3) {
		_, err := a.ec2Client.DescribeVpcEndpoints(ctx, &ec2.DescribeVpcEndpointsInput{
			VpcEndpointIds: []string{endpoint.ID},
		})
		if err != nil && strings.Contains(err.Error(), AwsNotFound) {
			a.log.Infof("No vpc endpoint found with ID: %s\n", endpoint.ID)
			continue
		}
		if err != nil {
			return errors.Wrap(err, "failed to describe vpc endpoint")
		}
		_, err = a.ec2Client.DeleteVpcEndpoints(ctx, &ec2.DeleteVpcEndpointsInput{
			VpcEndpointIds: []string{endpoint.ID},
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete vpc endpoint")
		}
	}
	a.cluster.DeleteCloudResource(biz.ResourceTypeVpcEndpointS3)

	// Step 1: Delete security group
	for _, sg := range a.cluster.GetCloudResource(biz.ResourceTypeSecurityGroup) {
		_, err := a.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
			GroupIds: []string{sg.ID},
		})
		if err != nil && strings.Contains(err.Error(), AwsNotFound) {
			a.log.Infof("No security group found with ID: %s\n", sg.ID)
			continue
		}
		if err != nil {
			return errors.Wrap(err, "failed to describe security group")
		}
		_, err = a.ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(sg.ID),
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete security group")
		}
	}
	a.cluster.DeleteCloudResource(biz.ResourceTypeSecurityGroup)

	// Step 2: Delete route tables
	rts := a.cluster.GetCloudResource(biz.ResourceTypeRouteTable)
	for _, rt := range rts {
		_, err := a.ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
			RouteTableIds: []string{rt.ID},
		})
		if err != nil && strings.Contains(err.Error(), AwsNotFound) {
			a.log.Infof("No route table found with ID: %s\n", rt.ID)
			continue
		}
		if err != nil {
			return errors.Wrap(err, "failed to describe route table")
		}
		for _, subRtassoc := range rt.SubResources {
			_, err = a.ec2Client.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
				AssociationId: aws.String(subRtassoc.ID),
			})
			if err != nil {
				return errors.Wrap(err, "failed to disassociate route table")
			}
		}
		_, err = a.ec2Client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
			RouteTableId: aws.String(rt.ID),
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete route table")
		}
	}
	a.cluster.DeleteCloudResource(biz.ResourceTypeRouteTable)

	// Step 4: Delete NAT Gateways
	natGwIDs := make([]string, 0)
	for _, natGw := range a.cluster.GetCloudResource(biz.ResourceTypeNATGateway) {
		_, err := a.ec2Client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []string{natGw.ID},
		})
		if err != nil && strings.Contains(err.Error(), AwsNotFound) {
			a.log.Infof("No NAT Gateway found with ID: %s\n", natGw.ID)
			continue
		}
		if err != nil {
			return errors.Wrap(err, "failed to describe NAT Gateway")
		}
		_, err = a.ec2Client.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{
			NatGatewayId: aws.String(natGw.ID),
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete NAT Gateway")
		}
		natGwIDs = append(natGwIDs, natGw.ID)
	}
	// Wait for NAT Gateway to be deleted
	waiter := ec2.NewNatGatewayDeletedWaiter(a.ec2Client)
	err := waiter.Wait(ctx, &ec2.DescribeNatGatewaysInput{
		NatGatewayIds: natGwIDs,
	}, time.Duration(len(natGwIDs))*TimeoutPerInstance)
	if err != nil {
		return fmt.Errorf("failed to wait for NAT Gateway deletion: %w", err)
	}
	a.cluster.DeleteCloudResource(biz.ResourceTypeNATGateway)

	// Release Elastic IPs associated with NAT Gateways
	for _, addr := range a.cluster.GetCloudResource(biz.ResourceTypeElasticIP) {
		_, err := a.ec2Client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
			AllocationIds: []string{addr.ID},
		})
		if err != nil && strings.Contains(err.Error(), AwsNotFound) {
			a.log.Infof("No Elastic IP found with ID: %s\n", addr.ID)
			continue
		}
		if err != nil {
			return errors.Wrap(err, "failed to describe Elastic IP")
		}
		_, err = a.ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
			AllocationId: aws.String(addr.ID),
		})
		if err != nil {
			return errors.Wrap(err, "failed to release Elastic IP")
		}
	}
	a.cluster.DeleteCloudResource(biz.ResourceTypeElasticIP)

	// Step 3: Delete Internet Gateway
	for _, igw := range a.cluster.GetCloudResource(biz.ResourceTypeInternetGateway) {
		_, err := a.ec2Client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
			InternetGatewayIds: []string{igw.ID},
		})
		if err != nil && strings.Contains(err.Error(), AwsNotFound) {
			a.log.Infof("No Internet Gateway found with ID: %s\n", igw.ID)
			continue
		}
		if err != nil {
			return errors.Wrap(err, "failed to describe Internet Gateway")
		}
		_, err = a.ec2Client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
			InternetGatewayId: aws.String(igw.ID),
			VpcId:             aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
		})
		if err != nil {
			return errors.Wrap(err, "failed to detach Internet Gateway")
		}
		_, err = a.ec2Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: aws.String(igw.ID),
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete Internet Gateway")
		}
	}
	a.cluster.DeleteCloudResource(biz.ResourceTypeInternetGateway)

	// // Step 5: Delete Subnets
	for _, subnet := range a.cluster.GetCloudResource(biz.ResourceTypeSubnet) {
		_, err := a.ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			SubnetIds: []string{subnet.ID},
		})
		if err != nil && strings.Contains(err.Error(), AwsNotFound) {
			a.log.Infof("No subnet found with ID: %s\n", subnet.ID)
			continue
		}
		if err != nil {
			return errors.Wrap(err, "failed to describe subnet")
		}
		_, err = a.ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
			SubnetId: aws.String(subnet.ID),
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete subnet")
		}
	}
	a.cluster.DeleteCloudResource(biz.ResourceTypeSubnet)

	// Step 6: Delete VPC
	_, err = a.ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
		VpcId: aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete VPC")
	}
	a.cluster.DeleteCloudResource(biz.ResourceTypeVPC)

	// step 7: Delete SLB
	for _, slb := range a.cluster.GetCloudResource(biz.ResourceTypeLoadBalancer) {
		_, err := a.elbv2Client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
			LoadBalancerArns: []string{slb.ID},
		})
		if err != nil && strings.Contains(err.Error(), AwsNotFound) {
			a.log.Infof("No SLB found with ID: %s\n", slb.ID)
			continue
		}
		if err != nil {
			return errors.Wrap(err, "failed to describe SLB")
		}
		_, err = a.elbv2Client.DeleteLoadBalancer(ctx, &elasticloadbalancingv2.DeleteLoadBalancerInput{
			LoadBalancerArn: &slb.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete SLB")
		}
	}
	a.cluster.DeleteCloudResource(biz.ResourceTypeLoadBalancer)
	return nil
}

type InstanceTypeResults []ec2Types.InstanceTypeInfo

// sort by vcpu and memory
func (a InstanceTypeResults) Len() int {
	return len(a)
}

func (a InstanceTypeResults) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a InstanceTypeResults) Less(i, j int) bool {
	if aws.ToInt32(a[i].VCpuInfo.DefaultVCpus) < aws.ToInt32(a[j].VCpuInfo.DefaultVCpus) {
		return true
	}
	if aws.ToInt32(a[i].VCpuInfo.DefaultVCpus) == aws.ToInt32(a[j].VCpuInfo.DefaultVCpus) {
		return aws.ToInt64(a[i].MemoryInfo.SizeInMiB) < aws.ToInt64(a[j].MemoryInfo.SizeInMiB)
	}
	return false
}

// get instance type familiy
func (a *AwsCloud) SetByNodeGroups(ctx context.Context) error {
	image, err := a.findImage(ctx)
	if err != nil {
		return err
	}
	for _, ng := range a.cluster.NodeGroups {
		platformDetails := strings.Split(aws.ToString(image.PlatformDetails), "/")
		if len(platformDetails) > 0 {
			ng.OS = strings.ToLower(platformDetails[0])
		}
		ng.Image = aws.ToString(image.ImageId)
		ng.ImageDescription = aws.ToString(image.Description)
		ng.ARCH = string(image.Architecture)
		ng.DefaultUsername = determineUsername(aws.ToString(image.Name), aws.ToString(image.Description))
		ng.RootDeviceName = aws.ToString(image.RootDeviceName)
		for _, dataDeivce := range image.BlockDeviceMappings {
			if dataDeivce.DeviceName != nil && aws.ToString(dataDeivce.DeviceName) != ng.RootDeviceName {
				ng.DataDeviceName = aws.ToString(dataDeivce.DeviceName)
				break
			}
		}
		a.log.Info(strings.Join([]string{"image found: ", aws.ToString(image.Name), aws.ToString(image.Description)}, " "))

		if ng.InstanceType != "" {
			continue
		}
		instanceTypeFamiliy := getIntanceTypeFamilies(ng)
		instanceInfo, err := a.findInstanceType(ctx, instanceTypeFamiliy, ng.CPU, ng.GPU, ng.Memory)
		if err != nil {
			return err
		}
		ng.InstanceType = string(instanceInfo.InstanceType)
		if instanceInfo.VCpuInfo != nil && instanceInfo.VCpuInfo.DefaultVCpus != nil {
			ng.CPU = aws.ToInt32(instanceInfo.VCpuInfo.DefaultVCpus)
		}
		if instanceInfo.MemoryInfo != nil && instanceInfo.MemoryInfo.SizeInMiB != nil {
			ng.Memory = int32(aws.ToInt64(instanceInfo.MemoryInfo.SizeInMiB) / 1024)
		}
		if ng.GPU != 0 && instanceInfo.GpuInfo != nil && len(instanceInfo.GpuInfo.Gpus) > 0 {
			for _, g := range instanceInfo.GpuInfo.Gpus {
				ng.GPU += aws.ToInt32(g.Count)
				ng.GpuSpec += fmt.Sprintf("-%s", aws.ToString(g.Name))
			}
		}
		a.log.Info("instance type found: ", ng.InstanceType)
	}
	return nil
}

// KeyPair
func (a *AwsCloud) ImportKeyPair(ctx context.Context) error {
	keyName := a.cluster.Name + "-keypair"
	tags := map[string]string{
		AwsTagKeyName: keyName,
	}
	keyPairOutputs, err := a.ec2Client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{
		KeyNames: []string{keyName},
	})
	if err != nil && !strings.Contains(err.Error(), AwsNotFound) {
		return fmt.Errorf("failed to describe key pair: %v", err)
	}
	if keyPairOutputs != nil && len(keyPairOutputs.KeyPairs) != 0 {
		for _, keyPair := range keyPairOutputs.KeyPairs {
			if keyPair.KeyPairId == nil {
				continue
			}
			if a.cluster.GetCloudResourceByID(biz.ResourceTypeKeyPair, aws.ToString(keyPair.KeyPairId)) != nil {
				continue
			}
			a.cluster.AddCloudResource(biz.ResourceTypeKeyPair, &biz.CloudResource{
				Name: aws.ToString(keyPair.KeyName),
				ID:   aws.ToString(keyPair.KeyPairId),
				Tags: tags,
			})
			a.log.Info("key pair found")
		}
		return nil
	}

	keyPairOutput, err := a.ec2Client.ImportKeyPair(ctx, &ec2.ImportKeyPairInput{
		KeyName:           aws.String(keyName),
		PublicKeyMaterial: []byte(a.cluster.PublicKey),
		TagSpecifications: []ec2Types.TagSpecification{
			{
				ResourceType: ec2Types.ResourceTypeKeyPair,
				Tags:         a.mapToEc2Tags(tags),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to import key pair: %v", err)
	}
	a.log.Info("key pair imported")
	a.cluster.AddCloudResource(biz.ResourceTypeKeyPair, &biz.CloudResource{
		Name: keyName,
		ID:   aws.ToString(keyPairOutput.KeyPairId),
		Tags: tags,
	})
	return nil
}

func (a *AwsCloud) DeleteKeyPair(ctx context.Context) error {
	for _, keyPair := range a.cluster.GetCloudResource(biz.ResourceTypeKeyPair) {
		_, err := a.ec2Client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{
			KeyNames: []string{keyPair.Name},
		})
		if err != nil && strings.Contains(err.Error(), AwsNotFound) {
			a.log.Infof("No key pair found with ID: %s\n", keyPair.ID)
			continue
		}
		_, err = a.ec2Client.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
			KeyName: aws.String(keyPair.Name),
		})
		if err != nil {
			return fmt.Errorf("failed to delete key pair: %v", err)
		}
		a.log.Info("key pair deleted")
	}
	a.cluster.DeleteCloudResource(biz.ResourceTypeKeyPair)
	return nil
}

func (a *AwsCloud) ManageInstance(ctx context.Context) error {
	// Delete instances
	needDeleteInstanceIDs := make([]string, 0)
	for _, node := range a.cluster.Nodes {
		if node.Status == biz.NodeStatusDeleting && node.InstanceID != "" {
			needDeleteInstanceIDs = append(needDeleteInstanceIDs, node.InstanceID)
		}
	}
	instances, err := a.getInstances(ctx, []string{}, []string{fmt.Sprintf("%s-node*", a.cluster.Name)})
	if err != nil {
		return err
	}
	deleteInstanceIDs := make([]string, 0)
	for _, instance := range instances {
		if utils.InArray(aws.ToString(instance.InstanceId), needDeleteInstanceIDs) {
			deleteInstanceIDs = append(deleteInstanceIDs, aws.ToString(instance.InstanceId))
		}
	}
	if len(deleteInstanceIDs) > 0 {
		_, err = a.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: deleteInstanceIDs,
		})
		if err != nil {
			return errors.Wrap(err, "failed to terminate instances")
		}
		waiter := ec2.NewInstanceTerminatedWaiter(a.ec2Client)
		err := waiter.Wait(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: deleteInstanceIDs,
		}, time.Duration(len(deleteInstanceIDs))*TimeoutPerInstance)
		if err != nil {
			return fmt.Errorf("failed to wait for instance termination: %w", err)
		}
		for _, node := range a.cluster.Nodes {
			if utils.InArray(node.InstanceID, deleteInstanceIDs) {
				node.Status = biz.NodeStatusDeleted
			}
		}
		a.log.Info("instances terminated")
	}

	// Create instances
	instanceIds := make([]string, 0)
	for index, node := range a.cluster.Nodes {
		if node.Status != biz.NodeStatusCreating {
			continue
		}

		nodeGroup := a.cluster.GetNodeGroup(node.NodeGroupID)
		nodeTags := make(map[string]string)
		if node.Labels != "" {
			err = json.Unmarshal([]byte(node.Labels), &nodeTags)
			if err != nil {
				return errors.Wrap(err, "failed to parse labels")
			}
		}
		nodeTags[AwsTagKeyName] = node.Name
		// root Volume
		blockDeviceMappings := []ec2Types.BlockDeviceMapping{
			{
				DeviceName: aws.String(nodeGroup.RootDeviceName),
				Ebs: &ec2Types.EbsBlockDevice{
					VolumeSize:          aws.Int32(30),
					VolumeType:          ec2Types.VolumeTypeGp3,
					DeleteOnTermination: aws.Bool(true),
				},
			},
		}
		if nodeGroup.DataDisk > 0 {
			blockDeviceMappings = append(blockDeviceMappings, ec2Types.BlockDeviceMapping{
				DeviceName: aws.String(nodeGroup.DataDeviceName),
				Ebs: &ec2Types.EbsBlockDevice{
					VolumeSize:          aws.Int32(nodeGroup.DataDisk),
					VolumeType:          ec2Types.VolumeTypeGp3,
					DeleteOnTermination: aws.Bool(true),
				},
			})
		}
		sgs := a.cluster.GetCloudResourceByTags(biz.ResourceTypeSecurityGroup, AwsTagKeyType, AwsResourceHttpSG)
		if sgs == nil {
			return errors.Wrap(err, "security group not found")
		}
		sgIDs := make([]string, 0)
		for _, v := range sgs {
			sgIDs = append(sgIDs, v.ID)
		}
		keyName := a.cluster.GetSingleCloudResource(biz.ResourceTypeKeyPair).Name
		privateSubnetID := a.distributeNodeSubnets(index, a.cluster.GetCloudResourceByTags(biz.ResourceTypeSubnet, AwsTagKeyType, AwsResourcePrivate))
		instanceOutput, err := a.ec2Client.RunInstances(ctx, &ec2.RunInstancesInput{
			ImageId:             aws.String(nodeGroup.Image),
			InstanceType:        ec2Types.InstanceType(nodeGroup.InstanceType),
			KeyName:             aws.String(keyName),
			MaxCount:            aws.Int32(1),
			MinCount:            aws.Int32(1),
			SecurityGroupIds:    sgIDs,
			SubnetId:            aws.String(privateSubnetID),
			BlockDeviceMappings: blockDeviceMappings,
			TagSpecifications: []ec2Types.TagSpecification{
				{
					ResourceType: ec2Types.ResourceTypeInstance,
					Tags:         a.mapToEc2Tags(nodeTags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to run instances")
		}
		for _, instance := range instanceOutput.Instances {
			a.log.Info("instance createing", "name", node.Name, "id", aws.ToString(instance.InstanceId))
			if instance.PrivateIpAddress != nil {
				node.InternalIP = aws.ToString(instance.PrivateIpAddress)
			}
			if instance.PublicIpAddress != nil {
				node.ExternalIP = aws.ToString(instance.PublicIpAddress)
			}
			node.InstanceID = aws.ToString(instance.InstanceId)
			instanceIds = append(instanceIds, aws.ToString(instance.InstanceId))
		}
		node.User = nodeGroup.DefaultUsername
		node.Status = biz.NodeStatusCreating
	}

	// wait for instance running
	if len(instanceIds) > 0 {
		waiter := ec2.NewInstanceRunningWaiter(a.ec2Client)
		err := waiter.Wait(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: instanceIds,
		}, time.Duration(len(instanceIds))*TimeoutPerInstance)
		if err != nil {
			return fmt.Errorf("failed to wait for instance running: %w", err)
		}
		for _, instanceId := range instanceIds {
			for _, node := range a.cluster.Nodes {
				if node.InstanceID == instanceId {
					node.Status = biz.NodeStatusRunning
					break
				}
			}
		}
	}
	return nil
}

// Manage BostionHost
func (a *AwsCloud) ManageBostionHost(ctx context.Context) error {
	if a.cluster.BostionHost == nil {
		return nil
	}
	if a.cluster.BostionHost.Status == biz.NodeStatusDeleting {
		if a.cluster.BostionHost.InstanceID == "" {
			return nil
		}
		_, err := a.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []string{a.cluster.BostionHost.InstanceID},
		})
		if err != nil {
			return errors.Wrap(err, "failed to terminate instances")
		}
		waiter := ec2.NewInstanceTerminatedWaiter(a.ec2Client)
		err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{a.cluster.BostionHost.InstanceID},
		}, time.Duration(1)*TimeoutPerInstance)
		if err != nil {
			return fmt.Errorf("failed to wait for instance termination: %w", err)
		}
		a.cluster.BostionHost.Status = biz.NodeStatusDeleted
		return nil
	}

	if a.cluster.BostionHost.Status != biz.NodeStatusCreating {
		return nil
	}

	// find image
	image, err := a.findImage(ctx)
	if err != nil {
		return err
	}
	platformDetails := strings.Split(aws.ToString(image.PlatformDetails), "/")
	if len(platformDetails) > 0 {
		a.cluster.BostionHost.OS = strings.ToLower(platformDetails[0])
	}
	a.cluster.BostionHost.ARCH = string(image.Architecture)
	a.cluster.BostionHost.Image = aws.ToString(image.ImageId)
	a.cluster.BostionHost.ImageDescription = aws.ToString(image.Description)

	// find instance type
	instanceType, err := a.findInstanceType(ctx, "t3.*", a.cluster.BostionHost.CPU, 0, a.cluster.BostionHost.Memory)
	if err != nil {
		return err
	}
	publicSubnet := a.cluster.GetCloudResourceByTags(biz.ResourceTypeSubnet, AwsTagKeyType, AwsResourcePublic)
	if publicSubnet == nil {
		return errors.New("public subnet not found in the ManageBostionHost")
	}
	sgs := a.cluster.GetCloudResourceByTags(biz.ResourceTypeSecurityGroup, AwsTagKeyType, AwsReousrceBostionHostSG)
	if sgs == nil {
		return errors.New("security group not found in the ManageBostionHost")
	}
	sgIds := make([]string, 0)
	for _, v := range sgs {
		sgIds = append(sgIds, v.ID)
	}

	keyPair := a.cluster.GetSingleCloudResource(biz.ResourceTypeKeyPair)
	if keyPair == nil {
		return errors.New("key pair not found in the ManageBostionHost")
	}

	bostionHostTag := map[string]string{
		AwsTagKeyName: fmt.Sprintf("%s-%s", a.cluster.Name, "bostion"),
	}
	instanceOutput, err := a.ec2Client.RunInstances(ctx, &ec2.RunInstancesInput{
		ImageId:      image.ImageId,
		InstanceType: ec2Types.InstanceType(instanceType.InstanceType),
		MaxCount:     aws.Int32(1),
		MinCount:     aws.Int32(1),
		KeyName:      aws.String(keyPair.Name),
		NetworkInterfaces: []ec2Types.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:              aws.Int32(0),
				AssociatePublicIpAddress: aws.Bool(true),
				DeleteOnTermination:      aws.Bool(true),
				SubnetId:                 aws.String(publicSubnet[0].ID),
				Groups:                   sgIds,
				Description:              aws.String("ManageBostionHost network interface"),
			},
		},
		TagSpecifications: []ec2Types.TagSpecification{
			{
				ResourceType: ec2Types.ResourceTypeInstance,
				Tags:         a.mapToEc2Tags(bostionHostTag),
			},
		},
		BlockDeviceMappings: []ec2Types.BlockDeviceMapping{
			{
				DeviceName: aws.String(aws.ToString(image.RootDeviceName)),
				Ebs: &ec2Types.EbsBlockDevice{
					VolumeSize:          aws.Int32(10),
					VolumeType:          ec2Types.VolumeTypeGp3,
					DeleteOnTermination: aws.Bool(true),
				},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to run instances in the ManageBostionHost")
	}

	instanceIds := make([]string, 0)
	for _, instance := range instanceOutput.Instances {
		instanceIds = append(instanceIds, aws.ToString(instance.InstanceId))
	}
	waiter := ec2.NewInstanceRunningWaiter(a.ec2Client)
	err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	}, time.Duration(1)*TimeoutPerInstance)
	if err != nil {
		return fmt.Errorf("failed to wait for instance running: %w", err)
	}
	instances, err := a.getInstances(ctx, instanceIds, []string{})
	if err != nil {
		return err
	}
	for _, instance := range instances {
		a.cluster.BostionHost.InternalIP = aws.ToString(instance.PrivateIpAddress)
		a.cluster.BostionHost.ExternalIP = aws.ToString(instance.PublicIpAddress)
		a.cluster.BostionHost.Status = biz.NodeStatusRunning
		a.cluster.BostionHost.InstanceID = aws.ToString(instance.InstanceId)
		a.cluster.BostionHost.User = determineUsername(aws.ToString(image.Name), aws.ToString(image.Description))
		// cpu
		if instanceType.VCpuInfo != nil && instanceType.VCpuInfo.DefaultVCpus != nil {
			a.cluster.BostionHost.CPU = aws.ToInt32(instanceType.VCpuInfo.DefaultVCpus)
		}
		// memory
		if instanceType.MemoryInfo != nil && instanceType.MemoryInfo.SizeInMiB != nil {
			a.cluster.BostionHost.Memory = int32(aws.ToInt64(instanceType.MemoryInfo.SizeInMiB) / 1024)
		}
	}
	return nil
}

// find image
func (a *AwsCloud) findImage(ctx context.Context) (ec2Types.Image, error) {
	image := ec2Types.Image{}
	images, err := a.ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners: []string{"amazon"},
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{"ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"},
			},
			{
				Name:   aws.String("architecture"),
				Values: []string{"x86_64"},
			},
			{
				Name:   aws.String("state"),
				Values: []string{"available"},
			},
		},
	})
	if err != nil || len(images.Images) == 0 {
		return image, errors.Wrap(err, "failed to describe images")
	}
	for _, image := range images.Images {
		return image, nil
	}
	return image, nil
}

func (a *AwsCloud) findInstanceType(ctx context.Context, instanceTypeFamiliy string, CPU, GPU, Memory int32) (ec2Types.InstanceTypeInfo, error) {
	instanceTypeInfo := ec2Types.InstanceTypeInfo{}
	instanceData := make(InstanceTypeResults, 0)
	instanceTypeInput := &ec2.DescribeInstanceTypesInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("current-generation"),
				Values: []string{"true"},
			},
			{
				Name:   aws.String("processor-info.supported-architecture"),
				Values: []string{"x86_64"},
			},
			{
				Name:   aws.String("instance-type"),
				Values: []string{instanceTypeFamiliy},
			},
		},
	}
	for {
		instanceTypes, err := a.ec2Client.DescribeInstanceTypes(ctx, instanceTypeInput)
		if err != nil {
			return instanceTypeInfo, errors.Wrap(err, "failed to describe instance types")
		}
		for _, instanceType := range instanceTypes.InstanceTypes {
			instanceData = append(instanceData, instanceType)
		}
		if instanceTypes.NextToken == nil {
			break
		}
		instanceTypeInput.NextToken = instanceTypes.NextToken
	}
	sort.Sort(instanceData)
	for _, instanceType := range instanceData {
		if aws.ToInt64(instanceType.MemoryInfo.SizeInMiB) == 0 {
			continue
		}
		memoryGBiSize := aws.ToInt64(instanceType.MemoryInfo.SizeInMiB) / 1024
		if int32(memoryGBiSize) >= Memory && aws.ToInt32(instanceType.VCpuInfo.DefaultVCpus) >= CPU {
			instanceTypeInfo = instanceType
		}
		if instanceTypeInfo.InstanceType == "" {
			continue
		}
		if GPU == 0 {
			break
		}
		for _, gpues := range instanceType.GpuInfo.Gpus {
			if aws.ToInt32(gpues.Count) >= GPU {
				break
			}
		}
	}
	if instanceTypeInfo.InstanceType == "" {
		return instanceTypeInfo, errors.New("no instance type found")
	}
	return instanceTypeInfo, nil
}

func (a *AwsCloud) getInstances(ctx context.Context, instanceIDs, tagNames []string) ([]ec2Types.Instance, error) {
	filters := []ec2Types.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []string{a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID},
		},
		{
			Name:   aws.String("instance-state-name"),
			Values: []string{string(ec2Types.InstanceStateNameRunning)},
		},
	}
	if len(tagNames) > 0 {
		filters = append(filters, ec2Types.Filter{
			Name:   aws.String("tag:Name"),
			Values: tagNames,
		})
	}
	input := &ec2.DescribeInstancesInput{
		Filters: filters,
	}
	if len(instanceIDs) > 0 {
		input.InstanceIds = instanceIDs
	}
	var instances []ec2Types.Instance
	for {
		output, err := a.ec2Client.DescribeInstances(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instances: %w", err)
		}

		for _, reservation := range output.Reservations {
			instances = append(instances, reservation.Instances...)
		}

		if output.NextToken == nil {
			break
		}
		input.NextToken = output.NextToken
	}

	return instances, nil
}

func (a *AwsCloud) distributeNodeSubnets(nodeIndex int, subnets []*biz.CloudResource) (subNetID string) {
	if len(subnets) == 0 {
		return ""
	}
	nodeSize := len(a.cluster.Nodes)
	subnetsSize := len(subnets)
	if nodeSize <= subnetsSize {
		return subnets[nodeIndex%subnetsSize].ID
	}
	interval := nodeSize / subnetsSize
	return subnets[(nodeIndex/interval)%subnetsSize].ID
}

// create vpc
func (a *AwsCloud) createVPC(ctx context.Context) error {
	if a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC) != nil {
		a.log.Info("vpc already exists ", "vpc ", a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).Name)
		return nil
	}
	vpcName := a.cluster.Name + "-vpc"
	vpcTags := map[string]string{
		AwsTagKeyName: vpcName,
	}
	a.cluster.AddCloudResource(biz.ResourceTypeVPC, &biz.CloudResource{
		Name: vpcName,
		Tags: vpcTags,
	})
	existingVpcs, err := a.ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).Name},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe VPCs")
	}

	if len(existingVpcs.Vpcs) != 0 {
		for _, vpc := range existingVpcs.Vpcs {
			for _, tag := range vpc.Tags {
				vpcTags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
			}
			vpcCloudResource := a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC)
			vpcCloudResource.ID = aws.ToString(vpc.VpcId)
			vpcCloudResource.Tags = vpcTags
		}
		a.log.Infof("vpc %s already exists", a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID)
		return nil
	}

	// Create VPC if it doesn't exist
	vpcOutput, err := a.ec2Client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: aws.String(a.cluster.IpCidr),
		TagSpecifications: []ec2Types.TagSpecification{
			{
				ResourceType: ec2Types.ResourceTypeVpc,
				Tags:         a.mapToEc2Tags(vpcTags),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create VPC")
	}
	vpcCloudResource := a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC)
	vpcCloudResource.ID = aws.ToString(vpcOutput.Vpc.VpcId)

	_, err = a.ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId: vpcOutput.Vpc.VpcId,
		EnableDnsSupport: &ec2Types.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to enable DNS support for VPC")
	}
	a.log.Infof("vpc %s created", a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID)
	return nil
}

// Check and Create subnets
func (a *AwsCloud) createSubnets(ctx context.Context) error {
	existingSubnets, err := a.ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe subnets")
	}

	zoneSubnets := make(map[string][]ec2Types.Subnet)
	for _, subnet := range existingSubnets.Subnets {
		if subnet.AvailabilityZone == nil {
			continue
		}
		_, ok := zoneSubnets[aws.ToString(subnet.AvailabilityZone)]
		if ok && len(zoneSubnets[aws.ToString(subnet.AvailabilityZone)]) >= 3 {
			continue
		}
		zoneSubnets[aws.ToString(subnet.AvailabilityZone)] = append(zoneSubnets[aws.ToString(subnet.AvailabilityZone)], subnet)
	}
	for zoneName, subzoneSubnets := range zoneSubnets {
		for i, subnet := range subzoneSubnets {
			if subnet.SubnetId == nil {
				continue
			}
			if a.cluster.GetCloudResourceByID(biz.ResourceTypeSubnet, aws.ToString(subnet.SubnetId)) != nil {
				a.log.Infof("subnet %s already exists", aws.ToString(subnet.SubnetId))
				continue
			}
			tags := make(map[string]string)
			name := ""
			for _, tag := range subnet.Tags {
				tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
			}
			tags[AwsTagKeyZone] = zoneName
			if i < 2 {
				name = fmt.Sprintf("%s-private-subnet-%s-%d", a.cluster.Name, aws.ToString(subnet.AvailabilityZone), i+1)
				tags[AwsTagKeyType] = AwsResourcePrivate
			} else {
				name = fmt.Sprintf("%s-public-subnet-%s", a.cluster.Name, aws.ToString(subnet.AvailabilityZone))
				tags[AwsTagKeyType] = AwsResourcePublic
			}
			if nameVal, ok := tags[AwsTagKeyName]; !ok || nameVal != name {
				tags[AwsTagKeyName] = name
				err = a.createTags(ctx, aws.ToString(subnet.SubnetId), biz.ResourceTypeSubnet, tags)
				if err != nil {
					return err
				}
			}
			tags[AwsTagKeyName] = name
			a.cluster.AddCloudResource(biz.ResourceTypeSubnet, &biz.CloudResource{
				Name: name,
				ID:   aws.ToString(subnet.SubnetId),
				Tags: tags,
			})
			a.log.Infof("subnet %s already exists", aws.ToString(subnet.SubnetId))
		}
	}

	// get subnet cidr
	privateSubnetCount := len(a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones)) * 2
	publicSubnetCount := len(a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones))
	subnetCidrRes, err := utils.GenerateSubnets(a.cluster.IpCidr, privateSubnetCount+publicSubnetCount+len(existingSubnets.Subnets))
	if err != nil {
		return errors.Wrap(err, "failed to generate subnet CIDRs")
	}
	subnetCidrs := make([]string, 0)
	existingSubnetCird := make(map[string]bool)
	for _, subnet := range existingSubnets.Subnets {
		existingSubnetCird[aws.ToString(subnet.CidrBlock)] = true
	}
	for _, subnetCidr := range subnetCidrRes {
		subnetCidrDecode := utils.DecodeCidr(subnetCidr)
		if subnetCidrDecode == "" {
			continue
		}
		ok := true
		for _, subnet := range existingSubnets.Subnets {
			existingSubnetCirdDecode := utils.DecodeCidr(aws.ToString(subnet.CidrBlock))
			if existingSubnetCirdDecode == "" {
				continue
			}
			if subnetCidrDecode == existingSubnetCirdDecode {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		subnetCidrs = append(subnetCidrs, subnetCidr)
	}

	// Create private subnets
	for i, az := range a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones) {
		for j := 0; j < 2; j++ {
			name := fmt.Sprintf("%s-private-subnet-%s-%d", a.cluster.Name, az.Name, j+1)
			tags := map[string]string{
				AwsTagKeyName: name,
				AwsTagKeyType: AwsResourcePrivate,
				AwsTagKeyZone: az.Name,
			}
			if a.cluster.GetCloudResourceByTags(biz.ResourceTypeSubnet, AwsTagKeyName, name) != nil {
				continue
			}
			cidr := subnetCidrs[i*2+j]
			subnetOutput, err := a.ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
				VpcId:            aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
				CidrBlock:        aws.String(cidr),
				AvailabilityZone: &az.Name,
				TagSpecifications: []ec2Types.TagSpecification{
					{
						ResourceType: ec2Types.ResourceTypeSubnet,
						Tags:         a.mapToEc2Tags(tags),
					},
				},
			})
			if err != nil {
				return errors.Wrap(err, "failed to create private subnet")
			}
			a.cluster.AddCloudResource(biz.ResourceTypeSubnet, &biz.CloudResource{
				Name: name,
				ID:   aws.ToString(subnetOutput.Subnet.SubnetId),
				Tags: tags,
			})
			a.log.Infof("private subnet %s created", aws.ToString(subnetOutput.Subnet.SubnetId))
		}

		name := fmt.Sprintf("%s-public-subnet-%s", a.cluster.Name, az.Name)
		tags := map[string]string{
			AwsTagKeyName: name,
			AwsTagKeyType: AwsResourcePublic,
			AwsTagKeyZone: az.Name,
		}
		if a.cluster.GetCloudResourceByTags(biz.ResourceTypeSubnet, AwsTagKeyName, name) != nil {
			continue
		}
		// Create public subnet
		cidr := subnetCidrs[privateSubnetCount+i]
		subnetOutput, err := a.ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
			VpcId:            aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
			CidrBlock:        aws.String(cidr),
			AvailabilityZone: &az.Name,
			TagSpecifications: []ec2Types.TagSpecification{
				{
					ResourceType: ec2Types.ResourceTypeSubnet,
					Tags:         a.mapToEc2Tags(tags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create public subnet")
		}
		a.cluster.AddCloudResource(biz.ResourceTypeSubnet, &biz.CloudResource{
			Name: name,
			ID:   aws.ToString(subnetOutput.Subnet.SubnetId),
			Tags: tags,
		})
		a.log.Infof("public subnet %s created", aws.ToString(subnetOutput.Subnet.SubnetId))
	}
	return nil
}

// Check and Create Internet Gateway
func (a *AwsCloud) createInternetGateway(ctx context.Context) error {
	existingIgws, err := a.ec2Client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("attachment.vpc-id"),
				Values: []string{a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe Internet Gateways")
	}

	if len(existingIgws.InternetGateways) != 0 {
		for _, igw := range existingIgws.InternetGateways {
			if igw.InternetGatewayId == nil {
				continue
			}
			if a.cluster.GetCloudResourceByID(biz.ResourceTypeInternetGateway, aws.ToString(igw.InternetGatewayId)) != nil {
				a.log.Infof("internet gateway %s already exists", aws.ToString(igw.InternetGatewayId))
				continue
			}
			name := ""
			tags := make(map[string]string)
			for _, tag := range igw.Tags {
				if aws.ToString(tag.Key) == AwsTagKeyName {
					name = aws.ToString(tag.Value)
				}
				tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
			}
			if name == "" {
				name = fmt.Sprintf("%s-igw", a.cluster.Name)
			}
			if nameVal, ok := tags[AwsTagKeyName]; !ok || nameVal != name {
				tags[AwsTagKeyName] = name
				err = a.createTags(ctx, aws.ToString(igw.InternetGatewayId), biz.ResourceTypeInternetGateway, tags)
				if err != nil {
					return err
				}
			}
			tags[AwsTagKeyName] = name
			a.cluster.AddCloudResource(biz.ResourceTypeInternetGateway, &biz.CloudResource{
				Name: name,
				ID:   aws.ToString(igw.InternetGatewayId),
				Tags: tags,
			})
			a.log.Infof("internet gateway %s already exists", aws.ToString(igw.InternetGatewayId))
		}
		return nil
	}

	// Create Internet Gateway if it doesn't exist
	name := fmt.Sprintf("%s-igw", a.cluster.Name)
	tags := map[string]string{
		AwsTagKeyName: name,
	}
	igwOutput, err := a.ec2Client.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{
		TagSpecifications: []ec2Types.TagSpecification{
			{
				ResourceType: ec2Types.ResourceTypeInternetGateway,
				Tags:         a.mapToEc2Tags(tags),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create Internet Gateway")
	}
	a.cluster.AddCloudResource(biz.ResourceTypeInternetGateway, &biz.CloudResource{
		Name: name,
		ID:   aws.ToString(igwOutput.InternetGateway.InternetGatewayId),
		Tags: tags,
	})

	_, err = a.ec2Client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeInternetGateway).ID),
		VpcId:             aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
	})
	if err != nil {
		return errors.Wrap(err, "failed to attach Internet Gateway")
	}
	a.log.Infof("internet gateway %s created", aws.ToString(igwOutput.InternetGateway.InternetGatewayId))
	return nil
}

// Check and Create NAT Gateways
func (a *AwsCloud) createNATGateways(ctx context.Context) error {
	if a.cluster.Level == biz.ClusterLevelBasic {
		return nil
	}
	existingNatGateways, err := a.ec2Client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
		Filter: []ec2Types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID}},
			{Name: aws.String("state"), Values: []string{"available"}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe NAT Gateways")
	}

	for _, natGateway := range existingNatGateways.NatGateways {
		if natGateway.SubnetId == nil || len(natGateway.NatGatewayAddresses) == 0 {
			continue
		}
		if a.cluster.GetCloudResourceByID(biz.ResourceTypeNATGateway, aws.ToString(natGateway.NatGatewayId)) != nil {
			a.log.Infof("nat gateway %s already exists", aws.ToString(natGateway.NatGatewayId))
			continue
		}
		// check public subnet
		subnetCloudResource := a.cluster.GetCloudResourceByID(biz.ResourceTypeSubnet, aws.ToString(natGateway.SubnetId))
		if subnetCloudResource == nil {
			continue
		}
		if val, ok := subnetCloudResource.Tags[AwsTagKeyType]; !ok || val != AwsResourcePublic {
			continue
		}
		tags := make(map[string]string)
		for _, tag := range natGateway.Tags {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
		name := fmt.Sprintf("%s-nat-gateway-%s", a.cluster.Name, subnetCloudResource.Tags[AwsTagKeyZone])
		tags[AwsTagKeyZone] = subnetCloudResource.Tags[AwsTagKeyZone]
		if nameVal, ok := tags[AwsTagKeyName]; !ok || nameVal != name {
			tags[AwsTagKeyName] = name
			err = a.createTags(ctx, aws.ToString(natGateway.NatGatewayId), biz.ResourceTypeNATGateway, tags)
			if err != nil {
				return err
			}
		}
		tags[AwsTagKeyName] = name
		a.cluster.AddCloudResource(biz.ResourceTypeNATGateway, &biz.CloudResource{
			Name: name,
			ID:   aws.ToString(natGateway.NatGatewayId),
			Tags: tags,
		})
		a.log.Infof("nat gateway %s already exists", aws.ToString(natGateway.NatGatewayId))
	}

	// Get Elastic IP
	eipRes, err := a.ec2Client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	if err != nil {
		return errors.Wrap(err, "failed to describe Elastic IPs")
	}
	for _, eip := range eipRes.Addresses {
		if eip.Domain != ec2Types.DomainTypeVpc {
			continue
		}
		if eip.AssociationId != nil || eip.InstanceId != nil || eip.NetworkInterfaceId != nil {
			continue
		}
		if a.cluster.GetCloudResourceByID(biz.ResourceTypeElasticIP, aws.ToString(eip.AllocationId)) != nil {
			a.log.Infof("elastic ip %s already exists", aws.ToString(eip.PublicIp))
			continue
		}
		name := ""
		tags := make(map[string]string)
		for _, tag := range eip.Tags {
			if aws.ToString(tag.Key) == AwsTagKeyName {
				name = aws.ToString(tag.Value)
			}
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
		a.cluster.AddCloudResource(biz.ResourceTypeElasticIP, &biz.CloudResource{
			ID:    aws.ToString(eip.AllocationId),
			Name:  name,
			Value: aws.ToString(eip.PublicIp),
			Tags:  tags,
		})
		a.log.Infof("elastic ip %s already exists", aws.ToString(eip.PublicIp))
	}

	// Allocate Elastic IP
	usedEipID := make([]string, 0)
	for _, az := range a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones) {
		natGatewayName := fmt.Sprintf("%s-nat-gateway-%s", a.cluster.Name, az.Name)
		if a.cluster.GetCloudResourceByName(biz.ResourceTypeNATGateway, natGatewayName) != nil {
			continue
		}
		eipName := fmt.Sprintf("%s-eip-%s", a.cluster.Name, az.Name)
		eipTags := map[string]string{AwsTagKeyName: eipName, AwsTagKeyZone: az.Name}
		for _, eipResource := range a.cluster.GetCloudResource(biz.ResourceTypeElasticIP) {
			if utils.InArray(eipResource.ID, usedEipID) {
				continue
			}
			if eipName != eipResource.Name {
				err = a.createTags(ctx, eipResource.ID, biz.ResourceTypeElasticIP, eipTags)
				if err != nil {
					return err
				}
			}
			eipResource.Name = eipName
			eipResource.Tags = eipTags
			usedEipID = append(usedEipID, eipResource.ID)
			break
		}

		if a.cluster.GetCloudResourceByTags(biz.ResourceTypeElasticIP, AwsTagKeyName, eipName) == nil {
			eipOutput, err := a.ec2Client.AllocateAddress(ctx, &ec2.AllocateAddressInput{
				Domain: ec2Types.DomainTypeVpc,
				TagSpecifications: []ec2Types.TagSpecification{
					{
						ResourceType: ec2Types.ResourceTypeElasticIp,
						Tags:         a.mapToEc2Tags(eipTags),
					},
				},
			})
			if err != nil {
				return errors.Wrap(err, "failed to allocate Elastic IP")
			}
			a.cluster.AddCloudResource(biz.ResourceTypeElasticIP, &biz.CloudResource{
				ID:    aws.ToString(eipOutput.AllocationId),
				Name:  eipName,
				Value: aws.ToString(eipOutput.PublicIp),
				Tags:  eipTags,
			})
			a.log.Infof("elastic ip %s allocated", aws.ToString(eipOutput.PublicIp))
		}
	}

	// Create NAT Gateways if they don't exist for each AZ
	natGateWayIds := make([]string, 0)
	for _, az := range a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones) {
		natGatewayName := fmt.Sprintf("%s-nat-gateway-%s", a.cluster.Name, az.Name)
		if a.cluster.GetCloudResourceByName(biz.ResourceTypeNATGateway, natGatewayName) != nil {
			continue
		}

		// Create NAT Gateway
		natGatewayTags := map[string]string{
			AwsTagKeyName: natGatewayName,
			AwsTagKeyType: AwsResourcePublic,
			AwsTagKeyZone: az.Name,
		}
		// eip
		eips := a.cluster.GetCloudResourceByTags(biz.ResourceTypeElasticIP, AwsTagKeyZone, az.Name)
		if len(eips) == 0 {
			return errors.New("no Elastic IP found for AZ " + az.Name)
		}
		// public subnet
		publickSubnets := a.cluster.GetCloudResourceByTags(biz.ResourceTypeSubnet, AwsTagKeyZone, az.Name, AwsTagKeyType, AwsResourcePublic)
		if len(publickSubnets) == 0 {
			return errors.New("no public subnet found for AZ " + az.Name)
		}
		natGatewayOutput, err := a.ec2Client.CreateNatGateway(ctx, &ec2.CreateNatGatewayInput{
			AllocationId:     &eips[0].ID,
			SubnetId:         &publickSubnets[0].ID,
			ConnectivityType: ec2Types.ConnectivityTypePublic,
			TagSpecifications: []ec2Types.TagSpecification{
				{
					ResourceType: ec2Types.ResourceTypeNatgateway, // natgateway
					Tags:         a.mapToEc2Tags(natGatewayTags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create NAT Gateway")
		}
		natGateWayIds = append(natGateWayIds, *natGatewayOutput.NatGateway.NatGatewayId)
		a.cluster.AddCloudResource(biz.ResourceTypeNATGateway, &biz.CloudResource{
			Name: natGatewayName,
			ID:   aws.ToString(natGatewayOutput.NatGateway.NatGatewayId),
			Tags: natGatewayTags,
		})
		a.log.Infof("nat gateway %s createing...", aws.ToString(natGatewayOutput.NatGateway.NatGatewayId))
	}

	if len(natGateWayIds) != 0 {
		a.log.Info("waiting for NAT Gateway availability")
		waiter := ec2.NewNatGatewayAvailableWaiter(a.ec2Client)
		err := waiter.Wait(ctx, &ec2.DescribeNatGatewaysInput{
			NatGatewayIds: natGateWayIds,
		}, time.Duration(len(natGateWayIds))*TimeoutPerInstance)
		if err != nil {
			return fmt.Errorf("failed to wait for NAT Gateway availability: %w", err)
		}
	}
	return nil
}

// Check and Create route tables
func (a *AwsCloud) createRouteTables(ctx context.Context) error {
	existingRouteTables, err := a.ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []ec2Types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe route tables")
	}

	for _, routeTable := range existingRouteTables.RouteTables {
		if routeTable.Tags == nil {
			continue
		}
		if a.cluster.GetCloudResourceByID(biz.ResourceTypeRouteTable, aws.ToString(routeTable.RouteTableId)) != nil {
			a.log.Infof("route table %s already exists", aws.ToString(routeTable.RouteTableId))
			continue
		}
		name := ""
		tags := make(map[string]string)
		for _, tag := range routeTable.Tags {
			if aws.ToString(tag.Key) == AwsTagKeyName {
				name = aws.ToString(tag.Value)
			}
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
		if val, ok := tags[AwsTagKeyType]; !ok || (val != AwsResourcePublic && val != AwsResourcePrivate) {
			continue
		}
		if tags[AwsTagKeyType] == AwsResourcePublic && name != fmt.Sprintf("%s-public-rt", a.cluster.Name) {
			continue
		}
		if tags[AwsTagKeyType] == AwsResourcePrivate {
			privateZoneName, ok := tags[AwsTagKeyZone]
			if !ok {
				continue
			}
			if name != fmt.Sprintf("%s-private-rt-%s", a.cluster.Name, privateZoneName) {
				continue
			}
		}
		a.cluster.AddCloudResource(biz.ResourceTypeRouteTable, &biz.CloudResource{
			Name: name,
			ID:   aws.ToString(routeTable.RouteTableId),
			Tags: tags,
		})
		a.log.Infof("route table %s already exists", aws.ToString(routeTable.RouteTableId))
	}

	// Create public route table
	publicRouteTableName := fmt.Sprintf("%s-public-rt", a.cluster.Name)
	publicRouteTableNameTags := map[string]string{
		AwsTagKeyName: publicRouteTableName,
		AwsTagKeyType: AwsResourcePublic,
	}
	if a.cluster.GetCloudResourceByName(biz.ResourceTypeRouteTable, publicRouteTableName) == nil {
		publicRouteTable, err := a.ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
			VpcId: aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
			TagSpecifications: []ec2Types.TagSpecification{
				{
					ResourceType: ec2Types.ResourceTypeRouteTable,
					Tags:         a.mapToEc2Tags(publicRouteTableNameTags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create public route table")
		}
		a.cluster.AddCloudResource(biz.ResourceTypeRouteTable, &biz.CloudResource{
			Name: publicRouteTableName,
			ID:   aws.ToString(publicRouteTable.RouteTable.RouteTableId),
			Tags: publicRouteTableNameTags,
		})
		a.log.Infof("public route table %s created", aws.ToString(publicRouteTable.RouteTable.RouteTableId))

		// Add route to Internet Gateway in public route table
		_, err = a.ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
			RouteTableId:         publicRouteTable.RouteTable.RouteTableId,
			DestinationCidrBlock: aws.String("0.0.0.0/0"),
			GatewayId:            aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeInternetGateway).ID),
		})
		if err != nil {
			return errors.Wrap(err, "failed to add route to Internet Gateway")
		}

		// Associate public subnets with public route table
		for i, subnetReource := range a.cluster.GetCloudResource(biz.ResourceTypeSubnet) {
			if typeVal, ok := subnetReource.Tags[AwsTagKeyType]; !ok || typeVal != AwsResourcePublic {
				continue
			}
			publicAssociateRouteTable, err := a.ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
				RouteTableId: publicRouteTable.RouteTable.RouteTableId,
				SubnetId:     aws.String(subnetReource.ID),
			})
			if err != nil {
				return errors.Wrap(err, "failed to associate public subnet with route table")
			}
			a.cluster.AddSubCloudResource(biz.ResourceTypeRouteTable, *publicRouteTable.RouteTable.RouteTableId, &biz.CloudResource{
				ID:   aws.ToString(publicAssociateRouteTable.AssociationId),
				Name: fmt.Sprintf("public associate routetable %d", i),
			})
		}
	}

	// Create private route tables (one per AZ)
	for _, az := range a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones) {
		privateRouteTableName := fmt.Sprintf("%s-private-rt-%s", a.cluster.Name, az.Name)
		tags := map[string]string{
			AwsTagKeyName: privateRouteTableName,
			AwsTagKeyType: AwsResourcePrivate,
			AwsTagKeyZone: az.Name,
		}
		if a.cluster.GetCloudResourceByTags(biz.ResourceTypeRouteTable, AwsTagKeyName, privateRouteTableName) != nil {
			continue
		}
		privateRouteTable, err := a.ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
			VpcId: aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
			TagSpecifications: []ec2Types.TagSpecification{
				{
					ResourceType: ec2Types.ResourceTypeRouteTable,
					Tags:         a.mapToEc2Tags(tags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create private route table for AZ "+az.Name)
		}
		a.cluster.AddCloudResource(biz.ResourceTypeRouteTable, &biz.CloudResource{
			Name: privateRouteTableName,
			ID:   aws.ToString(privateRouteTable.RouteTable.RouteTableId),
			Tags: tags,
		})
		a.log.Infof("private route table %s created for AZ %s", aws.ToString(privateRouteTable.RouteTable.RouteTableId), az.Name)

		// defalut local

		// Add route to NAT Gateway in private route table
		for _, natGateway := range a.cluster.GetCloudResource(biz.ResourceTypeNATGateway) {
			if zoneName, ok := natGateway.Tags[AwsTagKeyZone]; !ok || zoneName != az.Name {
				continue
			}
			_, err = a.ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
				RouteTableId:         privateRouteTable.RouteTable.RouteTableId,
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				NatGatewayId:         aws.String(natGateway.ID),
			})
			if err != nil {
				return errors.Wrap(err, "failed to add route to NAT Gateway for AZ "+az.Name)
			}
		}

		// Associate private subnets with private route table
		for _, subnet := range a.cluster.GetCloudResourceByTags(biz.ResourceTypeSubnet, AwsTagKeyType, AwsResourcePrivate, AwsTagKeyZone, az.Name) {
			privateAssociateRouteTable, err := a.ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
				RouteTableId: privateRouteTable.RouteTable.RouteTableId,
				SubnetId:     aws.String(subnet.ID),
			})
			if err != nil {
				return errors.Wrap(err, "failed to associate private subnet with route table in AZ "+az.Name)
			}
			a.cluster.AddSubCloudResource(biz.ResourceTypeRouteTable, *privateRouteTable.RouteTable.RouteTableId, &biz.CloudResource{
				ID:   aws.ToString(privateAssociateRouteTable.AssociationId),
				Name: fmt.Sprintf("%s-private-associate-routetable", subnet.Name),
			})
		}
	}
	return nil
}

// Check and Create security group
func (a *AwsCloud) createSecurityGroup(ctx context.Context) error {
	sgNames := []string{
		fmt.Sprintf("%s-%s-sg", a.cluster.Name, AwsResourceHttpSG),
		fmt.Sprintf("%s-%s-sg", a.cluster.Name, AwsReousrceBostionHostSG),
	}

	existingSecurityGroups, err := a.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []ec2Types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID}},
			{Name: aws.String("group-name"), Values: sgNames},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe security groups")
	}

	if len(existingSecurityGroups.SecurityGroups) != 0 {
		for _, securityGroup := range existingSecurityGroups.SecurityGroups {
			if securityGroup.GroupId == nil {
				continue
			}
			if a.cluster.GetCloudResourceByID(biz.ResourceTypeSecurityGroup, aws.ToString(securityGroup.GroupId)) != nil {
				a.log.Infof("security group %s already exists", aws.ToString(securityGroup.GroupId))
				continue
			}
			tags := make(map[string]string)
			for _, tag := range securityGroup.Tags {
				tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
			}
			a.cluster.AddCloudResource(biz.ResourceTypeSecurityGroup, &biz.CloudResource{
				Name: aws.ToString(securityGroup.GroupName),
				ID:   aws.ToString(securityGroup.GroupId),
				Tags: tags,
			})
			a.log.Infof("security group %s already exists", aws.ToString(securityGroup.GroupId))
		}
	}

	for _, sgName := range sgNames {
		if a.cluster.GetCloudResourceByName(biz.ResourceTypeSecurityGroup, sgName) != nil {
			continue
		}
		tags := map[string]string{AwsTagKeyName: sgName}
		if strings.Contains(sgName, AwsResourceHttpSG) {
			tags[AwsTagKeyType] = AwsResourceHttpSG
		}
		if strings.Contains(sgName, AwsReousrceBostionHostSG) {
			tags[AwsTagKeyType] = AwsReousrceBostionHostSG
		}
		sgOutput, err := a.ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(sgName),
			VpcId:       aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
			Description: aws.String(sgName),
			TagSpecifications: []ec2Types.TagSpecification{
				{
					ResourceType: ec2Types.ResourceTypeSecurityGroup,
					Tags:         a.mapToEc2Tags(tags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create security group")
		}
		a.cluster.AddCloudResource(biz.ResourceTypeSecurityGroup, &biz.CloudResource{
			Name: sgName,
			ID:   aws.ToString(sgOutput.GroupId),
			Tags: tags,
		})
		a.log.Infof("security group %s created", aws.ToString(sgOutput.GroupId))
		if v, ok := tags[AwsTagKeyType]; ok && v == AwsReousrceBostionHostSG {
			_, err = a.ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
				GroupId: sgOutput.GroupId,
				IpPermissions: []ec2Types.IpPermission{
					{
						IpProtocol: aws.String(string(ec2Types.ProtocolTcp)),
						FromPort:   aws.Int32(22),
						ToPort:     aws.Int32(22),
						IpRanges:   []ec2Types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
					},
					{
						IpProtocol: aws.String(string(ec2Types.ProtocolTcp)),
						FromPort:   aws.Int32(utils.GetPortByAddr(a.conf.Server.HTTP.Addr)),
						ToPort:     aws.Int32(utils.GetPortByAddr(a.conf.Server.HTTP.Addr)),
						IpRanges:   []ec2Types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
					},
					{
						IpProtocol: aws.String(string(ec2Types.ProtocolTcp)),
						FromPort:   aws.Int32(utils.GetPortByAddr(a.conf.Server.GRPC.Addr)),
						ToPort:     aws.Int32(utils.GetPortByAddr(a.conf.Server.GRPC.Addr)),
						IpRanges:   []ec2Types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
					},
				},
				TagSpecifications: []ec2Types.TagSpecification{
					{
						ResourceType: ec2Types.ResourceTypeSecurityGroupRule,
						Tags:         a.mapToEc2Tags(tags),
					},
				},
			})
			if err != nil {
				return errors.Wrap(err, "failed to add inbound rules to security group")
			}
		}
		if v, ok := tags[AwsTagKeyType]; ok && v == AwsResourceHttpSG {
			_, err = a.ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
				GroupId: sgOutput.GroupId,
				IpPermissions: []ec2Types.IpPermission{
					{
						IpProtocol: aws.String(string(ec2Types.ProtocolTcp)),
						FromPort:   aws.Int32(80),
						ToPort:     aws.Int32(80),
						IpRanges:   []ec2Types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
					},
					{
						IpProtocol: aws.String(string(ec2Types.ProtocolTcp)),
						FromPort:   aws.Int32(443),
						ToPort:     aws.Int32(443),
						IpRanges:   []ec2Types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
					},
				},
				TagSpecifications: []ec2Types.TagSpecification{
					{
						ResourceType: ec2Types.ResourceTypeSecurityGroupRule,
						Tags:         a.mapToEc2Tags(tags),
					},
				},
			})
			if err != nil {
				return errors.Wrap(err, "failed to add inbound rules to security group")
			}
		}
	}
	return nil
}

func (a *AwsCloud) createS3Endpoint(ctx context.Context) error {
	vpcResource := a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC)
	if vpcResource == nil {
		return errors.New("vpc resource not found")
	}
	privateRouterTable := a.cluster.GetCloudResourceByTags(biz.ResourceTypeRouteTable, AwsTagKeyType, AwsResourcePrivate)
	if privateRouterTable == nil {
		return errors.New("public route table not found")
	}
	routerTableids := make([]string, 0)
	for _, v := range privateRouterTable {
		routerTableids = append(routerTableids, v.ID)
	}

	if a.cluster.GetCloudResourceByTags(biz.ResourceTypeVpcEndpointS3, AwsTagKeyVpc, vpcResource.ID) != nil {
		a.log.Infof("s3 endpoint already exists")
		return nil
	}

	// s3 gateway
	name := fmt.Sprintf("%s-s3-endpoint", a.cluster.Name)
	tags := map[string]string{
		AwsTagKeyName: name,
		AwsTagKeyVpc:  vpcResource.ID,
	}
	serviceNmae := fmt.Sprintf("com.amazonaws.%s.s3", a.cluster.Region)
	endpointoutpus, err := a.ec2Client.DescribeVpcEndpoints(ctx, &ec2.DescribeVpcEndpointsInput{
		Filters: []ec2Types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcResource.ID}},
			{Name: aws.String("service-name"), Values: []string{serviceNmae}},
			{Name: aws.String("tag:Name"), Values: []string{name}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe s3 endpoint")
	}
	for _, endpoint := range endpointoutpus.VpcEndpoints {
		if endpoint.VpcEndpointId == nil {
			continue
		}
		a.cluster.AddCloudResource(biz.ResourceTypeVpcEndpointS3, &biz.CloudResource{
			ID:   aws.ToString(endpoint.VpcEndpointId),
			Name: name,
			Tags: tags,
		})
		a.log.Infof("s3 endpoint %s already exists", aws.ToString(endpoint.VpcEndpointId))
		return nil
	}
	s3enpointoutput, err := a.ec2Client.CreateVpcEndpoint(ctx, &ec2.CreateVpcEndpointInput{
		VpcId:           aws.String(vpcResource.ID),
		ServiceName:     aws.String(serviceNmae), // com.amazonaws.us-east-1.s3
		VpcEndpointType: ec2Types.VpcEndpointTypeGateway,
		RouteTableIds:   routerTableids,
		PolicyDocument:  aws.String("{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":\"*\",\"Action\":\"*\",\"Resource\":\"*\"}]}"),
		TagSpecifications: []ec2Types.TagSpecification{
			{
				ResourceType: ec2Types.ResourceTypeVpcEndpoint,
				Tags:         a.mapToEc2Tags(tags),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create s3 endpoint")
	}
	a.cluster.AddCloudResource(biz.ResourceTypeVpcEndpointS3, &biz.CloudResource{
		ID:   aws.ToString(s3enpointoutput.VpcEndpoint.VpcEndpointId),
		Name: name,
		Tags: tags,
	})
	a.log.Infof("s3 endpoint %s created", aws.ToString(s3enpointoutput.VpcEndpoint.VpcEndpointId))
	return nil
}

// create slb
func (a *AwsCloud) createSLB(ctx context.Context) error {
	if a.cluster.Level == biz.ClusterLevelBasic {
		a.log.Info("skip create slb for basic cluster")
		return nil
	}
	// Check if SLB already exists
	name := fmt.Sprintf("%s-slb", a.cluster.Name)
	if a.cluster.GetCloudResourceByName(biz.ResourceTypeLoadBalancer, name) != nil {
		a.log.Infof("slb %s already exists", name)
		return nil
	}
	publicSubnetIDs := make([]string, 0)
	for _, subnet := range a.cluster.GetCloudResource(biz.ResourceTypeSubnet) {
		if typeVal, ok := subnet.Tags[AwsTagKeyType]; !ok || typeVal != AwsResourcePublic {
			continue
		}
		publicSubnetIDs = append(publicSubnetIDs, subnet.ID)
	}
	if len(publicSubnetIDs) == 0 {
		return errors.New("failed to get public subnets")
	}
	sgs := a.cluster.GetCloudResourceByTags(biz.ResourceTypeSecurityGroup, AwsTagKeyType, AwsResourceHttpSG)
	if sgs == nil {
		return errors.New("failed to get security group")
	}
	sgIDs := make([]string, 0)
	for _, v := range sgs {
		sgIDs = append(sgIDs, v.ID)
	}

	loadBalancers, err := a.elbv2Client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
		Names: []string{name},
	})
	if err != nil && !strings.Contains(err.Error(), AwsNotFound) {
		return errors.Wrap(err, "failed to describe load balancers")
	}
	if loadBalancers != nil && loadBalancers.LoadBalancers != nil && len(loadBalancers.LoadBalancers) != 0 {
		for _, loadBalancer := range loadBalancers.LoadBalancers {
			if loadBalancer.LoadBalancerArn == nil {
				continue
			}
			if a.cluster.GetCloudResourceByID(biz.ResourceTypeLoadBalancer, aws.ToString(loadBalancer.LoadBalancerArn)) != nil {
				continue
			}
			a.cluster.AddCloudResource(biz.ResourceTypeLoadBalancer, &biz.CloudResource{
				Name: aws.ToString(loadBalancer.LoadBalancerName),
				ID:   aws.ToString(loadBalancer.LoadBalancerArn),
			})
			a.log.Infof("slb %s already exists", aws.ToString(loadBalancer.LoadBalancerName))
		}
		return nil
	}

	// Create SLB
	tags := map[string]string{AwsTagKeyName: name}
	slbOutput, err := a.elbv2Client.CreateLoadBalancer(ctx, &elasticloadbalancingv2.CreateLoadBalancerInput{
		Name:           aws.String(name),
		Subnets:        publicSubnetIDs,
		SecurityGroups: sgIDs,
		Scheme:         elasticloadbalancingv2Types.LoadBalancerSchemeEnumInternetFacing,
		Type:           elasticloadbalancingv2Types.LoadBalancerTypeEnumApplication,
		Tags:           a.mapToElbv2Tags(tags),
	})
	if err != nil || len(slbOutput.LoadBalancers) == 0 {
		return errors.Wrap(err, "failed to create SLB")
	}
	slb := slbOutput.LoadBalancers[0]
	a.cluster.AddCloudResource(biz.ResourceTypeLoadBalancer, &biz.CloudResource{
		Name: name,
		ID:   aws.ToString(slb.LoadBalancerArn),
		Tags: tags,
	})

	// Create target group
	taggetGroup, err := a.elbv2Client.CreateTargetGroup(ctx, &elasticloadbalancingv2.CreateTargetGroupInput{
		Name:       aws.String(fmt.Sprintf("%s-targetgroup", a.cluster.Name)),
		TargetType: elasticloadbalancingv2Types.TargetTypeEnumAlb,
		Port:       aws.Int32(6443),
		Protocol:   elasticloadbalancingv2Types.ProtocolEnumHttp,
		VpcId:      aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
		Tags:       a.mapToElbv2Tags(tags),
	})
	if err != nil || len(taggetGroup.TargetGroups) == 0 {
		return errors.Wrap(err, "failed to create target group")
	}
	targetGroup := taggetGroup.TargetGroups[0]
	a.log.Infof("target group %s created", aws.ToString(targetGroup.TargetGroupArn))

	// create listener
	_, err = a.elbv2Client.CreateListener(ctx, &elasticloadbalancingv2.CreateListenerInput{
		DefaultActions: []elasticloadbalancingv2Types.Action{
			{
				Type: elasticloadbalancingv2Types.ActionTypeEnumForward,
				ForwardConfig: &elasticloadbalancingv2Types.ForwardActionConfig{
					TargetGroups: []elasticloadbalancingv2Types.TargetGroupTuple{
						{
							TargetGroupArn: targetGroup.TargetGroupArn,
							Weight:         aws.Int32(100),
						},
					},
				},
			},
		},
		LoadBalancerArn: slb.LoadBalancerArn,
		Port:            aws.Int32(6443),
		Protocol:        elasticloadbalancingv2Types.ProtocolEnumHttp,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create listener")
	}
	return nil
}

// create Tags
func (a *AwsCloud) createTags(ctx context.Context, resourceID string, resourceType biz.ResourceType, tags map[string]string) error {
	_, err := a.ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{resourceID},
		Tags:      a.mapToEc2Tags(tags),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create tags for %s", resourceType)
	}
	return nil
}

// map to ec2 tags
func (a *AwsCloud) mapToEc2Tags(tags map[string]string) []ec2Types.Tag {
	ec2Tags := []ec2Types.Tag{}
	for key, value := range tags {
		ec2Tags = append(ec2Tags, ec2Types.Tag{Key: aws.String(key), Value: aws.String(value)})
	}
	return ec2Tags
}

// map to elbv2 tags
func (a *AwsCloud) mapToElbv2Tags(tags map[string]string) []elasticloadbalancingv2Types.Tag {
	elbv2Tags := []elasticloadbalancingv2Types.Tag{}
	for key, value := range tags {
		elbv2Tags = append(elbv2Tags, elasticloadbalancingv2Types.Tag{Key: aws.String(key), Value: aws.String(value)})
	}
	return elbv2Tags
}

func getIntanceTypeFamilies(nodeGroup *biz.NodeGroup) string {
	if nodeGroup == nil || nodeGroup.Type == "" {
		return "m5.*"
	}
	switch nodeGroup.Type {
	case biz.NodeGroupTypeNormal:
		return "m5.*"
	case biz.NodeGroupTypeHighComputation:
		return "c5.*"
	case biz.NodeGroupTypeGPUAcceleraterd:
		return "p3.*"
	case biz.NodeGroupTypeHighMemory:
		return "r5.*"
	case biz.NodeGroupTypeLargeHardDisk:
		return "i3.*"
	default:
		return "m5.*"
	}
}

func determineUsername(amiName, amiDescription string) string {
	amiName = strings.ToLower(amiName)
	amiDescription = strings.ToLower(amiDescription)

	if strings.Contains(amiName, "amazon linux") || strings.Contains(amiDescription, "amazon linux") {
		return "ec2-user"
	} else if strings.Contains(amiName, "ubuntu") || strings.Contains(amiDescription, "ubuntu") {
		return "ubuntu"
	} else if strings.Contains(amiName, "centos") || strings.Contains(amiDescription, "centos") {
		return "centos"
	} else if strings.Contains(amiName, "debian") || strings.Contains(amiDescription, "debian") {
		return "admin"
	} else if strings.Contains(amiName, "rhel") || strings.Contains(amiDescription, "red hat") {
		return "ec2-user"
	} else if strings.Contains(amiName, "suse") || strings.Contains(amiDescription, "suse") {
		return "ec2-user"
	} else if strings.Contains(amiName, "fedora") || strings.Contains(amiDescription, "fedora") {
		return "fedora"
	} else if strings.Contains(amiName, "bitnami") || strings.Contains(amiDescription, "bitnami") {
		return "bitnami"
	}

	// Default to ec2-user if we can't determine the username
	return "ec2-user"
}
