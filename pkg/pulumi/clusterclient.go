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

func (c *ClusterInfrastructure) alicloudServers(ctx context.Context, cluster *biz.Cluster, delete ...bool) error {
	args := AlicloudClusterArgs{
		Name:      cluster.Name,
		PublicKey: cluster.PublicKey,
		Nodes:     make([]AlicloudNodeArgs, 0),
	}
	for _, node := range cluster.Nodes {
		labels := make(map[string]string)
		if node.Labels != "" {
			err := json.Unmarshal([]byte(node.Labels), &labels)
			if err != nil {
				return err
			}
		}
		if node.NodeGroup == nil {
			return errors.New("node group is nil")
		}
		args.Nodes = append(args.Nodes, AlicloudNodeArgs{
			Name:                    node.Name,
			InstanceType:            node.NodeGroup.InstanceType,
			CPU:                     node.NodeGroup.CPU,
			Memory:                  node.NodeGroup.Memory,
			GPU:                     node.NodeGroup.GPU,
			GpuSpec:                 node.NodeGroup.GpuSpec,
			OSImage:                 node.NodeGroup.OSImage,
			InternetMaxBandwidthOut: node.NodeGroup.InternetMaxBandwidthOut,
			SystemDisk:              node.NodeGroup.SystemDisk,
			DataDisk:                node.NodeGroup.DataDisk,
			NodeInitScript:          node.NodeGroup.NodeInitScript,
			Labels:                  labels,
		})
	}
	var pulumiFunc PulumiFunc
	pulumiFunc = StartAlicloudCluster(args).StartServers
	if len(delete) > 0 {
		pulumiFunc = StartAlicloudCluster(args).Clear
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
