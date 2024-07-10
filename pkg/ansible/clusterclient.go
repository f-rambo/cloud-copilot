package ansible

import (
	"context"
	"encoding/json"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
)

type ClusterConstruct struct {
	log *log.Helper
	c   *conf.Bootstrap
}

func NewClusterConstruct(c *conf.Bootstrap, logger log.Logger) biz.ClusterConstruct {
	return &ClusterConstruct{
		log: log.NewHelper(logger),
		c:   c,
	}
}

func (cc *ClusterConstruct) GenerateInitialCluster(ctx context.Context, cluster *biz.Cluster) error {

	return nil
}

func (cc *ClusterConstruct) GenerateNodeLables(ctx context.Context, cluster *biz.Cluster, nodeGroup *biz.NodeGroup) (lables string, err error) {
	lableMap := make(map[string]string)
	lableMap["cluster"] = cluster.Name
	lableMap["cluster_type"] = cluster.Type
	lableMap["region"] = cluster.Region
	lableMap["nodegroup"] = nodeGroup.Name
	lableMap["nodegroup_type"] = nodeGroup.Type
	lableMap["instance_type"] = nodeGroup.InstanceType
	lablebytes, err := json.Marshal(lableMap)
	if err != nil {
		return "", err
	}
	return string(lablebytes), nil
}

func (cc *ClusterConstruct) MigrateToBostionHost(ctx context.Context, cluster *biz.Cluster) error {
	oceanResource := cc.c.GetOceanResource()
	migratePlaybook := GetMigratePlaybook()
	databasePath := cc.c.GetOceanData().GetDBFilePath()
	pulumiPath := cc.c.GetOceanResource().GetPulumiPath()
	migratePlaybook.AddSynchronize("database", databasePath, databasePath)
	migratePlaybook.AddSynchronize("pulumi", pulumiPath, pulumiPath)
	migratePlaybookPath, err := SavePlaybook(oceanResource.GetClusterPath(), migratePlaybook)
	if err != nil {
		return err
	}
	args := &ansibleArgs{
		servers: []Server{
			{Ip: cluster.BostionHost.ExternalIP, Username: "root", ID: cluster.BostionHost.InstanceID, Role: "bostion"},
		},
	}
	err = cc.exec(ctx,
		cluster,
		oceanResource.GetClusterPath(),
		migratePlaybookPath,
		args,
	)
	if err != nil {
		return err
	}
	return nil
}

func (cc *ClusterConstruct) InstallCluster(ctx context.Context, cluster *biz.Cluster) error {
	serversInitPlaybook := GetServerInitPlaybook()
	serversInitPlaybookPath, err := SavePlaybook(cc.c.GetOceanResource().GetClusterPath(), serversInitPlaybook)
	if err != nil {
		return err
	}
	err = cc.exec(ctx, cluster, cc.c.GetOceanResource().GetClusterPath(), serversInitPlaybookPath, nil)
	if err != nil {
		return err
	}
	return cc.kubespray(ctx, cluster, GetClusterPlaybookPath(), nil)
}

func (cc *ClusterConstruct) UnInstallCluster(ctx context.Context, cluster *biz.Cluster) error {
	return cc.kubespray(ctx, cluster, GetResetPlaybookPath(), nil)
}

func (cc *ClusterConstruct) AddNodes(ctx context.Context, cluster *biz.Cluster, nodes []*biz.Node) error {
	for _, node := range nodes {
		log.Info("add node", "name", node.Name, "ip", node.ExternalIP, "role", node.Role)
	}
	return cc.kubespray(ctx, cluster, GetScalePlaybookPath(), nil)
}

func (cc *ClusterConstruct) RemoveNodes(ctx context.Context, cluster *biz.Cluster, nodes []*biz.Node) error {
	for _, node := range nodes {
		log.Info("remove node", "name", node.Name, "ip", node.ExternalIP, "role", node.Role)
		args := &ansibleArgs{
			env: map[string]string{"node": node.Name},
		}
		err := cc.kubespray(ctx, cluster, GetRemoveNodePlaybookPath(), args)
		if err != nil {
			return err
		}
	}
	return nil
}

type ansibleArgs struct {
	servers []Server
	env     map[string]string
}

func (cc *ClusterConstruct) kubespray(ctx context.Context, cluster *biz.Cluster, playbook string, args *ansibleArgs) error {
	oceanResource := cc.c.GetOceanResource()
	kubespray, err := NewKubespray(&oceanResource)
	if err != nil {
		return errors.Wrap(err, "new kubespray error")
	}
	return cc.exec(ctx, cluster, kubespray.GetPackagePath(), playbook, args)
}

func (cc *ClusterConstruct) exec(ctx context.Context, cluster *biz.Cluster, playbook string, cmdRunDir string, args *ansibleArgs) error {
	servers := make([]Server, 0)
	for _, node := range cluster.Nodes {
		servers = append(servers, Server{Ip: node.ExternalIP, Username: node.User, ID: cast.ToString(node.ID), Role: node.Role})
	}
	if args != nil && len(args.servers) > 0 {
		servers = args.servers
	}
	env := make(map[string]string)
	if args != nil && len(args.env) > 0 {
		env = args.env
	}
	g := new(errgroup.Group)
	ansibleLog := make(chan string, 1024)
	g.Go(func() error {
		defer close(ansibleLog)
		return NewGoAnsiblePkg(cc.c).
			SetAnsiblePlaybookBinary(cc.c.GetOceanResource().GetAnsibleCli()).
			SetLogChan(ansibleLog).
			SetServers(servers...).
			SetCmdRunDir(cmdRunDir).
			SetPlaybooks(playbook).
			SetEnvMap(env).
			ExecPlayBooks(ctx)
	})
	g.Go(func() error {
		for {
			select {
			case log, ok := <-ansibleLog:
				if !ok {
					return nil
				}
				cluster.Logs += log
				cc.log.Info(log)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
	err := g.Wait()
	if err != nil {
		return err
	}
	return nil
}
