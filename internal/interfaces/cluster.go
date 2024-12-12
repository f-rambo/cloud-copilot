package interfaces

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/f-rambo/cloud-copilot/api/cluster/v1alpha1"
	"github.com/f-rambo/cloud-copilot/api/common"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	sidecarCluster "github.com/f-rambo/cloud-copilot/internal/repository/sidecar/api/cluster"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/fsnotify/fsnotify"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ClusterInterface struct {
	v1alpha1.UnimplementedClusterInterfaceServer
	clusterUc *biz.ClusterUsecase
	userUc    *biz.UserUseCase
	c         *conf.Bootstrap
	log       *log.Helper
}

func NewClusterInterface(clusterUc *biz.ClusterUsecase, userUc *biz.UserUseCase, c *conf.Bootstrap, logger log.Logger) *ClusterInterface {
	return &ClusterInterface{
		clusterUc: clusterUc,
		userUc:    userUc,
		c:         c,
		log:       log.NewHelper(logger),
	}
}

func (c *ClusterInterface) Ping(ctx context.Context, _ *emptypb.Empty) (*common.Msg, error) {
	return common.Response(), nil
}

func (c *ClusterInterface) GetClusterTypes(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ClusterTypes, error) {
	clusterTypes := c.clusterUc.GetClusterTypes()
	clusterTypeResponse := &v1alpha1.ClusterTypes{ClusterTypes: make([]*v1alpha1.ClusterType, 0)}
	for _, clusterType := range clusterTypes {
		clusterTypeResponse.ClusterTypes = append(clusterTypeResponse.ClusterTypes, &v1alpha1.ClusterType{
			Id:      int32(clusterType),
			Name:    clusterType.String(),
			IsCloud: clusterType.IsCloud(),
		})
	}
	return clusterTypeResponse, nil
}

func (c *ClusterInterface) GetClusterStatuses(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ClusterStatuses, error) {
	clusterStatuses := c.clusterUc.GetClusterStatus()
	clusterStatusResponse := &v1alpha1.ClusterStatuses{ClusterStatuses: make([]*v1alpha1.ClusterStatus, 0)}
	for _, clusterStatus := range clusterStatuses {
		clusterStatusResponse.ClusterStatuses = append(clusterStatusResponse.ClusterStatuses, &v1alpha1.ClusterStatus{
			Id:   int32(clusterStatus),
			Name: clusterStatus.String(),
		})
	}
	return clusterStatusResponse, nil
}

func (c *ClusterInterface) GetClusterLevels(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ClusterLevels, error) {
	clusterLevels := c.clusterUc.GetClusterLevels()
	clusterLevelResponse := &v1alpha1.ClusterLevels{ClusterLevels: make([]*v1alpha1.ClusterLevel, 0)}
	for _, clusterLevel := range clusterLevels {
		clusterLevelResponse.ClusterLevels = append(clusterLevelResponse.ClusterLevels, &v1alpha1.ClusterLevel{
			Id:   int32(clusterLevel),
			Name: clusterLevel.String(),
		})
	}
	return clusterLevelResponse, nil
}

func (c *ClusterInterface) GetNodeRoles(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.NodeRoles, error) {
	nodeRoles := c.clusterUc.GetNodeRoles()
	nodeRoleResponse := &v1alpha1.NodeRoles{NodeRoles: make([]*v1alpha1.NodeRole, 0)}
	for _, nodeRole := range nodeRoles {
		nodeRoleResponse.NodeRoles = append(nodeRoleResponse.NodeRoles, &v1alpha1.NodeRole{
			Id:   int32(nodeRole),
			Name: nodeRole.String(),
		})
	}
	return nodeRoleResponse, nil
}

