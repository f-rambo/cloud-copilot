package infrastructure

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	clusterv1alpha1 "github.com/f-rambo/ocean/api/cluster/v1alpha1"
	"github.com/f-rambo/ocean/api/common"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/metadata"
	mmd "github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

var ARCH_MAP = map[string]string{
	"x86_64":  "amd64",
	"aarch64": "arm64",
}

var (
	ServiceShell    string = "service.sh"
	SyncShell       string = "sync.sh"
	NodeInitShell   string = "nodeinit.sh"
	KubernetesShell string = "kubernetes.sh"
	systemInfoShell string = "systeminfo.sh"

	ClusterConfiguration        string = "cluster.yaml"
	NormalNodeJoinConfiguration string = "nodejoin.yaml"
	MasterNodeJoinConfiguration string = "masterjoin.yaml"
)

type ClusterInfrastructure struct {
	log  *log.Helper
	conf *conf.Bootstrap
}

func NewClusterInfrastructure(c *conf.Bootstrap, logger log.Logger) biz.ClusterInfrastructure {
	return &ClusterInfrastructure{
		log:  log.NewHelper(logger),
		conf: c,
	}
}

func (c *ClusterInfrastructure) GetRegions(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.Type.IsCloud() {
		return nil
	}
	if cluster.Type == biz.ClusterTypeAWSEc2 || cluster.Type == biz.ClusterTypeAWSEks {
		awsCloud, err := NewAwsCloud(ctx, cluster, c.conf, c.log)
		if err != nil {
			return err
		}
		return awsCloud.GetAvailabilityZones(ctx)
	}
	if cluster.Type == biz.ClusterTypeAliCloudEcs || cluster.Type == biz.ClusterTypeAliCloudAks {
		alicloud, err := NewAlicloud(cluster, c.log)
		if err != nil {
			return err
		}
		return alicloud.GetAvailabilityZones()
	}
	return errors.New("cluster type is not supported")
}

func (c *ClusterInfrastructure) Start(ctx context.Context, cluster *biz.Cluster) (err error) {
	if !cluster.Type.IsCloud() {
		return nil
	}
	if len(cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones)) == 0 {
		return errors.New("availability zones is empty")
	}
	if cluster.Type == biz.ClusterTypeAWSEc2 {
		awsCloud, err := NewAwsCloud(ctx, cluster, c.conf, c.log)
		if err != nil {
			return err
		}
		err = awsCloud.CreateNetwork(ctx)
		if err != nil {
			return err
		}
		err = awsCloud.SetByNodeGroups(ctx)
		if err != nil {
			return err
		}
		err = awsCloud.ImportKeyPair(ctx)
		if err != nil {
			return err
		}
		err = awsCloud.ManageInstance(ctx)
		if err != nil {
			return err
		}
		return awsCloud.ManageBostionHost(ctx)
	}
	return errors.New("cluster type is not supported")
}

func (c *ClusterInfrastructure) Stop(ctx context.Context, cluster *biz.Cluster) error {
	if !cluster.Type.IsCloud() {
		return nil
	}
	if cluster.Type == biz.ClusterTypeAWSEc2 {
		awsCloud, err := NewAwsCloud(ctx, cluster, c.conf, c.log)
		if err != nil {
			return err
		}
		err = awsCloud.ManageInstance(ctx)
		if err != nil {
			return err
		}
		err = awsCloud.ManageBostionHost(ctx)
		if err != nil {
			return err
		}
		err = awsCloud.DeleteKeyPair(ctx)
		if err != nil {
			return err
		}
		return awsCloud.DeleteNetwork(ctx)
	}
	return errors.New("cluster type is not supported")
}

