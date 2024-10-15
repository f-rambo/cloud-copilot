package infrastructure

import (
	"context"
	"encoding/json"
	"os"
	"sort"

	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

type AwsCloud struct {
	cluster     *biz.Cluster
	ec2Client   *ec2.EC2
	elbv2Client *elbv2.ELBV2
	log         *log.Helper
}

const (
	awsDefaultRegion   = "us-east-1"
	AwsTagKeyName      = "Name"
	AwsTagKeyType      = "Type"
	AwsTagZone         = "Zone"
	AwsResourcePublic  = "Public"
	AwsResourcePrivate = "Private"
)

func NewAwsCloud(cluster *biz.Cluster, log *log.Helper) (*AwsCloud, error) {
	if cluster.Region == "" {
		cluster.Region = awsDefaultRegion
	}
	os.Setenv("AWS_REGION", cluster.Region)
	os.Setenv("AWS_DEFAULT_REGION", cluster.Region)
	os.Setenv("AWS_ACCESS_KEY_ID", cluster.AccessID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", cluster.AccessKey)
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cluster.Region),
	})
	if err != nil {
		return nil, err
	}
	return &AwsCloud{
		cluster:     cluster,
		ec2Client:   ec2.New(sess),
		elbv2Client: elbv2.New(sess),
		log:         log,
	}, nil
}

func (a *AwsCloud) GetRegions(ctx context.Context) ([]string, error) {
	regions := make([]string, 0)
	err := a.getAvailabilityZones(ctx)
	if err != nil {
		return nil, err
	}
	for _, zone := range a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones) {
		if zone.Value == nil {
			continue
		}
		regions = append(regions, cast.ToString(zone.Value))
	}
	return regions, nil
}

// create network(vpc, subnet, internet gateway,nat gateway, route table, security group)
func (a *AwsCloud) CreateNetwork(ctx context.Context) error {
	// Step 1: Check and Create VPC
	err := a.createVPC(ctx)
	if err != nil {
		return err
	}

	// Step 2: Get availability zones
	err = a.getAvailabilityZones(ctx)
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

	// Step 8: Create Slb
	err = a.createSLB(ctx)
	if err != nil {
		return err
	}

	return nil
}

