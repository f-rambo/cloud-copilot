package infrastructure

import (
	"context"
	"fmt"
	"testing"
	"time"

	cloudv1alpha1 "github.com/f-rambo/ocean/api/cloud/v1alpha1"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func TestConnShip(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialInsecure(
		ctx,
		grpc.WithEndpoint(fmt.Sprintf("%s:%d", "localhost", 9000)),
		grpc.WithTimeout(time.Second*5),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := cloudv1alpha1.NewCloudInterfaceClient(conn)
	response, err := client.Ping(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(response)
}