func (c *ClusterInterface) GetNodeStatuses(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.NodeStatuses, error) {
	nodeStatuses := c.clusterUc.GetNodeStatuses()
	nodeStatusResponse := &v1alpha1.NodeStatuses{NodeStatuses: make([]*v1alpha1.NodeStatus, 0)}
	for _, nodeStatus := range nodeStatuses {
		nodeStatusResponse.NodeStatuses = append(nodeStatusResponse.NodeStatuses, &v1alpha1.NodeStatus{
			Id:   int32(nodeStatus),
			Name: nodeStatus.String(),
		})
	}
	return nodeStatusResponse, nil
}

func (c *ClusterInterface) GetNodeGroupTypes(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.NodeGroupTypes, error) {
	nodeGroupTypes := c.clusterUc.GetNodeGroupTypes()
	nodeGroupTypeResponse := &v1alpha1.NodeGroupTypes{NodeGroupTypes: make([]*v1alpha1.NodeGroupType, 0)}
	for _, nodeGroupType := range nodeGroupTypes {
		nodeGroupTypeResponse.NodeGroupTypes = append(nodeGroupTypeResponse.NodeGroupTypes, &v1alpha1.NodeGroupType{
			Id:   int32(nodeGroupType),
			Name: nodeGroupType.String(),
		})
	}
	return nodeGroupTypeResponse, nil
}

func (c *ClusterInterface) GetResourceTypes(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ResourceTypes, error) {
	resourceTypes := c.clusterUc.GetResourceTypes()
	resourceTypeResponse := &v1alpha1.ResourceTypes{ResourceTypes: make([]*v1alpha1.ResourceType, 0)}
	for _, resourceType := range resourceTypes {
		resourceTypeResponse.ResourceTypes = append(resourceTypeResponse.ResourceTypes, &v1alpha1.ResourceType{
			Id:   int32(resourceType),
			Name: resourceType.String(),
		})
	}
	return resourceTypeResponse, nil
}

