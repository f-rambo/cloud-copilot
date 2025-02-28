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
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				authorizations := md.Get(AuthorizationKey.String())
				for _, v := range authorizations {
					authorization = v
					break
				}
			}

			if header, ok := transport.FromServerContext(ctx); ok && authorization == "" {
				authorization = header.RequestHeader().Get(AuthorizationKey.String())
			}

			if authorization == "" {
				return nil, errors.New(AuthorizationKey.String() + " is null")
			}

			authorizationArr := strings.Split(authorization, " ")
			if len(authorizationArr) != 2 {
				return nil, errors.New(AuthorizationKey.String() + " is error")
			}

			userInfo, err := biz.ValidateJWT(authorizationArr[1])
			if err != nil {
				return nil, err
			}

			ctx = context.WithValue(ctx, biz.SignType, authorizationArr[0])
			ctx = context.WithValue(ctx, biz.TokenKey, authorizationArr[1])
			ctx = biz.WithUser(ctx, userInfo)
			return handler(ctx, req)
		}
	}
}

func BizContext(clusterApi *interfaces.ClusterInterface, projectApi *interfaces.ProjectInterface, workspaceApi *interfaces.WorkspaceInterface) func(handler middleware.Handler) middleware.Handler {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			param, ok := req.(proto.Message)
			if !ok {
				return handler(ctx, req)
			}
			var cluster *biz.Cluster
			var workspace *biz.Workspace
			var project *biz.Project
			msgReflection := param.ProtoReflect()
			descriptor := msgReflection.Descriptor()
			fields := descriptor.Fields()
			for i := 0; i < fields.Len(); i++ {
				field := fields.Get(i)
				fieldName := field.TextName()
				value := msgReflection.Get(field)
				switch fieldName {
				case "cluster_id", "clusterId":
					clusterId := cast.ToInt64(value.Interface())
					if clusterId > 0 {
						cluster, err = clusterApi.GetCluster(ctx, cast.ToInt64(value.Interface()))
						if err != nil {
							return nil, err
						}
						ctx = biz.WithCluster(ctx, cluster)
					}
				case "workspace_id", "workspaceId":
					workspaceId := cast.ToInt64(value.Interface())
					if workspaceId > 0 {
						workspace, err = workspaceApi.GetWorkspace(ctx, cast.ToInt64(value.Interface()))
						if err != nil {
							return nil, err
						}
						ctx = biz.WithWorkspace(ctx, workspace)
						if cluster == nil && workspace.ClusterId > 0 {
							cluster, err = clusterApi.GetCluster(ctx, workspace.ClusterId)
							if err != nil {
								return nil, err
							}
							ctx = biz.WithCluster(ctx, cluster)
						}
					}
				case "project_id", "projectId":
					projectId := cast.ToInt64(value.Interface())
					if projectId > 0 {
						project, err = projectApi.GetProject(ctx, cast.ToInt64(value.Interface()))
						if err != nil {
							return nil, err
						}
						ctx = biz.WithProject(ctx, project)
						if workspace == nil && project.WorkspaceId > 0 {
							workspace, err = workspaceApi.GetWorkspace(ctx, project.WorkspaceId)
							if err != nil {
								return nil, err
							}
							ctx = biz.WithWorkspace(ctx, workspace)
							if cluster == nil && workspace.ClusterId > 0 {
								cluster, err = clusterApi.GetCluster(ctx, workspace.ClusterId)
								if err != nil {
									return nil, err
								}
								ctx = biz.WithCluster(ctx, cluster)
							}
						}
					}
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
