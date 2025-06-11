package data

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/lib"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type ClusterRepo struct {
	handlerClusterEvent func(ctx context.Context, cluster *biz.Cluster) error
	handlerLogs         func(ctx context.Context, key biz.LogType, msg string) error

	locks     map[int64]*sync.Mutex
	locksMux  sync.Mutex
	eventChan chan *biz.Cluster

	data *Data
	log  *log.Helper
}

func NewClusterRepo(data *Data, logger log.Logger) biz.ClusterData {
	c := &ClusterRepo{
		data:      data,
		log:       log.NewHelper(logger),
		locks:     make(map[int64]*sync.Mutex),
		locksMux:  sync.Mutex{},
		eventChan: make(chan *biz.Cluster, 1024),
	}
	data.registerRunner(c)
	return c
}

type FilebeatLog struct {
	Host struct {
		Name string `json:"name"`
	} `json:"host"`
	Agent struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		ID      string `json:"id"`
		Version string `json:"version"`
	} `json:"agent"`
	Message string `json:"message"`
	Log     struct {
		Offset int64 `json:"offset"`
		File   struct {
			Path string `json:"path"`
		} `json:"file"`
	} `json:"log"`
	Input struct {
		Type string `json:"type"`
	}
	Timestamp string `json:"@timestamp"`
}

type HubbleLog struct {
	Flow struct {
		Time     string `json:"time"`
		UUID     string `json:"uuid"`
		Verdict  string `json:"verdict"`
		Ethernet struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
		} `json:"ethernet"`
		IP struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
			IPVersion   string `json:"ipVersion"`
		} `json:"IP"`
		L4 struct {
			TCP struct {
				SourcePort      int `json:"source_port"`
				DestinationPort int `json:"destination_port"`
				Flags           struct {
					ACK bool `json:"ACK"`
				} `json:"flags"`
			} `json:"TCP"`
		} `json:"l4"`
		Source struct {
			ID          int      `json:"ID"`
			Identity    int      `json:"identity"`
			ClusterName string   `json:"cluster_name"`
			Namespace   string   `json:"namespace"`
			Labels      []string `json:"labels"`
			PodName     string   `json:"pod_name"`
		} `json:"source"`
		Destination struct {
			ID          int      `json:"ID"`
			Identity    int      `json:"identity"`
			ClusterName string   `json:"cluster_name"`
			Namespace   string   `json:"namespace"`
			Labels      []string `json:"labels"`
			PodName     string   `json:"pod_name"`
			Workloads   []struct {
				Name string `json:"name"`
				Kind string `json:"kind"`
			} `json:"workloads"`
		} `json:"destination"`
		Type       string   `json:"Type"`
		NodeName   string   `json:"node_name"`
		NodeLabels []string `json:"node_labels"`
		EventType  struct {
			Type int `json:"type"`
		} `json:"event_type"`
		TrafficDirection      string `json:"traffic_direction"`
		TraceObservationPoint string `json:"trace_observation_point"`
		TraceReason           string `json:"trace_reason"`
		IsReply               bool   `json:"is_reply"`
		Interface             struct {
			Index int    `json:"index"`
			Name  string `json:"name"`
		} `json:"interface"`
		Summary string `json:"Summary"`
	} `json:"flow"`
	NodeName string `json:"node_name"`
	Time     string `json:"time"`
}

func (c *ClusterRepo) RegisterHandlerClusterEvent(handler func(ctx context.Context, cluster *biz.Cluster) error) {
	c.handlerClusterEvent = handler
}

func (c *ClusterRepo) RegisterHandlerLogs(handler func(ctx context.Context, key biz.LogType, msg string) error) {
	c.handlerLogs = handler
}

func (c *ClusterRepo) getLock(clusterID int64) *sync.Mutex {
	c.locksMux.Lock()
	defer c.locksMux.Unlock()

	if clusterID < 0 {
		c.log.Errorf("Invalid clusterID: %d", clusterID)
		return &sync.Mutex{}
	}

	if _, exists := c.locks[clusterID]; !exists {
		c.locks[clusterID] = &sync.Mutex{}
	}
	return c.locks[clusterID]
}

