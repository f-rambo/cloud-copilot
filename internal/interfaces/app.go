package interfaces

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	v1alpha1 "github.com/f-rambo/cloud-copilot/api/app/v1alpha1"
	"github.com/f-rambo/cloud-copilot/api/common"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AppInterface struct {
	v1alpha1.UnimplementedAppInterfaceServer
	uc     *biz.AppUsecase
	userUc *biz.UserUseCase
	log    *log.Helper
}

func NewAppInterface(uc *biz.AppUsecase, user *biz.UserUseCase, logger log.Logger) *AppInterface {
	return &AppInterface{uc: uc, userUc: user, log: log.NewHelper(logger)}
}

func (a *AppInterface) Ping(ctx context.Context, _ *emptypb.Empty) (*common.Msg, error) {
	return common.Response(), nil
}

func (a *AppInterface) Save(ctx context.Context, app *v1alpha1.App) (*common.Msg, error) {
	if app.Name == "" || len(app.Versions) == 0 {
		return common.Response("name and version is required"), nil
	}
	bizApp, err := a.appTobizApp(app)
	if err != nil {
		return nil, err
	}
	err = a.uc.Save(ctx, bizApp)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) Get(ctx context.Context, appReq *v1alpha1.AppReq) (*v1alpha1.App, error) {
	if appReq.Id == 0 {
		return nil, errors.New("app id is required")
	}
	bizApp, err := a.uc.Get(ctx, appReq.Id)
	if err != nil {
		return nil, err
	}
	if bizApp == nil {
		return nil, errors.New("app not found")
	}
	app, err := a.bizAppToApp(bizApp)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (a *AppInterface) List(ctx context.Context, appReq *v1alpha1.AppReq) (*v1alpha1.AppList, error) {
	if appReq.Page == 0 {
		appReq.Page = 1
	}
	if appReq.PageSize == 0 {
		appReq.PageSize = 10
	}
	bizAppReq := &biz.App{
		Id:        appReq.Id,
		Name:      appReq.Name,
		AppTypeId: appReq.AppTypeId,
	}
	appItems, itemCount, err := a.uc.List(ctx, bizAppReq, appReq.Page, appReq.PageSize)
	if err != nil {
		return nil, err
	}
	appList := &v1alpha1.AppList{
		Items:     make([]*v1alpha1.App, len(appItems)),
		PageCount: int32(math.Ceil(float64(itemCount) / float64(appReq.PageSize))),
		ItemCount: itemCount,
	}
	appTypes, err := a.uc.ListAppType(ctx)
	if err != nil {
		return nil, err
	}
	for index, appItem := range appItems {
		app, err := a.bizAppToApp(appItem)
		if err != nil {
			return nil, err
		}
		for _, v := range appTypes {
			if v.Id == app.AppTypeId {
				app.AppTypeName = v.Name
				break
			}
		}
		appList.Items[index] = app
	}
	return appList, nil
}

func (a *AppInterface) Delete(ctx context.Context, appReq *v1alpha1.AppReq) (*common.Msg, error) {
	if appReq.Id == 0 {
		return common.Response("app id is required"), nil
	}
	app, err := a.uc.Get(ctx, appReq.Id)
	if err != nil {
		return nil, err
	}
	if app == nil || app.Id == 0 {
		return common.Response("app not found"), nil
	}
	err = a.uc.Delete(ctx, appReq.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) DeleteAppVersion(ctx context.Context, appReq *v1alpha1.AppReq) (*common.Msg, error) {
	if appReq.Id == 0 || appReq.VersionId == 0 {
		return common.Response("app id and version id is required"), nil
	}
	app, err := a.uc.Get(ctx, appReq.Id)
	if err != nil {
		return nil, err
	}
	if app == nil || app.Id == 0 {
		return common.Response("app not found"), nil
	}
	appVersion, err := a.uc.GetAppVersion(ctx, app.Id, appReq.VersionId)
	if err != nil {
		return nil, err
	}
	err = a.uc.DeleteAppVersion(ctx, app, appVersion)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) CreateAppType(ctx context.Context, appType *v1alpha1.AppType) (*common.Msg, error) {
	if appType.Name == "" {
		return common.Response("app type name is required"), nil
	}
	err := a.uc.CreateAppType(ctx, &biz.AppType{
		Name: appType.Name,
	})
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) ListAppType(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.AppTypeList, error) {
	appTypes, err := a.uc.ListAppType(ctx)
	if err != nil {
		return nil, err
	}
	appTypeList := &v1alpha1.AppTypeList{
		Items: make([]*v1alpha1.AppType, len(appTypes)),
	}
	for index, appType := range appTypes {
		appTypeList.Items[index] = &v1alpha1.AppType{
			Id:   appType.Id,
			Name: appType.Name,
		}
	}
	return appTypeList, nil
}

func (a *AppInterface) DeleteAppType(ctx context.Context, appTypeReq *v1alpha1.AppTypeReq) (*common.Msg, error) {
	if appTypeReq.Id == 0 {
		return common.Response("app type id is required"), nil
	}
	err := a.uc.DeleteAppType(ctx, appTypeReq.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) SaveRepo(ctx context.Context, repo *v1alpha1.AppRepo) (*common.Msg, error) {
	if repo.Name == "" {
		return nil, errors.New("repo name is required")
	}
	if repo.Url == "" {
		return nil, errors.New("repo url is required")
	}
	err := a.uc.SaveRepo(ctx, &biz.AppRepo{
		Id:          repo.Id,
		Name:        repo.Name,
		Url:         repo.Url,
		Description: repo.Description,
	})
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) ListRepo(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.AppRepoList, error) {
	repos, err := a.uc.ListRepo(ctx)
	if err != nil {
		return nil, err
	}
	repoList := &v1alpha1.AppRepoList{
		Items: make([]*v1alpha1.AppRepo, len(repos)),
	}
	for index, repo := range repos {
		repoList.Items[index] = &v1alpha1.AppRepo{
			Id:          repo.Id,
			Name:        repo.Name,
			Url:         repo.Url,
			Description: repo.Description,
		}
	}
	return repoList, nil
}

func (a *AppInterface) DeleteRepo(ctx context.Context, repoReq *v1alpha1.AppRepoReq) (*common.Msg, error) {
	if repoReq.Id == 0 {
		return nil, errors.New("repo id is required")
	}
	err := a.uc.DeleteRepo(ctx, repoReq.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) GetAppsByRepo(ctx context.Context, repoReq *v1alpha1.AppRepoReq) (*v1alpha1.AppList, error) {
	if repoReq.Id == 0 {
		return nil, errors.New("repo id is required")
	}
	apps, err := a.uc.GetAppsByRepo(ctx, repoReq.Id)
	if err != nil {
		return nil, err
	}
	itemCount := len(apps)
	appList := &v1alpha1.AppList{
		Items:     make([]*v1alpha1.App, itemCount),
		ItemCount: int32(itemCount),
	}
	for index, app := range apps {
		dataApp, err := a.bizAppToApp(app)
		if err != nil {
			return nil, err
		}
		dataApp.Id = int64(index) + 1
		appList.Items[index] = dataApp
	}
	return appList, nil
}

func (a *AppInterface) GetAppDetailByRepo(ctx context.Context, repoReq *v1alpha1.AppRepoReq) (*v1alpha1.App, error) {
	if repoReq.Id == 0 {
		return nil, errors.New("repo id is required")
	}
	if repoReq.AppName == "" {
		return nil, errors.New("app name is required")
	}
	app, err := a.uc.GetAppDetailByRepo(ctx, repoReq.Id, repoReq.AppName, repoReq.Version)
	if err != nil {
		return nil, err
	}
	appRes, err := a.bizAppToApp(app)
	if err != nil {
		return nil, err
	}
	appRes.Id = repoReq.Id
	return appRes, nil
}

func (a *AppInterface) GetAppRelease(ctx context.Context, AppReleaseReq *v1alpha1.AppReleaseReq) (*v1alpha1.AppRelease, error) {
	if AppReleaseReq.Id == 0 {
		return nil, errors.New("app release id is required")
	}
	appReleaseRes, err := a.uc.GetAppRelease(ctx, AppReleaseReq.Id)
	if err != nil {
		return nil, err
	}
	appRelease := &v1alpha1.AppRelease{}
	err = utils.StructTransform(appReleaseRes, appRelease)
	if err != nil {
		return nil, err
	}
	appRelease.Id = appReleaseRes.Id
	user, err := a.userUc.GetUser(ctx, appRelease.UserId)
	if err != nil {
		return nil, err
	}
	appRelease.UserName = user.Name
	return appRelease, nil
}

func (a *AppInterface) AppReleaseList(ctx context.Context, appReleaseReq *v1alpha1.AppReleaseReq) (*v1alpha1.AppReleaseList, error) {
	if appReleaseReq.Page == 0 {
		appReleaseReq.Page = 1
	}
	if appReleaseReq.PageSize == 0 {
		appReleaseReq.PageSize = 10
	}
	if appReleaseReq.PageSize > 30 {
		appReleaseReq.PageSize = 30
	}
	appReleaseReqMap := make(map[string]string)
	appReleaseReqJsonStr, err := json.Marshal(appReleaseReq)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(appReleaseReqJsonStr, &appReleaseReqMap)
	if err != nil {
		return nil, err
	}
	appReleases, count, err := a.uc.AppReleaseList(ctx, appReleaseReqMap, appReleaseReq.Page, appReleaseReq.PageSize)
	if err != nil {
		return nil, err
	}
	appReleaseList := make([]*v1alpha1.AppRelease, len(appReleases))
	for index, appRelease := range appReleases {
		appReleaseList[index] = &v1alpha1.AppRelease{}
		err = utils.StructTransform(appRelease, appReleaseList[index])
		if err != nil {
			return nil, err
		}
		appReleaseList[index].Id = appRelease.Id
	}
	return &v1alpha1.AppReleaseList{
		Items: appReleaseList,
		Count: count,
	}, nil
}

func (a *AppInterface) GetAppReleaseResources(ctx context.Context, appReleaseReq *v1alpha1.AppReleaseReq) (*v1alpha1.AppReleasepResources, error) {
	if appReleaseReq.Id == 0 {
		return nil, errors.New("app release id is required")
	}
	resources, err := a.uc.GetAppReleaseResourcesInCluster(ctx, appReleaseReq.Id)
	if err != nil {
		return nil, err
	}
	data := &v1alpha1.AppReleasepResources{}
	items := make([]*v1alpha1.AppReleaseResource, 0)
	for _, resource := range resources {
		item := &v1alpha1.AppReleaseResource{}
		err = utils.StructTransform(resource, item)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	data.Items = items
	return data, nil
}

func (a *AppInterface) SaveAppRelease(ctx context.Context, appReleaseReq *v1alpha1.AppReleaseReq) (*v1alpha1.AppRelease, error) {
	if appReleaseReq.AppId == 0 {
		return nil, errors.New("app id is required")
	}
	if appReleaseReq.VersionId == 0 {
		return nil, errors.New("app version is required")
	}
	app, err := a.uc.Get(ctx, appReleaseReq.AppId)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, errors.New("app not found")
	}
	appVersion := app.GetVersionById(appReleaseReq.VersionId)
	if appVersion == nil {
		return nil, errors.New("app version not found")
	}
	user := biz.GetUserInfo(ctx)
	err = a.uc.CreateAppRelease(ctx, &biz.AppRelease{
		ReleaseName: appReleaseReq.ReleaseName,
		AppId:       app.Id,
		VersionId:   appVersion.Id,
		UserId:      user.Id,
		ProjectId:   appReleaseReq.ProjectId,
		ClusterId:   appReleaseReq.ClusterId,
		Namespace:   appReleaseReq.Namespace,
		Config:      appReleaseReq.Config,
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (a *AppInterface) DeleteAppRelease(ctx context.Context, appReleaseReq *v1alpha1.AppReleaseReq) (*common.Msg, error) {
	if appReleaseReq.Id == 0 {
		return nil, errors.New("app release id is required")
	}
	err := a.uc.DeleteAppRelease(ctx, appReleaseReq.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) UploadApp(ctx context.Context, _ *emptypb.Empty) (*common.Msg, error) {
	appPath := utils.GetServerStoragePathByNames("apps")
	appFileName, err := utils.AcceptingFile(ctx, appPath)
	if err != nil {
		return nil, err
	}
	app := &biz.App{Versions: make([]*biz.AppVersion, 0)}
	app.AddVersion(&biz.AppVersion{Chart: filepath.Join(appPath, appFileName)})
	err = a.uc.GetAppAndVersionInfo(ctx, app)
	if err != nil {
		return nil, err
	}
	appVersion := app.GetLastVersion()
	appTgzFileName := fmt.Sprintf("%s-%s.tgz", app.Name, appVersion.Version)
	appChartPath := filepath.Join(appPath, appTgzFileName)
	if utils.IsFileExist(appChartPath) {
		err = os.Remove(appChartPath)
		if err != nil {
			return nil, err
		}
	}
	err = os.Rename(filepath.Join(appPath, appFileName), appChartPath)
	if err != nil {
		return nil, err
	}
	appData, err := a.uc.GetAppByName(ctx, app.Name)
	if err != nil {
		return nil, err
	}
	if !appData.IsEmpty() {
		appData.DeleteVersion(appVersion.Version)
		appData.AddVersion(appVersion)
		appData.UpdateApp(app)
		err = a.uc.Save(ctx, appData)
		if err != nil {
			return nil, err
		}
		return common.Response(), nil
	}
	err = a.uc.Save(ctx, app)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) bizAppToApp(bizApp *biz.App) (*v1alpha1.App, error) {
	app := &v1alpha1.App{
		Id:        bizApp.Id,
		Name:      bizApp.Name,
		Icon:      bizApp.Icon,
		AppTypeId: bizApp.AppTypeId,
		Versions:  make([]*v1alpha1.AppVersion, len(bizApp.Versions)),
	}
	for index, v := range bizApp.Versions {
		appversion, err := a.bizAppVersionToAppVersion(v)
		if err != nil {
			return nil, err
		}
		appversion.Id = int64(index) + 1
		app.Versions[index] = appversion
	}
	return app, nil
}

func (a *AppInterface) bizAppVersionToAppVersion(bizAppVersion *biz.AppVersion) (*v1alpha1.AppVersion, error) {
	appVersion := &v1alpha1.AppVersion{}
	err := utils.StructTransform(bizAppVersion, appVersion)
	if err != nil {
		return nil, err
	}
	appVersion.Id = bizAppVersion.Id
	return appVersion, nil
}

func (a *AppInterface) appTobizApp(app *v1alpha1.App) (*biz.App, error) {
	bizApp := &biz.App{}
	err := utils.StructTransform(app, bizApp)
	if err != nil {
		return nil, err
	}
	bizApp.Id = app.Id
	for index, v := range app.Versions {
		appVersion, err := a.appVersionToBizAppVersion(v)
		if err != nil {
			return nil, err
		}
		bizApp.Versions[index] = appVersion
	}
	return bizApp, nil
}

func (a *AppInterface) appVersionToBizAppVersion(appVersion *v1alpha1.AppVersion) (*biz.AppVersion, error) {
	bizAppVersion := &biz.AppVersion{}
	err := utils.StructTransform(appVersion, bizAppVersion)
	if err != nil {
		return nil, err
	}
	bizAppVersion.Id = appVersion.Id
	return bizAppVersion, nil
}
