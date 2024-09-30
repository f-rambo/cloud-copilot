package infrastructure

import (
	"os"

	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/utils"
	"github.com/pkg/errors"
)

type AwsCloud struct {
	cluster   *biz.Cluster
	ec2Client *ec2.EC2
}

const (
	awsDefaultRegion = "us-east-1"
)

func NewAwsCloud(cluster *biz.Cluster) (*AwsCloud, error) {
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
	if a.cluster.GetCloudResource(biz.ResourceTypeVPC) == nil || len(a.cluster.GetCloudResource(biz.ResourceTypeVPC)) == 0 {
		a.cluster.AddCloudResource(biz.ResourceTypeVPC, &biz.CloudResource{
			Name: a.cluster.Name + "-vpc",
			Tags: map[string]string{
				"Name": a.cluster.Name + "-vpc",
			},
		})
	}

	existingVpcs, err := a.ec2Client.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).Name)},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe VPCs")
	}

	if len(existingVpcs.Vpcs) != 0 {
		for _, vpc := range existingVpcs.Vpcs {
			a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID = *vpc.VpcId
			break
		}
	}

	if len(existingVpcs.Vpcs) == 0 {
		// Create VPC if it doesn't exist
		vpcOutput, err := a.ec2Client.CreateVpc(&ec2.CreateVpcInput{
			CidrBlock: aws.String(a.cluster.IpCidr),
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String("vpc"),
					Tags:         a.mapToEc2Tags(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).Tags),
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create VPC")
		}
		a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID = *vpcOutput.Vpc.VpcId
	}

	// Step 2: Get availability zones
	azOutput, err := a.ec2Client.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		return errors.Wrap(err, "failed to describe availability zones")
	}
	if len(azOutput.AvailabilityZones) == 0 {
		return errors.New("no availability zones found")
	}

	// Step 3: Check and Create subnets
	existingSubnets, err := a.ec2Client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe subnets")
	}

	if len(existingSubnets.Subnets) == 0 {
		// Create subnets if they don't exist
		privateSubnetCount := len(azOutput.AvailabilityZones) * 2
		publicSubnetCount := len(azOutput.AvailabilityZones)
		subnetCidrs, err := utils.GenerateSubnets(a.cluster.IpCidr, privateSubnetCount+publicSubnetCount)
		if err != nil {
			return errors.Wrap(err, "failed to generate subnet CIDRs")
		}

		for i, az := range azOutput.AvailabilityZones {
			// Create private subnets
			for j := 0; j < 2; j++ {
				cidr := subnetCidrs[i*2+j]
				_, err := a.ec2Client.CreateSubnet(&ec2.CreateSubnetInput{
					VpcId:            aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID),
					CidrBlock:        aws.String(cidr),
					AvailabilityZone: az.ZoneName,
					TagSpecifications: []*ec2.TagSpecification{
						{
							ResourceType: aws.String("subnet"),
							Tags: []*ec2.Tag{
								{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-private-subnet-%s-%d", a.cluster.Name, *az.ZoneName, j+1))},
							},
						},
					},
				})
				if err != nil {
					return errors.Wrap(err, "failed to create private subnet")
				}
			}

			// Create public subnet
			cidr := subnetCidrs[privateSubnetCount+i]
			_, err := a.ec2Client.CreateSubnet(&ec2.CreateSubnetInput{
				VpcId:            aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID),
				CidrBlock:        aws.String(cidr),
				AvailabilityZone: az.ZoneName,
				TagSpecifications: []*ec2.TagSpecification{
					{
						ResourceType: aws.String("subnet"),
						Tags: []*ec2.Tag{
							{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-public-subnet-%s", a.cluster.Name, *az.ZoneName))},
							{Key: aws.String("Type"), Value: aws.String("public")},
						},
					},
				},
			})
			if err != nil {
				return errors.Wrap(err, "failed to create public subnet")
			}
		}
	}

	// Step 4: Check and Create Internet Gateway
	existingIgws, err := a.ec2Client.DescribeInternetGateways(&ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("attachment.vpc-id"),
				Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe Internet Gateways")
	}

	var igwId *string
	if len(existingIgws.InternetGateways) == 0 {
		// Create Internet Gateway if it doesn't exist
		igwOutput, err := a.ec2Client.CreateInternetGateway(&ec2.CreateInternetGatewayInput{
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String("internet-gateway"),
					Tags: []*ec2.Tag{
						{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-igw", a.cluster.Name))},
						{Key: aws.String("ocean-key"), Value: aws.String("ocean-cluster")},
					},
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create Internet Gateway")
		}
		igwId = igwOutput.InternetGateway.InternetGatewayId

		_, err = a.ec2Client.AttachInternetGateway(&ec2.AttachInternetGatewayInput{
			InternetGatewayId: igwId,
			VpcId:             aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID),
		})
		if err != nil {
			return errors.Wrap(err, "failed to attach Internet Gateway")
		}
	} else {
		igwId = existingIgws.InternetGateways[0].InternetGatewayId
	}

	// Step 5: Check and Create NAT Gateways
	existingNatGateways, err := a.ec2Client.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{
		Filter: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)}},
			{Name: aws.String("state"), Values: []*string{aws.String("available")}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe NAT Gateways")
	}

	if len(existingNatGateways.NatGateways) < len(azOutput.AvailabilityZones) {
		// Create NAT Gateways if they don't exist for each AZ
		for _, az := range azOutput.AvailabilityZones {
			// Check if NAT Gateway already exists for this AZ
			natGatewayExists := false
			for _, natGateway := range existingNatGateways.NatGateways {
				if *natGateway.SubnetId == *az.ZoneId {
					natGatewayExists = true
					break
				}
			}
			if natGatewayExists {
				continue
			}

			// Allocate Elastic IP
			eipOutput, err := a.ec2Client.AllocateAddress(&ec2.AllocateAddressInput{
				Domain: aws.String("vpc"),
			})
			if err != nil {
				return errors.Wrap(err, "failed to allocate Elastic IP")
			}

			// Find the public subnet for this AZ
			subnetOutput, err := a.ec2Client.DescribeSubnets(&ec2.DescribeSubnetsInput{
				Filters: []*ec2.Filter{
					{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)}},
					{Name: aws.String("availability-zone"), Values: []*string{az.ZoneName}},
					{Name: aws.String("tag:Type"), Values: []*string{aws.String("public")}},
				},
			})
			if err != nil || len(subnetOutput.Subnets) == 0 {
				return errors.Wrap(err, "failed to find public subnet for AZ "+*az.ZoneName)
			}

			// Create NAT Gateway
			_, err = a.ec2Client.CreateNatGateway(&ec2.CreateNatGatewayInput{
				AllocationId: eipOutput.AllocationId,
				SubnetId:     subnetOutput.Subnets[0].SubnetId,
				TagSpecifications: []*ec2.TagSpecification{
					{
						ResourceType: aws.String("natgateway"),
						Tags: []*ec2.Tag{
							{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-nat-gateway-%s", a.cluster.Name, *az.ZoneName))},
						},
					},
				},
			})
			if err != nil {
				return errors.Wrap(err, "failed to create NAT Gateway")
			}
		}
	}

	// Step 6: Check and Create route tables
	existingRouteTables, err := a.ec2Client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe route tables")
	}

	if len(existingRouteTables.RouteTables) == 0 {
		// Create route tables if they don't exist
		// Create public route table
		publicRouteTable, err := a.ec2Client.CreateRouteTable(&ec2.CreateRouteTableInput{
			VpcId: aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID),
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String("route-table"),
					Tags: []*ec2.Tag{
						{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-public-rt", a.cluster.Name))},
						{Key: aws.String("ocean-key"), Value: aws.String("ocean-cluster")},
					},
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create public route table")
		}

		// Add route to Internet Gateway in public route table
		_, err = a.ec2Client.CreateRoute(&ec2.CreateRouteInput{
			RouteTableId:         publicRouteTable.RouteTable.RouteTableId,
			DestinationCidrBlock: aws.String("0.0.0.0/0"),
			GatewayId:            igwId,
		})
		if err != nil {
			return errors.Wrap(err, "failed to add route to Internet Gateway")
		}

		// Associate public subnets with public route table
		publicSubnets, err := a.ec2Client.DescribeSubnets(&ec2.DescribeSubnetsInput{
			Filters: []*ec2.Filter{
				{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)}},
				{Name: aws.String("tag:Type"), Values: []*string{aws.String("public")}},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to describe public subnets")
		}

		for _, subnet := range publicSubnets.Subnets {
			_, err = a.ec2Client.AssociateRouteTable(&ec2.AssociateRouteTableInput{
				RouteTableId: publicRouteTable.RouteTable.RouteTableId,
				SubnetId:     subnet.SubnetId,
			})
			if err != nil {
				return errors.Wrap(err, "failed to associate public subnet with route table")
			}
		}

		// Create private route tables (one per AZ)
		for _, az := range azOutput.AvailabilityZones {
			privateRouteTable, err := a.ec2Client.CreateRouteTable(&ec2.CreateRouteTableInput{
				VpcId: aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID),
				TagSpecifications: []*ec2.TagSpecification{
					{
						ResourceType: aws.String("route-table"),
						Tags: []*ec2.Tag{
							{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-private-rt-%s", a.cluster.Name, *az.ZoneName))},
							{Key: aws.String("ocean-key"), Value: aws.String("ocean-cluster")},
						},
					},
				},
			})
			if err != nil {
				return errors.Wrap(err, "failed to create private route table for AZ "+*az.ZoneName)
			}

			// Find NAT Gateway for this AZ
			natGateways, err := a.ec2Client.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{
				Filter: []*ec2.Filter{
					{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)}},
					{Name: aws.String("state"), Values: []*string{aws.String("available")}},
					{Name: aws.String("tag:Name"), Values: []*string{aws.String(fmt.Sprintf("%s-nat-gateway-%s", a.cluster.Name, *az.ZoneName))}},
				},
			})
			if err != nil || len(natGateways.NatGateways) == 0 {
				return errors.Wrap(err, "failed to find NAT Gateway for AZ "+*az.ZoneName)
			}

			// Add route to NAT Gateway in private route table
			_, err = a.ec2Client.CreateRoute(&ec2.CreateRouteInput{
				RouteTableId:         privateRouteTable.RouteTable.RouteTableId,
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				NatGatewayId:         natGateways.NatGateways[0].NatGatewayId,
			})
			if err != nil {
				return errors.Wrap(err, "failed to add route to NAT Gateway for AZ "+*az.ZoneName)
			}

			// Associate private subnets with private route table
			privateSubnets, err := a.ec2Client.DescribeSubnets(&ec2.DescribeSubnetsInput{
				Filters: []*ec2.Filter{
					{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)}},
					{Name: aws.String("availability-zone"), Values: []*string{az.ZoneName}},
					{Name: aws.String("tag:Type"), Values: []*string{aws.String("private")}},
				},
			})
			if err != nil {
				return errors.Wrap(err, "failed to describe private subnets for AZ "+*az.ZoneName)
			}

			for _, subnet := range privateSubnets.Subnets {
				_, err = a.ec2Client.AssociateRouteTable(&ec2.AssociateRouteTableInput{
					RouteTableId: privateRouteTable.RouteTable.RouteTableId,
					SubnetId:     subnet.SubnetId,
				})
				if err != nil {
					return errors.Wrap(err, "failed to associate private subnet with route table in AZ "+*az.ZoneName)
				}
			}
		}
	}

	// Step 7: Check and Create security group
	existingSecurityGroups, err := a.ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)}},
			{Name: aws.String("group-name"), Values: []*string{aws.String(fmt.Sprintf("%s-sg", a.cluster.Name))}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe security groups")
	}

	if len(existingSecurityGroups.SecurityGroups) == 0 {
		// Create security group if it doesn't exist
		sgOutput, err := a.ec2Client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(fmt.Sprintf("%s-sg", a.cluster.Name)),
			Description: aws.String("Security group for Ocean cluster"),
			VpcId:       aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID),
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String("security-group"),
					Tags: []*ec2.Tag{
						{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-sg", a.cluster.Name))},
					},
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create security group")
		}

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
	securityGroups, err := a.ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)}},
			{Name: aws.String("group-name"), Values: []*string{aws.String(fmt.Sprintf("%s-sg", a.cluster.Name))}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe security groups")
	}
	for _, sg := range securityGroups.SecurityGroups {
		_, err = a.ec2Client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupId: sg.GroupId,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete security group")
		}
	}

	// Step 2: Delete route tables
	routeTables, err := a.ec2Client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)}},
			{Name: aws.String("tag:ocean-key"), Values: []*string{aws.String("ocean-cluster")}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe route tables")
	}
	for _, rt := range routeTables.RouteTables {
		// Disassociate subnets from route table
		for _, assoc := range rt.Associations {
			if assoc.Main != nil && *assoc.Main {
				continue // Skip the main route table
			}
			_, err = a.ec2Client.DisassociateRouteTable(&ec2.DisassociateRouteTableInput{
				AssociationId: assoc.RouteTableAssociationId,
			})
			if err != nil {
				return errors.Wrap(err, "failed to disassociate route table")
			}
		}
		// Delete route table
		_, err = a.ec2Client.DeleteRouteTable(&ec2.DeleteRouteTableInput{
			RouteTableId: rt.RouteTableId,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete route table")
		}
	}

	// Step 3: Delete Internet Gateway
	igws, err := a.ec2Client.DescribeInternetGateways(&ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("attachment.vpc-id"), Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe Internet Gateways")
	}
	for _, igw := range igws.InternetGateways {
		_, err = a.ec2Client.DetachInternetGateway(&ec2.DetachInternetGatewayInput{
			InternetGatewayId: igw.InternetGatewayId,
			VpcId:             aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID),
		})
		if err != nil {
			return errors.Wrap(err, "failed to detach Internet Gateway")
		}
		_, err = a.ec2Client.DeleteInternetGateway(&ec2.DeleteInternetGatewayInput{
			InternetGatewayId: igw.InternetGatewayId,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete Internet Gateway")
		}
	}

	// Step 4: Delete NAT Gateways
	natGateways, err := a.ec2Client.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{
		Filter: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe NAT Gateways")
	}
	for _, natGw := range natGateways.NatGateways {
		_, err = a.ec2Client.DeleteNatGateway(&ec2.DeleteNatGatewayInput{
			NatGatewayId: natGw.NatGatewayId,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete NAT Gateway")
		}
		// Wait for NAT Gateway to be deleted
		err = a.ec2Client.WaitUntilNatGatewayDeleted(&ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []*string{natGw.NatGatewayId},
		})
		if err != nil {
			return errors.Wrap(err, "failed to wait for NAT Gateway deletion")
		}
	}

	// Release Elastic IPs associated with NAT Gateways
	addresses, err := a.ec2Client.DescribeAddresses(&ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("domain"), Values: []*string{aws.String("vpc")}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe Elastic IPs")
	}
	for _, addr := range addresses.Addresses {
		if addr.AssociationId != nil {
			_, err = a.ec2Client.ReleaseAddress(&ec2.ReleaseAddressInput{
				AllocationId: addr.AllocationId,
			})
			if err != nil {
				return errors.Wrap(err, "failed to release Elastic IP")
			}
		}
	}

	// Step 5: Delete Subnets
	subnets, err := a.ec2Client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID)}},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to describe subnets")
	}
	for _, subnet := range subnets.Subnets {
		_, err = a.ec2Client.DeleteSubnet(&ec2.DeleteSubnetInput{
			SubnetId: subnet.SubnetId,
		})
		if err != nil {
			return errors.Wrap(err, "failed to delete subnet")
		}
	}

	// Step 6: Delete VPC
	_, err = a.ec2Client.DeleteVpc(&ec2.DeleteVpcInput{
		VpcId: aws.String(a.cluster.GetFirstCloudResource(biz.ResourceTypeVPC).ID),
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