func (c *ClusterRepo) Apply(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.IsEmpty() {
		return errors.New("invalid cluster")
	}
	select {
	case c.eventChan <- cluster:
		return nil
	default:
		return errors.New("cluster event channel is either full or closed")
	}
}

func (c *ClusterRepo) Start(ctx context.Context) error {
	if c.data.kafkaConsumer != nil {
		go func() {
			err := c.data.kafkaConsumer.ConsumeMessages(ctx, lib.NewDefaultMessageHandler(c.HandlerLogMsg))
			if err != nil {
				c.data.addRunnerError(err)
			}
		}()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case cluster, ok := <-c.eventChan:
			if !ok {
				return nil
			}
			c.getLock(cluster.Id).Lock()
			err := c.handlerClusterEvent(ctx, cluster)
			c.getLock(cluster.Id).Unlock()
			if err != nil {
				return err
			}
		}
	}
}

func (c *ClusterRepo) Stop(ctx context.Context) error {
	close(c.eventChan)
	return nil
}

func (c *ClusterRepo) getLogType(filebeatLog *FilebeatLog) biz.LogType {
	if filebeatLog == nil {
		return biz.LogType_UNSPECIFIED
	}
	if filebeatLog.Input.Type == "container" {
		return biz.LogType_POD
	}
	dirName := strings.Split(filebeatLog.Log.File.Path, "/")
	if slices.Contains(dirName, "hubble") {
		return biz.LogType_Trace
	}
	if slices.Contains(dirName, "service") {
		return biz.LogType_Service
	}
	return biz.LogType_UNSPECIFIED
}

func (c *ClusterRepo) HandlerLogMsg(ctx context.Context, _, val []byte) error {
	filebeatLog := &FilebeatLog{}
	err := json.Unmarshal(val, filebeatLog)
	if err != nil {
		return err
	}
	return c.handlerLogs(ctx, c.getLogType(filebeatLog), string(val))
}

func (c *ClusterRepo) CommitLogs(ctx context.Context, key biz.LogType, msg string) error {
	if key == biz.LogType_UNSPECIFIED {
		return errors.New("invalid log type")
	}
	if key == biz.LogType_POD {
		return c.commitPodLogs(ctx, msg)
	}
	if key == biz.LogType_Service {
		return c.commitServiceLogs(ctx, msg)
	}
	if key == biz.LogType_Trace {
		return c.commitTraceLogs(ctx, msg)
	}
	return nil

}

func (c *ClusterRepo) commitTraceLogs(_ context.Context, msg string) error {
	filebeatLog := &FilebeatLog{}
	err := json.Unmarshal([]byte(msg), filebeatLog)
	if err != nil {
		return err
	}
	hubbleLog := &HubbleLog{}
	err = json.Unmarshal([]byte(filebeatLog.Message), hubbleLog)
	if err != nil {
		return err
	}
	if hubbleLog.Flow.Verdict != "FORWARDED" || hubbleLog.Flow.IsReply {
		return nil
	}
	LablePrefix := "k8s:"
	var sourceServiceId int64 = 0
	for _, label := range hubbleLog.Flow.Source.Labels {
		if strings.HasPrefix(label, LablePrefix) {
			kv := strings.Split(label, "=")
			if len(kv) != 2 {
				continue
			}
			if kv[0] == "k8s:service_id" {
				sourceServiceId = cast.ToInt64(kv[1])
				break
			}
		}
	}
	var destinationServiceId int64 = 0
	for _, label := range hubbleLog.Flow.Destination.Labels {
		if strings.HasPrefix(label, LablePrefix) {
			kv := strings.Split(label, "=")
			if len(kv) != 2 {
				continue
			}
			if kv[0] == "k8s:service_id" {
				destinationServiceId = cast.ToInt64(kv[1])
				break
			}
		}
	}
	if sourceServiceId == 0 || destinationServiceId == 0 {
		return nil
	}
	trace := &biz.Trace{
		FromServiceId:   sourceServiceId,
		ToServiceId:     destinationServiceId,
		FromLabel:       strings.Join(hubbleLog.Flow.Source.Labels, ","),
		ToLabel:         strings.Join(hubbleLog.Flow.Destination.Labels, ","),
		RequestCount:    1,
		NodeName:        hubbleLog.Flow.NodeName,
		LastRequestTime: hubbleLog.Flow.Time,
	}
	dataTrace := &biz.Trace{}
	err = c.data.db.Model(&biz.Trace{}).Select("request_count").
		Where("from_service_id =? and to_service_id =?", sourceServiceId, destinationServiceId).First(&dataTrace).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.data.db.Model(&biz.Trace{}).Create(trace).Error
		}
		return err
	}
	trace.RequestCount += dataTrace.RequestCount
	return c.data.db.Model(&biz.Trace{}).Where("from_service_id =? and to_service_id =?", sourceServiceId, destinationServiceId).Updates(trace).Error
}

