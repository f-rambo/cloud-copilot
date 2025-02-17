package server

import (
	"context"
	"strings"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/interfaces"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

const (
	AuthorizationKey biz.ContextKey = "Authorization"
)

func NewAuthServer(user *interfaces.UserInterface) func(handler middleware.Handler) middleware.Handler {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var authorization string
			var userEmail string
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				authorizations := md.Get(AuthorizationKey.String())
				for _, v := range authorizations {
					authorization = v
					break
				}
				userEmails := md.Get(biz.UserEmailKey.String())
				for _, v := range userEmails {
					userEmail = v
					break
				}
			} else if header, ok := transport.FromServerContext(ctx); ok {
				authorization = header.RequestHeader().Get(AuthorizationKey.String())
				userEmail = header.RequestHeader().Get(biz.UserEmailKey.String())
			} else {
				return nil, errors.New(AuthorizationKey.String() + " is null")
			}
			authorizationArr := strings.Split(authorization, " ")
			if len(authorizationArr) != 2 {
				return nil, errors.New(AuthorizationKey.String() + " is error")
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
			return handler(ctx, req)
		}
	}
}

func BizContext(cluster *interfaces.ClusterInterface, project *interfaces.ProjectInterface, workspace *interfaces.WorkspaceInterface) func(handler middleware.Handler) middleware.Handler {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			param, ok := req.(proto.Message)
			if !ok {
				return handler(ctx, req)
			}
			msgReflection := param.ProtoReflect()
			descriptor := msgReflection.Descriptor()
			fields := descriptor.Fields()
			for i := 0; i < fields.Len(); i++ {
				field := fields.Get(i)
				fieldName := field.TextName()
				value := msgReflection.Get(field)
				switch fieldName {
				case "cluster_id", "clusterId":
					cluster, err := cluster.GetCluster(ctx, cast.ToInt64(value.Interface()))
					if err != nil {
						return nil, err
					}
					ctx = biz.WithCluster(ctx, cluster)
				case "workspace_id", "workspaceId":
					workspace, err := workspace.GetWorkspace(ctx, cast.ToInt64(value.Interface()))
					if err != nil {
						return nil, err
					}
					ctx = biz.WithWorkspace(ctx, workspace)
				case "project_id", "projectId":
					project, err := project.GetProject(ctx, cast.ToInt64(value.Interface()))
					if err != nil {
						return nil, err
					}
					ctx = biz.WithProject(ctx, project)
				default:
				}
			}
			return handler(ctx, req)
		}
	}
}

func NewWhiteListMatcher() selector.MatchFunc {
	whiteList := []string{
		"/user.v1alpha1.UserInterface/SignIn",
		"/cluster.v1alpha1.ClusterInterface/GetLogs",
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