// delete network(vpc, subnet, internet gateway, nat gateway, route table, security group)
func (a *AwsCloud) DeleteNetwork(ctx context.Context) error {
	// Step 1: Delete security group
	for _, sg := range a.cluster.GetCloudResource(biz.ResourceTypeSecurityGroup) {
		_, err := a.ec2Client.DeleteSecurityGroupWithContext(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: &sg.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete security group")
		}
		a.cluster.DeleteCloudResourceByID(biz.ResourceTypeSecurityGroup, sg.ID)
	}

	// Step 2: Delete route tables
	for _, rt := range a.cluster.GetCloudResource(biz.ResourceTypeRouteTable) {
		for _, subRtassoc := range rt.SubResources {
			_, err := a.ec2Client.DisassociateRouteTableWithContext(ctx, &ec2.DisassociateRouteTableInput{
				AssociationId: &subRtassoc.ID,
			})
			if err != nil {
				return errors.Wrap(err, "failed to disassociate route table")
			}
		}
		_, err := a.ec2Client.DeleteRouteTableWithContext(ctx, &ec2.DeleteRouteTableInput{
			RouteTableId: &rt.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete route table")
		}
		a.cluster.DeleteCloudResourceByID(biz.ResourceTypeRouteTable, rt.ID)
	}

	// Step 3: Delete Internet Gateway
	for _, igw := range a.cluster.GetCloudResource(biz.ResourceTypeInternetGateway) {
		_, err := a.ec2Client.DetachInternetGatewayWithContext(ctx, &ec2.DetachInternetGatewayInput{
			InternetGatewayId: &igw.ID,
			VpcId:             aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
		})
		if err != nil {
			return errors.Wrap(err, "failed to detach Internet Gateway")
		}
		_, err = a.ec2Client.DeleteInternetGatewayWithContext(ctx, &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: &igw.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete Internet Gateway")
		}
		a.cluster.DeleteCloudResourceByID(biz.ResourceTypeInternetGateway, igw.ID)
	}

	// Step 4: Delete NAT Gateways
	// client.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{})
	for _, natGw := range a.cluster.GetCloudResource(biz.ResourceTypeNATGateway) {
		_, err := a.ec2Client.DeleteNatGatewayWithContext(ctx, &ec2.DeleteNatGatewayInput{
			NatGatewayId: &natGw.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete NAT Gateway")
		}
		// Wait for NAT Gateway to be deleted
		err = a.ec2Client.WaitUntilNatGatewayDeletedWithContext(ctx, &ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []*string{&natGw.ID},
		})
		if err != nil {
			return errors.Wrap(err, "failed to wait for NAT Gateway deletion")
		}
		a.cluster.DeleteCloudResourceByID(biz.ResourceTypeNATGateway, natGw.ID)
	}

	// Release Elastic IPs associated with NAT Gateways
	for _, addr := range a.cluster.GetCloudResource(biz.ResourceTypeElasticIP) {
		_, err := a.ec2Client.ReleaseAddressWithContext(ctx, &ec2.ReleaseAddressInput{
			AllocationId: &addr.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to release Elastic IP")
		}
		a.cluster.DeleteCloudResourceByID(biz.ResourceTypeElasticIP, addr.ID)
	}

	// // Step 5: Delete Subnets
	for _, subnet := range a.cluster.GetCloudResource(biz.ResourceTypeSubnet) {
		_, err := a.ec2Client.DeleteSubnetWithContext(ctx, &ec2.DeleteSubnetInput{
			SubnetId: &subnet.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete subnet")
		}
		a.cluster.DeleteCloudResourceByID(biz.ResourceTypeSubnet, subnet.ID)
	}

	// Step 6: Delete VPC
	_, err := a.ec2Client.DeleteVpcWithContext(ctx, &ec2.DeleteVpcInput{
		VpcId: aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete VPC")
	}
	a.cluster.DeleteCloudResourceByID(biz.ResourceTypeVPC, a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID)

	// step 7: Delete SLB
	for _, slb := range a.cluster.GetCloudResource(biz.ResourceTypeLoadBalancer) {
		_, err := a.elbv2Client.DeleteLoadBalancerWithContext(ctx, &elbv2.DeleteLoadBalancerInput{
			LoadBalancerArn: &slb.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete SLB")
		}
		a.cluster.DeleteCloudResourceByID(biz.ResourceTypeLoadBalancer, slb.ID)
	}
	return nil
}

type GetInstanceTypeResults []*ec2.InstanceTypeInfo

// sort by vcpu and memory
func (a GetInstanceTypeResults) Len() int {
	return len(a)
}

func (a GetInstanceTypeResults) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a GetInstanceTypeResults) Less(i, j int) bool {
	if a[i] == nil {
		return true
	}
	if *a[i].VCpuInfo.DefaultVCpus < *a[j].VCpuInfo.DefaultVCpus {
		return true
	}
	if *a[i].VCpuInfo.DefaultVCpus == *a[j].VCpuInfo.DefaultVCpus {
		return *a[i].MemoryInfo.SizeInMiB < *a[j].MemoryInfo.SizeInMiB
	}
	return false
}

// get instance type familiy
func (a *AwsCloud) SetByNodeGroups(ctx context.Context) error {
	for _, ng := range a.cluster.NodeGroups {
		os := ng.OS
		if os == "" {
			os = "ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"
		}
		architecture := ng.ARCH
		if architecture == "" {
			architecture = "x86_64"
		}
		images, err := a.ec2Client.DescribeImagesWithContext(ctx, &ec2.DescribeImagesInput{
			Owners: []*string{aws.String("amazon")},
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("name"),
					Values: []*string{aws.String(os)},
				},
				{
					Name:   aws.String("architecture"),
					Values: []*string{aws.String(architecture)},
				},
				{
					Name:   aws.String("state"),
					Values: []*string{aws.String("available")},
				},
			},
		})
		if err != nil || len(images.Images) == 0 {
			return errors.Wrap(err, "failed to describe images")
		}
		image := images.Images[0]
		ng.Image = *image.ImageId
		ng.OS = *image.Name
		ng.ARCH = *image.Architecture
		a.log.Info("image found", "image", image.ImageId)

		// set instance type
		if ng.InstanceType != "" {
			continue
		}
		instanceTypeFamiliy := getIntanceTypeFamilies(ng)
		instanceTypes, err := a.ec2Client.DescribeInstanceTypesWithContext(ctx, &ec2.DescribeInstanceTypesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("current-generation"),
					Values: []*string{aws.String("true")},
				},
				{
					Name:   aws.String("processor-info.supported-architecture"),
					Values: []*string{aws.String(ng.ARCH)},
				},
				{
					Name:   aws.String("instance-type"),
					Values: []*string{aws.String(instanceTypeFamiliy)},
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe instance types")
		}
		if len(instanceTypes.InstanceTypes) == 0 {
			return errors.New("no instance types found")
		}
		instanceData := GetInstanceTypeResults(instanceTypes.InstanceTypes)
		sort.Sort(instanceData)
		for _, instanceType := range instanceData {
			if *instanceType.MemoryInfo.SizeInMiB == 0 {
				continue
			}
			memoryGBiSize := float64(*instanceType.MemoryInfo.SizeInMiB) / 1024.0
			if memoryGBiSize >= ng.Memory && int(*instanceType.VCpuInfo.DefaultVCpus) >= int(ng.CPU) {
				ng.InstanceType = *instanceType.InstanceType
			}
			if ng.InstanceType == "" {
				continue
			}
			if ng.GPU == 0 {
				break
			}
			for _, gpues := range instanceType.GpuInfo.Gpus {
				if *gpues.Count >= int64(ng.GPU) {
					break
				}
			}
		}
		if ng.InstanceType == "" {
			return errors.New("no instance type found")
		}
		a.log.Info("instance type found", "instanceType", ng.InstanceType)
	}
	return nil
}

// KeyPair
func (a *AwsCloud) ImportKeyPair(ctx context.Context) error {
	keyName := a.cluster.Name + "-keypair"
	tags := map[string]string{
		AwsTagKeyName: keyName,
	}
	keyPairOutput, err := a.ec2Client.ImportKeyPairWithContext(ctx, &ec2.ImportKeyPairInput{
		KeyName:           aws.String(keyName),
		PublicKeyMaterial: []byte(a.cluster.PublicKey),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String(ec2.ResourceTypeKeyPair),
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
		ID:   *keyPairOutput.KeyPairId,
		Tags: tags,
	})
	return nil
}

func (a *AwsCloud) DeleteKeyPair(ctx context.Context) error {
	keyPair := a.cluster.GetSingleCloudResource(biz.ResourceTypeKeyPair)
	if keyPair == nil {
		return nil
	}
	_, err := a.ec2Client.DeleteKeyPairWithContext(ctx, &ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyPair.Name),
	})
	if err != nil {
		return fmt.Errorf("failed to delete key pair: %v", err)
	}
	a.log.Info("key pair deleted")
	a.cluster.DeleteCloudResourceByID(biz.ResourceTypeKeyPair, keyPair.ID)
	return nil
}

