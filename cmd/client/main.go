package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/f-rambo/ocean/api/cluster/v1alpha1"
	"github.com/f-rambo/ocean/cmd/client/app"
	"github.com/f-rambo/ocean/cmd/client/cluster"
	"github.com/f-rambo/ocean/cmd/client/service"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	serverAddr string
)

func init() {
	flag.StringVar(&serverAddr, "server", "127.0.0.1:9000", "server address, eg: -server 127.0.0.1:9000")
}

func newRootCommand(conn *grpc.ClientConn, logger log.Logger) *cobra.Command {
	var (
		clusterGrpcAddr string
	)
	command := cobra.Command{
		Use:   "ocean",
		Short: "ocean client command",
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 0 {
				return c.Help()
			}
			// sync
			if args[0] != "sync" {
				return c.Help()
			}
			if clusterGrpcAddr == "" {
				return fmt.Errorf("cluster grpc address is empty")
			}
			clusterConn, err := grpc.Dial(clusterGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return err
			}
			defer func() {
				if conn != nil {
					conn.Close()
				}
			}()
			localClient := v1alpha1.NewClusterServiceClient(conn)
			clusterClient := v1alpha1.NewClusterServiceClient(clusterConn)
			clusters, err := localClient.Get(c.Context(), &emptypb.Empty{})
			if err != nil {
				return err
			}
			for _, cluster := range clusters.Clusters {
				cluster.Applyed = false
				_, err = clusterClient.Save(c.Context(), cluster)
				if err != nil {
					return err
				}
				cluster.Applyed = true
				_, err = clusterClient.Save(c.Context(), cluster)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.AddCommand(
		cluster.NewClusterCommand(conn, logger),
		app.NewAppommand(conn, logger),
		service.NewServiceCommand(conn, logger),
	)
	command.Flags().StringVar(&clusterGrpcAddr, "cluster-grpc-addr", "", "deployed cluster grpc address")
	return &command
}

func main() {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	if err := newRootCommand(conn, log.DefaultLogger).Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
