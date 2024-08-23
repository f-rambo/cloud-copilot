package infrastructure

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/utils"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ebs"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

/*

1. VPC（Virtual Private Cloud）

	•	作用: 提供隔离的网络环境，用于托管 Kubernetes 集群的所有资源。
	•	配置:
	•	创建一个新的 VPC。
	•	配置适当的 CIDR 块（例如 10.0.0.0/16）。

2. 子网（Subnets）

	•	作用: 子网用于将 VPC 划分成更小的网络段，通常需要创建多个子网，以便在不同的可用区中实现高可用性。
	•	配置:
	•	公共子网: 用于暴露对外服务的 Kubernetes 资源（如负载均衡器）。
	•	私有子网: 用于部署 Kubernetes 工作节点（Worker Nodes），这些子网中的实例通常不直接暴露到互联网。

3. 互联网网关（Internet Gateway）

	•	作用: 允许公共子网中的资源访问互联网。
	•	配置:
	•	为 VPC 创建一个互联网网关。
	•	将互联网网关附加到 VPC。
	•	配置路由表，将公共子网的路由表指向互联网网关。

7. Elastic IP（弹性 IP）

	•	作用: 如果需要为控制平面的 API 服务器或负载均衡器分配固定的公网 IP，可以使用弹性 IP。
	•	配置:
	•	通过 AWS 管理控制台分配弹性 IP，并将其绑定到相应的 EC2 实例或负载均衡器。

4. NAT 网关（NAT Gateway）

	•	作用: 允许私有子网中的资源（如 EC2 实例）访问互联网，同时保护它们不被外部直接访问。
	•	配置:
	•	创建 NAT 网关，并将其放置在公共子网中。
	•	更新私有子网的路由表，使其通过 NAT 网关访问互联网。

5. 安全组（Security Groups）

	•	作用: 控制进出 Kubernetes 节点的流量。
	•	配置:
	•	创建安全组以允许 Kubernetes 控制平面和节点之间的必要通信（如 API 服务器、节点端口、SSH 等）。
	•	配置相应的入站和出站规则。

6. EC2 实例（EC2 Instances）

	•	作用: 运行 Kubernetes 控制平面（Master Nodes）和工作节点（Worker Nodes）。
	•	配置:
	•	控制平面实例: 用于运行 Kubernetes API 服务器、调度器、etcd 等核心组件。通常需要配置为高可用。
	•	工作节点实例: 用于运行实际的容器化应用程序。
	•	选择适当的实例类型（如 t3.medium、m5.large 等）以匹配工作负载需求。

8. Elastic Load Balancer (ELB)

	•	作用: 为 Kubernetes 控制平面或应用程序提供负载均衡和高可用性。
	•	配置:
	•	使用 Application Load Balancer (ALB) 或 Network Load Balancer (NLB) 作为 Kubernetes 的外部入口点，尤其是对于 Kubernetes API 服务器。

9. IAM 角色和策略（IAM Roles and Policies）

	•	作用: 为 Kubernetes 控制平面和节点分配适当的权限，以访问 AWS 资源（如 S3 存储桶、EBS 卷等）。
	•	配置:
	•	创建 IAM 角色，并附加适当的策略。
	•	将这些 IAM 角色分配给 EC2 实例，允许它们与 AWS 资源交互。

10. S3 存储桶（S3 Bucket）

	•	作用: 如果使用 kops 等工具，可以使用 S3 存储桶来存储 Kubernetes 集群的配置文件和状态信息。
	•	配置:
	•	创建一个 S3 存储桶，并配置适当的访问权限。

11. EBS 卷（Elastic Block Store Volumes）

	•	作用: 为 Kubernetes 节点提供持久存储。
	•	配置:
	•	在创建 EC2 实例时，配置适当的 EBS 卷大小和类型（如 gp3、io1），以满足存储需求。

12. Route 53 (可选)

	•	作用: 如果你需要为 Kubernetes 集群提供 DNS 服务，可以使用 AWS 的 Route 53 服务。
	•	配置:
	•	创建一个托管区并配置 DNS 记录，指向 Kubernetes 集群的入口点。

13. Route Table (路由表)

	•	作用: 控制子网中流量的路由规则。
	•	配置:
	•	为 VPC 创建一个路由表，并将其与子网关联。
	•	配置适当的路由规则，以确保流量正确路由到目标。

*/

