package runtime

import (
	"context"
	"os"
	"path/filepath"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
)

const (
	CloudAppKind        = "CloudApp"
	CloudAppReleaseKind = "CloudAppRelease"

	AppRepoObjName = "app_repo"
	AppObjName     = "app"
)

type AppStatus int32

const (
	AppStatus_PENDING   AppStatus = 0
	AppStatus_COMPLETED AppStatus = 1
	AppStatus_FAILED    AppStatus = 2
)

func (a AppStatus) Int32() int32 {
	return int32(a)
}

type AppRuntime struct {
	log *log.Helper
}

func NewAppRuntime(logger log.Logger) biz.AppRuntime {
	return &AppRuntime{
		log: log.NewHelper(logger),
	}
}

func (a *AppRuntime) DeleteApp(ctx context.Context, app *biz.App) error {
	appPath := utils.GetServerStoragePathByNames(biz.AppsDir)
	err := os.Remove(appPath)
	if err != nil {
		return err
	}
	return nil
}

func (a *AppRuntime) DeleteAppVersion(ctx context.Context, app *biz.App, appVersion *biz.AppVersion) error {
	appPath := utils.GetServerStoragePathByNames(biz.AppsDir)
	for _, v := range app.Versions {
		if v.Chart == "" || appVersion.Id != v.Id {
			continue
		}
		chartPath := filepath.Join(appPath, v.Chart)
		if utils.IsFileExist(chartPath) {
			err := os.Remove(chartPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *AppRuntime) ReloadAppRepo(ctx context.Context, appRepo *biz.AppRepo) error {
	obj := NewUnstructured(CloudAppKind)
	obj.SetName(appRepo.Name)
	SetSpec(obj, map[string]any{AppRepoObjName: appRepo})
	obj, err := GetObjResrouce(ctx, obj, AppStatus_COMPLETED.Int32(), AppStatus_FAILED.Int32())
	if err != nil {
		return err
	}
	appRepoMap := map[string]*biz.AppRepo{}
	err = GetSpec(obj, &appRepoMap)
	if err != nil {
		return err
	}
	appRepo.IndexPath = appRepoMap[AppRepoObjName].IndexPath
	appRepo.Apps = make([]*biz.App, 0)
	appRepo.Apps = append(appRepo.Apps, appRepoMap[AppRepoObjName].Apps...)
	return nil
}

func (a *AppRuntime) GetAppAndVersionInfo(ctx context.Context, app *biz.App) error {
	obj := NewUnstructuredWithGenerateName(CloudAppKind, app.Name)
	SetSpec(obj, map[string]any{AppObjName: app})
	obj, err := GetObjResrouce(ctx, obj, AppStatus_COMPLETED.Int32(), AppStatus_FAILED.Int32())
	if err != nil {
		return err
	}
	appMap := map[string]*biz.App{}
	err = GetSpec(obj, &appMap)
	if err != nil {
		return err
	}
	err = utils.StructTransform(appMap[AppObjName], app)
	if err != nil {
		return err
	}
	return nil
}

func (a *AppRuntime) AppRelease(ctx context.Context, appRelease *biz.AppRelease) error {
	obj := NewUnstructured(CloudAppReleaseKind)
	obj.SetName(appRelease.ReleaseName)
	obj.SetNamespace(appRelease.Namespace)
	SetSpec(obj, appRelease)
	dynamicClient, err := GetKubeDynamicClient()
	if err != nil {
		return err
	}
	err = CreateResource(ctx, dynamicClient, obj)
	if err != nil {
		return err
	}
	return nil
}

func (a *AppRuntime) GetAppReleaseResources(ctx context.Context, appRelease *biz.AppRelease) error {
	obj := NewUnstructured(CloudAppReleaseKind)
	obj.SetName(appRelease.ReleaseName)
	obj.SetNamespace(appRelease.Namespace)
	dynamicClient, err := GetKubeDynamicClient()
	if err != nil {
		return err
	}
	obj, err = GetResource(ctx, dynamicClient, obj)
	if err != nil {
		return err
	}
	err = GetSpec(obj, appRelease)
	if err != nil {
		return err
	}
	return nil
}

func (a *AppRuntime) DeleteAppRelease(ctx context.Context, appRelease *biz.AppRelease) error {
	obj := NewUnstructured(CloudAppReleaseKind)
	obj.SetName(appRelease.ReleaseName)
	obj.SetNamespace(appRelease.Namespace)
	dynamicClient, err := GetKubeDynamicClient()
	if err != nil {
		return err
	}
	err = DeleteResource(ctx, dynamicClient, obj)
	if err != nil {
		return err
	}
	return nil
}
