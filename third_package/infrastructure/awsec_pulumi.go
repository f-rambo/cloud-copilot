package infrastructure

import (
	"fmt"
	"sort"
	"strings"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ebs"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	AWS_PROJECT = "aws-project"
	AWS_STACK   = "aws-stack"
)

const (
	TAG_KEY       = "ocean-key"
	TAG_VAL       = "ocean-cluster"
	UBUNTU_AMI_ID = "ami-04a81a99f5ec58529"
)

// cluster key
const (
	// PublicKey
	CLUSTER_PUBLIC_KEY = "cluster_public_key"
	// Region
	CLUSTER_REGION = "cluster_region"
	// VpcID
	CLUSTER_VPC_ID = "cluster_vpc_id"
	// ResourceGroupID
	CLUSTER_RESOURCE_GROUP_ID = "cluster_resource_group_id"
	// SecurityGroupIDs
	CLUSTER_SECURITY_GROUP_IDS = "cluster_security_group_ids"
	// ExternalIP
	CLUSTER_EXTERNAL_IP = "cluster_external_ip"
)

// nodegroup key
const (
	// InstanceType
	NODEGROUP_INSTANCE_TYPE = "nodegroup_instance_type"
	// OSImage
	NODEGROUP_OS_IMAGE = "nodegroup_os_image"
	// CPU
	NODEGROUP_CPU = "nodegroup_cpu"
	// Memory
	NODEGROUP_MEMORY = "nodegroup_memory"
	// GPU
	NODEGROUP_GPU = "nodegroup_gpu"
	// GpuSpec
	NODEGROUP_GPU_SPEC = "nodegroup_gpu_spec"
	// SystemDisk
	NODEGROUP_SYSTEM_DISK = "nodegroup_system_disk"
	// DataDisk
	NODEGROUP_DATA_DISK = "nodegroup_data_disk"
	// InternetMaxBandwidthOut
	NODEGROUP_INTERNET_MAX_BANDWIDTH_OUT = "nodegroup_internet_max_bandwidth_out"
	// NodePrice
	NODEGROUP_NODE_PRICE = "nodegroup_node_price"
)

// node key
const (
	// InstanceID
	NODE_INSTANCE_ID = "node_instance_id"
	// Labels
	NODE_LABELS = "node_labels"
	// Kernel
	NODE_KERNEL = "node_kernel"
	// InternalIP
	NODE_INTERNAL_IP = "node_internal_ip"
	// ExternalIP
	NODE_EXTERNAL_IP = "node_external_ip"
	// Status
	NODE_STATUS = "node_status"
	// SwitchId
	NODE_SWITCH_ID = "node_switch_id"
	// ZoneId
	NODE_ZONE_ID = "node_zone_id"
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
	cluster    *biz.Cluster
	vpc        *ec2.Vpc
	zones      *aws.GetAvailabilityZonesResult
	subnets    []*ec2.Subnet
	igw        *ec2.InternetGateway
	sg         *ec2.SecurityGroup
	ec2Profile *iam.InstanceProfile
	keyPair    *ec2.KeyPair
}

func StartEc2Instance(cluster *biz.Cluster) *AwsEc2Instance {
	return &AwsEc2Instance{
		cluster: cluster,
	}
}