func (a *AwsCloud) ManageInstance(ctx context.Context) error {
	instances, err := a.getInstances(ctx)
	if err != nil {
		return err
	}
	deleteInstanceIDs := make([]*string, 0)
	for _, instance := range instances {
		nodeExists := false
		for _, node := range a.cluster.Nodes {
			if node.InternalIP == *instance.InstanceId {
				nodeExists = true
				break
			}
		}
		if !nodeExists {
			deleteInstanceIDs = append(deleteInstanceIDs, instance.InstanceId)
		}
	}
	// Delete instances
	if len(deleteInstanceIDs) > 0 {
		_, err = a.ec2Client.TerminateInstancesWithContext(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: deleteInstanceIDs,
		})
		if err != nil {
			return errors.Wrap(err, "failed to terminate instances")
		}
		err = a.ec2Client.WaitUntilInstanceTerminatedWithContext(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: deleteInstanceIDs,
		})
		if err != nil {
			return errors.Wrap(err, "failed to wait for instance termination")
		}
		a.log.Info("instances terminated")
	}
	// Create instances
	instanceIds := make([]*string, 0)
	for index, node := range a.cluster.Nodes {
		instanceExits := false
		for _, instance := range instances {
			if node.InstanceID == *instance.InstanceId {
				instanceExits = true
				break
			}
		}
		if instanceExits {
			continue
		}
		nodeGroup := a.cluster.GetNodeGroup(node.NodeGroupID)
		subnetID := a.distributeNodeSubnets(index)
		nodeTags := make(map[string]string)
		if node.Labels != "" {
			err = json.Unmarshal([]byte(node.Labels), &nodeTags)
			if err != nil {
				return errors.Wrap(err, "failed to parse labels")
			}
		}
		nodeTags[AwsTagKeyName] = node.Name
		blockDeviceMappings := make([]*ec2.BlockDeviceMapping, 0)
		if node.SystemDisk > 0 {
			blockDeviceMappings = append(blockDeviceMappings, &ec2.BlockDeviceMapping{
				DeviceName: aws.String("/dev/xvda"),
				Ebs: &ec2.EbsBlockDevice{
					VolumeSize:          aws.Int64(int64(node.SystemDisk)),
					VolumeType:          aws.String(ec2.VolumeTypeGp2),
					DeleteOnTermination: aws.Bool(true),
				},
			})
		}
		if node.DataDisk > 0 {
			blockDeviceMappings = append(blockDeviceMappings, &ec2.BlockDeviceMapping{
				DeviceName: aws.String("/dev/sdf"),
				Ebs: &ec2.EbsBlockDevice{
					VolumeSize:          aws.Int64(int64(node.DataDisk)),
					VolumeType:          aws.String(ec2.VolumeTypeGp2),
					DeleteOnTermination: aws.Bool(true),
				},
			})
		}
		instanceOutput, err := a.ec2Client.RunInstancesWithContext(ctx, &ec2.RunInstancesInput{
			ImageId:             aws.String(nodeGroup.Image),
			InstanceType:        aws.String(nodeGroup.InstanceType),
			KeyName:             aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeKeyPair).Name),
			MaxCount:            aws.Int64(1),
			MinCount:            aws.Int64(1),
			SecurityGroupIds:    []*string{aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeSecurityGroup).ID)},
			SubnetId:            aws.String(subnetID),
			BlockDeviceMappings: blockDeviceMappings,
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String(ec2.ResourceTypeInstance),
					Tags:         a.mapToEc2Tags(nodeTags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to run instances")
		}
		for _, instance := range instanceOutput.Instances {
			a.cluster.AddCloudResource(biz.ResourceTypeInstance, &biz.CloudResource{
				Name: node.Name,
				ID:   *instance.InstanceId,
				Tags: nodeTags,
			})
			a.log.Info("instance createing", "name", node.Name, "id", *instance.InstanceId)
			instanceIds = append(instanceIds, instance.InstanceId)
		}
		node.Status = biz.NodeStatusCreating
	}
	err = a.ec2Client.WaitUntilInstanceRunningWithContext(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	})
	if err != nil {
		return errors.Wrap(err, "failed to wait for instance running")
	}
	// check instance status
	instancesOutPut, err := a.ec2Client.DescribeInstancesWithContext(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe instances")
	}
	for _, node := range a.cluster.Nodes {
		if node.Status != biz.NodeStatusCreating {
			continue
		}
		var instance *ec2.Instance
		for _, reservation := range instancesOutPut.Reservations {
			for _, instanceTmp := range reservation.Instances {
				if node.InstanceID == *instanceTmp.InstanceId {
					instance = instanceTmp
					break
				}
			}
		}
		if instance == nil {
			return errors.New("failed to find instance")
		}
		if *instance.State.Name != string(ec2.InstanceStateNameRunning) {
			return errors.New("failed to create instance")
		}
		node.InternalIP = *instance.PrivateIpAddress
		node.ExternalIP = *instance.PublicIpAddress
		node.Status = biz.NodeStatusRunning
		a.log.Info("instance created", "name", node.Name, "id", *instance.InstanceId)
	}
	return nil
}

func (a *AwsCloud) DeleteClusterAllInstance(ctx context.Context) error {
	instances, err := a.getInstances(ctx)
	if err != nil {
		return err
	}
	instanceIds := make([]*string, 0)
	for _, instance := range instances {
		instanceIds = append(instanceIds, instance.InstanceId)
	}
	_, err = a.ec2Client.TerminateInstancesWithContext(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
	})
	if err != nil {
		return errors.Wrap(err, "failed to terminate instances")
	}
	err = a.ec2Client.WaitUntilInstanceTerminatedWithContext(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	})
	if err != nil {
		return errors.Wrap(err, "failed to wait for instance termination")
	}
	for _, node := range a.cluster.Nodes {
		node.Status = biz.NodeStatusDeleted
	}
	a.cluster.DeleteCloudResource(biz.ResourceTypeInstance)
	a.log.Info("instances terminated")
	return nil
}

