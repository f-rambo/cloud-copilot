package interfaces

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/f-rambo/ocean/api/cluster/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ClusterInterface struct {
	v1alpha1.UnimplementedClusterInterfaceServer
	clusterUc *biz.ClusterUsecase
	projectUc *biz.ProjectUsecase
	appUc     *biz.AppUsecase
	c         *conf.Bootstrap
	log       *log.Helper
}

func NewClusterInterface(clusterUc *biz.ClusterUsecase, projectUc *biz.ProjectUsecase, appUc *biz.AppUsecase, c *conf.Bootstrap, logger log.Logger) *ClusterInterface {
	return &ClusterInterface{
		clusterUc: clusterUc,
		projectUc: projectUc,
		appUc:     appUc,
		c:         c,
		log:       log.NewHelper(logger),
	}
}

func (uc *ClusterInterface) StartReconcile(ctx context.Context) (err error) {
	for {
		uc.log.Info("start watching reconcile...")
		cluster, err := uc.clusterUc.Watch(ctx)
		if err != nil {
			return err
		}
		if cluster == nil {
			continue
		}
		err = uc.clusterUc.Reconcile(ctx, cluster)
		if err != nil {
			return err
		}
	}
}

func (uc *ClusterInterface) StopReconcile(ctx context.Context) error {
	uc.log.Info("stop watching reconcile...")
	return nil
}

func (c *ClusterInterface) Ping(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.Msg, error) {
	if appInfo, ok := kratos.FromContext(ctx); ok {
		fmt.Println(appInfo.Metadata())
	}
	if md, ok := metadata.FromServerContext(ctx); ok {
		fmt.Println(md)
	}
	return &v1alpha1.Msg{Message: "pong"}, nil
}

