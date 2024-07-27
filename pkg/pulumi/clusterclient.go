package pulumi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
)

type ClusterInfrastructure struct {
	log *log.Helper
	c   *conf.Bootstrap
}

func NewClusterInfrastructure(c *conf.Bootstrap, logger log.Logger) biz.Infrastructure {
	return &ClusterInfrastructure{
		log: log.NewHelper(logger),
		c:   c,
	}
}

// 在云厂商创建服务器
func (c *ClusterInfrastructure) SaveServers(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.GetType() == biz.ClusterTypeLocal {
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAliCloud {
		err := c.alicloudServers(ctx, cluster)
		if err != nil {
			return err
		}
	}
	if cluster.GetType() == biz.ClusterTypeAWS {
		return nil
	}
	return errors.New("not support cluster type")
}

// 删除云厂商服务器
func (c *ClusterInfrastructure) DeleteServers(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.GetType() == biz.ClusterTypeLocal {
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAliCloud {
		err := c.alicloudServers(ctx, cluster)
		if err != nil {
			return err
		}
	}
	if cluster.GetType() == biz.ClusterTypeAWS {
		return nil
	}
	return nil
}

// 创建阿里云服务器
func (c *ClusterInfrastructure) alicloudServers(ctx context.Context, cluster *biz.Cluster, delete ...bool) error {
	var pulumiFunc PulumiFunc
	pulumiFunc = StartAlicloudCluster(cluster).StartServers
	if len(delete) > 0 {
		pulumiFunc = StartAlicloudCluster(cluster).Clear
	}
	g := new(errgroup.Group)
	bostionHost := &biz.BostionHost{}
	pulumiOutput := make(chan string, 1024)
	pulumiOutput <- "starting alicloud servers..."
	g.Go(func() error {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("pulumi error: %s", err)
			}
			pulumiOutput <- "alicloud servers success"
			close(pulumiOutput)
		}()
		output, err := NewPulumiAPI(ctx, pulumiOutput).
			ProjectName(AlicloudProjectName).
			StackName(AlicloudStackName).
			Plugin(PulumiPlugin{Kind: "alicloud", Version: "3.56.0"}, PulumiPlugin{Kind: "kubernetes", Version: "4.12.0"}).
			Env(map[string]string{"ALICLOUD_ACCESS_KEY": cluster.AccessID, "ALICLOUD_SECRET_KEY": cluster.AccessKey, "ALICLOUD_REGION": cluster.Region}).
			RegisterDeployFunc(pulumiFunc).
			Up(ctx)
		if err != nil {
			return err
		}
		// 解析pulumi输出
		outputMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(output), &outputMap)
		if err != nil {
			return err
		}
		for k, v := range outputMap {
			switch k {
			case "vpc_id":
				cluster.VpcID = cast.ToString(v)
			case "external_ip":
				cluster.ExternalIP = cast.ToString(v)
				bostionHost.ExternalIP = cast.ToString(v)
			case "bostion_public_ip":
				bostionHost.PublicIP = cast.ToString(v)
			case "bostion_private_ip":
				bostionHost.PrivateIP = cast.ToString(v)
			case "bostion_id":
				bostionHost.InstanceID = cast.ToString(v)
			case "bostion_hostname":
				bostionHost.Hostname = cast.ToString(v)
			default:
				cluster.Logs += fmt.Sprintf("%s: %s\n", k, v)
			}
		}
		return nil
	})
	g.Go(func() error {
		for {
			select {
			case log, ok := <-pulumiOutput:
				if !ok {
					return nil
				}
				cluster.Logs += log
				c.log.Info(log)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
	cluster.BostionHost = bostionHost
	err := g.Wait()
	if err != nil {
		return err
	}
	return nil
}