// /var/log/pods/toolkit_cert-manager-54f9c9456d-h6lvf_3f45aa70-b311-47a6-bd8b-8f5e938126c4/cert-manager-controller/0.log
func (c *ClusterRepo) commitPodLogs(ctx context.Context, msg string) error {
	if c.data.esClient == nil {
		return nil
	}
	filebeatLog := &FilebeatLog{}
	err := json.Unmarshal([]byte(msg), filebeatLog)
	if err != nil {
		return err
	}
	parts := strings.Split(filebeatLog.Log.File.Path, "/")
	if len(parts) < 5 {
		return nil
	}
	namespacePodName := parts[4]
	namespacePodNameParts := strings.Split(namespacePodName, "_")
	if len(namespacePodNameParts) < 2 {
		return nil
	}
	namespace := namespacePodNameParts[0]
	podName := namespacePodNameParts[1]
	return c.data.esClient.IndexDocument(ctx, c.data.esClient.GetIndexWrite(PodLogIndexName),
		map[string]any{
			"message":   filebeatLog.Message,
			"host":      filebeatLog.Host.Name,
			"pod":       podName,
			"namespace": namespace,
		})
}

// /var/log/service/workspace_project_service_id/log.log
func (c *ClusterRepo) commitServiceLogs(ctx context.Context, msg string) error {
	if c.data.esClient == nil {
		return nil
	}
	filebeatLog := &FilebeatLog{}
	err := json.Unmarshal([]byte(msg), filebeatLog)
	if err != nil {
		return err
	}
	parts := strings.Split(filebeatLog.Log.File.Path, "/")
	if len(parts) < 5 {
		return nil
	}
	serviceInfoParts := strings.Split(parts[4], "_")
	if len(serviceInfoParts) < 4 {
		return nil
	}
	serviceId := serviceInfoParts[3]
	if serviceId == "" {
		return nil
	}
	serviceName := serviceInfoParts[2]
	if serviceName == "" {
		return nil
	}
	projectName := serviceInfoParts[1]
	if projectName == "" {
		return nil
	}
	workspaceName := serviceInfoParts[0]
	if workspaceName == "" {
		return nil
	}
	msgMap := make(map[string]any)
	err = json.Unmarshal([]byte(filebeatLog.Message), &msgMap)
	if err != nil {
		return err
	}
	msgMap[HostKeyWord] = filebeatLog.Host.Name
	msgMap[ServiceIdKeyWord] = serviceId
	msgMap[ServiceNameKeyWord] = serviceName
	msgMap[ProjectKeyWord] = projectName
	msgMap[WorkspaceKeyWord] = workspaceName
	return c.data.esClient.IndexDocument(ctx, c.data.esClient.GetIndexWrite(ServiceLogIndexName), msgMap)
}

func (c *ClusterRepo) Save(ctx context.Context, cluster *biz.Cluster) (err error) {
	tx := c.data.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
	}()
	if cluster.Id == 0 {
		err = tx.Model(&biz.Cluster{}).Create(cluster).Error
	} else {
		err = tx.Model(&biz.Cluster{}).Where("id =?", cluster.Id).Updates(cluster).Error
	}
	if err != nil {
		return err
	}
	funcs := []func(context.Context, *biz.Cluster, *gorm.DB) error{
		c.saveNodeGroup,
		c.saveNode,
		c.saveCloudResources,
		c.saveSecuritys,
		c.saveDisk,
	}
	for _, f := range funcs {
		getErr := f(ctx, cluster, tx)
		if getErr != nil {
			return getErr
		}
	}
	err = tx.Commit().Error
	if err != nil {
		return err
	}
	return nil
}

