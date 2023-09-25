package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/f-rambo/ocean/cmd/client/app"
	"github.com/f-rambo/ocean/cmd/client/cluster"
	"github.com/f-rambo/ocean/cmd/client/service"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	serverAddr string
)

func init() {
	flag.StringVar(&serverAddr, "server", "127.0.0.1:9000", "server address, eg: -server 127.0.0.1:9000")
}

func newRootCommand(conn *grpc.ClientConn, logger log.Logger) *cobra.Command {
	command := cobra.Command{
		Use:   "ocean",
		Short: "ocean client command",
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 0 {
				return c.Help()
			}
			return nil
		},
	}
	command.AddCommand(
		cluster.NewClusterCommand(conn, logger),
		app.NewAppommand(conn, logger),
		service.NewServiceCommand(conn, logger),
	)
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
