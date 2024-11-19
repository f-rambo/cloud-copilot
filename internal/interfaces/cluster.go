package interfaces

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/f-rambo/ocean/api/cluster/v1alpha1"
	"github.com/f-rambo/ocean/api/common"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	sidecarCluster "github.com/f-rambo/ocean/internal/repository/sidecar/api/cluster"
	"github.com/f-rambo/ocean/utils"
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

func (c *ClusterInterface) Get(ctx context.Context, clusterID *v1alpha1.ClusterArgs) (*v1alpha1.Cluster, error) {
	cluster, err := c.clusterUc.Get(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	data := &v1alpha1.Cluster{}
	if cluster == nil {
		return data, nil
	}
	return c.bizCLusterToCluster(cluster), nil
}
func (c *ClusterInterface) Save(ctx context.Context, clusterArgs *v1alpha1.ClusterArgs) (*v1alpha1.Cluster, error) {
	if err := c.validateClusterArgs(clusterArgs); err != nil {
		return nil, err
	}

	cluster, err := c.getOrCreateCluster(ctx, clusterArgs)
	if err != nil {
		return nil, err
	}

	c.updateClusterFromArgs(cluster, clusterArgs)
	user, _ := c.userUc.GetUserInfo(ctx)
	if user != nil {
		cluster.UserID = user.ID
	}
	err = c.clusterUc.Save(ctx, cluster)
	if err != nil {
		return nil, err
	}

	return c.bizCLusterToCluster(cluster), nil
}

func (c *ClusterInterface) validateClusterArgs(args *v1alpha1.ClusterArgs) error {
	if args.Name == "" {
		return errors.New("cluster name is required")
	}
	if args.PrivateKey == "" {
		return errors.New("private key is required")
	}
	if args.Type == "" {
		return errors.New("server type is required")
	}
	if biz.ClusterType(args.Type) != biz.ClusterTypeLocal {
		if args.AccessId == "" {
			return errors.New("access key id is required")
		}
		if args.AccessKey == "" {
			return errors.New("secret access key is required")
		}
		if args.PublicKey == "" {
			return errors.New("public key is required")
		}
	}
	if args.Id != 0 && args.Region == "" {
		return errors.New("region is required")
	}
	return nil
}

func (c *ClusterInterface) getOrCreateCluster(ctx context.Context, args *v1alpha1.ClusterArgs) (*biz.Cluster, error) {
	if args.Id != 0 {
		cluster, err := c.clusterUc.Get(ctx, args.Id)
		if err != nil {
			return nil, err
		}
		if cluster == nil {
			return nil, errors.New("cluster not found")
		}
		return cluster, nil
	}

	return &biz.Cluster{
		ID:         args.Id,
		Name:       args.Name,
		Type:       biz.ClusterType(args.Type),
		PublicKey:  args.PublicKey,
		PrivateKey: args.PrivateKey,
		Region:     args.Region,
		AccessID:   args.AccessId,
		AccessKey:  args.AccessKey,
		Nodes:      make([]*biz.Node, 0),
	}, nil
}

func (c *ClusterInterface) updateClusterFromArgs(cluster *biz.Cluster, args *v1alpha1.ClusterArgs) {
	cluster.Name = args.Name
	cluster.Type = biz.ClusterType(args.Type)
	cluster.PublicKey = args.PublicKey
	cluster.PrivateKey = args.PrivateKey
	cluster.Region = args.Region
	cluster.AccessID = args.AccessId
	cluster.AccessKey = args.AccessKey

	c.updateExistingNodes(cluster, args.Nodes)
	c.addNewNodes(cluster, args.Nodes)
}

func (c *ClusterInterface) updateExistingNodes(cluster *biz.Cluster, nodeArgs []*v1alpha1.NodeArgs) {
	for _, node := range cluster.Nodes {
		if node.ID == 0 {
			continue
		}
		for _, nodeArg := range nodeArgs {
			if node.ID == nodeArg.Id {
				node.InternalIP = nodeArg.Ip
				node.User = nodeArg.User
				node.Role = biz.NodeRole(nodeArg.Role)
			}
		}
	}
}

func (c *ClusterInterface) addNewNodes(cluster *biz.Cluster, nodeArgs []*v1alpha1.NodeArgs) {
	for _, nodeArg := range nodeArgs {
		if nodeArg.Id == 0 {
			cluster.Nodes = append(cluster.Nodes, &biz.Node{
				ID:         nodeArg.Id,
				InternalIP: nodeArg.Ip,
				User:       nodeArg.User,
				Role:       biz.NodeRole(nodeArg.Role),
			})
		}
	}
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

func (c *ClusterInterface) Delete(ctx context.Context, clusterID *v1alpha1.ClusterArgs) (*common.Msg, error) {
	if clusterID.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	err := c.clusterUc.Delete(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

// get regions
func (c *ClusterInterface) GetRegions(ctx context.Context, clusterID *v1alpha1.ClusterArgs) (*v1alpha1.Regions, error) {
	if clusterID == nil || clusterID.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	cluster, err := c.clusterUc.Get(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	regions, err := c.clusterUc.GetRegions(ctx, cluster)
	if err != nil {
		return nil, err
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

		// get ship logs
		shipLogContentChan := make(chan string)
		defer close(shipLogContentChan)
		cluster, err := c.clusterUc.Get(ctx, req.ClusterId)
		if err != nil {
			return err
		}
		if cluster != nil {
			for _, node := range cluster.Nodes {
				err = c.getShipLogContent(ctx, shipLogContentChan, node.InternalIP, 22)
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
				case shipLogContent, ok := <-shipLogContentChan:
					if !ok {
						c.log.Info("Ship GetLogs stream closed by ship content")
						return
					}
					err = stream.Send(&v1alpha1.ClusterLogsResponse{Logs: shipLogContent})
					if err != nil {
						c.log.Errorf("Error sending ship log message: %v", err)
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

func (c *ClusterInterface) getShipLogContent(ctx context.Context, contentChan chan string, nodeIp string, nodePort int32) error {
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
				c.log.Errorf("Error receiving ship log message: %v", err)
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
	if bizCluster.BostionHost != nil {
		bostionHost = c.bizBostionHostToBostionHost(bizCluster.BostionHost)
	}
	return &v1alpha1.Cluster{
		Id:                   bizCluster.ID,
		Name:                 bizCluster.Name,
		Connections:          bizCluster.Connections,
		CertificateAuthority: bizCluster.CertificateAuthority,
		Version:              bizCluster.Version,
		ApiServerAddress:     bizCluster.ApiServerAddress,
		Config:               bizCluster.Config,
		Addons:               bizCluster.Addons,
		AddonsConfig:         bizCluster.AddonsConfig,
		Status:               uint32(bizCluster.Status),
		Type:                 string(bizCluster.Type),
		PublicKey:            bizCluster.PublicKey,
		PrivateKey:           bizCluster.PrivateKey,
		Region:               bizCluster.Region,
		AccessId:             bizCluster.AccessID,
		AccessKey:            bizCluster.AccessKey,
		Nodes:                nodes,
		NodeGroups:           nodeGroups,
		BostionHost:          bostionHost,
		StatusString:         bizCluster.Status.String(),
	}
}

func (c *ClusterInterface) bizNodeToNode(bizNode *biz.Node) *v1alpha1.Node {
	return &v1alpha1.Node{
		Id:           bizNode.ID,
		Name:         bizNode.Name,
		Labels:       bizNode.Labels,
		InternalIp:   bizNode.InternalIP,
		ExternalIp:   bizNode.ExternalIP,
		User:         bizNode.User,
		Role:         string(bizNode.Role),
		Status:       uint32(bizNode.Status),
		ErrorInfo:    bizNode.ErrorInfo,
		ClusterId:    bizNode.ClusterID,
		NodeGroupId:  bizNode.NodeGroupID,
		StatusString: bizNode.Status.String(),
	}
}

func (c *ClusterInterface) bizNodeGroupToNodeGroup(bizNodeGroup *biz.NodeGroup) *v1alpha1.NodeGroup {
	return &v1alpha1.NodeGroup{
		Id:             bizNodeGroup.ID,
		Name:           bizNodeGroup.Name,
		Type:           string(bizNodeGroup.Type),
		Image:          bizNodeGroup.Image,
		Os:             bizNodeGroup.OS,
		Arch:           bizNodeGroup.ARCH,
		Cpu:            int32(bizNodeGroup.CPU),
		Memory:         float64(bizNodeGroup.Memory),
		Gpu:            int32(bizNodeGroup.GPU),
		NodeInitScript: bizNodeGroup.NodeInitScript,
		MinSize:        int32(bizNodeGroup.MinSize),
		MaxSize:        int32(bizNodeGroup.MaxSize),
		TargetSize:     int32(bizNodeGroup.TargetSize),
		ClusterId:      int64(bizNodeGroup.ClusterID),
	}
}

func (c *ClusterInterface) bizBostionHostToBostionHost(bizBostionHost *biz.BostionHost) *v1alpha1.BostionHost {
	return &v1alpha1.BostionHost{
		Id:         bizBostionHost.ID,
		User:       bizBostionHost.User,
		Image:      bizBostionHost.Image,
		Os:         bizBostionHost.OS,
		Arch:       bizBostionHost.ARCH,
		Hostname:   bizBostionHost.Hostname,
		ExternalIp: bizBostionHost.ExternalIP,
		InternalIp: bizBostionHost.InternalIP,
		SshPort:    int32(bizBostionHost.SshPort),
		ClusterId:  int64(bizBostionHost.ClusterID),
	}
}
