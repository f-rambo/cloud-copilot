package infrastructure

import (
	"os"

	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
)

type AwsCloud struct {
	cluster   *biz.Cluster
	ec2Client *ec2.EC2
	log       *log.Helper
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
		cluster:   cluster,
		ec2Client: ec2.New(sess),
		log:       log,
	}, nil
}

func (a *AwsCloud) GetRegions() (regions []string, err error) {
	result, err := a.ec2Client.DescribeRegions(nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to describe regions")
	}
	avaliableStatus := []string{"opt-in-not-required", "opted-in"}
	for _, region := range result.Regions {
		if !utils.InArray(*region.OptInStatus, avaliableStatus) {
			continue
		}
		regions = append(regions, *region.RegionName)
	}
	return regions, nil
}

func (a *AwsCloud) GetZoneNames() (zones []string, err error) {
	zoneResult, err := a.GetAvailableZones()
	if err != nil {
		return nil, err
	}
	for _, zone := range zoneResult {
		zones = append(zones, *zone.ZoneName)
	}
	return zones, nil
}

// get available zones
func (a *AwsCloud) GetAvailableZones() (zones []*ec2.AvailabilityZone, err error) {
	result, err := a.ec2Client.DescribeAvailabilityZones(nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to describe availability zones")
	}
	return result.AvailabilityZones, nil
}

// create network(vpc, subnet, internet gateway,nat gateway, route table, security group)
func (a *AwsCloud) CreateNetwork() error {
	// Step 1: Check and Create VPC
	err := a.createVPC()
	if err != nil {
		return err
	}

	// Step 2: Get availability zones
	err = a.getAvailabilityZones()
	if err != nil {
		return err
	}

	// Step 3: Check and Create subnets
	err = a.createSubnets()
	if err != nil {
		return err
	}

	// Step 4: Check and Create Internet Gateway
	err = a.createInternetGateway()
	if err != nil {
		return err
	}

	// Step 5: Check and Create NAT Gateways
	err = a.createNATGateways()
	if err != nil {
		return err
	}

	// Step 6: Check and Create route tables
	err = a.createRouteTables()
	if err != nil {
		return err
	}

	// Step 7: Check and Create security group
	err = a.createSecurityGroup()
	if err != nil {
		return err
	}
	return nil
}