func (a *AwsEc2Instance) Start(ctx *pulumi.Context) error {
	err := a.infrastructural(ctx)
	if err != nil {
		return err
	}
	err = a.bostionHost(ctx)
	if err != nil {
		return err
	}
	err = a.nodes(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsEc2Instance) getIntanceTypeFamilies(nodeGroup *biz.NodeGroup) string {
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

func (a *AwsEc2Instance) distributeNodeSubnetsFunc(nodeIndex int) *ec2.Subnet {
	if len(a.subnets) == 0 {
		return nil
	}
	nodeSize := len(a.cluster.Nodes)
	subnetsSize := len(a.subnets)
	if nodeSize <= subnetsSize {
		return a.subnets[nodeIndex%subnetsSize]
	}
	interval := nodeSize / subnetsSize
	return a.subnets[(nodeIndex/interval)%subnetsSize]
}

func (a *AwsEc2Instance) infrastructural(ctx *pulumi.Context) (err error) {
	// Create a VPC
	a.vpc, err = ec2.NewVpc(ctx, "k8s-vpc", &ec2.VpcArgs{
		CidrBlock: pulumi.String(a.cluster.VpcCidrBlock),
		Tags: pulumi.StringMap{
			"Name":  pulumi.String(a.cluster.Name + "-vpc"),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return err
	}

	// Get list of availability zones
	a.zones, err = aws.GetAvailabilityZones(ctx, &aws.GetAvailabilityZonesArgs{}, nil)
	if err != nil {
		return err
	}

	// Create a subnet in each availability zone
	a.subnets = make([]*ec2.Subnet, len(a.zones.Names))
	for i, zone := range a.zones.Names {
		subnet, err := ec2.NewSubnet(ctx, "k8s-subnet-"+zone, &ec2.SubnetArgs{
			VpcId:            a.vpc.ID(),
			CidrBlock:        pulumi.String(fmt.Sprintf("10.0.%d.0/24", i+1)),
			AvailabilityZone: pulumi.String(zone),
			Tags: pulumi.StringMap{
				"Name":  pulumi.String("k8s-subnet-" + zone),
				TAG_KEY: pulumi.String(TAG_VAL),
			},
		})
		if err != nil {
			return err
		}
		a.subnets[i] = subnet
	}

	// Create an Internet Gateway
	a.igw, err = ec2.NewInternetGateway(ctx, "k8s-igw", &ec2.InternetGatewayArgs{
		VpcId: a.vpc.ID(),
		Tags: pulumi.StringMap{
			"Name":  pulumi.String(a.cluster.Name + "-igw"),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return err
	}

	// Create a route table and a route
	rt, err := ec2.NewRouteTable(ctx, "k8s-rt", &ec2.RouteTableArgs{
		VpcId: a.vpc.ID(),
		Tags: pulumi.StringMap{
			"Name":  pulumi.String("k8s-rt"),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return err
	}

	// bind the route table to the internet gateway
	_, err = ec2.NewRoute(ctx, "k8s-route", &ec2.RouteArgs{
		RouteTableId:         rt.ID(),
		DestinationCidrBlock: pulumi.String("0.0.0.0/0"),
		GatewayId:            a.igw.ID(),
	})
	if err != nil {
		return err
	}

	// Associate route table with subnets
	for i, subnet := range a.subnets {
		_, err = ec2.NewRouteTableAssociation(ctx, "k8s-rta"+fmt.Sprint(i), &ec2.RouteTableAssociationArgs{
			SubnetId:     subnet.ID(),
			RouteTableId: rt.ID(),
		})
		if err != nil {
			return err
		}
	}

	// Security Group for Master and Worker nodes
	a.sg, err = ec2.NewSecurityGroup(ctx, "k8s-sg", &ec2.SecurityGroupArgs{
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

	a.ec2Profile, err = iam.NewInstanceProfile(ctx, "ec2Profile", &iam.InstanceProfileArgs{
		Role: ec2Role.Name,
		Tags: pulumi.StringMap{
			"Name":  pulumi.String("ec2Profile"),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return err
	}

	// key pair
	a.keyPair, err = ec2.NewKeyPair(ctx, "k8s-keypair", &ec2.KeyPairArgs{
		KeyName:   pulumi.String("k8s-keypair"),
		PublicKey: pulumi.String(a.cluster.PublicKey),
	})
	if err != nil {
		return err
	}
	return nil
}

// https://aws.amazon.com/cn/ec2/instance-types/
// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInstanceTypes.html

func (a *AwsEc2Instance) bostionHost(ctx *pulumi.Context) (err error) {
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
		VpcSecurityGroupIds: pulumi.StringArray{a.sg.ID()},
		SubnetId:            a.subnets[0].ID(),
		Ami:                 pulumi.String(UBUNTU_AMI_ID),
		IamInstanceProfile:  a.ec2Profile.Name,
		KeyName:             a.keyPair.KeyName,
		Tags: pulumi.StringMap{
			"Name":     pulumi.String("k8s-master-node"),
			TAG_KEY:    pulumi.String(TAG_VAL),
			"NodeRole": pulumi.String("BostionHost"),
		},
	})
	if err != nil {
		return err
	}
	ctx.Export("bostionHostNode_ID", bostionHostNode)
	ctx.Export("bostionHostNode_Pulic_IP", bostionHostNode.PublicIp)
	ctx.Export("bostionHostNode_Private_IP", bostionHostNode.PrivateIp)
	return nil
}

func (a *AwsEc2Instance) nodes(ctx *pulumi.Context) error {
	for _, nodeGroup := range a.cluster.NodeGroups {
		instanceTypeFamiliy := a.getIntanceTypeFamilies(nodeGroup)
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
			subnet := a.distributeNodeSubnetsFunc(index)
			nodeRes, err := ec2.NewInstance(ctx, node.Name, &ec2.InstanceArgs{
				InstanceType:        pulumi.String(nodeGroup.InstanceType),
				VpcSecurityGroupIds: pulumi.StringArray{a.sg.ID()},
				SubnetId:            subnet.ID(),
				Ami:                 pulumi.String(UBUNTU_AMI_ID),
				IamInstanceProfile:  a.ec2Profile.Name,
				KeyName:             a.keyPair.KeyName,
				RootBlockDevice: &ec2.InstanceRootBlockDeviceArgs{
					VolumeSize: pulumi.Int(nodeGroup.SystemDisk),
				},
				Tags: pulumi.StringMap{
					"Name":     pulumi.String(nodeGroup.Name),
					TAG_KEY:    pulumi.String(TAG_VAL),
					"NodeRole": pulumi.String(node.Role),
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

func (a *AwsEc2Instance) Get(ctx *pulumi.Context) error {
	instances, err := ec2.GetInstances(ctx, &ec2.GetInstancesArgs{})
	if err != nil {
		return err
	}
	subnetIds := make([]string, 0)
	instanceTypes := make([]string, 0)
	for _, instanceID := range instances.Ids {
		instance, err := ec2.GetInstance(ctx, "", pulumi.ID(instanceID), nil)
		if err != nil {
			return err
		}
		// cluster
		ctx.Export(strings.Join([]string{instanceID, CLUSTER_PUBLIC_KEY}, ","), instance.KeyName)
		ctx.Export(strings.Join([]string{instanceID, CLUSTER_SECURITY_GROUP_IDS}, ","), instance.VpcSecurityGroupIds)
		// nodegroup
		ctx.Export(strings.Join([]string{instanceID, NODEGROUP_INSTANCE_TYPE}, ","), instance.InstanceType)
		instance.InstanceType.ApplyT(func(v string) string {
			for _, vv := range instanceTypes {
				if vv == v {
					return v
				}
			}
			instanceTypes = append(instanceTypes, v)
			return v
		})
		ctx.Export(strings.Join([]string{instanceID, NODEGROUP_OS_IMAGE}, ","), instance.Ami)
		// node
		ctx.Export(strings.Join([]string{instanceID, NODE_INTERNAL_IP}, ","), instance.PrivateIp)
		ctx.Export(strings.Join([]string{instanceID, NODE_EXTERNAL_IP}, ","), instance.PublicIp)
		ctx.Export(strings.Join([]string{instanceID, NODE_STATUS}, ","), instance.InstanceState)
		ctx.Export(strings.Join([]string{instanceID, NODE_ZONE_ID}, ","), instance.AvailabilityZone)
		ctx.Export(strings.Join([]string{instanceID, NODE_SWITCH_ID}, ","), instance.SubnetId)
		instance.SubnetId.ApplyT(func(v string) string {
			for _, vv := range subnetIds {
				if vv == v {
					return v
				}
			}
			subnetIds = append(subnetIds, v)
			return v
		})
	}
	vpcID := ""
	for i, subnetID := range subnetIds {
		subnet, err := ec2.GetSubnet(ctx, fmt.Sprintf("subnet-%d", i), pulumi.ID(subnetID), nil)
		if err != nil {
			return err
		}
		if vpcID == "" {
			subnet.VpcId.ApplyT(func(v string) string {
				vpcID = v
				return v
			})
		}
	}
	for i, instanceType := range instanceTypes {
		instanceTypeRes, err := ec2.GetInstanceType(ctx, &ec2.GetInstanceTypeArgs{InstanceType: instanceType})
		if err != nil {
			return err
		}
		ctx.Export(strings.Join([]string{string(i), NODEGROUP_CPU}, ","), pulumi.Int(instanceTypeRes.DefaultVcpus))
		ctx.Export(strings.Join([]string{string(i), NODEGROUP_MEMORY}, ","), pulumi.Int(instanceTypeRes.MemorySize))
		for _, gpues := range instanceTypeRes.Gpuses {
			ctx.Export(strings.Join([]string{string(i), NODEGROUP_GPU_SPEC}, ","), pulumi.String(gpues.Name))
			ctx.Export(strings.Join([]string{string(i), NODEGROUP_GPU}, ","), pulumi.Int(gpues.Count))
		}
		ctx.Export(strings.Join([]string{string(i), NODEGROUP_DATA_DISK}, ","), pulumi.Int(instanceTypeRes.TotalInstanceStorage))
	}
	return nil
}

func (a *AwsEc2Instance) DecodeClusterInfomation(cluster *biz.Cluster, output string) error {

	return nil
}

func (a *AwsEc2Instance) Clear(ctx *pulumi.Context) error {
	return nil
}
