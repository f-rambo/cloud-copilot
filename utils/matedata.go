package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/metadata"
	mmd "github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	grpcConnect "google.golang.org/grpc"
)

type MatedataKey string

func (m MatedataKey) String() string {
	return string(m)
}

const (
	ServiceNameKey    MatedataKey = "service"
	ServiceVersionKey MatedataKey = "version"
	RuntimeKey        MatedataKey = "runtime"
	OSKey             MatedataKey = "os"
	ArchKey           MatedataKey = "arch"
	ConfKey           MatedataKey = "conf"
	ConfDirKey        MatedataKey = "confdir"
)

func GetFromContextByKey(ctx context.Context, key MatedataKey) string {
	appInfo, ok := kratos.FromContext(ctx)
	if !ok {
		return ""
	}
	value, ok := appInfo.Metadata()[key.String()]
	if !ok {
		return ""
	}
	return value
}

func GetFromContext(ctx context.Context) map[string]string {
	appInfo, ok := kratos.FromContext(ctx)
	if !ok {
		return nil
	}
	return appInfo.Metadata()
}

type GrpcConn struct {
	Conn *grpcConnect.ClientConn
	Ctx  context.Context
}

func (g *GrpcConn) OpenGrpcConn(ctx context.Context, addr string, port int32, timeoutsecond int64) (*GrpcConn, error) {
	if timeoutsecond == 0 {
		timeoutsecond = 10
	}
	conn, err := grpc.DialInsecure(ctx,
		grpc.WithEndpoint(fmt.Sprintf("%s:%d", addr, port)),
		grpc.WithMiddleware(mmd.Client()),
		grpc.WithTimeout(time.Duration(timeoutsecond)*time.Second),
	)
	if err != nil {
		return nil, err
	}
	appInfo := GetFromContext(ctx)
	for k, v := range appInfo {
		ctx = metadata.AppendToClientContext(ctx, k, v)
	}
	return &GrpcConn{
		Conn: conn,
		Ctx:  ctx,
	}, nil
}

func (g *GrpcConn) Close() {
	g.Conn.Close()
}