// delete network(vpc, subnet, internet gateway, nat gateway, route table, security group)
func (a *AwsCloud) DeleteNetwork() error {
	// Step 1: Delete security group
	for _, sg := range a.cluster.GetCloudResource(biz.ResourceTypeSecurityGroup) {
		_, err := a.ec2Client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupId: &sg.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete security group")
		}
	}

	// Step 2: Delete route tables
	for _, rt := range a.cluster.GetCloudResource(biz.ResourceTypeRouteTable) {
		for _, subRtassoc := range rt.SubResources {
			_, err := a.ec2Client.DisassociateRouteTable(&ec2.DisassociateRouteTableInput{
				AssociationId: &subRtassoc.ID,
			})
			if err != nil {
				return errors.Wrap(err, "failed to disassociate route table")
			}
		}
		_, err := a.ec2Client.DeleteRouteTable(&ec2.DeleteRouteTableInput{
			RouteTableId: &rt.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete route table")
		}
	}

	// Step 3: Delete Internet Gateway
	for _, igw := range a.cluster.GetCloudResource(biz.ResourceTypeInternetGateway) {
		_, err := a.ec2Client.DetachInternetGateway(&ec2.DetachInternetGatewayInput{
			InternetGatewayId: &igw.ID,
			VpcId:             aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
		})
		if err != nil {
			return errors.Wrap(err, "failed to detach Internet Gateway")
		}
		_, err = a.ec2Client.DeleteInternetGateway(&ec2.DeleteInternetGatewayInput{
			InternetGatewayId: &igw.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete Internet Gateway")
		}
	}

	// Step 4: Delete NAT Gateways
	// client.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{})
	for _, natGw := range a.cluster.GetCloudResource(biz.ResourceTypeNATGateway) {
		_, err := a.ec2Client.DeleteNatGateway(&ec2.DeleteNatGatewayInput{
			NatGatewayId: &natGw.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete NAT Gateway")
		}
		// Wait for NAT Gateway to be deleted
		err = a.ec2Client.WaitUntilNatGatewayDeleted(&ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []*string{&natGw.ID},
		})
		if err != nil {
			return errors.Wrap(err, "failed to wait for NAT Gateway deletion")
		}
	}

	// Release Elastic IPs associated with NAT Gateways
	for _, addr := range a.cluster.GetCloudResource(biz.ResourceTypeElasticIP) {
		_, err := a.ec2Client.ReleaseAddress(&ec2.ReleaseAddressInput{
			AllocationId: &addr.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to release Elastic IP")
		}
	}

	// Step 5: Delete Subnets
	for _, subnet := range a.cluster.GetCloudResource(biz.ResourceTypeSubnet) {
		_, err := a.ec2Client.DeleteSubnet(&ec2.DeleteSubnetInput{
			SubnetId: &subnet.ID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete subnet")
		}
	}

	// Step 6: Delete VPC
	_, err := a.ec2Client.DeleteVpc(&ec2.DeleteVpcInput{
		VpcId: aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete VPC")
	}
	return nil
}

// create vpc
func (a *AwsCloud) createVPC() error {
	if a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC) != nil {
		return nil
	}
	vpcName := a.cluster.Name + "-vpc"
	vpcTags := map[string]string{
		"Name": vpcName,
	}
	a.cluster.AddCloudResource(biz.ResourceTypeVPC, &biz.CloudResource{
		Name: vpcName,
		Tags: vpcTags,
	})
	existingVpcs, err := a.ec2Client.DescribeVpcs(&ec2.DescribeVpcsInput{
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
	vpcOutput, err := a.ec2Client.CreateVpc(&ec2.CreateVpcInput{
		CidrBlock: aws.String(a.cluster.IpCidr),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("vpc"),
				Tags:         a.mapToEc2Tags(vpcTags),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create VPC")
	}
	vpcCloudResource := a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC)
	vpcCloudResource.ID = *vpcOutput.Vpc.VpcId

	a.log.Infof("vpc %s created", a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID)
	return nil
}

// Get availability zones
func (a *AwsCloud) getAvailabilityZones() error {
	azOutput, err := a.ec2Client.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		return errors.Wrap(err, "failed to describe availability zones")
	}
	if len(azOutput.AvailabilityZones) == 0 {
		return errors.New("no availability zones found")
	}

	for _, az := range azOutput.AvailabilityZones {
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
func (a *AwsCloud) createSubnets() error {
	existingSubnets, err := a.ec2Client.DescribeSubnets(&ec2.DescribeSubnetsInput{
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
				tags := make(map[string]string)
				name := ""
				for _, tag := range subnet.Tags {
					tags[*tag.Key] = *tag.Value
				}
				tags[AwsTagZone] = zoneName
				if i < 2 {
					name = fmt.Sprintf("%s-private-subnet-%s-%d", a.cluster.Name, *subnet.AvailabilityZone, i+1)
					tags[AwsTagKeyName] = name
					tags[AwsTagKeyType] = AwsResourcePrivate
				} else {
					name = fmt.Sprintf("%s-public-subnet-%s", a.cluster.Name, *subnet.AvailabilityZone)
					tags[AwsTagKeyName] = name
					tags[AwsTagKeyType] = AwsResourcePublic
				}
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
			subnetOutput, err := a.ec2Client.CreateSubnet(&ec2.CreateSubnetInput{
				VpcId:            aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
				CidrBlock:        aws.String(cidr),
				AvailabilityZone: &az.Name,
				TagSpecifications: []*ec2.TagSpecification{
					{
						ResourceType: aws.String("subnet"),
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
		subnetOutput, err := a.ec2Client.CreateSubnet(&ec2.CreateSubnetInput{
			VpcId:            aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
			CidrBlock:        aws.String(cidr),
			AvailabilityZone: &az.Name,
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String("subnet"),
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

	a.log.Infof("subnets %v", a.cluster.GetCloudResource(biz.ResourceTypeSubnet))
	return nil
}

// Check and Create Internet Gateway
func (a *AwsCloud) createInternetGateway() error {
	existingIgws, err := a.ec2Client.DescribeInternetGateways(&ec2.DescribeInternetGatewaysInput{
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
				tags[AwsTagKeyName] = name
			}
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
		"Name": name,
	}
	igwOutput, err := a.ec2Client.CreateInternetGateway(&ec2.CreateInternetGatewayInput{
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("internet-gateway"),
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

	_, err = a.ec2Client.AttachInternetGateway(&ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeInternetGateway).ID),
		VpcId:             aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
	})
	if err != nil {
		return errors.Wrap(err, "failed to attach Internet Gateway")
	}

	return nil
}

// Check and Create NAT Gateways
func (a *AwsCloud) createNATGateways() error {
	existingNatGateways, err := a.ec2Client.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{
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
			tags[AwsTagKeyName] = name
			if _, ok := tags[AwsTagZone]; !ok {
				tags[AwsTagZone] = subnetCloudResource.Tags[AwsTagZone]
			}
			a.cluster.AddCloudResource(biz.ResourceTypeNATGateway, &biz.CloudResource{
				Name: name,
				ID:   *natGateway.NatGatewayId,
				Tags: tags,
			})
		}
	}

	// Create NAT Gateways if they don't exist for each AZ
	for _, az := range a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones) {
		// Allocate Elastic IP
		name := fmt.Sprintf("%s-eip-%s", a.cluster.Name, az.Name)
		tags := map[string]string{
			AwsTagKeyName: name,
			AwsTagZone:    az.Name,
		}
		eipCloudResouce := &biz.CloudResource{
			Name: name,
			Tags: tags,
		}
		// get Elastic IP
		eipRes, err := a.ec2Client.DescribeAddresses(&ec2.DescribeAddressesInput{})
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
		if a.cluster.GetCloudResourceByTags(biz.ResourceTypeElasticIP, AwsTagKeyName, name) == nil {
			// Allocate Elastic IP
			eipOutput, err := a.ec2Client.AllocateAddress(&ec2.AllocateAddressInput{
				Domain: aws.String("vpc"),
				TagSpecifications: []*ec2.TagSpecification{
					{
						ResourceType: aws.String("elastic-ip"),
						Tags:         a.mapToEc2Tags(tags),
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
		name = fmt.Sprintf("%s-nat-gateway-%s", a.cluster.Name, az.Name)
		tags = map[string]string{
			AwsTagKeyName: name,
			AwsTagKeyType: AwsResourcePublic,
			AwsTagZone:    az.Name,
		}
		if a.cluster.GetCloudResourceByName(biz.ResourceTypeNATGateway, name) != nil {
			continue
		}

		publickSubnet := a.cluster.GetCloudResourceByTags(biz.ResourceTypeSubnet, AwsTagZone, az.Name, AwsTagKeyType, AwsResourcePublic)
		if publickSubnet == nil {
			return errors.New("no public subnet found for AZ " + az.Name)
		}
		natGatewayOutput, err := a.ec2Client.CreateNatGateway(&ec2.CreateNatGatewayInput{
			AllocationId:     &eipCloudResouce.ID,
			ConnectivityType: aws.String("public"),
			SubnetId:         &publickSubnet.ID,
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String("natgateway"),
					Tags:         a.mapToEc2Tags(tags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create NAT Gateway")
		}
		a.cluster.AddCloudResource(biz.ResourceTypeNATGateway, &biz.CloudResource{
			Name: name,
			ID:   *natGatewayOutput.NatGateway.NatGatewayId,
			Tags: tags,
		})
	}
	return nil
}

// Check and Create route tables
func (a *AwsCloud) createRouteTables() error {
	existingRouteTables, err := a.ec2Client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
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
		publicRouteTable, err := a.ec2Client.CreateRouteTable(&ec2.CreateRouteTableInput{
			VpcId: aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String("route-table"),
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
		_, err = a.ec2Client.CreateRoute(&ec2.CreateRouteInput{
			RouteTableId:         publicRouteTable.RouteTable.RouteTableId,
			DestinationCidrBlock: aws.String("0.0.0.0/0"),
			GatewayId:            aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeInternetGateway).ID),
		})
		if err != nil {
			return errors.Wrap(err, "failed to add route to Internet Gateway")
		}

		// Associate public subnets with public route table
		for i, subnetReource := range a.cluster.GetCloudResource(biz.ResourceTypeSubnet) {
			if typeVal, ok := subnetReource.Tags["Type"]; !ok || typeVal != "public" {
				continue
			}
			publicAssociateRouteTable, err := a.ec2Client.AssociateRouteTable(&ec2.AssociateRouteTableInput{
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
		privateRouteTable, err := a.ec2Client.CreateRouteTable(&ec2.CreateRouteTableInput{
			VpcId: aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String("route-table"),
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
			if zoneName, ok := natGateway.Tags["Zone"]; !ok || zoneName != az.Name {
				continue
			}
			_, err = a.ec2Client.CreateRoute(&ec2.CreateRouteInput{
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
			if typeVal, ok := subnet.Tags["Type"]; !ok || typeVal != "private" {
				continue
			}
			privateAssociateRouteTable, err := a.ec2Client.AssociateRouteTable(&ec2.AssociateRouteTableInput{
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
func (a *AwsCloud) createSecurityGroup() error {
	sgGroupName := fmt.Sprintf("%s-sg", a.cluster.Name)
	tags := map[string]string{
		AwsTagKeyName: sgGroupName,
	}
	existingSecurityGroups, err := a.ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
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
	sgOutput, err := a.ec2Client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(sgGroupName),
		Description: aws.String("Security group for kubernetes cluster"),
		VpcId:       aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("security-group"),
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
	_, err = a.ec2Client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
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

// map to ec2 tags
func (a *AwsCloud) mapToEc2Tags(tags map[string]string) []*ec2.Tag {
	ec2Tags := []*ec2.Tag{}
	for key, value := range tags {
		ec2Tags = append(ec2Tags, &ec2.Tag{Key: aws.String(key), Value: aws.String(value)})
	}
	return ec2Tags
}
