package pulumi

import (
	"fmt"
	"sort"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ebs"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	ec2OceanTagKey  = "ocean-key"
	ec2OceanTagVal  = "ocean-cluster"
	ec2VpcName      = "ocean-vpc"
	ec2VpcCidrBlock = "192.168.0.0/16"
	ec2InternetGw   = "ocean-igw"
	ec2UbuntuAmiId  = "ami-04a81a99f5ec58529"
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

type AwsEc2Instance struct {
	cluster *biz.Cluster
}

func StartEc2Instance(cluster *biz.Cluster) func(*pulumi.Context) error {
	awsEc2Instance := &AwsEc2Instance{
		cluster: cluster,
	}
	return awsEc2Instance.Start
}

// ec2 instance
func (a *AwsEc2Instance) Start(ctx *pulumi.Context) error {
	// Create a VPC
	vpc, err := ec2.NewVpc(ctx, "k8s-vpc", &ec2.VpcArgs{
		CidrBlock: pulumi.String(ec2VpcCidrBlock),
		Tags: pulumi.StringMap{
			"Name":         pulumi.String(ec2VpcName),
			ec2OceanTagKey: pulumi.String(ec2OceanTagVal),
		},
	})
	if err != nil {
		return err
	}

	// Get list of availability zones
	zones, err := aws.GetAvailabilityZones(ctx, &aws.GetAvailabilityZonesArgs{}, nil)
	if err != nil {
		return err
	}

	// Create a subnet in each availability zone
	subnets := make([]*ec2.Subnet, len(zones.Names))
	for i, zone := range zones.Names {
		subnet, err := ec2.NewSubnet(ctx, "k8s-subnet-"+zone, &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String(fmt.Sprintf("10.0.%d.0/24", i+1)),
			AvailabilityZone: pulumi.String(zone),
			Tags: pulumi.StringMap{
				"Name":         pulumi.String("k8s-subnet-" + zone),
				ec2OceanTagKey: pulumi.String(ec2OceanTagVal),
			},
		})
		if err != nil {
			return err
		}
		subnets[i] = subnet
	}
	distributeNodeSubnetsFunc := func(nodeIndex int) *ec2.Subnet {
		nodeSize := len(a.cluster.Nodes)
		subnetsSize := len(subnets)
		if nodeSize <= subnetsSize {
			return subnets[nodeIndex%subnetsSize]
		}
		interval := nodeSize / subnetsSize
		return subnets[(nodeIndex/interval)%subnetsSize]
	}

	// Create an Internet Gateway
	igw, err := ec2.NewInternetGateway(ctx, "k8s-igw", &ec2.InternetGatewayArgs{
		VpcId: vpc.ID(),
		Tags: pulumi.StringMap{
			"Name":         pulumi.String(ec2InternetGw),
			ec2OceanTagKey: pulumi.String(ec2OceanTagVal),
		},
	})
	if err != nil {
		return err
	}

	// Create a route table and a route
	rt, err := ec2.NewRouteTable(ctx, "k8s-rt", &ec2.RouteTableArgs{
		VpcId: vpc.ID(),
		Tags: pulumi.StringMap{
			"Name":         pulumi.String("k8s-rt"),
			ec2OceanTagKey: pulumi.String(ec2OceanTagVal),
		},
	})
	if err != nil {
		return err
	}

	// bind the route table to the internet gateway
	_, err = ec2.NewRoute(ctx, "k8s-route", &ec2.RouteArgs{
		RouteTableId:         rt.ID(),
		DestinationCidrBlock: pulumi.String("0.0.0.0/0"),
		GatewayId:            igw.ID(),
	})
	if err != nil {
		return err
	}

	// Associate route table with subnets
	for i, subnet := range subnets {
		_, err = ec2.NewRouteTableAssociation(ctx, "k8s-rta"+fmt.Sprint(i), &ec2.RouteTableAssociationArgs{
			SubnetId:     subnet.ID(),
			RouteTableId: rt.ID(),
		})
		if err != nil {
			return err
		}
	}

	// Security Group for Master and Worker nodes
	sg, err := ec2.NewSecurityGroup(ctx, "k8s-sg", &ec2.SecurityGroupArgs{
		VpcId: vpc.ID(),
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
			// Add other necessary ingress rules here
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

	// IAM Role
	ec2Role, err := iam.NewRole(ctx, "ec2Role", &iam.RoleArgs{
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
	})
	if err != nil {
		return err
	}

	// IAM Role and Policy
	_, err = iam.NewRolePolicy(ctx, "ec2Policy", &iam.RolePolicyArgs{
		Role: ec2Role.ID(),
		Policy: pulumi.String(`{
	    "Version": "2012-10-17",
	    "Statement": [
		  {
			"Effect": "Allow",
			"Action": [
			    "ec2:Describe*",
			    "ecr:GetDownloadUrlForLayer",
			    "ecr:BatchGetImage",
			    "ecr:BatchCheckLayerAvailability",
			    "autoscaling:Describe*",
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

	ec2Profile, err := iam.NewInstanceProfile(ctx, "ec2Profile", &iam.InstanceProfileArgs{
		Role: ec2Role.Name,
		Tags: pulumi.StringMap{
			"Name":         pulumi.String("ec2Profile"),
			ec2OceanTagKey: pulumi.String(ec2OceanTagVal),
		},
	})
	if err != nil {
		return err
	}

	// key pair
	keyRes, err := ec2.NewKeyPair(ctx, "k8s-keypair", &ec2.KeyPairArgs{
		KeyName:   pulumi.String("k8s-keypair"),
		PublicKey: pulumi.String(a.cluster.PublicKey),
	})
	if err != nil {
		return err
	}

	// https://aws.amazon.com/cn/ec2/instance-types/
	// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInstanceTypes.html

	//  bostionHost
	bostionHostInstanceTypes, err := ec2.GetInstanceTypes(ctx, &ec2.GetInstanceTypesArgs{
		Filters: []ec2.GetInstanceTypesFilter{
			{
				Name:   "processor-info.supported-architecture",
				Values: []string{"x86_64"},
			},
			{
				Name:   "instance-type",
				Values: []string{"t2.*"},
			},
			{
				Name:   "vcpu-info.default-vcpus",
				Values: []string{"2"},
			},
		},
	})
	if err != nil {
		return err
	}
	instanceTypeArr := make(GetInstanceTypeResults, 0)
	for _, instanceType := range bostionHostInstanceTypes.InstanceTypes {
		instanceTypeRes, err := ec2.GetInstanceType(ctx, &ec2.GetInstanceTypeArgs{InstanceType: instanceType})
		if err != nil {
			return err
		}
		instanceTypeArr = append(instanceTypeArr, instanceTypeRes)
	}
	sort.Sort(instanceTypeArr)
	bostionHostInstanceType := ""
	for _, instanceType := range instanceTypeArr {
		bostionHostInstanceType = instanceType.InstanceType
		break
	}
	bostionHostNode, err := ec2.NewInstance(ctx, "bostionHost-node", &ec2.InstanceArgs{
		InstanceType:        pulumi.String(bostionHostInstanceType),
		VpcSecurityGroupIds: pulumi.StringArray{sg.ID()},
		SubnetId:            subnets[0].ID(),
		Ami:                 pulumi.String(ec2UbuntuAmiId),
		IamInstanceProfile:  ec2Profile.Name,
		KeyName:             keyRes.KeyName,
		Tags: pulumi.StringMap{
			"Name":         pulumi.String("k8s-master-node"),
			ec2OceanTagKey: pulumi.String(ec2OceanTagVal),
			"NodeRole":     pulumi.String("BostionHost"),
		},
	})
	if err != nil {
		return err
	}
	ctx.Export("bostionHostNode_ID", bostionHostNode)
	ctx.Export("bostionHostNode_Pulic_IP", bostionHostNode.PublicIp)
	ctx.Export("bostionHostNode_Private_IP", bostionHostNode.PrivateIp)

	// cluster nodes
	for _, nodeGroup := range a.cluster.NodeGroups {
		instanceTypeFamiliy := "m5.*"
		if nodeGroup.GPU > 0 {
			instanceTypeFamiliy = "p3.*"
		}
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
		if nodeGroup.InstanceType == "" {
			return fmt.Errorf("no instance type found for node group %s", nodeGroup.Name)
		}
		for index, node := range a.cluster.Nodes {
			if node.NodeGroupID != nodeGroup.ID {
				continue
			}
			subnet := distributeNodeSubnetsFunc(index)
			nodeRes, err := ec2.NewInstance(ctx, node.Name, &ec2.InstanceArgs{
				InstanceType:        pulumi.String(nodeGroup.InstanceType),
				VpcSecurityGroupIds: pulumi.StringArray{sg.ID()},
				SubnetId:            subnet.ID(),
				Ami:                 pulumi.String(ec2UbuntuAmiId),
				IamInstanceProfile:  ec2Profile.Name,
				KeyName:             keyRes.KeyName,
				RootBlockDevice: &ec2.InstanceRootBlockDeviceArgs{
					VolumeSize: pulumi.Int(nodeGroup.SystemDisk),
				},
				Tags: pulumi.StringMap{
					"Name":         pulumi.String(nodeGroup.Name),
					ec2OceanTagKey: pulumi.String(ec2OceanTagVal),
					"NodeRole":     pulumi.String(node.Role),
				},
			})
			if err != nil {
				return err
			}
			ctx.Export(node.Name+"_ID", nodeRes.ID())
			ctx.Export(node.Name+"_Pulic_IP", nodeRes.PublicIp)
			ctx.Export(node.Name+"_Private_IP", nodeRes.PrivateIp)

			if nodeGroup.DataDisk > 0 {
				// Create an additional EBS volume
				volume, err := ebs.NewVolume(ctx, "myVolume", &ebs.VolumeArgs{
					AvailabilityZone: nodeRes.AvailabilityZone,
					Size:             pulumi.Int(nodeGroup.DataDisk), // Set additional disk size to 100 GiB
				})
				if err != nil {
					return err
				}

				// Attach the additional EBS volume to the instance
				_, err = ec2.NewVolumeAttachment(ctx, "myVolumeAttachment", &ec2.VolumeAttachmentArgs{
					InstanceId: nodeRes.ID(),
					VolumeId:   volume.ID(),
					DeviceName: pulumi.String("/dev/sdf"), // Device name for the attached volume
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