func (cc *ClusterInfrastructure) MigrateToBostionHost(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.BostionHost.User == "" {
		return errors.New("bostion host username is empty")
	}
	if cluster.BostionHost.ExternalIP == "" {
		return errors.New("bostion host external ip is empty")
	}
	remoteBash := NewBash(Server{
		Name:       "bostion-host",
		Host:       cluster.BostionHost.ExternalIP,
		User:       cluster.BostionHost.User,
		Port:       cluster.BostionHost.SshPort,
		PrivateKey: cluster.PrivateKey,
	}, cc.log)
	stdout, err := remoteBash.Run("uname -m")
	if err != nil {
		return err
	}
	arch := strings.TrimSpace(stdout)
	if _, ok := ARCH_MAP[arch]; !ok {
		return errors.New("bostion host arch is not supported")
	}
	cluster.BostionHost.ARCH = ARCH_MAP[arch]
	shellPath, err := utils.GetFromContextByKey(ctx, utils.ShellKey)
	if err != nil {
		return err
	}
	resourcePath, err := utils.GetFromContextByKey(ctx, utils.ResourceKey)
	if err != nil {
		return err
	}
	installPath, err := utils.GetFromContextByKey(ctx, utils.InstallKey)
	if err != nil {
		return err
	}
	syncShellPath := utils.MergePath(shellPath, SyncShell)
	oceanHomePath, err := utils.GetPackageStorePathByNames()
	if err != nil {
		return err
	}
	err = cc.runCommandWithLogging("bash", syncShellPath,
		cluster.BostionHost.ExternalIP,
		cast.ToString(cluster.BostionHost.SshPort),
		cluster.BostionHost.User,
		cluster.PrivateKey,
		oceanHomePath,
		shellPath,
		resourcePath,
		installPath,
	)
	if err != nil {
		return err
	}
	serviceShellPath := utils.MergePath(shellPath, ServiceShell)
	err = remoteBash.RunWithLogging("bash", serviceShellPath, conf.EnvBostionHost.String())
	if err != nil {
		return err
	}
	conn, err := grpc.DialInsecure(ctx,
		grpc.WithEndpoint(fmt.Sprintf("%s:%d", cluster.BostionHost.ExternalIP, utils.GetPortByAddr(cc.conf.Server.GRPC.Addr))),
		grpc.WithMiddleware(mmd.Client()),
	)
	if err != nil {
		return err
	}
	defer conn.Close()
	client := clusterv1alpha1.NewClusterInterfaceClient(conn)
	appInfo := utils.GetFromContext(ctx)
	for k, v := range appInfo {
		ctx = metadata.AppendToClientContext(ctx, k, v)
	}
	msg, err := client.Ping(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	if msg.Reason != common.ErrorReason_SUCCEED {
		return errors.New(msg.Message)
	}
	return nil
}

func (cc *ClusterInfrastructure) Install(ctx context.Context, cluster *biz.Cluster) error {
	remoteBash := NewBash(Server{Name: cluster.Name, Host: cluster.MasterIP, User: cluster.MasterUser, Port: 22, PrivateKey: cluster.PrivateKey}, cc.log)
	shellPath, err := utils.GetFromContextByKey(ctx, utils.ShellKey)
	if err != nil {
		return err
	}
	err = remoteBash.RunWithLogging("bash", utils.MergePath(shellPath, NodeInitShell))
	if err != nil {
		return err
	}
	installPath, err := utils.GetFromContextByKey(ctx, utils.InstallKey)
	if err != nil {
		return err
	}
	clusterConfigData, err := utils.ReadFile(utils.MergePath(installPath, ClusterConfiguration))
	if err != nil {
		return err
	}
	clusterConfigMap := map[string]string{
		"CLUSTER_NAME":    cluster.Name,
		"CLUSTER_VERSION": cluster.Version,
		"MASTER_IP":       cluster.MasterIP,
		"IMAGE_REPO":      "",
	}
	clusterConfigDataStr := utils.DecodeYaml(string(clusterConfigData), clusterConfigMap)
	clusterConfigPath := fmt.Sprintf("$HOME/%s", ClusterConfiguration)
	err = remoteBash.RunWithLogging("echo", clusterConfigDataStr, ">", clusterConfigPath)
	if err != nil {
		return err
	}
	errGroup, _ := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		err = remoteBash.RunWithLogging("kubeadm init --config", clusterConfigPath, "--v=5")
		if err != nil {
			remoteBash.RunWithLogging("kubeadm reset --force")
			return err
		}
		return nil
	})
	errGroup.Go(func() error {
		return cc.restartKubelet(remoteBash)
	})
	err = errGroup.Wait()
	if err != nil {
		return err
	}
	err = remoteBash.RunWithLogging("rm -f $HOME/.kube/config && mkdir -p $HOME/.kube && cp -i /etc/kubernetes/admin.conf $HOME/.kube/config && chown $(id -u):$(id -g) $HOME/.kube/config")
	if err != nil {
		return err
	}
	token, err := remoteBash.Run("kubeadm token create")
	if err != nil {
		return err
	}
	cluster.Token = token
	ca, err := remoteBash.Run("cat /etc/kubernetes/pki/ca.crt")
	if err != nil {
		return err
	}
	cluster.CAData = ca
	cert, err := remoteBash.Run("cat /etc/kubernetes/pki/apiserver.crt")
	if err != nil {
		return err
	}
	cluster.CertData = cert
	key, err := remoteBash.Run("cat /etc/kubernetes/pki/apiserver.key")
	if err != nil {
		return err
	}
	cluster.KeyData = key
	return nil
}

