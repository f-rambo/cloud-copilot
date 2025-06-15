package biz

import (
	"strings"

	"github.com/google/wire"
)

// common context key
type ContextKey string

func (c ContextKey) String() string {
	return string(c)
}

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewClusterUseCase, NewAppUsecase, NewServicesUseCase, NewUseUser, NewProjectUseCase, NewWorkspaceUsecase)

const (
	WorkspaceName string = "workspace"
	ProjectName   string = "project"
	ClusterName   string = "cluster"
	WorkspaceId   string = "workspace_id"
	ProjectId     string = "project_id"
	ClusterId     string = "cluster_id"
	ServiceName   string = "service"
	ServiceId     string = "service_id"
	AppName       string = "app"
	AppId         string = "app_id"
	UserId        string = "user_id"
	UserName      string = "user_name"
)

func getBaseLableKeys() []string {
	return []string{
		WorkspaceName,
		ProjectName,
		ClusterName,
		ServiceName,
		AppName,
		UserId,
		UserName,
		AppId,
		ServiceId,
		ClusterId,
		WorkspaceId,
		ProjectId,
		ClusterId,
	}
}

func LablesToMap(labels string) map[string]string {
	if labels == "" {
		return make(map[string]string)
	}
	m := make(map[string]string)
	for _, label := range strings.Split(labels, ",") {
		kv := strings.Split(label, "=")
		if len(kv) == 2 {
			m[kv[0]] = kv[1]
		}
	}
	return m
}

func mapToLables(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	var labels []string
	for k, v := range m {
		labels = append(labels, k+"="+v)
	}
	return strings.Join(labels, ",")
}
