package interfaces

import "github.com/google/wire"

// ProviderSet is interface providers.
var ProviderSet = wire.NewSet(NewClusterInterface, NewAppInterface, NewServicesInterface, NewUserInterface, NewProjectInterface, NewAutoscaler)
