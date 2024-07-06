package pulumi

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// https://docs.aws.amazon.com/zh_cn/eks/latest/userguide/what-is-eks.html

const (
	eksServicePolicyArn     = "arn:aws:iam::aws:policy/AmazonEKSServicePolicy"
	eksClusterPolicyArn     = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
	eksWorkerNodePolicyArn  = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
	eksCNIPolicyArn         = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
	ec2ContainerRegistryArn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"

	eksRoleArgsAssumeRolePolicy = `{
		"Version": "2008-10-17",
		"Statement": [{
		    "Sid": "",
		    "Effect": "Allow",
		    "Principal": {
			  "Service": "eks.amazonaws.com"
		    },
		    "Action": "sts:AssumeRole"
		}]
	  }`
	nodeGroupRoleArgsAssumeRolePolicy = `{
		"Version": "2012-10-17",
		"Statement": [{
		    "Sid": "",
		    "Effect": "Allow",
		    "Principal": {
			  "Service": "ec2.amazonaws.com"
		    },
		    "Action": "sts:AssumeRole"
		}]
	  }`

	eskClusterName             = "ocean-cluster"
	eskNodeGroupName           = "ocean-nodegroup"
	eskSecurityGroupName       = "ocean-cluster-sg"
	eskNodeDefaultInstanceType = "t3.medium"
	eskNodeMinSize             = 1
	eskNodeMaxSize             = 3
	eskNodeDesiredSize         = 2
	eskClusterRoleName         = "ocean-cluster-role"
	eskNodeGroupRoleName       = "ocean-nodegroup-role"
)

type ClusterNodeGroupArgs struct {
	ClusterName      string
	Region           string
	NodeGroupOptions []NodeGroupOptions
}

type NodeGroupOptions struct {
	Name         string
	InstanceType string
	DesiredSize  int
	MaxSize      int
	MinSize      int
}

type AwsEksCluster struct {
	clusterNodeGroupArgs ClusterNodeGroupArgs
}

func StartAwsEksCluster(clusterNodeGroupArgs ClusterNodeGroupArgs) func(*pulumi.Context) error {
	awsCluster := &AwsEksCluster{
		clusterNodeGroupArgs: clusterNodeGroupArgs,
	}
	return awsCluster.Start
}

