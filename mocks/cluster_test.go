package mocks

import (
	"context"
	"testing"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang/mock/gomock"
)

func TestGetCurrentCluster(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	clusterRepo := NewMockClusterRepo(ctl)
	infrastructure := NewMockInfrastructure(ctl)
	clusterConstruct := NewMockClusterConstruct(ctl)
	clusterRuntime := NewMockClusterRuntime(ctl)
	clusterUsecase := biz.NewClusterUseCase(clusterRepo, infrastructure, clusterConstruct, clusterRuntime, log.DefaultLogger)
	cluster, err := clusterUsecase.GetCurrentCluster(context.TODO())
	if err != nil {
		t.Error(err)
	}
	t.Log(cluster)
}
