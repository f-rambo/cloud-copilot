package infrastructure

import (
	"context"
	"fmt"
	"os"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs "github.com/alibabacloud-go/ecs-20140526/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
)

type Alicloud struct {
	cluster   *biz.Cluster
	ecsClient *ecs.Client
	log       *log.Helper
	conf      *conf.Bootstrap
}

const (
	alicloudDefaultRegion = "cn-hangzhou"
)

func NewAlicloud(cluster *biz.Cluster, log *log.Helper, conf *conf.Bootstrap) (*Alicloud, error) {
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
		conf:      conf,
	}, nil
}

func (a *Alicloud) GetAvailabilityZones() error {
	a.cluster.DeleteCloudResource(biz.ResourceTypeAvailabilityZones)
	describeRegionsRequest := &ecs.DescribeRegionsRequest{
		AcceptLanguage: tea.String("zh-CN"),
	}
	response, err := a.ecsClient.DescribeRegions(describeRegionsRequest)
	if err != nil {
		return err
	}
	for _, region := range response.Body.Regions.Region {
		if region.RegionId == nil {
			continue
		}

		if strings.ToLower(tea.StringValue(region.Status)) != "available" {
			continue
		}
		a.cluster.AddCloudResource(biz.ResourceTypeAvailabilityZones, &biz.CloudResource{
			ID:   tea.StringValue(region.RegionId),
			Name: tea.StringValue(region.LocalName),
		})
	}
	if len(a.cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones)) == 0 {
		return errors.New("no availability zones found")
	}
	return nil
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

func (a *Alicloud) CreateNetwork(ctx context.Context) error {
	fs := []func(context.Context) error{
		a.createVPC,
		a.createSubnets,
		a.createInternetGateway,
		a.createNatGateway,
		a.createRouteTables,
		a.createSecurityGroup,
		a.createSLB,
	}
	for _, f := range fs {
		if err := f(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (a *Alicloud) SetByNodeGroups(ctx context.Context) error {
	return nil
}

func (a *Alicloud) ImportKeyPair(ctx context.Context) error {
	return nil
}

func (a *Alicloud) DeleteKeyPair(ctx context.Context) error {
	return nil
}

func (a *Alicloud) ManageInstance(ctx context.Context) error {
	return nil
}

func (a *Alicloud) ManageBostionHost(ctx context.Context) error {
	return nil
}

func (a *Alicloud) DeleteNetwork(ctx context.Context) error {
	return nil
}

func (a *Alicloud) createVPC(ctx context.Context) error {
	return nil
}

func (a *Alicloud) createSubnets(ctx context.Context) error {
	return nil
}

func (a *Alicloud) createInternetGateway(ctx context.Context) error {
	return nil
}

func (a *Alicloud) createNatGateway(ctx context.Context) error {
	return nil
}

func (a *Alicloud) createRouteTables(ctx context.Context) error {
	return nil
}

func (a *Alicloud) createSecurityGroup(ctx context.Context) error {
	return nil
}

func (a *Alicloud) createSLB(ctx context.Context) error {
	return nil
}