func (a *AwsCloud) getInstances(ctx context.Context) ([]*ec2.Instance, error) {
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(fmt.Sprintf("%s-node-*", a.cluster.Name))},
			},
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID)},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String(string(ec2.InstanceStateNameRunning))},
			},
		},
	}

	var instances []*ec2.Instance
	for {
		output, err := a.ec2Client.DescribeInstancesWithContext(ctx, input)
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

func (a *AwsCloud) distributeNodeSubnets(nodeIndex int) (subNetID string) {
	subnets := make([]*biz.CloudResource, 0)
	for _, subnet := range a.cluster.GetCloudResource(biz.ResourceTypeSubnet) {
		if typeValue, ok := subnet.Tags[AwsTagKeyType]; ok && typeValue == AwsResourcePrivate {
			subnets = append(subnets, subnet)
		}
	}
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
	existingVpcs, err := a.ec2Client.DescribeVpcsWithContext(ctx, &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).Name)},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe VPCs")
	}

	if len(existingVpcs.Vpcs) != 0 {
		for _, vpc := range existingVpcs.Vpcs {
			for _, tag := range vpc.Tags {
				vpcTags[*tag.Key] = *tag.Value
			}
			vpcCloudResource := a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC)
			vpcCloudResource.ID = *vpc.VpcId
			vpcCloudResource.Tags = vpcTags
		}
		return nil
	}

	// Create VPC if it doesn't exist
	vpcOutput, err := a.ec2Client.CreateVpcWithContext(ctx, &ec2.CreateVpcInput{
		CidrBlock: aws.String(a.cluster.IpCidr),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String(ec2.ResourceTypeVpc),
				Tags:         a.mapToEc2Tags(vpcTags),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create VPC")
	}
	vpcCloudResource := a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC)
	vpcCloudResource.ID = *vpcOutput.Vpc.VpcId

	_, err = a.ec2Client.ModifyVpcAttributeWithContext(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId: vpcOutput.Vpc.VpcId,
		EnableDnsSupport: &ec2.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to enable DNS support for VPC")
	}
	a.log.Infof("vpc %s created", a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID)
	return nil
}

// Get availability zones
func (a *AwsCloud) getAvailabilityZones(ctx context.Context) error {
	a.cluster.DeleteCloudResource(biz.ResourceTypeAvailabilityZones)
	result, err := a.ec2Client.DescribeAvailabilityZonesWithContext(ctx, &ec2.DescribeAvailabilityZonesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String("available")},
			},
			{
				Name:   aws.String("region-name"),
				Values: []*string{aws.String(a.cluster.Region)},
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
			Name:  *az.ZoneName,
			ID:    *az.ZoneId,
			Type:  biz.ResourceTypeAvailabilityZones,
			Value: *az.RegionName,
		})
	}
	return nil
}