func (cc *ClusterInfrastructure) UnInstall(ctx context.Context, cluster *biz.Cluster) error {
	for _, node := range cluster.Nodes {
		if node.Role != biz.NodeRoleWorker {
			continue
		}
		remoteBash := NewBash(Server{Name: node.Name, Host: node.InternalIP, User: node.User, Port: 22, PrivateKey: cluster.PrivateKey}, cc.log)
		err := cc.uninstallNode(remoteBash)
		if err != nil {
			return err
		}
		node.Status = biz.NodeStatusDeleted
	}
	for _, node := range cluster.Nodes {
		if node.Role != biz.NodeRoleMaster {
			continue
		}
		remoteBash := NewBash(Server{Name: node.Name, Host: node.InternalIP, User: node.User, Port: 22, PrivateKey: cluster.PrivateKey}, cc.log)
		err := cc.uninstallNode(remoteBash)
		if err != nil {
			return err
		}
		node.Status = biz.NodeStatusDeleted
	}
	return nil
}

func (cc *ClusterInfrastructure) HandlerNodes(ctx context.Context, cluster *biz.Cluster) error {
	installPath, err := utils.GetFromContextByKey(ctx, utils.InstallKey)
	if err != nil {
		return err
	}
	normalNodeJoinConfig, err := utils.ReadFile(utils.MergePath(installPath, NormalNodeJoinConfiguration))
	if err != nil {
		return err
	}
	masterNodeJoinConfig, err := utils.ReadFile(utils.MergePath(installPath, MasterNodeJoinConfiguration))
	if err != nil {
		return err
	}
	for _, node := range cluster.Nodes {
		remoteBash := NewBash(Server{Name: node.Name, Host: node.InternalIP, User: node.User, Port: 22, PrivateKey: cluster.PrivateKey}, cc.log)
		if node.Status == biz.NodeStatusCreating {
			var nodeJoinYamlData string
			var controlPlane string
			if node.Role == biz.NodeRoleWorker {
				nodeJoinYamlData = utils.DecodeYaml(string(normalNodeJoinConfig), map[string]string{})
			}
			if node.Role == biz.NodeRoleMaster {
				controlPlane = "--control-plane"
				nodeJoinYamlData = utils.DecodeYaml(string(masterNodeJoinConfig), map[string]string{})
			}
			if nodeJoinYamlData == "" {
				return errors.New("node join yaml data is empty")
			}
			err = remoteBash.RunWithLogging("echo", nodeJoinYamlData, "> $HOME/nodejoin.yaml")
			if err != nil {
				return err
			}
			remoteBashShell := fmt.Sprintf("kubeadm join --config $HOME/nodejoin.yaml %s", controlPlane)
			errGroup, _ := errgroup.WithContext(ctx)
			errGroup.Go(func() error {
				err = remoteBash.RunWithLogging(remoteBashShell)
				if err != nil {
					remoteBash.RunWithLogging("kubeadm reset --force")
					return err
				}
				return nil
			})
			errGroup.Go(func() error {
				return cc.restartKubelet(remoteBash)
			})
			err = errGroup.Wait()
			if err != nil {
				return err
			}
			node.Status = biz.NodeStatusRunning
		}
		if node.Status == biz.NodeStatusDeleting {
			err = cc.uninstallNode(remoteBash)
			if err != nil {
				return err
			}
			node.Status = biz.NodeStatusDeleted
		}
	}
	return nil
}