func (c *ClusterRepo) Get(ctx context.Context, id int64) (*biz.Cluster, error) {
	cluster := &biz.Cluster{}
	err := c.data.db.Model(&biz.Cluster{}).Where("id = ?", id).First(cluster).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if cluster.Id == 0 {
		return nil, nil
	}
	nodeGroups := make([]*biz.NodeGroup, 0)
	err = c.data.db.Model(&biz.NodeGroup{}).Where("cluster_id = ?", cluster.Id).Find(&nodeGroups).Error
	if err != nil {
		return nil, err
	}
	if len(nodeGroups) != 0 {
		cluster.NodeGroups = nodeGroups
	}
	nodes := make([]*biz.Node, 0)
	err = c.data.db.Model(&biz.Node{}).Where("cluster_id = ?", cluster.Id).Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	if len(nodes) != 0 {
		cluster.Nodes = nodes
	}
	cloudResources := make([]*biz.CloudResource, 0)
	err = c.data.db.Model(&biz.CloudResource{}).Where("cluster_id = ?", cluster.Id).Find(&cloudResources).Error
	if err != nil {
		return nil, err
	}
	if len(cloudResources) != 0 {
		cluster.CloudResources = cloudResources
	}
	securitys := make([]*biz.Security, 0)
	err = c.data.db.Model(&biz.Security{}).Where("cluster_id = ?", cluster.Id).Find(&securitys).Error
	if err != nil {
		return nil, err
	}
	if len(securitys) != 0 {
		cluster.Securitys = securitys
	}
	disks := make([]*biz.Disk, 0)
	err = c.data.db.Model(&biz.Disk{}).Where("cluster_id =?", cluster.Id).Find(&disks).Error
	if err != nil {
		return nil, err
	}
	if len(disks) != 0 {
		for _, node := range nodes {
			node.Disks = make([]*biz.Disk, 0)
			for _, disk := range disks {
				if disk.NodeId == node.Id {
					node.Disks = append(node.Disks, disk)
				}
			}
		}
	}
	return cluster, nil
}

func (c *ClusterRepo) GetByName(ctx context.Context, name string) (*biz.Cluster, error) {
	cluster := &biz.Cluster{}
	err := c.data.db.Model(&biz.Cluster{}).Where("name = ?", name).First(cluster).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if cluster.Id != 0 {
		return c.Get(ctx, cluster.Id)
	}
	return cluster, nil
}