// Check and Create subnets
func (a *AwsCloud) createSubnets(ctx context.Context) error {
	existingSubnets, err := a.ec2Client.DescribeSubnetsWithContext(ctx, &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID)},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe subnets")
	}

	if len(existingSubnets.Subnets) != 0 {
		zoneSubnets := make(map[string][]*ec2.Subnet)
		for _, subnet := range existingSubnets.Subnets {
			if subnet.AvailabilityZone == nil {
				continue
			}
			_, ok := zoneSubnets[*subnet.AvailabilityZone]
			if ok && len(zoneSubnets[*subnet.AvailabilityZone]) >= 3 {
				continue
			}
			zoneSubnets[*subnet.AvailabilityZone] = append(zoneSubnets[*subnet.AvailabilityZone], subnet)
		}
		for zoneName, subzoneSubnets := range zoneSubnets {
			for i, subnet := range subzoneSubnets {
				if subnet == nil || subnet.SubnetId == nil {
					continue
				}
				if a.cluster.GetCloudResourceByID(biz.ResourceTypeSubnet, *subnet.SubnetId) != nil {
					continue
				}
				tags := make(map[string]string)
				name := ""
				for _, tag := range subnet.Tags {
					tags[*tag.Key] = *tag.Value
				}
				tags[AwsTagZone] = zoneName
				if i < 2 {
					name = fmt.Sprintf("%s-private-subnet-%s-%d", a.cluster.Name, *subnet.AvailabilityZone, i+1)
					tags[AwsTagKeyType] = AwsResourcePrivate
				} else {
					name = fmt.Sprintf("%s-public-subnet-%s", a.cluster.Name, *subnet.AvailabilityZone)
					tags[AwsTagKeyType] = AwsResourcePublic
				}
				if nameVal, ok := tags[AwsTagKeyName]; !ok || nameVal != name {
					tags[AwsTagKeyName] = name
					err = a.createTags(ctx, *subnet.SubnetId, biz.ResourceTypeSubnet, tags)
					if err != nil {
						return err
					}
				}
				tags[AwsTagKeyName] = name
				a.cluster.AddCloudResource(biz.ResourceTypeSubnet, &biz.CloudResource{
					Name: name,
					ID:   *subnet.SubnetId,
					Tags: tags,
				})
			}
		}
	}

	privateSubnetCount := len(a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones)) * 2
	publicSubnetCount := len(a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones))
	subnetCidrRes, err := utils.GenerateSubnets(a.cluster.IpCidr, privateSubnetCount+publicSubnetCount+len(existingSubnets.Subnets))
	if err != nil {
		return errors.Wrap(err, "failed to generate subnet CIDRs")
	}
	subnetCidrs := make([]string, 0)
	existingSubnetCird := make(map[string]bool)
	for _, subnet := range existingSubnets.Subnets {
		existingSubnetCird[*subnet.CidrBlock] = true
	}
	for _, subnetCidr := range subnetCidrRes {
		if _, ok := existingSubnetCird[subnetCidr]; ok {
			continue
		}
		subnetCidrs = append(subnetCidrs, subnetCidr)
	}

	for i, az := range a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones) {
		// Create private subnets
		for j := 0; j < 2; j++ {
			name := fmt.Sprintf("%s-private-subnet-%s-%d", a.cluster.Name, az.Name, j+1)
			tags := map[string]string{
				AwsTagKeyName: name,
				AwsTagKeyType: AwsResourcePrivate,
				AwsTagZone:    az.Name,
			}
			if a.cluster.GetCloudResourceByTags(biz.ResourceTypeSubnet, AwsTagKeyName, name) != nil {
				continue
			}
			cidr := subnetCidrs[i*2+j]
			subnetOutput, err := a.ec2Client.CreateSubnetWithContext(ctx, &ec2.CreateSubnetInput{
				VpcId:            aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
				CidrBlock:        aws.String(cidr),
				AvailabilityZone: &az.Name,
				TagSpecifications: []*ec2.TagSpecification{
					{
						ResourceType: aws.String(ec2.ResourceTypeSubnet),
						Tags:         a.mapToEc2Tags(tags),
					},
				},
			})
			if err != nil {
				return errors.Wrap(err, "failed to create private subnet")
			}
			a.cluster.AddCloudResource(biz.ResourceTypeSubnet, &biz.CloudResource{
				Name: name,
				ID:   *subnetOutput.Subnet.SubnetId,
				Tags: tags,
			})
		}

		name := fmt.Sprintf("%s-public-subnet-%s", a.cluster.Name, az.Name)
		tags := map[string]string{
			AwsTagKeyName: name,
			AwsTagKeyType: AwsResourcePublic,
			AwsTagZone:    az.Name,
		}
		if a.cluster.GetCloudResourceByTags(biz.ResourceTypeSubnet, AwsTagKeyName, name) != nil {
			continue
		}
		// Create public subnet
		cidr := subnetCidrs[privateSubnetCount+i]
		subnetOutput, err := a.ec2Client.CreateSubnetWithContext(ctx, &ec2.CreateSubnetInput{
			VpcId:            aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
			CidrBlock:        aws.String(cidr),
			AvailabilityZone: &az.Name,
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String(ec2.ResourceTypeSubnet),
					Tags:         a.mapToEc2Tags(tags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create public subnet")
		}
		a.cluster.AddCloudResource(biz.ResourceTypeSubnet, &biz.CloudResource{
			Name: name,
			ID:   *subnetOutput.Subnet.SubnetId,
			Tags: tags,
		})
	}

	a.log.Info("subnet finished.")
	return nil
}

// Check and Create Internet Gateway
func (a *AwsCloud) createInternetGateway(ctx context.Context) error {
	existingIgws, err := a.ec2Client.DescribeInternetGatewaysWithContext(ctx, &ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("attachment.vpc-id"),
				Values: []*string{aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID)},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe Internet Gateways")
	}

	if len(existingIgws.InternetGateways) != 0 {
		for _, igw := range existingIgws.InternetGateways {
			if igw == nil || igw.InternetGatewayId == nil {
				continue
			}
			if a.cluster.GetCloudResourceByID(biz.ResourceTypeInternetGateway, *igw.InternetGatewayId) != nil {
				continue
			}
			name := ""
			tags := make(map[string]string)
			for _, tag := range igw.Tags {
				if *tag.Key == AwsTagKeyName {
					name = *tag.Value
				}
				tags[*tag.Key] = *tag.Value
			}
			if name == "" {
				name = fmt.Sprintf("%s-igw", a.cluster.Name)
			}
			if nameVal, ok := tags[AwsTagKeyName]; !ok || nameVal != name {
				tags[AwsTagKeyName] = name
				err = a.createTags(ctx, *igw.InternetGatewayId, biz.ResourceTypeInternetGateway, tags)
				if err != nil {
					return err
				}
			}
			tags[AwsTagKeyName] = name
			a.cluster.AddCloudResource(biz.ResourceTypeInternetGateway, &biz.CloudResource{
				Name: name,
				ID:   *igw.InternetGatewayId,
				Tags: tags,
			})
		}
		return nil
	}

	// Create Internet Gateway if it doesn't exist
	name := fmt.Sprintf("%s-igw", a.cluster.Name)
	tags := map[string]string{
		AwsTagKeyName: name,
	}
	igwOutput, err := a.ec2Client.CreateInternetGatewayWithContext(ctx, &ec2.CreateInternetGatewayInput{
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String(ec2.ResourceTypeInternetGateway),
				Tags:         a.mapToEc2Tags(tags),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create Internet Gateway")
	}
	a.cluster.AddCloudResource(biz.ResourceTypeInternetGateway, &biz.CloudResource{
		Name: name,
		ID:   *igwOutput.InternetGateway.InternetGatewayId,
		Tags: tags,
	})

	_, err = a.ec2Client.AttachInternetGatewayWithContext(ctx, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeInternetGateway).ID),
		VpcId:             aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
	})
	if err != nil {
		return errors.Wrap(err, "failed to attach Internet Gateway")
	}

	return nil
}

