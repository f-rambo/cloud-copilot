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
	awsDefaultRegion = "us-east-1"
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
		name := ""
		tags := make(map[string]string)
		for _, vpc := range existingVpcs.Vpcs {
			for _, tag := range vpc.Tags {
				if *tag.Key == "Name" {
					name = *tag.Value
				}
				tags[*tag.Key] = *tag.Value
			}
			a.cluster.AddCloudResource(biz.ResourceTypeVPC, &biz.CloudResource{
				Name: name,
				ID:   *vpc.VpcId,
				Tags: tags,
			})
		}
	}

	if len(existingVpcs.Vpcs) == 0 {
		// Create VPC if it doesn't exist
		name := a.cluster.Name + "-vpc"
		tags := map[string]string{
			"Name": name,
		}
		vpcOutput, err := a.ec2Client.CreateVpc(&ec2.CreateVpcInput{
			CidrBlock: aws.String(a.cluster.IpCidr),
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String("vpc"),
					Tags:         a.mapToEc2Tags(tags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create VPC")
		}

		a.cluster.AddCloudResource(biz.ResourceTypeVPC, &biz.CloudResource{
			Name: name,
			ID:   *vpcOutput.Vpc.VpcId,
			Tags: tags,
		})
	}

	a.log.Infof("vpc %s created", a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID)

	// Step 2: Get availability zones
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

	// Step 3: Check and Create subnets
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
		for _, subnet := range existingSubnets.Subnets {
			tags := make(map[string]string)
			name := ""
			for _, tag := range subnet.Tags {
				if *tag.Key == "Name" {
					name = *tag.Value
				}
				tags[*tag.Key] = *tag.Value
			}
			a.cluster.AddCloudResource(biz.ResourceTypeSubnet, &biz.CloudResource{
				Name: name,
				ID:   *subnet.SubnetId,
				Tags: tags,
			})
		}
	}

	if len(existingSubnets.Subnets) == 0 {
		// Create subnets if they don't exist
		privateSubnetCount := len(azOutput.AvailabilityZones) * 2
		publicSubnetCount := len(azOutput.AvailabilityZones)
		subnetCidrs, err := utils.GenerateSubnets(a.cluster.IpCidr, privateSubnetCount+publicSubnetCount)
		if err != nil {
			return errors.Wrap(err, "failed to generate subnet CIDRs")
		}

		for i, az := range a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones) {
			// Create private subnets
			for j := 0; j < 2; j++ {
				name := fmt.Sprintf("%s-private-subnet-%s-%d", a.cluster.Name, az.Name, j+1)
				tags := map[string]string{
					"Name": name,
					"Type": "private",
					"Zone": az.Name,
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
				"Name": name,
				"Type": "public",
				"Zone": az.Name,
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
	}

	a.log.Infof("subnets %v", a.cluster.GetCloudResource(biz.ResourceTypeSubnet))

	// Step 4: Check and Create Internet Gateway
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
				if *tag.Key == "Name" {
					name = *tag.Value
				}
				tags[*tag.Key] = *tag.Value
			}
			a.cluster.AddCloudResource(biz.ResourceTypeInternetGateway, &biz.CloudResource{
				Name: name,
				ID:   *igw.InternetGatewayId,
				Tags: tags,
			})
		}
	}

	if len(existingIgws.InternetGateways) == 0 {
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
	}

	// Step 5: Check and Create NAT Gateways
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
			name := ""
			tags := make(map[string]string)
			for _, tag := range natGateway.Tags {
				if *tag.Key == "Name" {
					name = *tag.Value
				}
				tags[*tag.Key] = *tag.Value
			}
			a.cluster.AddCloudResource(biz.ResourceTypeNATGateway, &biz.CloudResource{
				Name: name,
				ID:   *natGateway.NatGatewayId,
				Tags: tags,
			})
		}
	}

	if len(existingNatGateways.NatGateways) == 0 {
		// Create NAT Gateways if they don't exist for each AZ
		for _, az := range a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones) {
			// Allocate Elastic IP
			name := fmt.Sprintf("%s-eip-%s", a.cluster.Name, az.Name)
			tags := map[string]string{
				"Name": name,
			}
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
			a.cluster.AddCloudResource(biz.ResourceTypeElasticIP, &biz.CloudResource{
				Name: name,
				ID:   *eipOutput.AllocationId,
				Tags: tags,
			})

			// Create NAT Gateway
			name = fmt.Sprintf("%s-nat-gateway-%s", a.cluster.Name, az.Name)
			tags = map[string]string{
				"Name": name,
				"Type": "private",
				"Zone": az.Name,
			}
			natGatewayOutput, err := a.ec2Client.CreateNatGateway(&ec2.CreateNatGatewayInput{
				AllocationId: eipOutput.AllocationId,
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
	}

	// Step 6: Check and Create route tables
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
			name := ""
			tags := make(map[string]string)
			for _, tag := range routeTable.Tags {
				if *tag.Key == "Name" {
					name = *tag.Value
				}
				tags[*tag.Key] = *tag.Value
			}
			a.cluster.AddCloudResource(biz.ResourceTypeRouteTable, &biz.CloudResource{
				Name: name,
				ID:   *routeTable.RouteTableId,
				Tags: tags,
			})
		}
	}

	if len(existingRouteTables.RouteTables) == 0 {
		// Create route tables if they don't exist
		// Create public route table
		publicRouteTableName := fmt.Sprintf("%s-public-rt", a.cluster.Name)
		tags := map[string]string{
			"Name": publicRouteTableName,
			"Type": "public",
		}
		publicRouteTable, err := a.ec2Client.CreateRouteTable(&ec2.CreateRouteTableInput{
			VpcId: aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID),
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String("route-table"),
					Tags:         a.mapToEc2Tags(tags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create public route table")
		}
		a.cluster.AddCloudResource(biz.ResourceTypeRouteTable, &biz.CloudResource{
			Name: publicRouteTableName,
			ID:   *publicRouteTable.RouteTable.RouteTableId,
			Tags: tags,
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

		// Create private route tables (one per AZ)
		for _, az := range a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones) {
			privateRouteTableName := fmt.Sprintf("%s-private-rt-%s", a.cluster.Name, az.Name)
			tags := map[string]string{
				"Name": privateRouteTableName,
				"Type": "private",
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
			},
			)

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
	}

	// Step 7: Check and Create security group
	existingSecurityGroups, err := a.ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetSingleCloudResource(biz.ResourceTypeVPC).ID)}},
			{Name: aws.String("group-name"), Values: []*string{aws.String(fmt.Sprintf("%s-sg", a.cluster.Name))}},
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
				if *tag.Key == "Name" {
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
	}

	if len(existingSecurityGroups.SecurityGroups) == 0 {
		// Create security group if it doesn't exist
		sgGroupName := fmt.Sprintf("%s-sg", a.cluster.Name)
		tags := map[string]string{
			"Name": sgGroupName,
		}
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

// map to ec2 tags
func (a *AwsCloud) mapToEc2Tags(tags map[string]string) []*ec2.Tag {
	ec2Tags := []*ec2.Tag{}
	for key, value := range tags {
		ec2Tags = append(ec2Tags, &ec2.Tag{Key: aws.String(key), Value: aws.String(value)})
	}
	return ec2Tags
}