func (c *ClusterRepo) GetClustersByIds(ctx context.Context, ids []int64) ([]*biz.Cluster, error) {
	clusters := make([]*biz.Cluster, 0)
	err := c.data.db.Model(&biz.Cluster{}).Where("id IN ?", ids).Find(&clusters).Error
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

func (c *ClusterRepo) List(ctx context.Context, name string, page, pageSize int32) ([]*biz.Cluster, int64, error) {
	var clusters []*biz.Cluster
	var total int64

	query := c.data.db.Model(&biz.Cluster{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(int(offset)).Limit(int(pageSize)).Find(&clusters).Error
	if err != nil {
		return nil, 0, err
	}

	return clusters, total, nil
}

func (c *ClusterRepo) Delete(ctx context.Context, id int64) (err error) {
	tx := c.data.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	err = tx.Model(&biz.Cluster{}).Where("id = ?", id).Delete(&biz.Cluster{}).Error
	if err != nil {
		return err
	}
	err = tx.Model(&biz.Node{}).Where("cluster_id = ?", id).Delete(&biz.Node{}).Error
	if err != nil {
		return err
	}
	err = tx.Model(&biz.NodeGroup{}).Where("cluster_id = ?", id).Delete(&biz.NodeGroup{}).Error
	if err != nil {
		return err
	}
	err = tx.Model(&biz.CloudResource{}).Where("cluster_id = ?", id).Delete(&biz.CloudResource{}).Error
	if err != nil {
		return err
	}
	err = tx.Model(&biz.Security{}).Where("cluster_id = ?", id).Delete(&biz.Security{}).Error
	if err != nil {
		return err
	}
	err = tx.Model(&biz.Disk{}).Where("cluster_id =?", id).Delete(&biz.Disk{}).Error
	if err != nil {
		return err
	}
	return tx.Commit().Error
}

func (c *ClusterRepo) saveNodeGroup(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	for _, nodeGroup := range cluster.NodeGroups {
		nodeGroup.ClusterId = cluster.Id
		err := tx.Model(&biz.NodeGroup{}).Where("id = ?", nodeGroup.Id).Save(nodeGroup).Error
		if err != nil {
			return err
		}
	}
	nodeGroups := make([]*biz.NodeGroup, 0)
	err := tx.Model(&biz.NodeGroup{}).Where("cluster_id = ?", cluster.Id).Find(&nodeGroups).Error
	if err != nil {
		return err
	}
	for _, nodeGroup := range nodeGroups {
		ok := false
		for _, nodeGroup2 := range cluster.NodeGroups {
			if nodeGroup.Id == nodeGroup2.Id {
				ok = true
				break
			}
		}
		if !ok {
			err = tx.Model(&biz.NodeGroup{}).Where("id = ?", nodeGroup.Id).Delete(nodeGroup).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *ClusterRepo) saveNode(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	for _, node := range cluster.Nodes {
		node.ClusterId = cluster.Id
		err := tx.Model(&biz.Node{}).Where("id = ?", node.Id).Save(node).Error
		if err != nil {
			return err
		}
	}
	nodes := make([]*biz.Node, 0)
	err := tx.Model(&biz.Node{}).Where("cluster_id = ?", cluster.Id).Find(&nodes).Error
	if err != nil {
		return err
	}
	for _, node := range nodes {
		ok := false
		for _, node2 := range cluster.Nodes {
			if node.Id == node2.Id {
				ok = true
				break
			}
		}
		if !ok {
			err = tx.Model(&biz.Node{}).Where("id = ?", node.Id).Delete(node).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *ClusterRepo) saveCloudResources(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	for _, cloudResource := range cluster.CloudResources {
		cloudResource.ClusterId = cluster.Id
		err := tx.Model(&biz.CloudResource{}).Where("id = ?", cloudResource.Id).Save(cloudResource).Error
		if err != nil {
			return err
		}
	}
	cloudResources := make([]*biz.CloudResource, 0)
	err := tx.Model(&biz.CloudResource{}).Where("cluster_id = ?", cluster.Id).Find(&cloudResources).Error
	if err != nil {
		return err
	}
	for _, v := range cloudResources {
		ok := false
		for _, cloudResource := range cluster.CloudResources {
			if v.Id == cloudResource.Id {
				ok = true
				break
			}
		}
		if !ok {
			err := tx.Model(&biz.CloudResource{}).Where("id = ?", v.Id).Delete(v).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *ClusterRepo) saveSecuritys(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	for _, v := range cluster.Securitys {
		v.ClusterId = cluster.Id
		err := tx.Model(&biz.Security{}).Where("id = ?", v.Id).Save(v).Error
		if err != nil {
			return err
		}
	}
	sgs := make([]*biz.Security, 0)
	err := tx.Model(&biz.Security{}).Where("cluster_id = ?", cluster.Id).Find(&sgs).Error
	if err != nil {
		return err
	}
	for _, v := range sgs {
		isExist := false
		for _, v1 := range cluster.Securitys {
			if v.Id == v1.Id {
				isExist = true
				break
			}
		}
		if !isExist {
			err := tx.Model(&biz.Security{}).Where("id = ?", v.Id).Delete(v).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// save disk
func (c *ClusterRepo) saveDisk(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	for _, node := range cluster.Nodes {
		for _, disk := range node.Disks {
			disk.NodeId = node.Id
			disk.ClusterId = cluster.Id
			err := tx.Model(&biz.Disk{}).Where("id =?", disk.Id).Save(disk).Error
			if err != nil {
				return err
			}
		}
	}
	disks := make([]*biz.Disk, 0)
	err := tx.Model(&biz.Disk{}).Where("cluster_id =?", cluster.Id).Find(&disks).Error
	if err != nil {
		return err
	}
	for _, disk := range disks {
		ok := false
		for _, node := range cluster.Nodes {
			for _, disk2 := range node.Disks {
				if disk.Id == disk2.Id {
					ok = true
					break
				}
			}
		}
		if !ok {
			err = tx.Model(&biz.Disk{}).Where("id =?", disk.Id).Delete(disk).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}
