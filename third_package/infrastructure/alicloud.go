package infrastructure

import (
	"fmt"
	"os"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs "github.com/alibabacloud-go/ecs-20140526/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/f-rambo/ocean/internal/biz"
	pulumiAlb "github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/alb"
	pulumiEcs "github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/ecs"
	pulumiRam "github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/ram"
	pulumiResourceManager "github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/resourcemanager"
	pulumiVpc "github.com/pulumi/pulumi-alicloud/sdk/v3/go/alicloud/vpc"
)

type Alicloud struct {
	cluster       *biz.Cluster
	resourceGroup *pulumiResourceManager.ResourceGroup
	vpcNetWork    *pulumiVpc.Network
	vSwitchs      []*pulumiVpc.Switch
	sgs           []*pulumiEcs.SecurityGroup
	eipAddress    *pulumiEcs.EipAddress
	lb            *pulumiAlb.LoadBalancer
	role          *pulumiRam.Role
	natGateway    *pulumiVpc.NatGateway
	keyPair       *pulumiEcs.KeyPair
	endpoint      string
}

func NewAlicloud(cluster *biz.Cluster) *Alicloud {
	if cluster.Region == "" {
		cluster.Region = "cn-hangzhou"
	}
	endpoint := fmt.Sprintf("ecs-%s.aliyuncs.com", cluster.Region)
	os.Setenv("ALICLOUD_ACCESS_KEY", cluster.AccessID)
	os.Setenv("ALICLOUD_SECRET_KEY", cluster.AccessKey)
	os.Setenv("ALICLOUD_REGION", cluster.Region)
	os.Setenv("ALICLOUD_DEFAULT_REGION", cluster.Region)
	return &Alicloud{
		cluster:  cluster,
		endpoint: endpoint,
	}
}

func (a *Alicloud) GetRegions() (regions []string, err error) {
	describeRegionsRequest := &ecs.DescribeRegionsRequest{
		AcceptLanguage: tea.String("zh-CN"),
	}
	client, err := ecs.NewClient(&openapi.Config{
		AccessKeyId:     &a.cluster.AccessID,
		AccessKeySecret: &a.cluster.AccessKey,
		Endpoint:        &a.endpoint,
	})
	if err != nil {
		return nil, err
	}
	response, err := client.DescribeRegions(describeRegionsRequest)
	if err != nil {
		return nil, err
	}

	for _, region := range response.Body.Regions.Region {
		if region.RegionId == nil {
			continue
		}
		regions = append(regions, *region.RegionId)
	}
	return regions, nil
}

func (a *Alicloud) GetZones() (zones []string, err error) {
	describeZonesRequest := &ecs.DescribeZonesRequest{
		AcceptLanguage: tea.String("zh-CN"),
		RegionId:       tea.String(a.cluster.Region),
	}
	client, err := ecs.NewClient(&openapi.Config{
		AccessKeyId:     &a.cluster.AccessID,
		AccessKeySecret: &a.cluster.AccessKey,
		Endpoint:        &a.endpoint,
	})
	if err != nil {
		return nil, err
	}
	response, err := client.DescribeZones(describeZonesRequest)
	if err != nil {
		return nil, err
	}

	for _, zone := range response.Body.Zones.Zone {
		if zone.ZoneId == nil {
			continue
		}
		zones = append(zones, *zone.ZoneId)
	}
	return zones, nil
}
