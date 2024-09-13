package infrastructure

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awsEc2 "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/utils"
	"github.com/pkg/errors"
	pulumiec2 "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	pulumiIam "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
)

type AwsCloud struct {
	cluster          *biz.Cluster
	vpc              *pulumiec2.Vpc
	vpcCidrBlock     string
	pulicSubnet      *pulumiec2.Subnet
	privateSubnets   []*pulumiec2.Subnet
	zoneNames        []string
	igw              *pulumiec2.InternetGateway
	publicNatGateWay *pulumiec2.NatGateway
	sg               *pulumiec2.SecurityGroup
	ec2Profile       *pulumiIam.InstanceProfile
	keyPair          *pulumiec2.KeyPair
	eip              *pulumiec2.Eip
}

func NewAwsCloud(cluster *biz.Cluster) *AwsCloud {
	if cluster.Region == "" {
		cluster.Region = "us-east-1"
	}
	os.Setenv("AWS_REGION", cluster.Region)
	os.Setenv("AWS_DEFAULT_REGION", cluster.Region)
	os.Setenv("AWS_ACCESS_KEY_ID", cluster.AccessID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", cluster.AccessKey)
	return &AwsCloud{
		vpcCidrBlock: cluster.VpcCidr,
		cluster:      cluster,
	}
}

func (a *AwsCloud) GetRegions() (regions []string, err error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(a.cluster.Region),
	})
	if err != nil {
		return nil, err
	}
	svc := awsEc2.New(sess)
	result, err := svc.DescribeRegions(nil)
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

func (a *AwsCloud) GetZones() (zones []string, err error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(a.cluster.Region),
	})
	if err != nil {
		return nil, err
	}
	svc := awsEc2.New(sess)
	result, err := svc.DescribeAvailabilityZones(nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to describe availability zones")
	}
	for _, zone := range result.AvailabilityZones {
		if *zone.State != "available" {
			continue
		}
		zones = append(zones, *zone.ZoneName)
	}
	return zones, nil
}
