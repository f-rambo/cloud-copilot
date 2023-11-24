package server

import (
	"context"
	"errors"
	"strings"

	"github.com/f-rambo/ocean/internal/service"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func NewAuthServer(user *service.UserService) func(handler middleware.Handler) middleware.Handler {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var jwtToken string
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				jwtToken = md.Get("token")[0]
			} else if header, ok := transport.FromServerContext(ctx); ok {
				jwtToken = strings.SplitN(header.RequestHeader().Get("Authorization"), " ", 2)[1]
			} else {
				return nil, errors.New("token err")
			}
			ctx = context.WithValue(ctx, "token", jwtToken)
			user, err := user.GetUserInfo(ctx, &emptypb.Empty{})
			if err != nil {
				return nil, err
			}
			ctx = context.WithValue(ctx, "id", user.Id)
			reply, err = handler(ctx, req)
			return
		}
	}
}
