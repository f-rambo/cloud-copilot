package interfaces

import (
	"context"
	"encoding/base64"
	"math"
	"path/filepath"
	"strings"

	v1alpha1 "github.com/f-rambo/ocean/api/app/v1alpha1"
	"github.com/f-rambo/ocean/api/common"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AppInterface struct {
	v1alpha1.UnimplementedAppInterfaceServer
	uc   *biz.AppUsecase
	user *biz.UserUseCase
	c    *conf.Bootstrap
	log  *log.Helper
}

func NewAppInterface(uc *biz.AppUsecase, user *biz.UserUseCase, c *conf.Bootstrap, logger log.Logger) *AppInterface {
	return &AppInterface{uc: uc, c: c, user: user, log: log.NewHelper(logger)}
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
	bizApp, err := a.uc.Get(ctx, appReq.Id, appReq.VersionId)
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
		ID:        appReq.Id,
		Name:      appReq.Name,
		AppTypeID: appReq.AppTypeId,
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
			if v.ID == app.AppTypeId {
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
	app, err := a.uc.Get(ctx, appReq.Id, appReq.VersionId)
	if err != nil {
		return nil, err
	}
	if app == nil || app.ID == 0 {
		return common.Response("app not found"), nil
	}
	err = a.uc.Delete(ctx, appReq.Id, appReq.VersionId)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) UploadApp(ctx context.Context, req *v1alpha1.FileUploadRequest) (*v1alpha1.App, error) {
	if filepath.Ext(req.GetFileName()) != ".tgz" {
		return nil, errors.New("file type is not supported")
	}
	appPath, err := utils.GetPackageStorePathByNames(biz.AppPackageName)
	if err != nil {
		return nil, err
	}
	filePathName, err := a.upload(appPath, req.GetFileName(), req.GetChunk())
	if err != nil {
		return nil, err
	}
	appVersion := &biz.AppVersion{Chart: filePathName, State: biz.AppUntested}
	err = a.uc.GetAppVersionChartInfomation(ctx, appVersion)
	if err != nil {
		return nil, err
	}
	app, err := a.uc.GetAppByName(ctx, appVersion.AppName)
	if err != nil {
		return nil, err
	}
	if app == nil {
		app = &biz.App{Name: appVersion.AppName}
	}
	if req.Icon != "" {
		app.Icon = req.Icon
	}
	if app.GetVersion(appVersion.Version) != nil {
		app.UpdateVersion(appVersion)
	} else {
		app.AddVersion(appVersion)
	}
	return a.bizAppToApp(app)
}

// 上传文件
func (a *AppInterface) upload(path, filename, chunk string) (string, error) {
	// 从base64转换为文件 []byte
	data, err := base64.StdEncoding.DecodeString(chunk[strings.IndexByte(chunk, ',')+1:])
	if err != nil {
		return "", err
	}
	file, err := utils.NewFile(path, filename, false)
	if err != nil {
		return "", err
	}
	defer func() {
		if file == nil {
			return
		}
		err := file.Close()
		if err != nil {
			a.log.Error(err)
		}
	}()
	err = file.Write(data)
	if err != nil {
		return "", err
	}
	return file.GetFileName(), nil
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
			Id:   appType.ID,
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

func (a *AppInterface) AppTest(ctx context.Context, deployAppReq *v1alpha1.DeployAppReq) (*v1alpha1.DeployApp, error) {
	if deployAppReq.AppId == 0 || deployAppReq.VersionId == 0 {
		return nil, errors.New("app id and version id is required")
	}
	appDeployRes, err := a.uc.AppTest(ctx, deployAppReq.AppId, deployAppReq.VersionId)
	if err != nil {
		return nil, err
	}
	appDeploy := &v1alpha1.DeployApp{}
	err = utils.StructTransform(appDeployRes, appDeploy)
	if err != nil {
		return nil, err
	}
	appDeploy.Id = appDeployRes.ID
	return appDeploy, nil
}

func (a *AppInterface) GetAppDeployed(ctx context.Context, appDeployId *v1alpha1.DeployApp) (*v1alpha1.DeployApp, error) {
	if appDeployId.Id == 0 {
		return nil, errors.New("app deploy id is required")
	}
	appDeployRes, err := a.uc.GetAppDeployed(ctx, appDeployId.Id)
	if err != nil {
		return nil, err
	}
	appDeploy := &v1alpha1.DeployApp{}
	err = utils.StructTransform(appDeployRes, appDeploy)
	if err != nil {
		return nil, err
	}
	appDeploy.Id = appDeployRes.ID
	user, err := a.user.GetUserByID(ctx, appDeploy.UserId)
	if err != nil {
		return nil, err
	}
	appDeploy.UserName = user.Name
	appDeploy.CreateTime = appDeployRes.CreatedAt.Format("2006-01-02 15:04:05")
	appDeploy.UpdateTime = appDeployRes.UpdatedAt.Format("2006-01-02 15:04:05")
	return appDeploy, nil
}

func (a *AppInterface) ListDeployedApp(ctx context.Context, deployAppReq *v1alpha1.DeployAppReq) (*v1alpha1.DeployAppList, error) {
	bizDeployApp := biz.DeployApp{}
	err := utils.StructTransform(deployAppReq, &bizDeployApp)
	if err != nil {
		return nil, err
	}
	bizDeployApp.ID = deployAppReq.Id
	if deployAppReq.Page == 0 {
		deployAppReq.Page = 1
	}
	if deployAppReq.PageSize == 0 {
		deployAppReq.PageSize = 10
	}
	if deployAppReq.PageSize > 30 {
		deployAppReq.PageSize = 30
	}
	deployApps, count, err := a.uc.DeployAppList(ctx, bizDeployApp, deployAppReq.Page, deployAppReq.PageSize)
	if err != nil {
		return nil, err
	}
	deployAppList := make([]*v1alpha1.DeployApp, len(deployApps))
	for index, deployApp := range deployApps {
		deployAppList[index] = &v1alpha1.DeployApp{}
		err = utils.StructTransform(deployApp, deployAppList[index])
		if err != nil {
			return nil, err
		}
		deployAppList[index].Id = deployApp.ID
	}
	return &v1alpha1.DeployAppList{
		Items: deployAppList,
		Count: count,
	}, nil
}

func (a *AppInterface) StopApp(ctx context.Context, deployAppReq *v1alpha1.DeployAppReq) (*common.Msg, error) {
	if deployAppReq.Id == 0 {
		return nil, errors.New("app deploy id is required")
	}
	err := a.uc.StopApp(ctx, deployAppReq.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) DeployApp(ctx context.Context, deployAppReq *v1alpha1.DeployAppReq) (*v1alpha1.DeployApp, error) {
	if deployAppReq.ClusterId == 0 {
		return nil, errors.New("cluster id is required")
	}
	if deployAppReq.ProjectId == 0 {
		return nil, errors.New("project id is required")
	}
	if deployAppReq.AppTypeId == biz.AppTypeRepo {
		if deployAppReq.AppName == "" || deployAppReq.RepoId == 0 || deployAppReq.Version == "" {
			return nil, errors.New("app name / repo id / version is required")
		}
	} else {
		if deployAppReq.AppId == 0 || deployAppReq.VersionId == 0 {
			return nil, errors.New("app id / version id is required")
		}
	}
	user, err := a.user.GetUserInfo(ctx)
	if err != nil {
		return nil, err
	}
	appDeployRes, err := a.uc.DeployApp(ctx, &biz.DeployApp{
		ID:        deployAppReq.Id,
		ClusterID: deployAppReq.ClusterId,
		ProjectID: deployAppReq.ProjectId,
		AppID:     deployAppReq.AppId,
		VersionID: deployAppReq.VersionId,
		AppName:   deployAppReq.AppName,
		AppTypeID: deployAppReq.AppTypeId,
		RepoID:    deployAppReq.RepoId,
		Version:   deployAppReq.Version,
		UserID:    user.ID,
		Config:    deployAppReq.Config,
	})
	if err != nil {
		return nil, err
	}
	appDeploy := &v1alpha1.DeployApp{}
	err = utils.StructTransform(appDeployRes, appDeploy)
	if err != nil {
		return nil, err
	}
	appDeploy.Id = appDeployRes.ID
	return appDeploy, nil
}

func (a *AppInterface) GetDeployedAppResources(ctx context.Context, deployAppReq *v1alpha1.DeployAppReq) (*v1alpha1.DeployAppResources, error) {
	if deployAppReq.Id == 0 {
		return nil, errors.New("app deploy id is required")
	}
	resources, err := a.uc.GetDeployedResources(ctx, deployAppReq.Id)
	if err != nil {
		return nil, err
	}
	data := &v1alpha1.DeployAppResources{}
	items := make([]*v1alpha1.AppDeployedResource, 0)
	err = utils.StructTransform(resources, &items)
	if err != nil {
		return nil, err
	}
	data.Items = items
	return data, nil
}

func (a *AppInterface) DeleteDeployedApp(ctx context.Context, deployAppReq *v1alpha1.DeployAppReq) (*common.Msg, error) {
	if deployAppReq.Id == 0 {
		return nil, errors.New("app deploy id is required")
	}
	err := a.uc.DeleteDeployedApp(ctx, deployAppReq.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) SaveRepo(ctx context.Context, repo *v1alpha1.AppHelmRepo) (*common.Msg, error) {
	if repo.Name == "" {
		return nil, errors.New("repo name is required")
	}
	if repo.Url == "" {
		return nil, errors.New("repo url is required")
	}
	err := a.uc.SaveRepo(ctx, &biz.AppHelmRepo{
		ID:          repo.Id,
		Name:        repo.Name,
		Url:         repo.Url,
		Description: repo.Description,
	})
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) ListRepo(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.AppHelmRepoList, error) {
	repos, err := a.uc.ListRepo(ctx)
	if err != nil {
		return nil, err
	}
	repoList := &v1alpha1.AppHelmRepoList{
		Items: make([]*v1alpha1.AppHelmRepo, len(repos)),
	}
	for index, repo := range repos {
		repoList.Items[index] = &v1alpha1.AppHelmRepo{
			Id:          repo.ID,
			Name:        repo.Name,
			Url:         repo.Url,
			Description: repo.Description,
		}
	}
	return repoList, nil
}

func (a *AppInterface) DeleteRepo(ctx context.Context, repoReq *v1alpha1.AppHelmRepoReq) (*common.Msg, error) {
	if repoReq.Id == 0 {
		return nil, errors.New("repo id is required")
	}
	err := a.uc.DeleteRepo(ctx, repoReq.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (a *AppInterface) GetAppsByRepo(ctx context.Context, repoReq *v1alpha1.AppHelmRepoReq) (*v1alpha1.AppList, error) {
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
		dataApp.UpdateTime = app.UpdatedAt.Format("2006/01/02")
		appList.Items[index] = dataApp
	}
	return appList, nil
}

func (a *AppInterface) GetAppDetailByRepo(ctx context.Context, repoReq *v1alpha1.AppHelmRepoReq) (*v1alpha1.App, error) {
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
	// mock id
	appRes.Id = repoReq.Id
	return appRes, nil
}

func (a *AppInterface) bizAppToApp(bizApp *biz.App) (*v1alpha1.App, error) {
	app := &v1alpha1.App{
		Id:         bizApp.ID,
		Name:       bizApp.Name,
		Icon:       bizApp.Icon,
		AppTypeId:  bizApp.AppTypeID,
		Versions:   make([]*v1alpha1.AppVersion, len(bizApp.Versions)),
		UpdateTime: bizApp.UpdatedAt.Format("2006/01/02"),
	}
	for index, v := range bizApp.Versions {
		appversion, err := a.bizAppVersionToAppVersion(v)
		if err != nil {
			return nil, err
		}
		// mock id
		appversion.Id = int64(index) + 1
		app.Versions[index] = appversion
	}
	return app, nil
}

func (a *AppInterface) bizAppVersionToAppVersion(bizAppVersion *biz.AppVersion) (*v1alpha1.AppVersion, error) {
	appVersion := &v1alpha1.AppVersion{}
	err := utils.StructTransform(bizAppVersion, &appVersion)
	if err != nil {
		return nil, err
	}
	appVersion.Id = bizAppVersion.ID
	return appVersion, nil
}

func (a *AppInterface) appTobizApp(app *v1alpha1.App) (*biz.App, error) {
	bizApp := &biz.App{}
	err := utils.StructTransform(app, &bizApp)
	if err != nil {
		return nil, err
	}
	bizApp.ID = app.Id
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
	err := utils.StructTransform(appVersion, &bizAppVersion)
	if err != nil {
		return nil, err
	}
	bizAppVersion.ID = appVersion.Id
	return bizAppVersion, nil
}
