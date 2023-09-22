package cluster

import (
	"context"
	"fmt"

	v1alpha1 "github.com/f-rambo/ocean/api/cluster/v1alpha1"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.in/yaml.v3"
)

/*
Get(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*Clusters, error)
Save(ctx context.Context, in *Cluster, opts ...grpc.CallOption) (*Msg, error)
Delete(ctx context.Context, in *ClusterID, opts ...grpc.CallOption) (*Msg, error)
Apply(ctx context.Context, in *ClusterName, opts ...grpc.CallOption) (*Msg, error)
*/
var (
	client            v1alpha1.ClusterServiceClient
	l                 *log.Helper
	clusterDirPath    = "./cluster"
	clusterYamlPath   = fmt.Sprintf("%s/%s", clusterDirPath, "cluster.yaml")
	clusterConfigPath = fmt.Sprintf("%s/%s", clusterDirPath, "config.yaml")
	clusterAddonsPath = fmt.Sprintf("%s/%s", clusterDirPath, "addons.yaml")
)

func NewClusterCommand(conn *grpc.ClientConn, logger log.Logger) *cobra.Command {
	client = v1alpha1.NewClusterServiceClient(conn)
	l = log.NewHelper(logger)
	command := &cobra.Command{
		Use:   "cluster",
		Short: `Manage the k8s cluster`,
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 0 {
				return c.Help()
			}
			return nil
		},
	}
	command.AddCommand(apply(), list(), delete(), example())
	return command
}

func apply() *cobra.Command {
	var (
		clusterPath string
		configPath  string
		addonsPath  string
	)
	command := &cobra.Command{
		Use: "apply",
		Short: `
		ocean cluster apply --cluster=./cluster/cluster.yaml --config=./cluster/config.yaml --addons=./cluster/addons.yaml
		default cluster yaml file is ./cluster/cluster.yaml
		default cluster config yaml file is ./cluster/config.yaml
		default cluster addons yaml file is ./cluster/addons.yaml
		`,
		RunE: func(c *cobra.Command, args []string) error {
			// savev / apply
			if !utils.CheckFileIsExist(clusterPath) {
				return fmt.Errorf("cluster yaml file not exist")
			}
			if !utils.CheckFileIsExist(configPath) {
				return fmt.Errorf("cluster config yaml file not exist")
			}
			if !utils.CheckFileIsExist(clusterAddonsPath) {
				return fmt.Errorf("cluster addons yaml file not exist")
			}
			clusterContent, err := utils.ReadFile(clusterPath)
			if err != nil {
				return err
			}
			var clusterData v1alpha1.ClusterV1Alpha1
			err = yaml.Unmarshal([]byte(clusterContent), &clusterData)
			if err != nil {
				return err
			}
			configContent, err := utils.ReadFile(configPath)
			if err != nil {
				return err
			}
			addonsContent, err := utils.ReadFile(addonsPath)
			if err != nil {
				return err
			}
			clusterData.Spec.Config = configContent
			clusterData.Spec.Addons = addonsContent
			_, err = client.Save(context.TODO(), clusterData.Spec)
			if err != nil {
				return err
			}
			_, err = client.Apply(context.TODO(), &v1alpha1.ClusterName{Name: clusterData.Spec.ClusterName})
			if err != nil {
				return err
			}
			return nil
		},
	}
	command.Flags().StringVar(&clusterPath, "cluster", clusterYamlPath, "cluster yaml flile")
	command.Flags().StringVar(&configPath, "config", clusterConfigPath, "cluster config yaml flile")
	command.Flags().StringVar(&addonsPath, "addons", clusterAddonsPath, "cluster addons yaml flile")
	return command
}

func list() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "ocean cluster list",
		RunE: func(c *cobra.Command, args []string) error {
			clusters, err := client.Get(context.Background(), &emptypb.Empty{})
			if err != nil {
				return err
			}
			for _, cluster := range clusters.Clusters {
				fmt.Printf("id:%d, cluster name: %s, applyed %v\n", cluster.Id, cluster.ClusterName, cluster.Applyed)
			}
			l.Info("list cluster success")
			return nil
		},
	}
}

func delete() *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "ocean cluster delete 'cluster name or id'",
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 0 {
				return c.Help()
			}
			clusterIDOrName := args[0]
			clusters, err := client.Get(context.Background(), &emptypb.Empty{})
			if err != nil {
				return err
			}
			clusterId := cast.ToInt32(clusterIDOrName)
			var deleteId int32
			for _, v := range clusters.Clusters {
				if clusterId == 0 && v.ClusterName == clusterIDOrName {
					deleteId = v.Id
				}
				if clusterId != 0 && clusterId == v.Id {
					deleteId = v.Id
				}
			}
			if deleteId == 0 {
				return fmt.Errorf("cluster %s not exist", clusterIDOrName)
			}
			_, err = client.Delete(context.Background(), &v1alpha1.ClusterID{Id: int32(deleteId)})
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func example() *cobra.Command {
	return &cobra.Command{
		Use: "example",
		Short: `
		generate example
		-------------
		./cluster/cluster.yaml
		./cluster/config.yaml
		./cluster/addons.yaml
		`,
		RunE: func(c *cobra.Command, args []string) error {
			// 写入到本地文件
			clusters, err := client.Get(context.TODO(), &emptypb.Empty{})
			if err != nil {
				return err
			}
			var config, addons string
			var cluster *v1alpha1.Cluster
			for _, v := range clusters.Clusters {
				cluster = v
				break
			}
			metaData := v1alpha1.MetaData{
				Name: cluster.ClusterName,
			}
			config = cluster.Config
			cluster.Config = ""
			addons = cluster.Addons
			cluster.Addons = ""
			data := &v1alpha1.ClusterV1Alpha1{
				MetaData: &metaData,
				Kind:     "cluster",
				Spec:     cluster,
			}
			yamlStr, err := yaml.Marshal(data)
			if err != nil {
				return err
			}
			if !utils.CheckFileIsExist(clusterDirPath) {
				err = utils.CreateDir(clusterDirPath)
				if err != nil {
					return err
				}
			}
			err = utils.WriteFile(clusterConfigPath, config)
			if err != nil {
				return err
			}
			err = utils.WriteFile(clusterAddonsPath, addons)
			if err != nil {
				return err
			}
			err = utils.WriteFile(clusterYamlPath, string(yamlStr))
			if err != nil {
				return err
			}
			return nil
		},
	}
}