func (a *AwsEksCluster) Start(ctx *pulumi.Context) error {
	// Read back the default VPC and public subnets, which we will use.
	vpcDefault := true
	vpcResult, err := ec2.LookupVpc(ctx, &ec2.LookupVpcArgs{Default: &vpcDefault})
	if err != nil {
		return err
	}
	if vpcResult == nil || vpcResult.Id == "" {
		return errors.New("VPC Not found")
	}

	subnetResult, err := ec2.GetSubnets(ctx, &ec2.GetSubnetsArgs{
		Filters: []ec2.GetSubnetsFilter{
			{Name: "vpc-id", Values: []string{vpcResult.Id}},
		},
	})
	if err != nil {
		return err
	}
	if subnetResult == nil || len(subnetResult.Ids) == 0 {
		return errors.New("No public subnets found in VPC")
	}

	// Create the EKS Cluster Role
	eksRoleResult, err := iam.NewRole(ctx, eskClusterRoleName, &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(eksRoleArgsAssumeRolePolicy),
	})
	if err != nil {
		return err
	}
	eksPolicies := []string{eksServicePolicyArn, eksClusterPolicyArn}
	for i, eksPolicy := range eksPolicies {
		_, err := iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("%s-rpa-%d", eskClusterRoleName, i), &iam.RolePolicyAttachmentArgs{
			PolicyArn: pulumi.String(eksPolicy),
			Role:      eksRoleResult.Name,
		})
		if err != nil {
			return err
		}
	}

	// Create the EC2 NodeGroup Role
	nodeGroupRole, err := iam.NewRole(ctx, eskNodeGroupRoleName, &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(nodeGroupRoleArgsAssumeRolePolicy),
	})
	if err != nil {
		return err
	}
	nodeGroupPolicies := []string{eksWorkerNodePolicyArn, eksCNIPolicyArn, ec2ContainerRegistryArn}
	for i, nodeGroupPolicy := range nodeGroupPolicies {
		_, err := iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("%s-ngpa-%d", eskNodeGroupRoleName, i), &iam.RolePolicyAttachmentArgs{
			Role:      nodeGroupRole.Name,
			PolicyArn: pulumi.String(nodeGroupPolicy),
		})
		if err != nil {
			return err
		}
	}

	// Create a Security Group that we can use to actually connect to our cluster
	clusterSg, err := ec2.NewSecurityGroup(ctx, eskSecurityGroupName, &ec2.SecurityGroupArgs{
		VpcId: pulumi.String(vpcResult.Id),
		Egress: ec2.SecurityGroupEgressArray{
			ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Ingress: ec2.SecurityGroupIngressArray{
			ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(80),
				ToPort:     pulumi.Int(80),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Description: pulumi.String("Managed by Ocean"),
	})
	if err != nil {
		return err
	}

	// Create EKS Cluster
	if a.clusterNodeGroupArgs.ClusterName == "" {
		a.clusterNodeGroupArgs.ClusterName = eskClusterName
	}
	eksCluster, err := eks.NewCluster(ctx, a.clusterNodeGroupArgs.ClusterName, &eks.ClusterArgs{
		RoleArn: pulumi.StringInput(eksRoleResult.Arn),
		VpcConfig: &eks.ClusterVpcConfigArgs{
			PublicAccessCidrs: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			SecurityGroupIds:  pulumi.StringArray{clusterSg.ID().ToStringOutput()},
			SubnetIds:         toPulumiStringArray(subnetResult.Ids),
		},
	})
	if err != nil {
		return err
	}

	// Create a NodeGroup for our cluster defalut : t3.medium instance type
	if len(a.clusterNodeGroupArgs.NodeGroupOptions) == 0 {
		a.clusterNodeGroupArgs.NodeGroupOptions = append(a.clusterNodeGroupArgs.NodeGroupOptions, NodeGroupOptions{
			Name:         eskNodeGroupName,
			InstanceType: eskNodeDefaultInstanceType,
			DesiredSize:  eskNodeDesiredSize,
			MaxSize:      eskNodeMaxSize,
			MinSize:      eskNodeMinSize,
		})
	}
	nodeGroupResults := make([]pulumi.Resource, 0)
	for _, nodeGroupOptions := range a.clusterNodeGroupArgs.NodeGroupOptions {
		instanceType := eskNodeDefaultInstanceType
		if nodeGroupOptions.InstanceType != "" {
			instanceType = nodeGroupOptions.InstanceType
		}
		nodeGroupResult, err := eks.NewNodeGroup(ctx, nodeGroupOptions.Name, &eks.NodeGroupArgs{
			ClusterName:   eksCluster.Name,
			NodeGroupName: pulumi.String(nodeGroupOptions.Name),
			NodeRoleArn:   pulumi.StringInput(nodeGroupRole.Arn),
			SubnetIds:     toPulumiStringArray(subnetResult.Ids),
			InstanceTypes: pulumi.StringArray{
				pulumi.String(instanceType),
			},
			ScalingConfig: &eks.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(nodeGroupOptions.DesiredSize),
				MaxSize:     pulumi.Int(nodeGroupOptions.MaxSize),
				MinSize:     pulumi.Int(nodeGroupOptions.MinSize),
			},
		})
		if err != nil {
			return err
		}
		nodeGroupResults = append(nodeGroupResults, nodeGroupResult)
	}

	kubeConfig := generateKubeconfig(eksCluster.Endpoint, eksCluster.CertificateAuthority.Data().Elem(), eksCluster.Name, eksCluster.Arn, pulumi.String(a.clusterNodeGroupArgs.Region).ToStringOutput())

	ctx.Export("kubeconfig", kubeConfig)
	ctx.Export("version", eksCluster.Version)

	// Create a Kubernetes provider for our cluster
	k8sProvider, err := kubernetes.NewProvider(ctx, "k8sprovider", &kubernetes.ProviderArgs{
		Kubeconfig: kubeConfig,
	}, pulumi.DependsOn(nodeGroupResults))
	if err != nil {
		return err
	}

	// Create a Kubernetes Namespace for our app
	namespace, err := corev1.NewNamespace(ctx, "ocean", &corev1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("ocean"),
		},
	}, pulumi.Provider(k8sProvider))
	if err != nil {
		return err
	}

	ctx.Export("namespace", namespace.Metadata.Name())

	// todo deploy ocean service

	return nil
}

func toPulumiStringArray(a []string) pulumi.StringArrayInput {
	var res []pulumi.StringInput
	for _, s := range a {
		res = append(res, pulumi.String(s))
	}
	return pulumi.StringArray(res)
}

// Create the KubeConfig Structure as per https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html
func generateKubeconfig(clusterEndpoint pulumi.StringOutput, certData pulumi.StringOutput, clusterName pulumi.StringOutput, arn pulumi.StringOutput, region pulumi.StringOutput) pulumi.StringOutput {
	return pulumi.Sprintf(`
apiVersion: v1
clusters:
- cluster:
   certificate-authority-data: %s
   server: %s
  name: %s
contexts:
- context:
   cluster: %s
   user: %s
   name: %s
current-context: %s
kind: Config
preferences: {}
users:
- name: %s
  user:
   exec:
    apiVersion: client.authentication.k8s.io/v1beta1
    args:
    - --region
    - %s
    - eks
    - get-token
    - --cluster-name
    - %s
    - --output
    - json
    command: aws
`, certData, clusterEndpoint, arn, arn, arn, arn, arn, arn, region, clusterName)
}

/*

	// Create a Kubernetes provider for our cluster
	k8sProvider, err := kubernetes.NewProvider(ctx, "k8sprovider", &kubernetes.ProviderArgs{
		Kubeconfig: generateKubeconfig(eksCluster.Endpoint,
			eksCluster.CertificateAuthority.Data().Elem(), eksCluster.Name),
	}, pulumi.DependsOn([]pulumi.Resource{nodeGroup}))
	if err != nil {
		return err
	}

	// Create a Kubernetes Namespace for our app
	namespace, err := corev1.NewNamespace(ctx, "app-ns", &corev1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("joe-duffy"),
		},
	}, pulumi.Provider(k8sProvider))
	if err != nil {
		return err
	}

*/