func (c *ClusterInterface) Get(ctx context.Context, clusterID *v1alpha1.ClusterID) (*v1alpha1.Cluster, error) {
	cluster, err := c.clusterUc.Get(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	data := &v1alpha1.Cluster{}
	if cluster == nil {
		return data, nil
	}
	data = c.bizCLusterToCluster(cluster)
	return data, nil
}

func (c *ClusterInterface) Save(ctx context.Context, clusterArgs *v1alpha1.ClusterArgs) (*v1alpha1.Cluster, error) {
	if clusterArgs.Name == "" {
		return nil, errors.New("cluster name is required")
	}
	if clusterArgs.PublicKey == "" {
		return nil, errors.New("public key is required")
	}
	if clusterArgs.ServerType == "" {
		return nil, errors.New("server type is required")
	}
	if biz.ClusterType(clusterArgs.ServerType) != biz.ClusterTypeLocal {
		if clusterArgs.AccessKeyId == "" {
			return nil, errors.New("access key id is required")
		}
		if clusterArgs.SecretAccessKey == "" {
			return nil, errors.New("secret access key is required")
		}
		if clusterArgs.Region == "" {
			return nil, errors.New("region is required")
		}
	}
	cluster := &biz.Cluster{
		Name:      clusterArgs.Name,
		Type:      biz.ClusterType(clusterArgs.ServerType),
		PublicKey: clusterArgs.PublicKey,
		Region:    clusterArgs.Region,
		AccessID:  clusterArgs.AccessKeyId,
		AccessKey: clusterArgs.SecretAccessKey,
		Nodes:     make([]*biz.Node, 0),
	}
	if biz.ClusterType(clusterArgs.ServerType) == biz.ClusterTypeLocal {
		if len(clusterArgs.Nodes) == 0 {
			return nil, errors.New("at least one node is required")
		}
		for _, node := range clusterArgs.Nodes {
			cluster.Nodes = append(cluster.Nodes, &biz.Node{
				ExternalIP: node.Ip,
				User:       node.User,
			})
		}
	}
	err := c.clusterUc.Save(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return c.bizCLusterToCluster(cluster), nil
}

func (c *ClusterInterface) List(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ClusterList, error) {
	data := &v1alpha1.ClusterList{}
	clusters, err := c.clusterUc.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(clusters) == 0 {
		return data, nil
	}
	for _, v := range clusters {
		data.Clusters = append(data.Clusters, c.bizCLusterToCluster(v))
	}
	return data, nil
}

func (c *ClusterInterface) Delete(ctx context.Context, clusterID *v1alpha1.ClusterID) (*v1alpha1.Msg, error) {
	if clusterID.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	err := c.clusterUc.Delete(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

func (c *ClusterInterface) ImportResouce(ctx context.Context, clusterArgs *v1alpha1.ClusterID) (*v1alpha1.Msg, error) {
	if clusterArgs == nil || clusterArgs.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	cluster, err := c.clusterUc.Get(ctx, clusterArgs.Id)
	if err != nil {
		return nil, err
	}
	err = c.clusterUc.ImportResource(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

func (c *ClusterInterface) StartCluster(ctx context.Context, clusterID *v1alpha1.ClusterID) (*v1alpha1.Msg, error) {
	if clusterID == nil || clusterID.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	cluster, err := c.clusterUc.Get(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	err = c.clusterUc.Apply(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

// check bostion host data and resources
func (c *ClusterInterface) CheckBostionHost(ctx context.Context, req *v1alpha1.CheckBostionHostRequest) (*v1alpha1.Msg, error) {
	if req.Arch == "" {
		return nil, errors.New("arch is required")
	}
	if req.OceanVersion == "" {
		return nil, errors.New("ocean version is required")
	}
	if req.ShipVersion == "" {
		return nil, errors.New("ship version is required")
	}
	if req.OceanDataTarGzPackagePath == "" {
		return nil, errors.New("ocean data tar gz package path is required")
	}
	if req.OceanDataTarGzPackageSha256SumPath == "" {
		return nil, errors.New("ocean data tar gz package sha256sum path is required")
	}
	if req.OceanPath == "" {
		return nil, errors.New("ocean path is required")
	}
	if req.ShipPath == "" {
		return nil, errors.New("ship path is required")
	}
	if req.Arch != runtime.GOOS {
		return nil, errors.New("arch is wrong")
	}
	// check ocean data tar gz package
	if ok := utils.IsFileExist(req.OceanDataTarGzPackagePath); !ok {
		return nil, errors.New("ocean data tar gz package is not exist")
	}
	// check ship
	if ok := utils.IsFileExist(req.ShipPath); !ok {
		return nil, errors.New("ship is not exist")
	}
	// check ocean data tar gz package sha256sum
	output, err := exec.Command("sudo", "sha256sum", "-c", req.OceanDataTarGzPackageSha256SumPath).CombinedOutput()
	if err != nil {
		return nil, errors.New(string(output))
	}
	// check ocean
	if ok := utils.IsFileExist(req.OceanPath); !ok {
		return nil, errors.New("ocean is not exist")
	}
	return &v1alpha1.Msg{}, nil
}

// get logs
func (c *ClusterInterface) GetLogs(stream grpc.BidiStreamingServer[v1alpha1.ClusterLogsRequest, v1alpha1.ClusterLogsResponse]) error {
	var lastReadPos int64

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		reqJson, err := json.Marshal(req)
		if err != nil {
			return err
		}
		clusterLogReq := &v1alpha1.ClusterLogsRequest{}
		err = json.Unmarshal(reqJson, clusterLogReq)
		if err != nil {
			return err
		}
		clusterLogPath, err := utils.GetPackageStorePathByNames("log", fmt.Sprintf("cluster-%d.log", clusterLogReq.ClusterId))
		if err != nil {
			return err
		}
		if ok := utils.IsFileExist(clusterLogPath); !ok {
			return errors.New("cluster log does not exist")
		}

		file, err := os.Open(clusterLogPath)
		if err != nil {
			return err
		}
		defer file.Close()

		var logs string
		if lastReadPos == 0 {
			// Read the last 30 lines
			logs, err = utils.ReadLastNLines(file, 30)
			if err != nil {
				return err
			}
		} else {
			// Read from the last read position
			_, err = file.Seek(lastReadPos, io.SeekStart)
			if err != nil {
				return err
			}
			newLogs, err := io.ReadAll(file)
			if err != nil {
				return err
			}
			logs = string(newLogs)
		}

		// If logs are empty, send a "." character
		if logs == "" {
			logs = "."
		}

		err = stream.Send(&v1alpha1.ClusterLogsResponse{
			Logs: logs,
		})
		if err != nil {
			return err
		}

		lastReadPos, err = file.Seek(0, io.SeekEnd)
		if err != nil {
			return err
		}
	}
}

func (c *ClusterInterface) bizCLusterToCluster(bizCluster *biz.Cluster) *v1alpha1.Cluster {
	cluster := &v1alpha1.Cluster{
		Id:               bizCluster.ID,
		Name:             bizCluster.Name,
		ServerVersion:    bizCluster.Version,
		ApiServerAddress: bizCluster.ApiServerAddress,
		Config:           bizCluster.Config,
		Addons:           bizCluster.Addons,
	}
	for _, node := range bizCluster.Nodes {
		cluster.Nodes = append(cluster.Nodes, c.bizNodeToNode(node))
	}
	return cluster
}

func (c *ClusterInterface) bizNodeToNode(bizNode *biz.Node) *v1alpha1.Node {
	node := &v1alpha1.Node{
		Id:         bizNode.ID,
		Name:       bizNode.Name,
		Labels:     bizNode.Labels,
		Kernel:     bizNode.Kernel,
		Container:  bizNode.ContainerRuntime,
		Kubelet:    bizNode.Kubelet,
		KubeProxy:  bizNode.KubeProxy,
		InternalIp: bizNode.InternalIP,
		ExternalIp: bizNode.ExternalIP,
		User:       bizNode.User,
		Role:       bizNode.Role.String(),
		ClusterId:  bizNode.ClusterID,
	}
	return node
}
