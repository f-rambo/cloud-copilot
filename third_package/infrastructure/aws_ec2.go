package infrastructure

import (
	"fmt"
	"sort"
	"strings"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/utils"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ebs"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	AwsProject = "aws-project"
	AwsStack   = "aws-stack"
)

const (
	awsTagkey = "ocean-key"
	awsTagVal = "ocean-cluster"

	awsVpcName = "aws-vpc-name"

	awsPrivateSubnetName = "private-subnet-" // + zone
	awsPubulilSubnetName = "public-subnet"

	awsInterneteGatewayName = "internetgateway"

	awsPublicNatgatewayEipName = "public-natgateway-eip"

	awsPrivateNatgatewayName = "private-natgateway-" // + zone
	awsPublicNatgatewayName  = "public-natgateway"

	awsPublicNatewayRouteTableName               = "public-natgateway-route-table"
	awsPublicInternetgatewayRouteTableName       = "public-internetgateway-route-table"
	awsPublicNatgatewayRouteTableAssctition      = "public-natgateway-route-table-association"
	awsPublicInternetgatewayRouteTableAssctition = "public-internetgateway-route-table-association"

	awsSecurityGroupStack = "security-group-stack"

	awsEc2RoleStack        = "ec2-role-stack"
	awsEc2RolePolicyStack  = "ec2-role-policy-stack"
	awsEc2RoleProfileStack = "ec2-role-profile-stack"

	awsEc2InstanceStack = "ec2-instance-stack"

	awsKeyPairStack = "key-pair-stack"

	awsBostionhostNetworkInterfaceStack = "bostionhost-network-interface-stack"
	awsBostionhostEipAssociationStack   = "bostionhost-eip-association-stack"
	awsBostionhostEipName               = "bostionhost-eip"

	defaultVpcCidrBlock = "10.0.0.0/16"

	awsAppLoadBalancerStack            = "app-load-balancer-stack"
	awsAppLoadBalancerListenerStack    = "app-load-balancer-listener-stack"
	awsAppLoadBalancerTargetGroupStack = "app-load-balancer-target-group-stack"
)

type GetInstanceTypeResults []*ec2.GetInstanceTypeResult

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
	if a[i].DefaultVcpus < a[j].DefaultVcpus {
		return true
	}
	if a[i].DefaultVcpus == a[j].DefaultVcpus {
		return a[i].MemorySize < a[j].MemorySize
	}
	return false
}

type InstanceTypeGpus []ec2.GetInstanceTypeGpus

// sort by grp count
func (g InstanceTypeGpus) Len() int {
	return len(g)
}

func (g InstanceTypeGpus) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}

func (g InstanceTypeGpus) Less(i, j int) bool {
	return g[i].Count < g[j].Count
}

func (a *AwsCloud) Start(ctx *pulumi.Context) (err error) {
	err = a.infrastructural(ctx)
	if err != nil {
		return err
	}
	// err = a.startNodes(ctx)
	// if err != nil {
	// 	return err
	// }
	return nil
}

func (a *AwsCloud) infrastructural(ctx *pulumi.Context) (err error) {
	err = a.getClusterInfoByInstance(ctx)
	if err != nil {
		return err
	}
	err = a.createVpc(ctx)
	if err != nil {
		return err
	}
	err = a.createSubnets(ctx)
	if err != nil {
		return err
	}
	// err = a.createGateway(ctx)
	// if err != nil {
	// 	return err
	// }
	// err = a.createRouteTable(ctx)
	// if err != nil {
	// 	return err
	// }
	// err = a.startSecurityGroup(ctx)
	// if err != nil {
	// 	return err
	// }
	// err = a.createSLB(ctx)
	// if err != nil {
	// 	return err
	// }
	// err = a.createIAM(ctx)
	// if err != nil {
	// 	return err
	// }
	// err = a.startSshKey(ctx)
	// if err != nil {
	// 	return err
	// }
	// err = a.setImageByNodeGroups(ctx)
	// if err != nil {
	// 	return err
	// }
	// err = a.setInstanceTypeByNodeGroups(ctx)
	// if err != nil {
	// 	return err
	// }
	return nil
}