const (
	AWS_PROJECT = "aws-project"
	AWS_STACK   = "aws-stack"
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

func StartEc2Instance(cluster *biz.Cluster) *AwsEc2Instance {
	return &AwsEc2Instance{
		cluster: cluster,
	}
}

func (a *AwsEc2Instance) Start(ctx *pulumi.Context) error {
	vpcCidrBlock := VPC_CIDR
	if a.cluster.VpcCidr != "" {
		vpcCidrBlock = a.cluster.VpcCidr
	}
	vpcFunc := &AwsVpc{
		cluster:   a.cluster,
		cidrBlock: vpcCidrBlock,
	}
	subnetFunc, err := vpcFunc.startVpc(ctx)
	if err != nil {
		return err
	}
	gatewayFunc, err := subnetFunc.startSubnets(ctx)
	if err != nil {
		return err
	}
	routeTableFunc, err := gatewayFunc.startGateway(ctx)
	if err != nil {
		return err
	}
	securityGroupFunc, err := routeTableFunc.startRouteTable(ctx)
	if err != nil {
		return err
	}
	slbFunc, err := securityGroupFunc.startSecurityGroup(ctx)
	if err != nil {
		return err
	}
	iamFunc, err := slbFunc.startSLB(ctx)
	if err != nil {
		return err
	}
	keyFunc, err := iamFunc.startIAM(ctx)
	if err != nil {
		return err
	}
	amiFunc, err := keyFunc.startSshKey(ctx)
	if err != nil {
		return err
	}
	bostionHostFunc, err := amiFunc.findAmi(ctx)
	if err != nil {
		return err
	}
	nodeFunc, err := bostionHostFunc.startBostionHost(ctx)
	if err != nil {
		return err
	}
	err = nodeFunc.startNodes(ctx)
	if err != nil {
		return err
	}
	return nil
}

type AwsVpc struct {
	cluster   *biz.Cluster
	cidrBlock string
}

func (v *AwsVpc) startVpc(ctx *pulumi.Context) (*AwsSubnet, error) {
	vpcres, err := ec2.NewVpc(ctx, VPC_STACK, &ec2.VpcArgs{
		CidrBlock: pulumi.String(v.cidrBlock),
		Tags: pulumi.StringMap{
			"Name":  pulumi.String(v.cluster.Name + "-vpc"),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return nil, err
	}
	return &AwsSubnet{
		cluster:      v.cluster,
		vpc:          vpcres,
		vpcCidrBlock: v.cidrBlock,
	}, nil
}

type AwsSubnet struct {
	cluster      *biz.Cluster
	vpc          *ec2.Vpc
	vpcCidrBlock string
}

func (s *AwsSubnet) startSubnets(ctx *pulumi.Context) (*AwsGateWay, error) {
	// Get list of availability zones
	zones, err := aws.GetAvailabilityZones(ctx, &aws.GetAvailabilityZonesArgs{}, nil)
	if err != nil {
		return nil, err
	}
	if len(zones.Names) == 0 {
		return nil, fmt.Errorf("no availability zones found")
	}

	// Create a subnet in each availability zone
	zoneCidrMap := make(map[string]string)
	for _, v := range s.cluster.Nodes {
		if v.Zone == "" || v.SubnetCidr == "" {
			continue
		}
		zoneCidrMap[v.Zone] = v.SubnetCidr
	}
	usEsubnetCidrs := make([]string, 0)
	subnetCidrs, err := utils.GenerateSubnets(s.vpcCidrBlock, len(zones.Names)+len(zoneCidrMap)+1)
	if err != nil {
		return nil, err
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
	privateSubnets := make([]*ec2.Subnet, len(zones.Names))
	for i, zone := range zones.Names {
		cidr := usEsubnetCidrs[i]
		if _, ok := zoneCidrMap[zone]; ok {
			cidr = zoneCidrMap[zone]
		}
		subnetArgs := &ec2.SubnetArgs{
			VpcId:            s.vpc.ID(),
			CidrBlock:        pulumi.String(cidr),
			AvailabilityZone: pulumi.String(zone),
			Tags: pulumi.StringMap{
				"Name":  pulumi.String(fmt.Sprintf("%s-private-subnet-%s", s.cluster.Name, getSubnetName(zone))),
				TAG_KEY: pulumi.String(TAG_VAL),
				"Zone":  pulumi.String(zone),
				"Type":  pulumi.String("private"),
			},
		}
		subnet, err := ec2.NewSubnet(ctx, getSubnetName(zone), subnetArgs)
		if err != nil {
			return nil, err
		}
		privateSubnets[i] = subnet
	}

	// public subnet
	pulicSubnet, err := ec2.NewSubnet(ctx, PUBLIC_SUBNET_STACK, &ec2.SubnetArgs{
		VpcId:     s.vpc.ID(),
		CidrBlock: pulumi.String(usEsubnetCidrs[len(usEsubnetCidrs)-1]),
		Tags: pulumi.StringMap{
			"Name":  pulumi.String(s.cluster.Name + "-public-subnet"),
			TAG_KEY: pulumi.String(TAG_VAL),
			"Type":  pulumi.String("public"),
		},
	})
	if err != nil {
		return nil, err
	}
	return &AwsGateWay{
		cluster:        s.cluster,
		vpc:            s.vpc,
		pulicSubnet:    pulicSubnet,
		privateSubnets: privateSubnets,
		zoneNames:      zones.Names,
	}, nil
}

type AwsGateWay struct {
	cluster        *biz.Cluster
	vpc            *ec2.Vpc
	pulicSubnet    *ec2.Subnet
	privateSubnets []*ec2.Subnet
	zoneNames      []string
}

func (g *AwsGateWay) startGateway(ctx *pulumi.Context) (*AwsRouteTable, error) {
	// Create an Internet Gateway
	interneteGateway, err := ec2.NewInternetGateway(ctx, INTERNETGATEWAY_STACK, &ec2.InternetGatewayArgs{
		VpcId: g.vpc.ID(),
		Tags: pulumi.StringMap{
			"Name":  pulumi.String(g.cluster.Name + "-igw"),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return nil, err
	}

	// create eip
	eip, err := ec2.NewEip(ctx, PUBLIC_NATGATEWAY_EIP_STACK, &ec2.EipArgs{
		Domain:             pulumi.String("vpc"),
		NetworkBorderGroup: pulumi.String(g.cluster.Region),
		PublicIpv4Pool:     pulumi.String("amazon"),
		Tags: pulumi.StringMap{
			"Name":  pulumi.String(fmt.Sprintf("%s-public-natgateway-eip", g.cluster.Name)),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return nil, err
	}

	publicNetGateway, err := ec2.NewNatGateway(ctx, PUBLIC_NATGATEWAY_STACK, &ec2.NatGatewayArgs{
		SubnetId:         g.pulicSubnet.ID(),
		AllocationId:     eip.ID(),
		ConnectivityType: pulumi.String("public"),
		Tags: pulumi.StringMap{
			"Name":  pulumi.String(g.cluster.Name + "-public-natgateway"),
			TAG_KEY: pulumi.String(TAG_VAL),
			"Type":  pulumi.String("public"),
		},
	})
	if err != nil {
		return nil, err
	}

	return &AwsRouteTable{
		cluster:          g.cluster,
		vpc:              g.vpc,
		pulicSubnet:      g.pulicSubnet,
		privateSubnets:   g.privateSubnets,
		igw:              interneteGateway,
		publicNatGateWay: publicNetGateway,
		zoneNames:        g.zoneNames,
	}, nil
}

type AwsRouteTable struct {
	cluster          *biz.Cluster
	vpc              *ec2.Vpc
	pulicSubnet      *ec2.Subnet
	privateSubnets   []*ec2.Subnet
	igw              *ec2.InternetGateway
	publicNatGateWay *ec2.NatGateway
	zoneNames        []string
}

func (r *AwsRouteTable) startRouteTable(ctx *pulumi.Context) (*AwsSecurityGroup, error) {
	// Create a route table with a route for the public nat gateway
	privateRouteTable, err := ec2.NewRouteTable(ctx, PUBLIC_NATGATEWAY_ROUTE_TABLE, &ec2.RouteTableArgs{
		VpcId: r.vpc.ID(),
		Routes: ec2.RouteTableRouteArray{
			&ec2.RouteTableRouteArgs{
				NatGatewayId: r.publicNatGateWay.ID(),
				CidrBlock:    pulumi.String("0.0.0.0/0"),
			},
		},
		Tags: pulumi.StringMap{
			"Name":  pulumi.String(fmt.Sprintf("%s-%s", r.cluster.Name, PUBLIC_NATGATEWAY_ROUTE_TABLE)),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return nil, err
	}
	for index, privateSubnet := range r.privateSubnets {
		_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-%s", PUBLIC_NATGATEWAY_ROUTE_TABLE_ASSOCIATION, getZoneName(r.zoneNames, index)), &ec2.RouteTableAssociationArgs{
			RouteTableId: privateRouteTable.ID(),
			SubnetId:     privateSubnet.ID(),
		})
		if err != nil {
			return nil, err
		}
	}

	// Create a route table and a route for the public subnet
	pulicRouteTable, err := ec2.NewRouteTable(ctx, PUBLIC__INTERNETGATEWAY_ROUTE_TABLE, &ec2.RouteTableArgs{
		VpcId: r.vpc.ID(),
		Routes: ec2.RouteTableRouteArray{
			&ec2.RouteTableRouteArgs{
				GatewayId: r.igw.ID(),
				CidrBlock: pulumi.String("0.0.0.0/0"),
			},
		},
		Tags: pulumi.StringMap{
			"Name":  pulumi.String(fmt.Sprintf("%s-%s", r.cluster.Name, PUBLIC__INTERNETGATEWAY_ROUTE_TABLE)),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = ec2.NewRouteTableAssociation(ctx, PUBLIC_INTERNETGATEWAY_ROUTE_TABLE_ASSOCIATION, &ec2.RouteTableAssociationArgs{
		RouteTableId: pulicRouteTable.ID(),
		SubnetId:     r.pulicSubnet.ID(),
	})
	if err != nil {
		return nil, err
	}
	return &AwsSecurityGroup{
		cluster:        r.cluster,
		vpc:            r.vpc,
		pulicSubnet:    r.pulicSubnet,
		privateSubnets: r.privateSubnets,
	}, nil
}

type AwsSecurityGroup struct {
	cluster        *biz.Cluster
	vpc            *ec2.Vpc
	pulicSubnet    *ec2.Subnet
	privateSubnets []*ec2.Subnet
}

func (s *AwsSecurityGroup) startSecurityGroup(ctx *pulumi.Context) (*AwsSlb, error) {
	// Security Group for Master and Worker nodes
	sg, err := ec2.NewSecurityGroup(ctx, SECURITY_GROUP_STACK, &ec2.SecurityGroupArgs{
		VpcId: s.vpc.ID(),
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
		return nil, err
	}
	return &AwsSlb{
		cluster:        s.cluster,
		sg:             sg,
		pulicSubnet:    s.pulicSubnet,
		privateSubnets: s.privateSubnets,
	}, nil
}

type AwsSlb struct {
	cluster        *biz.Cluster
	sg             *ec2.SecurityGroup
	pulicSubnet    *ec2.Subnet
	privateSubnets []*ec2.Subnet
}

func (s *AwsSlb) startSLB(_ *pulumi.Context) (*AwsIAM, error) {
	// todo
	return &AwsIAM{
		cluster:        s.cluster,
		sg:             s.sg,
		pulicSubnet:    s.pulicSubnet,
		privateSubnets: s.privateSubnets,
	}, nil
}

type AwsIAM struct {
	cluster        *biz.Cluster
	sg             *ec2.SecurityGroup
	pulicSubnet    *ec2.Subnet
	privateSubnets []*ec2.Subnet
}

func (a *AwsIAM) startIAM(ctx *pulumi.Context) (*AWSKey, error) {
	// IAM Role
	ec2Role, err := iam.NewRole(ctx, EC2_ROLE_STACK, &iam.RoleArgs{
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
			"Name":  pulumi.String(fmt.Sprintf("%s-ec2-role", a.cluster.Name)),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return nil, err
	}

	// IAM Role and Policy
	_, err = iam.NewRolePolicy(ctx, EC2_ROLE_POLICY_STACK, &iam.RolePolicyArgs{
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
		return nil, err
	}

	ec2Profile, err := iam.NewInstanceProfile(ctx, EC2_ROLE_PROFILE_STACK, &iam.InstanceProfileArgs{
		Role: ec2Role.Name,
		Tags: pulumi.StringMap{
			"Name":  pulumi.String(fmt.Sprintf("%s-ec2-profile", a.cluster.Name)),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return nil, err
	}
	return &AWSKey{
		cluster:        a.cluster,
		sg:             a.sg,
		pulicSubnet:    a.pulicSubnet,
		privateSubnets: a.privateSubnets,
		ec2Profile:     ec2Profile,
	}, nil
}

type AWSKey struct {
	cluster        *biz.Cluster
	sg             *ec2.SecurityGroup
	pulicSubnet    *ec2.Subnet
	privateSubnets []*ec2.Subnet
	ec2Profile     *iam.InstanceProfile
}

func (k *AWSKey) startSshKey(ctx *pulumi.Context) (*AwsAMI, error) {
	// key pair
	keyPair, err := ec2.NewKeyPair(ctx, KEY_PAIR_STACK, &ec2.KeyPairArgs{
		KeyName:   pulumi.String(fmt.Sprintf("%s-key-pair", k.cluster.Name)),
		PublicKey: pulumi.String(k.cluster.PublicKey),
		Tags: pulumi.StringMap{
			"Name":  pulumi.String(fmt.Sprintf("%s-key-pair", k.cluster.Name)),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return nil, err
	}
	return &AwsAMI{
		cluster:        k.cluster,
		sg:             k.sg,
		pulicbSubnet:   k.pulicSubnet,
		privateSubnets: k.privateSubnets,
		ec2Profile:     k.ec2Profile,
		keyPair:        keyPair,
	}, nil
}

type AwsAMI struct {
	cluster        *biz.Cluster
	sg             *ec2.SecurityGroup
	pulicbSubnet   *ec2.Subnet
	privateSubnets []*ec2.Subnet
	ec2Profile     *iam.InstanceProfile
	keyPair        *ec2.KeyPair
}

func (a *AwsAMI) findAmi(ctx *pulumi.Context) (*AwsBostionHost, error) {
	amiImageID := ""
	for _, nodegroup := range a.cluster.NodeGroups {
		if nodegroup.ImageID != "" {
			amiImageID = nodegroup.ImageID
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
			return nil, err
		}
		if ubuntuAmi == nil || ubuntuAmi.Id == "" {
			return nil, fmt.Errorf("unable to find ubuntu ami")
		}
		amiImageID = ubuntuAmi.Id
	}
	ctx.Export("ubuntuAmi", pulumi.String(amiImageID))
	for _, nodegroup := range a.cluster.NodeGroups {
		if nodegroup.ImageID == "" {
			nodegroup.ImageID = amiImageID
		}
	}
	a.cluster.BostionHost.ImageID = amiImageID
	return &AwsBostionHost{
		cluster:        a.cluster,
		sg:             a.sg,
		pulicbSubnet:   a.pulicbSubnet,
		privateSubnets: a.privateSubnets,
		ec2Profile:     a.ec2Profile,
		keyPair:        a.keyPair,
	}, nil
}

type AwsBostionHost struct {
	cluster        *biz.Cluster
	sg             *ec2.SecurityGroup
	pulicbSubnet   *ec2.Subnet
	privateSubnets []*ec2.Subnet
	ec2Profile     *iam.InstanceProfile
	keyPair        *ec2.KeyPair
}

func (b *AwsBostionHost) startBostionHost(ctx *pulumi.Context) (*AwsNode, error) {
	if b.cluster.BostionHost == nil {
		return &AwsNode{
			cluster:        b.cluster,
			privateSubnets: b.privateSubnets,
			sg:             b.sg,
			ec2Profile:     b.ec2Profile,
			keyPair:        b.keyPair,
		}, nil
	}
	if b.cluster.BostionHost.InstanceType == "" {
		// find suitable instance type
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
			},
		})
		if err != nil {
			return nil, err
		}
		// sort by vcpu and memory
		instanceTypeArr := make(GetInstanceTypeResults, 0)
		for _, instanceType := range bostionHostInstanceTypes.InstanceTypes {
			instanceTypeRes, err := ec2.GetInstanceType(ctx, &ec2.GetInstanceTypeArgs{InstanceType: instanceType})
			if err != nil {
				return nil, err
			}
			instanceTypeArr = append(instanceTypeArr, instanceTypeRes)
		}
		sort.Sort(instanceTypeArr)
		for _, instanceType := range instanceTypeArr {
			if instanceType.MemorySize == 0 {
				continue
			}
			memoryGBiSize := float64(instanceType.MemorySize) / 1024.0
			if memoryGBiSize >= b.cluster.BostionHost.Memory && instanceType.DefaultVcpus >= int(b.cluster.BostionHost.CPU) {
				b.cluster.BostionHost.InstanceType = instanceType.InstanceType
				break
			}
		}
	}
	// create eip
	eip, err := ec2.NewEip(ctx, BOSTIONHOST_EIP_STACK, &ec2.EipArgs{
		Domain:             pulumi.String("vpc"),
		NetworkBorderGroup: pulumi.String(b.cluster.Region),
		PublicIpv4Pool:     pulumi.String("amazon"),
		Tags: pulumi.StringMap{
			"Name":  pulumi.String(fmt.Sprintf("%s-bostionhost-eip", b.cluster.Name)),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return nil, err
	}
	// create bostion host
	bostionHostNodeNi, err := ec2.NewNetworkInterface(ctx, BOSTIONHOST_NETWORK_INTERFACE_STACK, &ec2.NetworkInterfaceArgs{
		SubnetId:       b.pulicbSubnet.ID(),
		SecurityGroups: pulumi.StringArray{b.sg.ID()},
		Tags: pulumi.StringMap{
			"Name":  pulumi.String(fmt.Sprintf("%s-bostionHost-ni", b.cluster.Name)),
			TAG_KEY: pulumi.String(TAG_VAL),
		},
	})
	if err != nil {
		return nil, err
	}
	bostionHost, err := ec2.NewInstance(ctx, BOSTIONHOST_STACK, &ec2.InstanceArgs{
		InstanceType: pulumi.String(b.cluster.BostionHost.InstanceType),
		NetworkInterfaces: ec2.InstanceNetworkInterfaceArray{&ec2.InstanceNetworkInterfaceArgs{
			NetworkInterfaceId: bostionHostNodeNi.ID(),
			DeviceIndex:        pulumi.Int(0),
		}},
		Ami:                pulumi.String(b.cluster.BostionHost.ImageID),
		IamInstanceProfile: b.ec2Profile.Name,
		KeyName:            b.keyPair.KeyName,
		Tags: pulumi.StringMap{
			"Name":     pulumi.String(fmt.Sprintf("%s-bostionHost", b.cluster.Name)),
			TAG_KEY:    pulumi.String(TAG_VAL),
			"NodeRole": pulumi.String("BostionHost"),
		},
	})
	if err != nil {
		return nil, err
	}
	_, err = ec2.NewEipAssociation(ctx, BOSTIONHOST_EIP_ASSOCIATION_STACK, &ec2.EipAssociationArgs{
		AllocationId:       eip.ID(),
		NetworkInterfaceId: bostionHostNodeNi.ID(),
	})
	if err != nil {
		return nil, err
	}
	ctx.Export(BOSTIONHOST_EIP, eip.PublicIp)
	ctx.Export(BOSTIONHOST_INSTANCE_ID, bostionHost.ID())
	ctx.Export(BOSTIONHOST_USERNAME, pulumi.String("ubuntu"))
	ctx.Export(BOSTIONHOST_PRIVATE_IP, bostionHost.PrivateIp)
	return &AwsNode{
		cluster:        b.cluster,
		privateSubnets: b.privateSubnets,
		sg:             b.sg,
		ec2Profile:     b.ec2Profile,
		keyPair:        b.keyPair,
	}, nil
}

type AwsNode struct {
	cluster        *biz.Cluster
	privateSubnets []*ec2.Subnet
	sg             *ec2.SecurityGroup
	ec2Profile     *iam.InstanceProfile
	keyPair        *ec2.KeyPair
}

func (n *AwsNode) startNodes(ctx *pulumi.Context) error {
	if len(n.cluster.Nodes) == 0 || len(n.cluster.NodeGroups) == 0 {
		return nil
	}
	for _, nodeGroup := range n.cluster.NodeGroups {
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
		ctx.Export(fmt.Sprintf("%s-instance-type", nodeGroup.Name), pulumi.String(nodeGroup.InstanceType))
		for index, node := range n.cluster.Nodes {
			if node.NodeGroupID != nodeGroup.ID {
				continue
			}
			subnet := distributeNodeSubnetsFunc(index, n.privateSubnets, n.cluster.Nodes)
			nodeNi, err := ec2.NewNetworkInterface(ctx, node.Name+"_NI", &ec2.NetworkInterfaceArgs{
				SubnetId:       subnet.ID(),
				SecurityGroups: pulumi.StringArray{n.sg.ID()},
				Tags: pulumi.StringMap{
					"Name":  pulumi.String(node.Name + "-ni"),
					TAG_KEY: pulumi.String(TAG_VAL),
				},
			})
			if err != nil {
				return err
			}
			nodeRes, err := ec2.NewInstance(ctx, node.Name, &ec2.InstanceArgs{
				InstanceType: pulumi.String(nodeGroup.InstanceType),
				NetworkInterfaces: ec2.InstanceNetworkInterfaceArray{&ec2.InstanceNetworkInterfaceArgs{
					NetworkInterfaceId: nodeNi.ID(),
					DeviceIndex:        pulumi.Int(0),
				}},
				Ami:                pulumi.String(nodeGroup.ImageID),
				IamInstanceProfile: n.ec2Profile.Name,
				KeyName:            n.keyPair.KeyName,
				RootBlockDevice: &ec2.InstanceRootBlockDeviceArgs{
					VolumeSize: pulumi.Int(nodeGroup.SystemDisk),
				},
				Tags: pulumi.StringMap{
					"Name":     pulumi.String(node.Name),
					TAG_KEY:    pulumi.String(TAG_VAL),
					"NodeRole": pulumi.String(node.Role),
				},
			})
			if err != nil {
				return err
			}
			ctx.Export(fmt.Sprintf("%s-instance-id", node.Name), nodeRes.ID())
			if nodeGroup.DataDisk > 0 {
				// Create an additional EBS volume
				volume, err := ebs.NewVolume(ctx, fmt.Sprintf("%s-data-volume", node.Name), &ebs.VolumeArgs{
					AvailabilityZone: nodeRes.AvailabilityZone,
					Size:             pulumi.Int(nodeGroup.DataDisk), // Set additional disk size to 100 GiB
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
		}
	}
	return nil
}

// 把现有的资源导入到 pulumi 中
func (a *AwsEc2Instance) Import(ctx *pulumi.Context) error {
	instances, err := ec2.GetInstances(ctx, &ec2.GetInstancesArgs{})
	if err != nil {
		return err
	}
	subnetIds := make([]string, 0)
	instanceTypes := make([]string, 0)
	instanceTypeOsMap := make(map[string]string)
	publikeys := make([]string, 0)
	sgids := make([]string, 0)
	publicIps := make([]string, 0)
	for _, instanceId := range instances.Ids {
		instance, err := ec2.LookupInstance(ctx, &ec2.LookupInstanceArgs{
			InstanceId: pulumi.StringRef(instanceId),
		})
		if err != nil {
			return err
		}
		var node *biz.Node
		for _, v := range a.cluster.Nodes {
			if v.InternalIP == instance.PrivateIp {
				node = v
				break
			}
		}
		if node == nil || node.Name == "" {
			continue
		}
		instanceTags := make(pulumi.StringMap)
		for k, v := range instance.Tags {
			instanceTags[k] = pulumi.String(v)
		}
		vpcSgIDs := make(pulumi.StringArray, 0)
		for _, sgID := range instance.VpcSecurityGroupIds {
			vpcSgIDs = append(vpcSgIDs, pulumi.String(sgID))
		}
		// pulumi import aws:ec2/instance:Instance myInstance i-092b8bf00cf03a72d --generate-code
		_, err = ec2.NewInstance(ctx, node.Name, &ec2.InstanceArgs{
			Ami:                 pulumi.String(instance.Ami),
			InstanceType:        pulumi.String(instance.InstanceType),
			KeyName:             pulumi.String(instance.KeyName),
			SubnetId:            pulumi.String(instance.SubnetId),
			Tags:                instanceTags,
			VpcSecurityGroupIds: vpcSgIDs,
		}, pulumi.Import(pulumi.ID(instanceId)))
		if err != nil {
			return err
		}
		// cluster
		publikeys = append(publikeys, instance.KeyName)
		sgids = append(sgids, instance.SecurityGroups...)
		publicIps = append(publicIps, instance.PublicIp)
		// nodegroup
		instanceTypes = append(instanceTypes, instance.InstanceType)
		instanceTypeOsMap[instance.InstanceType] = instance.Ami
		// node
		node.ClusterID = a.cluster.ID
		node.InstanceID = instanceId
		node.PublicKey = instance.KeyName
		tags, err := json.Marshal(instance.Tags)
		if err != nil {
			return err
		}
		node.Labels = string(tags)
		node.InternalIP = instance.PrivateIp
		node.ExternalIP = instance.PublicIp
		//  `pending`, `running`, `shutting-down`, `terminated`, `stopping`, `stopped`
		if instance.InstanceState == "running" {
			node.Status = biz.NodeStatusRunning
		} else {
			node.Status = biz.NodeStatusUnspecified
		}
		node.Zone = instance.AvailabilityZone
		node.SubnetId = instance.SubnetId
		subnetIds = append(subnetIds, instance.SubnetId)
		if node.NodeGroup == nil {
			node.NodeGroup = &biz.NodeGroup{InstanceType: instance.InstanceType}
		} else {
			node.NodeGroup.InstanceType = instance.InstanceType
		}
	}
	publikeys = utils.RemoveDuplicateString(publikeys)
	if len(publikeys) == 1 {
		a.cluster.PublicKey = publikeys[0]
	}
	sgids = utils.RemoveDuplicateString(sgids)
	a.cluster.SecurityGroupIDs = strings.Join(sgids, ",")
	publicIps = utils.RemoveDuplicateString(publicIps)
	if len(publicIps) == 1 {
		a.cluster.ExternalIP = publicIps[0]
	}
	// subnet
	for _, subnetID := range subnetIds {
		subnet, err := ec2.LookupSubnet(ctx, &ec2.LookupSubnetArgs{
			Id: pulumi.StringRef(subnetID),
		})
		if err != nil {
			return err
		}
		// import subnet pulumi import aws:ec2/subnet:Subnet mySubnet subnet-075eea802912b4a60 --generate-code
		tags := make(pulumi.StringMap)
		for k, v := range subnet.Tags {
			tags[k] = pulumi.String(v)
		}
		_, err = ec2.NewSubnet(ctx, "k8s-subnet-"+subnetID, &ec2.SubnetArgs{
			AvailabilityZone:               pulumi.String(subnet.AvailabilityZone),
			CidrBlock:                      pulumi.String(subnet.CidrBlock),
			MapPublicIpOnLaunch:            pulumi.Bool(subnet.MapPublicIpOnLaunch),
			PrivateDnsHostnameTypeOnLaunch: pulumi.String(subnet.PrivateDnsHostnameTypeOnLaunch),
			VpcId:                          pulumi.String(subnet.VpcId),
			Tags:                           tags,
		}, pulumi.Import(pulumi.ID(subnetID)))
		if err != nil {
			return err
		}
		if a.cluster.VpcID == "" {
			vpc, err := ec2.LookupVpc(ctx, &ec2.LookupVpcArgs{
				Id: pulumi.StringRef(subnet.VpcId),
			})
			if err != nil {
				return err
			}
			a.cluster.VpcID = subnet.VpcId
			a.cluster.VpcCidr = vpc.CidrBlock
		}
		for _, node := range a.cluster.Nodes {
			if node.SubnetId == subnetID {
				node.SubnetCidr = subnet.CidrBlock
			}
		}
	}
	// vpc
	vpc, err := ec2.LookupVpc(ctx, &ec2.LookupVpcArgs{})
	if err != nil {
		return err
	}
	// import vpc pulumi import aws:ec2/vpc:Vpc myVpc vpc-0483055d1fc806937 --generate-code
	tags := make(pulumi.StringMap)
	for k, v := range vpc.Tags {
		tags[k] = pulumi.String(v)
	}
	_, err = ec2.NewVpc(ctx, "k8s-vpc", &ec2.VpcArgs{
		CidrBlock:          pulumi.String(vpc.CidrBlock),
		EnableDnsHostnames: pulumi.Bool(vpc.EnableDnsHostnames),
		InstanceTenancy:    pulumi.String(vpc.InstanceTenancy),
		Tags:               tags,
	}, pulumi.Import(pulumi.ID(vpc.Id)))
	if err != nil {
		return err
	}
	nodeGroups := make([]*biz.NodeGroup, 0)
	for _, instanceType := range instanceTypes {
		ng := &biz.NodeGroup{}
		for _, v := range a.cluster.NodeGroups {
			if v.InstanceType == instanceType {
				ng = v
				break
			}
		}
		instanceTypeRes, err := ec2.GetInstanceType(ctx, &ec2.GetInstanceTypeArgs{InstanceType: instanceType})
		if err != nil {
			return err
		}
		ng.ClusterID = a.cluster.ID
		ng.InstanceType = instanceType
		ng.OSImage = instanceTypeOsMap[instanceType]
		ng.CPU = int32(instanceTypeRes.DefaultVcpus)
		ng.Memory = float64(instanceTypeRes.MemorySize)
		ng.DataDisk = int32(instanceTypeRes.TotalInstanceStorage)
		for _, gpues := range instanceTypeRes.Gpuses {
			ng.GPU += int32(gpues.Count)
			ng.GpuSpec = gpues.Name
		}
		nodeGroups = append(nodeGroups, ng)
	}
	a.cluster.NodeGroups = nodeGroups
	for _, node := range a.cluster.Nodes {
		for _, ng := range a.cluster.NodeGroups {
			if node.NodeGroup.InstanceType == ng.InstanceType {
				node.NodeGroup = ng
				node.NodeGroupID = ng.ID
				break
			}
		}
	}
	return nil
}

func (a *AwsEc2Instance) Clear(ctx *pulumi.Context) error {
	return nil
}

func getSubnetName(zone string) string {
	return fmt.Sprintf("%s%s", PRIVATE_SUBNET_STACK, zone)
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

func distributeNodeSubnetsFunc(nodeIndex int, subnets []*ec2.Subnet, nodes []*biz.Node) *ec2.Subnet {
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
