package server

import (
	"context"
	"strings"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/interfaces"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
)

func NewAuthServer(user *interfaces.UserInterface) func(handler middleware.Handler) middleware.Handler {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var authorization string
			var userEmail string
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				authorizations := md.Get("Authorization")
				for _, v := range authorizations {
					authorization = v
					break
				}
				userEmails := md.Get("User-Email")
				for _, v := range userEmails {
					userEmail = v
					break
				}
			} else if header, ok := transport.FromServerContext(ctx); ok {
				authorization = header.RequestHeader().Get("Authorization")
				userEmail = header.RequestHeader().Get("User-Email")
			} else {
				return nil, errors.New("authorization is null")
			}
			authorizationArr := strings.Split(authorization, " ")
			if len(authorizationArr) != 2 {
				return nil, errors.New("authorization is error")
			}
			for index, authorization := range authorizationArr {
				if index == 0 {
					ctx = context.WithValue(ctx, biz.SignType, authorization)
				}
				if index == 1 {
					ctx = context.WithValue(ctx, biz.TokenKey, authorization)
				}
			}
			ctx = context.WithValue(ctx, biz.UserEmailKey, userEmail)
			reply, err = handler(ctx, req)
			return
		}
	}
}

func NewWhiteListMatcher() selector.MatchFunc {
	whiteList := []string{
		"/user.v1alpha1.UserInterface/SignIn",
		"/app.v1alpha1.AppInterface/Ping",
		"/cluster.v1alpha1.ClusterInterface/Ping",
		"/cluster.v1alpha1.ClusterInterface/PollingLogs",
		"/service.v1alpha1.ServiceInterface/Ping",
		"/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/",
	}
	return func(ctx context.Context, operation string) bool {
		for _, v := range whiteList {
			if strings.Contains(operation, v) {
				return false
			}
		}
		return true
	}
}
