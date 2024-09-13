package infrastructure

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	awsEksClusterName   = "aws-eks-cluster"
	awsEksNodeGroupName = "aws-eks-node-group"
)

func (a *AwsInstance) StartEks(ctx *pulumi.Context) (err error) {
	var eksCluster *eks.Cluster
	subnetIds := make(pulumi.StringArray, 0)

	eksCLusterRes, err := eks.LookupCluster(ctx, &eks.LookupClusterArgs{
		Name: a.cluster.Name,
	})
	if err != nil {
		return fmt.Errorf("failed to get EKS cluster: %w", err)
	}
	if eksCLusterRes != nil && eksCLusterRes.Id != "" {
		// import
		a.cluster.CloudID = eksCLusterRes.Id
		a.cluster.VpcID = eksCLusterRes.VpcConfig.VpcId
		for _, v := range eksCLusterRes.VpcConfig.SubnetIds {
			subnetIds = append(subnetIds, pulumi.String(v))
		}
		sgIds := make(pulumi.StringArray, 0)
		for _, v := range eksCLusterRes.VpcConfig.SecurityGroupIds {
			sgIds = append(sgIds, pulumi.String(v))
		}
		eksCluster, err = eks.NewCluster(ctx, awsEksClusterName, &eks.ClusterArgs{
			Name:    pulumi.String(a.cluster.Name),
			RoleArn: pulumi.String(eksCLusterRes.Arn),
			VpcConfig: eks.ClusterVpcConfigArgs{
				VpcId:            pulumi.String(eksCLusterRes.VpcConfig.VpcId),
				SecurityGroupIds: sgIds,
				SubnetIds:        subnetIds,
			},
		}, pulumi.Import(pulumi.ID(eksCLusterRes.Id)))
		if err != nil {
			return fmt.Errorf("failed to create EKS cluster: %w", err)
		}
	} else {
		err = a.infrastructural(ctx)
		if err != nil {
			return err
		}
		// create eks cluster
		subnetIds := make(pulumi.StringArray, 0)
		for _, v := range a.privateSubnets {
			subnetIds = append(subnetIds, v.ID())
		}
		eksCluster, err = eks.NewCluster(ctx, awsEksClusterName, &eks.ClusterArgs{
			Name:    pulumi.String(a.cluster.Name),
			RoleArn: pulumi.StringInput(a.ec2Profile.Arn),
			VpcConfig: &eks.ClusterVpcConfigArgs{
				VpcId:             a.vpc.ID(),
				PublicAccessCidrs: pulumi.StringArray{pulumi.String(a.vpcCidrBlock)},
				SecurityGroupIds:  pulumi.StringArray{a.sg.ID()},
				SubnetIds:         subnetIds,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create EKS cluster: %w", err)
		}
	}

	// create node group
	for _, nodegroup := range a.cluster.NodeGroups {
		nodeGroupResult, err := eks.NewNodeGroup(ctx, fmt.Sprintf("%s-%s", awsEksClusterName, nodegroup.Name), &eks.NodeGroupArgs{
			ClusterName:   eksCluster.Name,
			NodeGroupName: pulumi.String(nodegroup.Name),
			NodeRoleArn:   pulumi.StringInput(a.ec2Profile.Arn),
			SubnetIds:     subnetIds,
			InstanceTypes: pulumi.StringArray{
				pulumi.String(nodegroup.InstanceType),
			},
			ScalingConfig: &eks.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(nodegroup.TargetSize),
				MaxSize:     pulumi.Int(nodegroup.MaxSize),
				MinSize:     pulumi.Int(nodegroup.MinSize),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create node group %s: %w", nodegroup.Name, err)
		}
		ctx.Export(getCloudNodeGroupID(nodegroup.Name), nodeGroupResult.ID())

	}
	ctx.Export(getClusterCloudID(), eksCluster.ID())
	ctx.Export(getConnections(), eksCluster.AccessConfig)
	ctx.Export(getCertificateAuthority(), eksCluster.CertificateAuthority)
	return nil
}
