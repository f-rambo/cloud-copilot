package infrastructure

import (
	"fmt"
	"os"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs "github.com/alibabacloud-go/ecs-20140526/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
)

type Alicloud struct {
	cluster   *biz.Cluster
	ecsClient *ecs.Client
	log       *log.Helper
}

const (
	alicloudDefaultRegion = "cn-hangzhou"
)

func NewAlicloud(cluster *biz.Cluster, log *log.Helper) (*Alicloud, error) {
	if cluster.Region == "" {
		cluster.Region = alicloudDefaultRegion
	}
	endpoint := fmt.Sprintf("ecs-%s.aliyuncs.com", cluster.Region)
	os.Setenv("ALICLOUD_ACCESS_KEY", cluster.AccessID)
	os.Setenv("ALICLOUD_SECRET_KEY", cluster.AccessKey)
	os.Setenv("ALICLOUD_REGION", cluster.Region)
	os.Setenv("ALICLOUD_DEFAULT_REGION", cluster.Region)
	client, err := ecs.NewClient(&openapi.Config{
		AccessKeyId:     &cluster.AccessID,
		AccessKeySecret: &cluster.AccessKey,
		Endpoint:        &endpoint,
	})
	if err != nil {
		return nil, err
	}
	return &Alicloud{
		cluster:   cluster,
		ecsClient: client,
		log:       log,
	}, nil
}

func (a *Alicloud) GetRegions() (regions []string, err error) {
	describeRegionsRequest := &ecs.DescribeRegionsRequest{
		AcceptLanguage: tea.String("zh-CN"),
	}
	response, err := a.ecsClient.DescribeRegions(describeRegionsRequest)
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
	response, err := a.ecsClient.DescribeZones(describeZonesRequest)
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
