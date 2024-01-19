package server

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/f-rambo/ocean/internal/interfaces"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func NewAuthServer(user *interfaces.UserInterface) func(handler middleware.Handler) middleware.Handler {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var authorization string
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				authorizations := md.Get("Authorization")
				for _, v := range authorizations {
					authorization = v
					break
				}
			} else if header, ok := transport.FromServerContext(ctx); ok {
				authorization = header.RequestHeader().Get("Authorization")
			} else {
				return nil, errors.New("authorization is null")
			}
			authorizationArr := strings.Split(authorization, " ")
			if len(authorizationArr) != 2 {
				return nil, errors.New("authorization is error")
			}
			// 获取token
			jwtToken := authorizationArr[1]
			ctx = context.WithValue(ctx, utils.TokenKey, jwtToken)
			user, err := user.GetUserInfo(ctx, &emptypb.Empty{})
			if err != nil {
				return nil, err
			}
			ctx = context.WithValue(ctx, utils.UserIDKey, user.Id)
			reply, err = handler(ctx, req)
			return
		}
	}
}

func NewWhiteListMatcher() selector.MatchFunc {
	whiteList := make(map[string]struct{})
	whiteList["/user.v1alpha1.UserInterface/SignIn"] = struct{}{}
	whiteList["/app.v1alpha1.AppInterface/Ping"] = struct{}{}
	whiteList["/cluster.v1alpha1.ClusterInterface/Ping"] = struct{}{}
	whiteList["/service.v1alpha1.ServiceInterface/Ping"] = struct{}{}
	return func(ctx context.Context, operation string) bool {
		if _, ok := whiteList[operation]; ok {
			return false
		}
		return true
	}
}