func (c *ClusterInterface) Get(ctx context.Context, clusterIdArgs *v1alpha1.ClusterIdMessge) (*v1alpha1.Cluster, error) {
	if clusterIdArgs.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	cluster, err := c.clusterUc.Get(ctx, clusterIdArgs.Id)
	if err != nil {
		return nil, err
	}
	data := &v1alpha1.Cluster{}
	if cluster == nil {
		return data, nil
	}
	return c.bizCLusterToCluster(cluster), nil
}
func (c *ClusterInterface) Save(ctx context.Context, clusterArgs *v1alpha1.ClusterSaveArgs) (*v1alpha1.ClusterIdMessge, error) {
	if clusterArgs.Name == "" || clusterArgs.PrivateKey == "" || clusterArgs.Type == 0 || clusterArgs.PublicKey == "" {
		return nil, errors.New("cluster name, private key, type and public key are required")
	}
	if biz.ClusterType(clusterArgs.Type) != biz.ClusterType_LOCAL {
		if clusterArgs.AccessId == "" || clusterArgs.AccessKey == "" {
			return nil, errors.New("access key id and secret access key are required")
		}
	}
	cluster := &biz.Cluster{}
	var err error
	if clusterArgs.Id != 0 {
		cluster, err = c.clusterUc.Get(ctx, clusterArgs.Id)
		if err != nil {
			return nil, err
		}
		if cluster == nil || cluster.Id == 0 {
			return nil, errors.New("cluster not found")
		}
		if cluster.Type.IsCloud() && clusterArgs.Region == "" {
			return nil, errors.New("region is required")
		}
	} else {
		cluster, err = c.clusterUc.GetByName(ctx, clusterArgs.Name)
		if err != nil {
			return nil, err
		}
		if cluster.Id != 0 {
			return nil, errors.New("cluster already exists")
		}
	}
	cluster.Name = clusterArgs.Name
	cluster.Type = biz.ClusterType(clusterArgs.Type)
	cluster.PublicKey = clusterArgs.PublicKey
	cluster.PrivateKey = clusterArgs.PrivateKey
	cluster.Region = clusterArgs.Region
	cluster.AccessId = clusterArgs.AccessId
	cluster.AccessKey = clusterArgs.AccessKey
	for _, nodeArgs := range clusterArgs.Nodes {
		if nodeArgs.Id == -1 {
			continue
		}
		if nodeArgs.Id == 0 {
			cluster.Nodes = append(cluster.Nodes, &biz.Node{
				InternalIp: nodeArgs.Ip,
				User:       nodeArgs.User,
				Role:       biz.NodeRole(nodeArgs.Role),
			})
			continue
		}
		for _, v := range cluster.Nodes {
			if v.Id == nodeArgs.Id {
				v.InternalIp = nodeArgs.Ip
				v.User = nodeArgs.User
				v.Role = biz.NodeRole(nodeArgs.Role)
				break
			}
		}
	}
	user, err := c.userUc.GetUserInfo(ctx)
	if err != nil {
		return nil, err
	}
	cluster.UserId = user.Id
	err = c.clusterUc.Save(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.ClusterIdMessge{Id: cluster.Id}, nil
}

func (c *ClusterInterface) Start(ctx context.Context, clusterArgs *v1alpha1.ClusterIdMessge) (*common.Msg, error) {
	if clusterArgs.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	cluster, err := c.clusterUc.Get(ctx, clusterArgs.Id)
	if err != nil {
		return nil, err
	}
	if cluster == nil || cluster.Id == 0 {
		return nil, errors.New("cluster not found")
	}
	if cluster.Status != biz.ClusterStatus_UNSPECIFIED && cluster.Status != biz.ClusterStatus_STOPPED {
		return nil, errors.New("cluster is not in stopped state")
	}
	cluster.Status = biz.ClusterStatus_STARTING
	err = c.clusterUc.Apply(cluster)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (c *ClusterInterface) Stop(ctx context.Context, clusterArgs *v1alpha1.ClusterIdMessge) (*common.Msg, error) {
	if clusterArgs.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	cluster, err := c.clusterUc.Get(ctx, clusterArgs.Id)
	if err != nil {
		return nil, err
	}
	if cluster == nil || cluster.Id == 0 {
		return nil, errors.New("cluster not found")
	}
	if cluster.Status != biz.ClusterStatus_UNSPECIFIED && cluster.Status != biz.ClusterStatus_RUNNING {
		return nil, errors.New("cluster is not in running state")
	}
	cluster.Status = biz.ClusterStatus_STOPPING
	err = c.clusterUc.Apply(cluster)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
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

func (c *ClusterInterface) Delete(ctx context.Context, clusterID *v1alpha1.ClusterIdMessge) (*common.Msg, error) {
	if clusterID.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	cluster, err := c.clusterUc.Get(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	if cluster == nil || cluster.Id == 0 {
		return nil, errors.New("cluster not found")
	}
	err = c.clusterUc.Delete(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

// get regions
func (c *ClusterInterface) GetRegions(ctx context.Context, clusterArgs *v1alpha1.ClusterIdMessge) (*v1alpha1.Regions, error) {
	if clusterArgs == nil || clusterArgs.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	cluster, err := c.clusterUc.Get(ctx, clusterArgs.Id)
	if err != nil {
		return nil, err
	}
	if cluster.Id == 0 {
		return nil, errors.New("cluster not found")
	}
	regionsCloudResources, err := c.clusterUc.GetRegions(ctx, cluster)
	if err != nil {
		return nil, err
	}
	regions := make([]*v1alpha1.Region, 0)
	for _, v := range regionsCloudResources {
		regions = append(regions, &v1alpha1.Region{
			Id:   v.RefId,
			Name: v.Name,
		})
	}
	return &v1alpha1.Regions{Regions: regions}, nil
}

// polling logs
func (c *ClusterInterface) PollingLogs(ctx context.Context, req *v1alpha1.ClusterLogsRequest) (*v1alpha1.ClusterLogsResponse, error) {
	if req.TailLines == 0 || req.TailLines > 30 {
		req.TailLines = 30
	}

	clusterLogPath, err := utils.GetLogFilePath(c.c.Server.Name)
	if err != nil {
		return nil, err
	}
	if ok := utils.IsFileExist(clusterLogPath); !ok {
		return nil, errors.New("cluster log does not exist")
	}
	if req.CurrentLine == 0 {
		file, err := os.Open(clusterLogPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		initialLogs, lastLine, err := utils.ReadLastNLines(file, int(req.TailLines))
		if err != nil {
			return nil, err
		}
		return &v1alpha1.ClusterLogsResponse{Logs: initialLogs, LastLine: int32(lastLine + 1)}, nil
	}
	logs, lastLine, err := utils.ReadFileFromLine(clusterLogPath, int64(req.CurrentLine))
	if err != nil {
		return nil, err
	}
	return &v1alpha1.ClusterLogsResponse{Logs: logs, LastLine: int32(lastLine + 1)}, nil
}

// get logs
func (c *ClusterInterface) GetLogs(stream v1alpha1.ClusterInterface_GetLogsServer) error {
	i := 0
	for {
		ctx, cancel := context.WithCancel(stream.Context())
		defer cancel()

		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if i > 0 {
			c.log.Info("repeat message, don't need to process")
			continue
		}
		i++
		if req.TailLines == 0 {
			req.TailLines = 30
		}

		clusterLogPath, err := utils.GetLogFilePath(c.c.Server.Name)
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

		// Read initial lines if TailLines is specified
		if req.TailLines > 0 {
			initialLogs, _, err := utils.ReadLastNLines(file, int(req.TailLines))
			if err != nil {
				return err
			}
			err = stream.Send(&v1alpha1.ClusterLogsResponse{Logs: initialLogs})
			if err != nil {
				return err
			}
		}

		// Move to the end of the file
		_, err = file.Seek(0, io.SeekEnd)
		if err != nil {
			return err
		}

		// Start watching for new logs
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		defer watcher.Close()

		err = watcher.Add(clusterLogPath)
		if err != nil {
			return err
		}

		sidecarLogContentChan := make(chan string)
		defer close(sidecarLogContentChan)
		cluster, err := c.clusterUc.Get(ctx, req.ClusterId)
		if err != nil {
			return err
		}
		if cluster != nil {
			for _, node := range cluster.Nodes {
				err = c.getSidecarLogContent(ctx, sidecarLogContentChan, node.InternalIp, 22)
				if err != nil {
					return err
				}
			}
		}

		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Op&fsnotify.Write == fsnotify.Write {
						newLogs, err := readNewLines(file)
						if err != nil {
							return
						}
						if newLogs != "" {
							err = stream.Send(&v1alpha1.ClusterLogsResponse{Logs: newLogs})
							if err != nil {
								return
							}
						}
					}
				case sidecarLogContent, ok := <-sidecarLogContentChan:
					if !ok {
						c.log.Info("Sidecar GetLogs stream closed by sidecar content")
						return
					}
					err = stream.Send(&v1alpha1.ClusterLogsResponse{Logs: sidecarLogContent})
					if err != nil {
						c.log.Errorf("Error sending sidecar log message: %v", err)
						return
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					c.log.Errorf("error watching log file: %v", err)
				case <-ctx.Done():
					c.log.Info("GetLogs stream closed by client")
					return
				}
			}
		}()
	}
}

func (c *ClusterInterface) getSidecarLogContent(ctx context.Context, contentChan chan string, nodeIp string, nodePort int32) error {
	conn, err := grpc.DialInsecure(ctx, grpc.WithEndpoint(fmt.Sprintf("%s:%d", nodeIp, nodePort)))
	if err != nil {
		return err
	}
	client := sidecarCluster.NewClusterInterfaceClient(conn)
	stream, err := client.GetLogs(ctx)
	if err != nil {
		return err
	}

	go func() {
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				c.log.Errorf("Error receiving sidecar log message: %v", err)
				return
			}
			contentChan <- msg.Log
		}
	}()

	err = stream.Send(&sidecarCluster.LogRequest{
		TailLines: 30,
	})
	if err != nil {
		return err
	}
	return nil
}

func readNewLines(file *os.File) (string, error) {
	currentPos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", err
	}

	newContent, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	if len(newContent) > 0 {
		_, err = file.Seek(currentPos+int64(len(newContent)), io.SeekStart)
		if err != nil {
			return "", err
		}
		return string(newContent), nil
	}

	return "", nil
}

func (c *ClusterInterface) bizCLusterToCluster(bizCluster *biz.Cluster) *v1alpha1.Cluster {
	nodes := make([]*v1alpha1.Node, 0)
	for _, v := range bizCluster.Nodes {
		if v == nil {
			continue
		}
		nodes = append(nodes, c.bizNodeToNode(v))
	}
	nodeGroups := make([]*v1alpha1.NodeGroup, 0)
	for _, v := range bizCluster.NodeGroups {
		if v == nil {
			continue
		}
		nodeGroups = append(nodeGroups, c.bizNodeGroupToNodeGroup(v))
	}
	var bostionHost *v1alpha1.BostionHost
	if bizCluster.BostionHost != nil && bizCluster.BostionHost.Id != "" {
		bostionHost = c.bizBostionHostToBostionHost(bizCluster.BostionHost)
	}
	return &v1alpha1.Cluster{
		Id:               bizCluster.Id,
		Name:             bizCluster.Name,
		Version:          bizCluster.Version,
		ApiServerAddress: bizCluster.ApiServerAddress,
		Status:           int32(bizCluster.Status),
		Type:             int32(bizCluster.Type),
		PublicKey:        bizCluster.PublicKey,
		PrivateKey:       bizCluster.PrivateKey,
		Region:           bizCluster.Region,
		AccessId:         bizCluster.AccessId,
		AccessKey:        bizCluster.AccessKey,
		CreateAt:         bizCluster.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdateAt:         bizCluster.UpdatedAt.Format("2006-01-02 15:04:05"),
		Nodes:            nodes,
		NodeGroups:       nodeGroups,
		BostionHost:      bostionHost,
	}
}

func (c *ClusterInterface) bizNodeToNode(node *biz.Node) *v1alpha1.Node {
	return &v1alpha1.Node{
		Id:         node.Id,
		Ip:         node.InternalIp,
		Name:       node.Name,
		Role:       int32(node.Role),
		User:       node.User,
		Status:     int32(node.Status),
		InstanceId: node.InstanceId,
		UpdateAt:   node.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func (c *ClusterInterface) bizNodeGroupToNodeGroup(nodeGroup *biz.NodeGroup) *v1alpha1.NodeGroup {
	return &v1alpha1.NodeGroup{
		Id:             nodeGroup.Id,
		Name:           nodeGroup.Name,
		Type:           int32(nodeGroup.Type),
		Os:             nodeGroup.Os,
		Arch:           nodeGroup.Arch.String(),
		Cpu:            nodeGroup.Cpu,
		Memory:         nodeGroup.Memory,
		Gpu:            nodeGroup.Gpu,
		GpuSpec:        nodeGroup.GpuSpec.String(),
		SystemDiskSize: nodeGroup.SystemDiskSize,
		DataDiskSize:   nodeGroup.DataDiskSize,
		MinSize:        nodeGroup.MinSize,
		MaxSize:        nodeGroup.MaxSize,
		TargetSize:     nodeGroup.TargetSize,
		UpdateAt:       nodeGroup.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func (c *ClusterInterface) bizBostionHostToBostionHost(bh *biz.BostionHost) *v1alpha1.BostionHost {
	return &v1alpha1.BostionHost{
		Id:         bh.Id,
		User:       bh.User,
		Os:         bh.Os,
		Arch:       bh.Arch.String(),
		Cpu:        bh.Cpu,
		Memory:     bh.Memory,
		Hostname:   bh.Hostname,
		ExternalIp: bh.ExternalIp,
		InternalIp: bh.InternalIp,
		SshPort:    bh.SshPort,
		Status:     int32(bh.Status),
		InstanceId: bh.InstanceId,
		UpdateAt:   bh.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