// Check and Create NAT Gateways
func (a *AwsCloud) createNATGateways(ctx context.Context) error {
	existingNatGateways, err := a.ec2Client.DescribeNatGatewaysWithContext(ctx, &ec2.DescribeNatGatewaysInput{
		Filter: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID)}},
			{Name: aws.String("state"), Values: []*string{aws.String("available")}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe NAT Gateways")
	}

	if len(existingNatGateways.NatGateways) != 0 {
		for _, natGateway := range existingNatGateways.NatGateways {
			if natGateway == nil || natGateway.SubnetId == nil || len(natGateway.NatGatewayAddresses) == 0 {
				continue
			}
			if a.cluster.GetCloudResourceByID(biz.ResourceTypeNATGateway, *natGateway.NatGatewayId) != nil {
				continue
			}
			// check public subnet
			subnetCloudResource := a.cluster.GetCloudResourceByID(biz.ResourceTypeSubnet, *natGateway.SubnetId)
			if subnetCloudResource == nil {
				continue
			}
			if val, ok := subnetCloudResource.Tags[AwsTagKeyType]; !ok || val != AwsResourcePublic {
				continue
			}
			tags := make(map[string]string)
			for _, tag := range natGateway.Tags {
				tags[*tag.Key] = *tag.Value
			}
			name := fmt.Sprintf("%s-nat-gateway-%s", a.cluster.Name, subnetCloudResource.Tags[AwsTagZone])
			tags[AwsTagZone] = subnetCloudResource.Tags[AwsTagZone]
			if nameVal, ok := tags[AwsTagKeyName]; !ok || nameVal != name {
				tags[AwsTagKeyName] = name
				err = a.createTags(ctx, *natGateway.NatGatewayId, biz.ResourceTypeNATGateway, tags)
				if err != nil {
					return err
				}
			}
			tags[AwsTagKeyName] = name
			a.cluster.AddCloudResource(biz.ResourceTypeNATGateway, &biz.CloudResource{
				Name: name,
				ID:   *natGateway.NatGatewayId,
				Tags: tags,
			})
		}
	}

	// Create NAT Gateways if they don't exist for each AZ
	natGateWayIds := make([]*string, 0)
	for _, az := range a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones) {
		natGatewayName := fmt.Sprintf("%s-nat-gateway-%s", a.cluster.Name, az.Name)
		if a.cluster.GetCloudResourceByName(biz.ResourceTypeNATGateway, natGatewayName) != nil {
			continue
		}
		// Allocate Elastic IP
		eipName := fmt.Sprintf("%s-eip-%s", a.cluster.Name, az.Name)
		eipTags := map[string]string{
			AwsTagKeyName: eipName,
			AwsTagZone:    az.Name,
		}
		eipCloudResouce := &biz.CloudResource{
			Name: eipName,
			Tags: eipTags,
		}
		// get Elastic IP
		eipRes, err := a.ec2Client.DescribeAddressesWithContext(ctx, &ec2.DescribeAddressesInput{})
		if err != nil {
			return errors.Wrap(err, "failed to describe Elastic IPs")
		}
		for _, eip := range eipRes.Addresses {
			if eip == nil {
				continue
			}
			if eip.Domain == nil {
				continue
			}
			if *eip.Domain != "vpc" {
				continue
			}
			if eip.AssociationId != nil || eip.InstanceId != nil || eip.NetworkInterfaceId != nil {
				continue
			}
			if a.cluster.GetCloudResourceByID(biz.ResourceTypeElasticIP, *eip.AllocationId) != nil {
				continue
			}
			eipCloudResouce.ID = *eip.AllocationId
			eipCloudResouce.Value = *eip.PublicIp
			a.cluster.AddCloudResource(biz.ResourceTypeElasticIP, eipCloudResouce)
			break
		}
		if a.cluster.GetCloudResourceByTags(biz.ResourceTypeElasticIP, AwsTagKeyName, eipName) == nil {
			// Allocate Elastic IP
			eipOutput, err := a.ec2Client.AllocateAddressWithContext(ctx, &ec2.AllocateAddressInput{
				Domain: aws.String("vpc"),
				TagSpecifications: []*ec2.TagSpecification{
					{
						ResourceType: aws.String(ec2.ResourceTypeElasticIp),
						Tags:         a.mapToEc2Tags(eipTags),
					},
				},
			})
			if err != nil {
				return errors.Wrap(err, "failed to allocate Elastic IP")
			}
			eipCloudResouce.ID = *eipOutput.AllocationId
			eipCloudResouce.Value = *eipOutput.PublicIp
			a.cluster.AddCloudResource(biz.ResourceTypeElasticIP, eipCloudResouce)
		}
		a.log.Info("created Elastic IP for NAT Gateway")

		// Create NAT Gateway
		natGatewayTags := map[string]string{
			AwsTagKeyName: natGatewayName,
			AwsTagKeyType: AwsResourcePublic,
			AwsTagZone:    az.Name,
		}
		if a.cluster.GetCloudResourceByName(biz.ResourceTypeNATGateway, natGatewayName) != nil {
			continue
		}

		publickSubnet := a.cluster.GetCloudResourceByTags(biz.ResourceTypeSubnet, AwsTagZone, az.Name, AwsTagKeyType, AwsResourcePublic)
		if publickSubnet == nil {
			return errors.New("no public subnet found for AZ " + az.Name)
		}
		natGatewayOutput, err := a.ec2Client.CreateNatGatewayWithContext(ctx, &ec2.CreateNatGatewayInput{
			AllocationId:     &eipCloudResouce.ID,
			ConnectivityType: aws.String("public"),
			SubnetId:         &publickSubnet.ID,
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String(ec2.ResourceTypeNatgateway), // natgateway
					Tags:         a.mapToEc2Tags(natGatewayTags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create NAT Gateway")
		}
		natGateWayIds = append(natGateWayIds, natGatewayOutput.NatGateway.NatGatewayId)
		a.cluster.AddCloudResource(biz.ResourceTypeNATGateway, &biz.CloudResource{
			Name: natGatewayName,
			ID:   *natGatewayOutput.NatGateway.NatGatewayId,
			Tags: natGatewayTags,
		})
	}

	if len(natGateWayIds) != 0 {
		// Wait for NAT Gateway availability
		a.log.Info("waiting for NAT Gateway availability")
		err = a.ec2Client.WaitUntilNatGatewayAvailableWithContext(ctx, &ec2.DescribeNatGatewaysInput{
			NatGatewayIds: natGateWayIds,
		})
		if err != nil {
			return errors.Wrap(err, "failed to wait for NAT Gateway availability")
		}
	}
	return nil
}

// Check and Create route tables
func (a *AwsCloud) createRouteTables(ctx context.Context) error {
	existingRouteTables, err := a.ec2Client.DescribeRouteTablesWithContext(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID)}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe route tables")
	}

	if len(existingRouteTables.RouteTables) != 0 {
		for _, routeTable := range existingRouteTables.RouteTables {
			if routeTable == nil || routeTable.Tags == nil {
				continue
			}
			if a.cluster.GetCloudResourceByID(biz.ResourceTypeRouteTable, *routeTable.RouteTableId) != nil {
				continue
			}
			name := ""
			tags := make(map[string]string)
			for _, tag := range routeTable.Tags {
				if *tag.Key == AwsTagKeyName {
					name = *tag.Value
				}
				tags[*tag.Key] = *tag.Value
			}
			if val, ok := tags[AwsTagKeyType]; !ok || (val != AwsResourcePublic && val != AwsResourcePrivate) {
				continue
			}
			if tags[AwsTagKeyType] == AwsResourcePublic && name != fmt.Sprintf("%s-public-rt", a.cluster.Name) {
				continue
			}
			if tags[AwsTagKeyType] == AwsResourcePrivate {
				privateZoneName, ok := tags[AwsTagZone]
				if !ok {
					continue
				}
				if name != fmt.Sprintf("%s-private-rt-%s", a.cluster.Name, privateZoneName) {
					continue
				}
			}
			a.cluster.AddCloudResource(biz.ResourceTypeRouteTable, &biz.CloudResource{
				Name: name,
				ID:   *routeTable.RouteTableId,
				Tags: tags,
			})
		}
	}

	// Create public route table
	publicRouteTableName := fmt.Sprintf("%s-public-rt", a.cluster.Name)
	publicRouteTableNameTags := map[string]string{
		AwsTagKeyName: publicRouteTableName,
		AwsTagKeyType: AwsResourcePublic,
	}
	if a.cluster.GetCloudResourceByName(biz.ResourceTypeRouteTable, publicRouteTableName) == nil {
		publicRouteTable, err := a.ec2Client.CreateRouteTableWithContext(ctx, &ec2.CreateRouteTableInput{
			VpcId: aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String(ec2.ResourceTypeRouteTable),
					Tags:         a.mapToEc2Tags(publicRouteTableNameTags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create public route table")
		}
		a.cluster.AddCloudResource(biz.ResourceTypeRouteTable, &biz.CloudResource{
			Name: publicRouteTableName,
			ID:   *publicRouteTable.RouteTable.RouteTableId,
			Tags: publicRouteTableNameTags,
		})
		a.log.Info("created public rotertable")

		// Add route to Internet Gateway in public route table
		_, err = a.ec2Client.CreateRouteWithContext(ctx, &ec2.CreateRouteInput{
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
			publicAssociateRouteTable, err := a.ec2Client.AssociateRouteTableWithContext(ctx, &ec2.AssociateRouteTableInput{
				RouteTableId: publicRouteTable.RouteTable.RouteTableId,
				SubnetId:     aws.String(subnetReource.ID),
			})
			if err != nil {
				return errors.Wrap(err, "failed to associate public subnet with route table")
			}
			a.cluster.AddSubCloudResource(biz.ResourceTypeRouteTable, *publicRouteTable.RouteTable.RouteTableId, &biz.CloudResource{
				ID:   *publicAssociateRouteTable.AssociationId,
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
			AwsTagZone:    az.Name,
		}
		if a.cluster.GetCloudResourceByTags(biz.ResourceTypeRouteTable, AwsTagKeyName, privateRouteTableName) != nil {
			continue
		}
		privateRouteTable, err := a.ec2Client.CreateRouteTableWithContext(ctx, &ec2.CreateRouteTableInput{
			VpcId: aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String(ec2.ResourceTypeRouteTable),
					Tags:         a.mapToEc2Tags(tags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create private route table for AZ "+az.Name)
		}
		a.cluster.AddCloudResource(biz.ResourceTypeRouteTable, &biz.CloudResource{
			Name: privateRouteTableName,
			ID:   *privateRouteTable.RouteTable.RouteTableId,
			Tags: tags,
		})

		// Add route to NAT Gateway in private route table
		for _, natGateway := range a.cluster.GetCloudResource(biz.ResourceTypeNATGateway) {
			if zoneName, ok := natGateway.Tags[AwsTagZone]; !ok || zoneName != az.Name {
				continue
			}
			_, err = a.ec2Client.CreateRouteWithContext(ctx, &ec2.CreateRouteInput{
				RouteTableId:         privateRouteTable.RouteTable.RouteTableId,
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				NatGatewayId:         aws.String(natGateway.ID),
			})
			if err != nil {
				return errors.Wrap(err, "failed to add route to NAT Gateway for AZ "+az.Name)
			}
		}

		// Associate private subnets with private route table
		for i, subnet := range a.cluster.GetCloudResource(biz.ResourceTypeSubnet) {
			if typeVal, ok := subnet.Tags[AwsTagKeyType]; !ok || typeVal != AwsResourcePrivate {
				continue
			}
			if zonename, ok := subnet.Tags[AwsTagZone]; !ok || zonename != az.Name {
				continue
			}
			privateAssociateRouteTable, err := a.ec2Client.AssociateRouteTableWithContext(ctx, &ec2.AssociateRouteTableInput{
				RouteTableId: privateRouteTable.RouteTable.RouteTableId,
				SubnetId:     aws.String(subnet.ID),
			})
			if err != nil {
				return errors.Wrap(err, "failed to associate private subnet with route table in AZ "+az.Name)
			}
			a.cluster.AddSubCloudResource(biz.ResourceTypeRouteTable, *privateRouteTable.RouteTable.RouteTableId, &biz.CloudResource{
				ID:   *privateAssociateRouteTable.AssociationId,
				Name: fmt.Sprintf("private associate routetable %d", i),
			})
		}
	}
	return nil
}

// Check and Create security group
func (a *AwsCloud) createSecurityGroup(ctx context.Context) error {
	sgGroupName := fmt.Sprintf("%s-sg", a.cluster.Name)
	tags := map[string]string{
		AwsTagKeyName: sgGroupName,
	}
	existingSecurityGroups, err := a.ec2Client.DescribeSecurityGroupsWithContext(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID)}},
			{Name: aws.String("group-name"), Values: []*string{aws.String(sgGroupName)}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe security groups")
	}

	if len(existingSecurityGroups.SecurityGroups) != 0 {
		for _, securityGroup := range existingSecurityGroups.SecurityGroups {
			if securityGroup == nil || securityGroup.GroupId == nil {
				continue
			}
			if a.cluster.GetCloudResourceByID(biz.ResourceTypeSecurityGroup, *securityGroup.GroupId) != nil {
				continue
			}
			name := ""
			tags := make(map[string]string)
			for _, tag := range securityGroup.Tags {
				if *tag.Key == AwsTagKeyName {
					name = *tag.Value
				}
				tags[*tag.Key] = *tag.Value
			}
			a.cluster.AddCloudResource(biz.ResourceTypeSecurityGroup, &biz.CloudResource{
				Name: name,
				ID:   *securityGroup.GroupId,
				Tags: tags,
			})
		}
		return nil
	}

	// Create security group if it doesn't exist
	sgOutput, err := a.ec2Client.CreateSecurityGroupWithContext(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(sgGroupName),
		Description: aws.String("Security group for kubernetes cluster"),
		VpcId:       aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String(ec2.ResourceTypeSecurityGroup), // security-group
				Tags:         a.mapToEc2Tags(tags),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create security group")
	}
	a.cluster.AddCloudResource(biz.ResourceTypeSecurityGroup, &biz.CloudResource{
		Name: sgGroupName,
		ID:   *sgOutput.GroupId,
		Tags: tags,
	})

	// Add inbound rules to the security group
	_, err = a.ec2Client.AuthorizeSecurityGroupIngressWithContext(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: sgOutput.GroupId,
		IpPermissions: []*ec2.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int64(22),
				ToPort:     aws.Int64(22),
				IpRanges:   []*ec2.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
			},
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int64(6443),
				ToPort:     aws.Int64(6443),
				IpRanges:   []*ec2.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to add inbound rules to security group")
	}

	a.log.Info("added inbound rules to security group")
	return nil
}

// create slb
func (a *AwsCloud) createSLB(ctx context.Context) error {
	// Check if SLB already exists
	name := fmt.Sprintf("%s-slb", a.cluster.Name)
	if a.cluster.GetCloudResourceByName(biz.ResourceTypeLoadBalancer, name) != nil {
		return nil
	}
	publicSubnetIDs := make([]*string, 0)
	for _, subnet := range a.cluster.GetCloudResource(biz.ResourceTypeSubnet) {
		if typeVal, ok := subnet.Tags[AwsTagKeyType]; !ok || typeVal != AwsResourcePublic {
			continue
		}
		publicSubnetIDs = append(publicSubnetIDs, aws.String(subnet.ID))
	}
	if len(publicSubnetIDs) == 0 {
		return errors.New("failed to get public subnets")
	}
	sg := a.cluster.GetSingleCloudResource(biz.ResourceTypeSecurityGroup)
	if sg == nil {
		return errors.New("failed to get security group")
	}

	// Create SLB
	tags := map[string]string{AwsTagKeyName: name}
	slbOutput, err := a.elbv2Client.CreateLoadBalancerWithContext(ctx, &elbv2.CreateLoadBalancerInput{
		Name:           aws.String(name),
		Subnets:        publicSubnetIDs,
		SecurityGroups: []*string{aws.String(sg.ID)},
		Tags:           a.mapToElbv2Tags(tags),
		Scheme:         aws.String("Internet-facing"),
		Type:           aws.String("application"),
	})
	if err != nil || len(slbOutput.LoadBalancers) == 0 {
		return errors.Wrap(err, "failed to create SLB")
	}
	slb := slbOutput.LoadBalancers[0]
	a.cluster.AddCloudResource(biz.ResourceTypeLoadBalancer, &biz.CloudResource{
		Name: name,
		ID:   *slb.LoadBalancerArn,
		Tags: tags,
	})

	// Create target group
	taggetGroup, err := a.elbv2Client.CreateTargetGroupWithContext(ctx, &elbv2.CreateTargetGroupInput{
		Name:                       aws.String(fmt.Sprintf("%s-targetgroup", a.cluster.Name)),
		Port:                       aws.Int64(6443),
		Protocol:                   aws.String(elbv2.ProtocolEnumTcp),
		VpcId:                      aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
		Tags:                       a.mapToElbv2Tags(tags),
		HealthCheckProtocol:        aws.String(elbv2.ProtocolEnumTcp),
		HealthCheckPort:            aws.String("6443"),
		HealthCheckPath:            aws.String("/healthz"),
		HealthCheckIntervalSeconds: aws.Int64(30),
		HealthCheckTimeoutSeconds:  aws.Int64(5),
		HealthyThresholdCount:      aws.Int64(5),
		UnhealthyThresholdCount:    aws.Int64(2),
	})
	if err != nil || len(taggetGroup.TargetGroups) == 0 {
		return errors.Wrap(err, "failed to create target group")
	}
	targetGroup := taggetGroup.TargetGroups[0]

	// create listener
	_, err = a.elbv2Client.CreateListenerWithContext(ctx, &elbv2.CreateListenerInput{
		DefaultActions: []*elbv2.Action{
			{
				Type: aws.String(elbv2.ActionTypeEnumForward),
				ForwardConfig: &elbv2.ForwardActionConfig{
					TargetGroups: []*elbv2.TargetGroupTuple{
						{
							TargetGroupArn: aws.String(*targetGroup.TargetGroupArn),
							Weight:         aws.Int64(100),
						},
					},
				},
			},
		},
		LoadBalancerArn: slb.LoadBalancerArn,
		Port:            aws.Int64(6443),
		Protocol:        aws.String(elbv2.ProtocolEnumTcp),
	})
	if err != nil {
		return errors.Wrap(err, "failed to create listener")
	}
	return nil
}

// create Tags
func (a *AwsCloud) createTags(ctx context.Context, resourceID string, resourceType biz.ResourceType, tags map[string]string) error {
	_, err := a.ec2Client.CreateTagsWithContext(ctx, &ec2.CreateTagsInput{
		Resources: []*string{aws.String(resourceID)},
		Tags:      a.mapToEc2Tags(tags),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create tags for %s", resourceType)
	}
	return nil
}

// map to ec2 tags
func (a *AwsCloud) mapToEc2Tags(tags map[string]string) []*ec2.Tag {
	ec2Tags := []*ec2.Tag{}
	for key, value := range tags {
		ec2Tags = append(ec2Tags, &ec2.Tag{Key: aws.String(key), Value: aws.String(value)})
	}
	return ec2Tags
}

// map to elbv2 tags
func (a *AwsCloud) mapToElbv2Tags(tags map[string]string) []*elbv2.Tag {
	elbv2Tags := []*elbv2.Tag{}
	for key, value := range tags {
		elbv2Tags = append(elbv2Tags, &elbv2.Tag{Key: aws.String(key), Value: aws.String(value)})
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
