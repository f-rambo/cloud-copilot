package runtime

import "github.com/google/wire"

var ProviderSet = wire.NewSet(NewAppRuntime, NewClusterRuntime, NewProjectRuntime, NewServiceRuntime, NewWorkflowRuntime, NewWorkspaceRuntime)