func (cc *ClusterInfrastructure) uninstallNode(remoteBash *Bash) error {
	err := remoteBash.RunWithLogging("kubeadm reset --force")
	if err != nil {
		return err
	}
	err = remoteBash.RunWithLogging("rm -rf $HOME/.kube && rm -rf /etc/kubernetes && rm -rf /etc/cni")
	if err != nil {
		return err
	}
	err = remoteBash.RunWithLogging("systemctl stop containerd && systemctl disable containerd && rm -rf /var/lib/containerd")
	if err != nil {
		return err
	}
	err = remoteBash.RunWithLogging("systemctl stop kubelet && systemctl disable kubelet")
	if err != nil {
		return err
	}
	return nil
}

func (cc *ClusterInfrastructure) restartKubelet(remoteBash *Bash) error {
	for {
		err := remoteBash.RunWithLogging("systemctl disable kubelet && systemctl stop kubelet")
		if err != nil {
			return err
		}
		time.Sleep(time.Second * 10)
		output, err := remoteBash.Run("ll /etc/kubernetes/kubelet.conf | wc -l")
		if err != nil {
			return err
		}
		if cast.ToInt(output) == 0 {
			continue
		}
		err = remoteBash.RunWithLogging("systemctl enable kubelet && systemctl restart kubelet")
		if err != nil {
			return err
		}
		return nil
	}
}

func (cc *ClusterInfrastructure) GetNodesSystemInfo(ctx context.Context, cluster *biz.Cluster) error {
	errGroup, _ := errgroup.WithContext(ctx)
	shellPath, err := utils.GetFromContextByKey(ctx, utils.ShellKey)
	if err != nil {
		return err
	}
	for _, node := range cluster.Nodes {
		if node.InternalIP == "" || node.User == "" {
			continue
		}
		nodegroup := cluster.NewNodeGroup()
		node := node
		errGroup.Go(func() error {
			remoteBash := NewBash(Server{Name: node.Name, Host: node.InternalIP, User: node.User, Port: 22, PrivateKey: cluster.PrivateKey}, cc.log)
			systemInfoOutput, err := remoteBash.Run("bash", utils.MergePath(shellPath, systemInfoShell))
			if err != nil {
				return err
			}
			systemInfoMap := make(map[string]any)
			if err := json.Unmarshal([]byte(systemInfoOutput), &systemInfoMap); err != nil {
				return err
			}
			for key, val := range systemInfoMap {
				switch key {
				case "os":
					nodegroup.OS = cast.ToString(val)
				case "arch":
					nodegroup.ARCH = cast.ToString(val)
				case "mem":
					nodegroup.Memory = cast.ToInt32(val)
				case "cpu":
					nodegroup.CPU = cast.ToInt32(val)
				case "gpu":
					nodegroup.GPU = cast.ToInt32(val)
				case "gpu_info":
					nodegroup.GpuSpec = cast.ToString(val)
				case "disk":
					nodegroup.DataDisk = cast.ToInt32(val)
				case "inner_ip":
					node.InternalIP = cast.ToString(val)
				}
			}
			cluster.GenerateNodeGroupName(nodegroup)
			exitsNodeGroup := cluster.GetNodeGroupByName(nodegroup.Name)
			if exitsNodeGroup == nil {
				cluster.NodeGroups = append(cluster.NodeGroups, nodegroup)
			} else {
				nodegroup.ID = exitsNodeGroup.ID
			}
			node.NodeGroupID = nodegroup.ID
			return nil
		})
	}
	err = errGroup.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (c *ClusterInfrastructure) Write(content []byte) (n int, err error) {
	c.log.Info(string(content))
	return len(content), nil
}

func (c *ClusterInfrastructure) runCommandWithLogging(command string, args ...string) error {
	c.log.Info("exec command: ", fmt.Sprintf("%s %s", command, strings.Join(args, " ")))
	cmd := exec.Command(command, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "failed to get stdout pipe")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "failed to get stderr pipe")
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start command")
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			c.log.Info(scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			c.log.Warn(scanner.Text())
		}
	}()

	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, "command failed")
	}

	return nil
}