func (a *AwsCloud) createVpc(ctx *pulumi.Context) (err error) {
	if a.vpcCidrBlock == "" {
		a.vpcCidrBlock = defaultVpcCidrBlock
	}
	if a.cluster.VpcID != "" {
		a.vpc, err = ec2.GetVpc(ctx, awsVpcName, pulumi.ID(a.cluster.VpcID), nil)
		if err != nil {
			return err
		}
		a.vpc, err = ec2.NewVpc(ctx, awsVpcName, &ec2.VpcArgs{
			CidrBlock: a.vpc.CidrBlock,
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(a.cluster.Name + "-vpc"),
				awsTagkey: pulumi.String(awsTagVal),
			},
		}, pulumi.Import(pulumi.ID(a.cluster.VpcID)))
		if err != nil {
			return err
		}
		return nil
	}
	a.vpc, err = ec2.NewVpc(ctx, awsVpcName, &ec2.VpcArgs{
		CidrBlock: pulumi.String(a.vpcCidrBlock),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(a.cluster.Name + "-vpc"),
			awsTagkey: pulumi.String(awsTagVal),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsCloud) createSubnets(ctx *pulumi.Context) (err error) {
	// Get list of availability zones
	zones, err := aws.GetAvailabilityZones(ctx, &aws.GetAvailabilityZonesArgs{}, nil)
	if err != nil {
		return err
	}
	if len(zones.Names) == 0 {
		return fmt.Errorf("no availability zones found")
	}
	a.zoneNames = zones.Names

	// Create a subnet in each availability zone
	zoneCidrMap := make(map[string]string)
	for _, v := range a.cluster.Nodes {
		if v.Zone == "" || v.SubnetCidr == "" {
			continue
		}
		zoneCidrMap[v.Zone] = v.SubnetCidr
	}
	usEsubnetCidrs := make([]string, 0)
	subnetCidrs, err := utils.GenerateSubnets(a.vpcCidrBlock, len(zones.Names)+len(zoneCidrMap)+1)
	if err != nil {
		return err
	}
	for _, v := range subnetCidrs {
		exits := false
		for _, s := range zoneCidrMap {
			if s == v {
				exits = true
			}
		}
		if !exits {
			usEsubnetCidrs = append(usEsubnetCidrs, v)
		}
	}
	a.privateSubnets = make([]*ec2.Subnet, len(zones.Names))
	for i, zone := range zones.Names {
		cidr := usEsubnetCidrs[i]
		if _, ok := zoneCidrMap[zone]; ok {
			cidr = zoneCidrMap[zone]
		}
		subnetArgs := &ec2.SubnetArgs{
			VpcId:            a.vpc.ID(),
			CidrBlock:        pulumi.String(cidr),
			AvailabilityZone: pulumi.String(zone),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-private-subnet-%s", a.cluster.Name, getSubnetName(zone))),
				awsTagkey: pulumi.String(awsTagVal),
				"Zone":    pulumi.String(zone),
				"Type":    pulumi.String("private"),
			},
		}
		subnet, err := ec2.NewSubnet(ctx, getSubnetName(zone), subnetArgs)
		if err != nil {
			return err
		}
		a.privateSubnets[i] = subnet
	}

	// public subnet
	a.pulicSubnet, err = ec2.NewSubnet(ctx, awsPubulilSubnetName, &ec2.SubnetArgs{
		VpcId:     a.vpc.ID(),
		CidrBlock: pulumi.String(usEsubnetCidrs[len(usEsubnetCidrs)-1]),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(a.cluster.Name + "-public-subnet"),
			awsTagkey: pulumi.String(awsTagVal),
			"Type":    pulumi.String("public"),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsCloud) createGateway(ctx *pulumi.Context) (err error) {
	// Create an Internet Gateway
	a.igw, err = ec2.NewInternetGateway(ctx, awsInterneteGatewayName, &ec2.InternetGatewayArgs{
		VpcId: a.vpc.ID(),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(a.cluster.Name + "-igw"),
			awsTagkey: pulumi.String(awsTagVal),
		},
	})
	if err != nil {
		return err
	}

	// create eip
	a.eip, err = ec2.NewEip(ctx, awsPublicNatgatewayEipName, &ec2.EipArgs{
		Domain:             pulumi.String("vpc"),
		NetworkBorderGroup: pulumi.String(a.cluster.Region),
		PublicIpv4Pool:     pulumi.String("amazon"),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-public-natgateway-eip", a.cluster.Name)),
			awsTagkey: pulumi.String(awsTagVal),
		},
	})
	if err != nil {
		return err
	}

	a.publicNatGateWay, err = ec2.NewNatGateway(ctx, awsPublicNatgatewayName, &ec2.NatGatewayArgs{
		SubnetId:         a.pulicSubnet.ID(),
		AllocationId:     a.eip.ID(),
		ConnectivityType: pulumi.String("public"),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(a.cluster.Name + "-public-natgateway"),
			awsTagkey: pulumi.String(awsTagVal),
			"Type":    pulumi.String("public"),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsCloud) createRouteTable(ctx *pulumi.Context) (err error) {
	// Create a route table with a route for the public nat gateway
	privateRouteTable, err := ec2.NewRouteTable(ctx, awsPublicNatewayRouteTableName, &ec2.RouteTableArgs{
		VpcId: a.vpc.ID(),
		Routes: ec2.RouteTableRouteArray{
			&ec2.RouteTableRouteArgs{
				NatGatewayId: a.publicNatGateWay.ID(),
				CidrBlock:    pulumi.String("0.0.0.0/0"),
			},
		},
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-%s", a.cluster.Name, awsPublicNatewayRouteTableName)),
			awsTagkey: pulumi.String(awsTagVal),
		},
	})
	if err != nil {
		return err
	}
	for index, privateSubnet := range a.privateSubnets {
		_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-%s", awsPublicNatgatewayRouteTableAssctition, getZoneName(a.zoneNames, index)), &ec2.RouteTableAssociationArgs{
			RouteTableId: privateRouteTable.ID(),
			SubnetId:     privateSubnet.ID(),
		})
		if err != nil {
			return err
		}
	}

	// Create a route table and a route for the public subnet
	pulicRouteTable, err := ec2.NewRouteTable(ctx, awsPublicInternetgatewayRouteTableName, &ec2.RouteTableArgs{
		VpcId: a.vpc.ID(),
		Routes: ec2.RouteTableRouteArray{
			&ec2.RouteTableRouteArgs{
				GatewayId: a.igw.ID(),
				CidrBlock: pulumi.String("0.0.0.0/0"),
			},
		},
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-%s", a.cluster.Name, awsPublicInternetgatewayRouteTableName)),
			awsTagkey: pulumi.String(awsTagVal),
		},
	})
	if err != nil {
		return err
	}

	_, err = ec2.NewRouteTableAssociation(ctx, awsPublicInternetgatewayRouteTableAssctition, &ec2.RouteTableAssociationArgs{
		RouteTableId: pulicRouteTable.ID(),
		SubnetId:     a.pulicSubnet.ID(),
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsCloud) startSecurityGroup(ctx *pulumi.Context) (err error) {
	// Security Group for Master and Worker nodes
	a.sg, err = ec2.NewSecurityGroup(ctx, awsSecurityGroupStack, &ec2.SecurityGroupArgs{
		VpcId: a.vpc.ID(),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(22),
				ToPort:     pulumi.Int(22),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
			&ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(6443),
				ToPort:     pulumi.Int(6443),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
			&ec2.SecurityGroupIngressArgs{
				FromPort:   pulumi.Int(443),
				ToPort:     pulumi.Int(443),
				Protocol:   pulumi.String("tcp"),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsCloud) createSLB(ctx *pulumi.Context) (err error) {
	// Create Application Load Balancer
	alb, err := lb.NewLoadBalancer(ctx, awsAppLoadBalancerStack, &lb.LoadBalancerArgs{
		Name:             pulumi.String(fmt.Sprintf("%s-alb", a.cluster.Name)),
		Internal:         pulumi.Bool(false),
		LoadBalancerType: pulumi.String("application"),
		SecurityGroups:   pulumi.StringArray{a.sg.ID()},
		Subnets:          pulumi.StringArray{a.pulicSubnet.ID()},
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-alb", a.cluster.Name)),
			awsTagkey: pulumi.String(awsTagVal),
		},
	})
	if err != nil {
		return err
	}

	// Create Target Group
	targetGroup, err := lb.NewTargetGroup(ctx, awsAppLoadBalancerListenerStack, &lb.TargetGroupArgs{
		Name:     pulumi.String(fmt.Sprintf("%s-tg", a.cluster.Name)),
		Port:     pulumi.Int(6443),
		Protocol: pulumi.String("TCP"),
		VpcId:    a.pulicSubnet.VpcId,
		HealthCheck: &lb.TargetGroupHealthCheckArgs{
			Path:               pulumi.String("/healthz"),
			Port:               pulumi.String("6443"),
			Protocol:           pulumi.String("HTTPS"),
			HealthyThreshold:   pulumi.Int(3),
			UnhealthyThreshold: pulumi.Int(3),
			Interval:           pulumi.Int(30),
			Timeout:            pulumi.Int(5),
		},
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-tg", a.cluster.Name)),
			awsTagkey: pulumi.String(awsTagVal),
		},
	})
	if err != nil {
		return err
	}

	// Create Listener
	_, err = lb.NewListener(ctx, awsAppLoadBalancerTargetGroupStack, &lb.ListenerArgs{
		LoadBalancerArn: alb.Arn,
		Port:            pulumi.Int(6443),
		Protocol:        pulumi.String("TCP"),
		DefaultActions: lb.ListenerDefaultActionArray{
			&lb.ListenerDefaultActionArgs{
				Type:           pulumi.String("forward"),
				TargetGroupArn: targetGroup.Arn,
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsCloud) createIAM(ctx *pulumi.Context) (err error) {
	// IAM Role
	ec2Role, err := iam.NewRole(ctx, awsEc2RoleStack, &iam.RoleArgs{
		Name: pulumi.String(fmt.Sprintf("%s-ec2-role", a.cluster.Name)),
		AssumeRolePolicy: pulumi.String(`{
"Version": "2012-10-17",
"Statement": [
  {
	"Effect": "Allow",
	"Principal": {
	    "Service": "ec2.amazonaws.com"
	},
	"Action": "sts:AssumeRole"
  }
]
}`),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-ec2-role", a.cluster.Name)),
			awsTagkey: pulumi.String(awsTagVal),
		},
	})
	if err != nil {
		return err
	}

	// IAM Role and Policy
	_, err = iam.NewRolePolicy(ctx, awsEc2RolePolicyStack, &iam.RolePolicyArgs{
		Name: pulumi.String(fmt.Sprintf("%s-ec2-role-policy", a.cluster.Name)),
		Role: ec2Role.ID(),
		Policy: pulumi.String(`{
"Version": "2012-10-17",
"Statement": [
  {
	"Effect": "Allow",
	"Action": [
	    "ec2:*",
	    "ecr:GetDownloadUrlForLayer",
	    "ecr:BatchGetImage",
	    "ecr:BatchCheckLayerAvailability",
	    "autoscaling:*",
	    "cloudwatch:PutMetricData",
	    "logs:*",
	    "s3:*"
	],
	"Resource": "*"
  }
]
}`),
	})
	if err != nil {
		return err
	}

	a.ec2Profile, err = iam.NewInstanceProfile(ctx, awsEc2RoleProfileStack, &iam.InstanceProfileArgs{
		Role: ec2Role.Name,
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-ec2-profile", a.cluster.Name)),
			awsTagkey: pulumi.String(awsTagVal),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsCloud) startSshKey(ctx *pulumi.Context) (err error) {
	// key pair
	a.keyPair, err = ec2.NewKeyPair(ctx, awsKeyPairStack, &ec2.KeyPairArgs{
		KeyName:   pulumi.String(fmt.Sprintf("%s-key-pair", a.cluster.Name)),
		PublicKey: pulumi.String(a.cluster.PublicKey),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-key-pair", a.cluster.Name)),
			awsTagkey: pulumi.String(awsTagVal),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsCloud) setImageByNodeGroups(ctx *pulumi.Context) (err error) {
	amiImageID := ""
	for _, nodegroup := range a.cluster.NodeGroups {
		if nodegroup.Image != "" {
			amiImageID = nodegroup.Image
			break
		}
	}
	if amiImageID == "" {
		// find AMI image
		ubuntuAmi, err := ec2.LookupAmi(ctx, &ec2.LookupAmiArgs{
			NameRegex: pulumi.StringRef("ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"),
			Filters: []ec2.GetAmiFilter{
				{
					Name:   "virtualization-type",
					Values: []string{"hvm"},
				},
				{
					Name:   "root-device-type",
					Values: []string{"ebs"},
				},
				{
					Name:   "architecture",
					Values: []string{"x86_64"},
				},
			},
			Owners:     []string{"amazon"},
			MostRecent: pulumi.BoolRef(true),
		})
		if err != nil {
			return err
		}
		if ubuntuAmi == nil || ubuntuAmi.Id == "" {
			return fmt.Errorf("unable to find ubuntu ami")
		}
		amiImageID = ubuntuAmi.Id
	}
	for _, nodegroup := range a.cluster.NodeGroups {
		if nodegroup.Image == "" {
			nodegroup.Image = amiImageID
		}
	}
	if a.cluster.BostionHost == nil {
		a.cluster.BostionHost = &biz.BostionHost{}
	}
	a.cluster.BostionHost.ImageID = amiImageID
	return nil
}

// setInstanceTypeByNodeGroups
func (a *AwsCloud) setInstanceTypeByNodeGroups(ctx *pulumi.Context) (err error) {
	for _, nodeGroup := range a.cluster.NodeGroups {
		// find suitable instance type
		if nodeGroup.InstanceType == "" {
			instanceTypeFamiliy := getIntanceTypeFamilies(nodeGroup)
			nodeInstanceTypes, err := ec2.GetInstanceTypes(ctx, &ec2.GetInstanceTypesArgs{
				Filters: []ec2.GetInstanceTypesFilter{
					{
						Name:   "processor-info.supported-architecture",
						Values: []string{"x86_64"},
					},
					{
						Name:   "instance-type",
						Values: []string{instanceTypeFamiliy},
					},
				},
			})
			if err != nil {
				return err
			}
			instanceTypeArr := make(GetInstanceTypeResults, 0)
			for _, instanceType := range nodeInstanceTypes.InstanceTypes {
				instanceTypeRes, err := ec2.GetInstanceType(ctx, &ec2.GetInstanceTypeArgs{InstanceType: instanceType})
				if err != nil {
					return err
				}
				instanceTypeArr = append(instanceTypeArr, instanceTypeRes)
			}
			sort.Sort(instanceTypeArr)
			for _, instanceType := range instanceTypeArr {
				if instanceType.MemorySize == 0 {
					continue
				}
				memoryGBiSize := float64(instanceType.MemorySize) / 1024.0
				if memoryGBiSize >= nodeGroup.Memory && instanceType.DefaultVcpus >= int(nodeGroup.CPU) {
					nodeGroup.InstanceType = instanceType.InstanceType
				}
				if nodeGroup.InstanceType == "" {
					continue
				}
				if nodeGroup.GPU == 0 {
					break
				}
				sort.Sort(InstanceTypeGpus(instanceType.Gpuses))
				for _, gpues := range instanceType.Gpuses {
					if gpues.Count >= int(nodeGroup.GPU) {
						break
					}
				}
			}
		}
		if nodeGroup.InstanceType == "" {
			return fmt.Errorf("no instance type found for node group %s", nodeGroup.Name)
		}
	}
	return nil
}

func (a *AwsCloud) startNodes(ctx *pulumi.Context) (err error) {
	if len(a.cluster.Nodes) == 0 || len(a.cluster.NodeGroups) == 0 {
		return nil
	}
	selectedBostionHost := false
	for index, node := range a.cluster.Nodes {
		// import instance
		if node.InstanceID != "" {
			instance, err := ec2.GetInstance(ctx, fmt.Sprintf("%s-%s", awsEc2InstanceStack, node.Name), pulumi.ID(node.InstanceID), nil)
			if err != nil {
				return err
			}
			nodeRes, err := ec2.NewInstance(ctx, fmt.Sprintf("%s-%s", awsEc2InstanceStack, node.Name), &ec2.InstanceArgs{
				InstanceType:       instance.InstanceType,
				NetworkInterfaces:  instance.NetworkInterfaces,
				Ami:                instance.Ami,
				IamInstanceProfile: a.ec2Profile.Name,
				KeyName:            a.keyPair.KeyName,
				RootBlockDevice:    instance.RootBlockDevice,
				Tags: pulumi.StringMap{
					"Name":     pulumi.String(node.Name),
					awsTagkey:  pulumi.String(awsTagVal),
					"NodeRole": pulumi.String(node.Role),
				},
			}, pulumi.Import(pulumi.ID(node.InstanceID)))
			if err != nil {
				return err
			}
			ctx.Export(GetKey(InstanceID, node.Name), nodeRes.ID())
			ctx.Export(GetKey(InstanceUser, node.Name), pulumi.String("ubuntu"))
			ctx.Export(GetKey(InstanceInternalIP, node.Name), nodeRes.PrivateIp)
			ctx.Export(GetKey(InstancePublicIP, node.Name), nodeRes.PublicIp)
			continue
		}
		// create node
		nodeGroup := a.cluster.GetNodeGroup(node.NodeGroupID)
		if nodeGroup == nil {
			return fmt.Errorf("node group %s not found", node.NodeGroupID)
		}
		subnet := distributeNodeSubnets(index, a.privateSubnets, a.cluster.Nodes)
		nodeNi, err := ec2.NewNetworkInterface(ctx, node.Name+"_NI", &ec2.NetworkInterfaceArgs{
			SubnetId:       subnet.ID(),
			SecurityGroups: pulumi.StringArray{a.sg.ID()},
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(node.Name + "-ni"),
				awsTagkey: pulumi.String(awsTagVal),
			},
		})
		if err != nil {
			return err
		}
		nodeRes, err := ec2.NewInstance(ctx, fmt.Sprintf("%s-%s", awsEc2InstanceStack, node.Name), &ec2.InstanceArgs{
			InstanceType: pulumi.String(nodeGroup.InstanceType),
			NetworkInterfaces: ec2.InstanceNetworkInterfaceArray{&ec2.InstanceNetworkInterfaceArgs{
				NetworkInterfaceId: nodeNi.ID(),
				DeviceIndex:        pulumi.Int(0),
			}},
			Ami:                pulumi.String(nodeGroup.Image),
			IamInstanceProfile: a.ec2Profile.Name,
			KeyName:            a.keyPair.KeyName,
			RootBlockDevice: &ec2.InstanceRootBlockDeviceArgs{
				VolumeSize: pulumi.Int(node.SystemDisk),
			},
			Tags: pulumi.StringMap{
				"Name":     pulumi.String(node.Name),
				awsTagkey:  pulumi.String(awsTagVal),
				"NodeRole": pulumi.String(node.Role),
			},
		})
		if err != nil {
			return err
		}
		if node.DataDisk > 0 {
			// Create an additional EBS volume
			volume, err := ebs.NewVolume(ctx, fmt.Sprintf("%s-data-volume", node.Name), &ebs.VolumeArgs{
				AvailabilityZone: nodeRes.AvailabilityZone,
				Size:             pulumi.Int(node.DataDisk), // Set additional disk size to 100 GiB
			})
			if err != nil {
				return err
			}

			// Attach the additional EBS volume to the instance myVolumeAttachment
			_, err = ec2.NewVolumeAttachment(ctx, fmt.Sprintf("%s-volume-attachment", node.Name), &ec2.VolumeAttachmentArgs{
				InstanceId: nodeRes.ID(),
				VolumeId:   volume.ID(),
				DeviceName: pulumi.String("/dev/sdf"), // Attach the volume as /dev/sdh
			})
			if err != nil {
				return err
			}
		}
		if !selectedBostionHost && node.Role == biz.NodeRoleMaster {
			ctx.Export(GetKey(BostionHostInstanceID), nodeRes.ID())
			selectedBostionHost = true
			// bind eip to instance
			_, err = ec2.NewEipAssociation(ctx, awsBostionhostEipAssociationStack, &ec2.EipAssociationArgs{
				AllocationId: a.eip.ID(),
				InstanceId:   nodeRes.ID(),
			})
			if err != nil {
				return err
			}
		}
		ctx.Export(GetKey(InstanceID, node.Name), nodeRes.ID())
		ctx.Export(GetKey(InstanceUser, node.Name), pulumi.String("ubuntu"))
		ctx.Export(GetKey(InstanceInternalIP, node.Name), nodeRes.PrivateIp)
		ctx.Export(GetKey(InstancePublicIP, node.Name), nodeRes.PublicIp)
	}
	return nil
}

func (a *AwsCloud) getClusterInfoByInstance(ctx *pulumi.Context) error {
	isNotUnspecifiedNodes := make([]*biz.Node, 0)
	for _, node := range a.cluster.Nodes {
		if node.Status != biz.NodeStatusUnspecified {
			isNotUnspecifiedNodes = append(isNotUnspecifiedNodes, node)
		}
	}
	if len(isNotUnspecifiedNodes) == 0 {
		return nil
	}
	instances, err := ec2.GetInstances(ctx, &ec2.GetInstancesArgs{})
	if err != nil {
		return err
	}
	sgids := make([]string, 0)
	subnetIds := make([]string, 0)
	for _, instanceID := range instances.Ids {
		instance, err := ec2.LookupInstance(ctx, &ec2.LookupInstanceArgs{
			InstanceId: pulumi.StringRef(instanceID),
		})
		if err != nil {
			return err
		}
		if instance == nil {
			continue
		}
		for _, node := range isNotUnspecifiedNodes {
			if node.InternalIP == instance.PrivateIp {
				node.InstanceID = instanceID
				node.SubnetId = instance.SubnetId
				node.Zone = instance.AvailabilityZone
				node.ExternalIP = instance.PublicIp
				node.InternalIP = instance.PrivateIp
				sgids = append(sgids, instance.VpcSecurityGroupIds...)
				subnetIds = append(subnetIds, instance.SubnetId)
				break
			}
		}
	}
	if len(sgids) == 0 || len(subnetIds) == 0 {
		return fmt.Errorf("no instance found")
	}
	sgids = utils.RemoveDuplicateString(sgids)
	a.cluster.SecurityGroupIDs = strings.Join(sgids, ",")
	// get vpc by subnet
	var subnetId string
	for _, subnetID := range subnetIds {
		subnetId = subnetID
		break
	}
	subnet, err := ec2.LookupSubnet(ctx, &ec2.LookupSubnetArgs{
		Id: pulumi.StringRef(subnetId),
	})
	if err != nil {
		return err
	}
	a.cluster.VpcID = subnet.VpcId
	vpc, err := ec2.LookupVpc(ctx, &ec2.LookupVpcArgs{
		Id: pulumi.StringRef(subnet.VpcId),
	})
	if err != nil {
		return err
	}
	a.cluster.VpcCidr = vpc.CidrBlock
	return nil
}

func getSubnetName(zone string) string {
	return fmt.Sprintf("%s%s", awsPrivateSubnetName, zone)
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

func distributeNodeSubnets(nodeIndex int, subnets []*ec2.Subnet, nodes []*biz.Node) *ec2.Subnet {
	if len(subnets) == 0 {
		return nil
	}
	nodeSize := len(nodes)
	subnetsSize := len(subnets)
	if nodeSize <= subnetsSize {
		return subnets[nodeIndex%subnetsSize]
	}
	interval := nodeSize / subnetsSize
	return subnets[(nodeIndex/interval)%subnetsSize]
}

func getZoneName(zoneNames []string, index int) string {
	if index >= len(zoneNames) {
		return fmt.Sprintf("%s-%d", zoneNames[len(zoneNames)-1], index)
	}
	return zoneNames[index]
}
