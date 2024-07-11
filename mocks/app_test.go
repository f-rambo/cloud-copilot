package mocks

import (
	"context"
	"fmt"
	"testing"

	v1alpha1 "github.com/f-rambo/ocean/api/app/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/internal/interfaces"
	"github.com/go-kratos/kratos/v2/log"
	gomock "github.com/golang/mock/gomock"
)

func Test_CreateAppType(t *testing.T) {
	ctl := gomock.NewController(&testing.T{})
	defer ctl.Finish()
	appRepo := NewMockAppRepo(ctl)
	clusterRepo := NewMockClusterRepo(ctl)
	projectRepo := NewMockProjectRepo(ctl)
	sailorRepo := NewMockSailorRepo(ctl)
	appRuntime := NewMockAppRuntime(ctl)
	appConstruct := NewMockAppConstruct(ctl)
	appuse := biz.NewAppUsecase(appRepo, clusterRepo, projectRepo, sailorRepo, appRuntime, appConstruct, log.DefaultLogger, &conf.Bootstrap{})
	appInterface := interfaces.NewAppInterface(appuse, nil, &conf.Bootstrap{}, log.DefaultLogger)
	call := appRepo.EXPECT().CreateAppType(gomock.Any(), gomock.Any())
	// call.Return(nil)
	// call.Do(func(ctx context.Context, appType *biz.AppType) {
	// 	t.Log(appType.Name)
	// 	t.Log("CreateAppType called")
	// })
	call.DoAndReturn(func(ctx context.Context, appType *biz.AppType) error {
		return fmt.Errorf("CreateAppType error")
	})
	_, err := appInterface.CreateAppType(context.Background(), &v1alpha1.AppType{Name: "test"})
	if err != nil {
		t.Error(err)
	}
	// err := appuse.CreateAppType(context.Background(), &biz.AppType{})
	// if err != nil {
	// 	t.Error(err)
	// }
}
