package biz

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

const (
	AppPackageName     = "app"
	AppPackageRepoName = "repo"
	AppPathckageIcon   = "icon"
)

type AppType struct {
	ID   int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name string `json:"name" gorm:"column:name; default:''; NOT NULL"`
	gorm.Model
}

const (
	AppTypeAll        = 0
	AppTypeAppPackage = -1
	AppTypeRepo       = -2
)

func DefaultAppType() []*AppType {
	return []*AppType{
		{Name: "All", ID: AppTypeAll},
		{Name: "App Package", ID: AppTypeAppPackage},
		{Name: "Repo", ID: AppTypeRepo},
	}
}

type AppHelmRepo struct {
	ID          int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Url         string `json:"url" gorm:"column:url; default:''; NOT NULL"`
	IndexPath   string `json:"index_path" gorm:"column:index_path; default:''; NOT NULL"`
	Description string `json:"description" gorm:"column:description; default:''; NOT NULL"`
	gorm.Model
}

func (a *AppHelmRepo) SetIndexPath(path string) {
	a.IndexPath = path
}

type App struct {
	ID            int64         `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name          string        `json:"name" gorm:"column:name; default:''; NOT NULL; index"`
	Icon          string        `json:"icon,omitempty" gorm:"column:icon; default:''; NOT NULL"`
	AppTypeID     int64         `json:"app_type_id,omitempty" gorm:"column:app_type_id; default:0; NOT NULL"`
	AppHelmRepoID int64         `json:"app_helm_repo_id,omitempty" gorm:"column:app_helm_repo_id; default:0; NOT NULL"`
	Versions      []*AppVersion `json:"versions,omitempty" gorm:"-"`
	gorm.Model
}

type AppVersion struct {
	ID          int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	AppID       int64  `json:"app_id" gorm:"column:app_id; default:0; NOT NULL; index"`
	AppName     string `json:"app_name,omitempty" gorm:"column:app_name; default:''; NOT NULL"`
	Name        string `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Chart       string `json:"chart,omitempty" gorm:"column:chart; default:''; NOT NULL"`
	Version     string `json:"version,omitempty" gorm:"column:version; default:''; NOT NULL; index"`
	Config      string `json:"config,omitempty" gorm:"column:config; default:''; NOT NULL"`
	Readme      string `json:"readme,omitempty" gorm:"-"`
	State       string `json:"state,omitempty" gorm:"column:state; default:''; NOT NULL"`
	TestResult  string `json:"test_result,omitempty" gorm:"column:test_result; default:''; NOT NULL"` // 哪些资源部署成功，哪些失败
	Description string `json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`
	Metadata    []byte `json:"metadata,omitempty" gorm:"-"`
	gorm.Model
}

type DeployApp struct {
	ID          int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	ReleaseName string `json:"release_name,omitempty" gorm:"column:release_name; default:''; NOT NULL"`
	AppID       int64  `json:"app_id" gorm:"column:app_id; default:0; NOT NULL; index"`
	VersionID   int64  `json:"version_id" gorm:"column:version_id; default:0; NOT NULL; index"`
	Version     string `json:"version,omitempty" gorm:"column:version; default:''; NOT NULL"`
	RepoID      int64  `json:"repo_id,omitempty" gorm:"column:repo_id; default:0; NOT NULL"`
	AppName     string `json:"app_name,omitempty" gorm:"column:app_name; default:''; NOT NULL"`
	AppTypeID   int64  `json:"app_type_id,omitempty" gorm:"column:app_type_id; default:0; NOT NULL"`
	Chart       string `json:"chart,omitempty" gorm:"column:chart; default:''; NOT NULL"`
	ClusterID   int64  `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL; index"`
	ProjectID   int64  `json:"project_id" gorm:"column:project_id; default:0; NOT NULL; index"`
	UserID      int64  `json:"user_id" gorm:"column:user_id; default:0; NOT NULL; index"`
	Namespace   string `json:"namespace,omitempty" gorm:"column:namespace; default:''; NOT NULL"`
	Config      string `json:"config,omitempty" gorm:"column:config; default:''; NOT NULL"`
	State       string `json:"state,omitempty" gorm:"column:state; default:''; NOT NULL"`
	IsTest      bool   `json:"is_test,omitempty" gorm:"column:is_test; default:false; NOT NULL"`
	Manifest    string `json:"manifest,omitempty" gorm:"column:manifest; default:''; NOT NULL"` // also template | yaml
	Notes       string `json:"notes,omitempty" gorm:"column:notes; default:''; NOT NULL"`
	Logs        string `json:"logs,omitempty" gorm:"column:logs; default:''; NOT NULL"`
	gorm.Model
}

const (
	AppUntested   = "untested"
	AppTested     = "tested"
	AppTestFailed = "test_failed"
)

func (a *App) AddVersion(version *AppVersion) {
	if a.Versions == nil {
		a.Versions = make([]*AppVersion, 0)
	}
	a.Versions = append(a.Versions, version)
}

func (a *App) UpdateVersion(version *AppVersion) {
	for index, v := range a.Versions {
		if v.Version == version.Version {
			versionID := v.ID
			appID := v.AppID
			appName := v.AppName
			a.Versions[index] = version
			a.Versions[index].ID = versionID
			a.Versions[index].AppID = appID
			a.Versions[index].AppName = appName
			return
		}
	}
}

func (a *App) GetVersion(version string) *AppVersion {
	for _, v := range a.Versions {
		if version == "" {
			return v
		}
		if v.Version == version {
			return v
		}
	}
	return nil
}

func (a *App) GetVersionById(id int64) *AppVersion {
	for _, v := range a.Versions {
		if id == 0 {
			return v
		}
		if v.ID == id {
			return v
		}
	}
	return nil
}

func (a *App) DeleteVersion(version string) {
	for index, v := range a.Versions {
		if v.Version == version {
			a.Versions = append(a.Versions[:index], a.Versions[index+1:]...)
			return
		}
	}
}

func (v *AppVersion) GetAppDeployed() *DeployApp {
	releaseName := fmt.Sprintf("%s-%s", v.AppName, strings.ReplaceAll(v.Version, ".", "-"))
	return &DeployApp{
		AppID:       v.AppID,
		VersionID:   v.ID,
		Version:     v.Version,
		Chart:       v.Chart,
		AppName:     v.AppName,
		Namespace:   "default",
		Config:      v.Config,
		ReleaseName: releaseName,
	}
}

type AppRepo interface {
	Save(context.Context, *App) error
	List(ctx context.Context, appReq *App, page, pageSize int32) ([]*App, int32, error)
	Get(ctx context.Context, appID int64) (*App, error)
	GetByName(ctx context.Context, name string) (*App, error)
	Delete(ctx context.Context, appID, versionID int64) error
	CreateAppType(ctx context.Context, appType *AppType) error
	ListAppType(ctx context.Context) ([]*AppType, error)
	DeleteAppType(ctx context.Context, appTypeID int64) error
	SaveDeployApp(ctx context.Context, appDeployed *DeployApp) error
	DeleteDeployApp(ctx context.Context, id int64) error
	DeployAppList(ctx context.Context, appDeployedReq DeployApp, page, pageSuze int32) ([]*DeployApp, int32, error)
	GetDeployApp(ctx context.Context, id int64) (*DeployApp, error)
	SaveRepo(ctx context.Context, helmRepo *AppHelmRepo) error
	ListRepo(ctx context.Context) ([]*AppHelmRepo, error)
	GetRepo(ctx context.Context, helmRepoID int64) (*AppHelmRepo, error)
	GetRepoByName(ctx context.Context, repoName string) (*AppHelmRepo, error)
	DeleteRepo(ctx context.Context, helmRepoID int64) error
}

type SailorRepo interface {
	Create(context.Context, *DeployApp) error
}

type AppRuntime interface {
	GetPodResources(context.Context, *DeployApp) ([]*AppDeployedResource, error)
	GetNetResouces(context.Context, *DeployApp) ([]*AppDeployedResource, error)
	GetAppsReouces(context.Context, *DeployApp) ([]*AppDeployedResource, error)
}

type AppConstruct interface {
	GetAppVersionChartInfomation(context.Context, *AppVersion) error
	DeployingApp(context.Context, *DeployApp) error
	UnDeployingApp(context.Context, *DeployApp) error
	AddAppRepo(context.Context, *AppHelmRepo) error
	GetAppDetailByRepo(ctx context.Context, apprepo *AppHelmRepo, appName, version string) (*App, error)
	GetAppsByRepo(context.Context, *AppHelmRepo) ([]*App, error)
	DeleteAppChart(ctx context.Context, app *App, versionId int64) (err error)
}

type AppUsecase struct {
	repo         AppRepo
	log          *log.Helper
	c            *conf.Bootstrap
	clusterRepo  ClusterRepo
	projectRepo  ProjectRepo
	sailorRepo   SailorRepo
	appRuntime   AppRuntime
	appConstruct AppConstruct
}

func NewAppUsecase(repo AppRepo,
	clusterRepo ClusterRepo,
	projectRepo ProjectRepo,
	sailorRepo SailorRepo,
	appRuntime AppRuntime,
	appConstruct AppConstruct, logger log.Logger, c *conf.Bootstrap) *AppUsecase {
	return &AppUsecase{repo, log.NewHelper(logger), c, clusterRepo, projectRepo, sailorRepo, appRuntime, appConstruct}
}

func (uc *AppUsecase) GetAppByName(ctx context.Context, name string) (app *App, err error) {
	return uc.repo.GetByName(ctx, name)
}

func (uc *AppUsecase) List(ctx context.Context, appReq *App, page, pageSize int32) ([]*App, int32, error) {
	return uc.repo.List(ctx, appReq, page, pageSize)
}

func (uc *AppUsecase) Get(ctx context.Context, id, versionId int64) (*App, error) {
	app, err := uc.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	appVersion := app.GetVersionById(versionId)
	if appVersion == nil {
		return app, nil
	}
	err = uc.appConstruct.GetAppVersionChartInfomation(ctx, appVersion)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (uc *AppUsecase) Save(ctx context.Context, app *App) error {
	return uc.repo.Save(ctx, app)
}

func (uc *AppUsecase) Delete(ctx context.Context, id, versionId int64) error {
	app, err := uc.Get(ctx, id, versionId)
	if err != nil {
		return err
	}
	err = uc.repo.Delete(ctx, id, versionId)
	if err != nil {
		return err
	}
	return uc.appConstruct.DeleteAppChart(ctx, app, versionId)
}

func (uc *AppUsecase) CreateAppType(ctx context.Context, appType *AppType) error {
	return uc.repo.CreateAppType(ctx, appType)
}

func (uc *AppUsecase) ListAppType(ctx context.Context) ([]*AppType, error) {
	return uc.repo.ListAppType(ctx)
}

func (uc *AppUsecase) DeleteAppType(ctx context.Context, appTypeID int64) error {
	return uc.repo.DeleteAppType(ctx, appTypeID)
}

func (uc *AppUsecase) GetAppDeployed(ctx context.Context, id int64) (*DeployApp, error) {
	return uc.repo.GetDeployApp(ctx, id)
}

func (uc *AppUsecase) DeployAppList(ctx context.Context, appDeployedReq DeployApp, page, pageSize int32) ([]*DeployApp, int32, error) {
	return uc.repo.DeployAppList(ctx, appDeployedReq, page, pageSize)
}

func (uc *AppUsecase) AppOperation(ctx context.Context, deployedApp *DeployApp) error {
	return uc.sailorRepo.Create(ctx, deployedApp)
}

func (uc *AppUsecase) GetAppVersionChartInfomation(ctx context.Context, appVersion *AppVersion) error {
	return uc.appConstruct.GetAppVersionChartInfomation(ctx, appVersion)
}

func (uc *AppUsecase) AppTest(ctx context.Context, appID, versionID int64) (*DeployApp, error) {
	app, err := uc.Get(ctx, appID, versionID)
	if err != nil {
		return nil, err
	}
	appVersion := app.GetVersionById(versionID)
	if appVersion == nil {
		return nil, errors.New("app version not found")
	}
	appDeployed := appVersion.GetAppDeployed()
	appDeployed.IsTest = true
	deployAppErr := uc.appConstruct.DeployingApp(ctx, appDeployed)
	if deployAppErr != nil {
		appVersion.State = AppTestFailed
	}
	if deployAppErr == nil {
		appVersion.State = AppTested
		appVersion.TestResult = "success"
	}
	err = uc.repo.Save(ctx, app)
	if err != nil {
		return nil, err
	}
	return appDeployed, deployAppErr
}

func (uc *AppUsecase) DeployApp(ctx context.Context, deployAppReq *DeployApp) (*DeployApp, error) {
	var app *App
	var appVersion *AppVersion
	var err error
	if deployAppReq.AppTypeID == AppTypeRepo {
		app, err = uc.GetAppDetailByRepo(ctx, deployAppReq.RepoID, deployAppReq.AppName, deployAppReq.Version)
		if err != nil {
			return nil, err
		}
		appVersion = app.GetVersion(deployAppReq.Version)
	}
	if deployAppReq.AppTypeID != AppTypeRepo {
		app, err = uc.Get(ctx, deployAppReq.AppID, deployAppReq.VersionID)
		if err != nil {
			return nil, err
		}
		appVersion = app.GetVersionById(deployAppReq.VersionID)
	}
	appDeployed := appVersion.GetAppDeployed()
	appDeployed.ID = deployAppReq.ID
	appDeployed.RepoID = deployAppReq.RepoID
	appDeployed.AppTypeID = app.AppTypeID
	appDeployed.ClusterID = deployAppReq.ClusterID
	appDeployed.ProjectID = deployAppReq.ProjectID
	appDeployed.Namespace = deployAppReq.Namespace
	appDeployed.Config = deployAppReq.Config
	appDeployed.UserID = deployAppReq.UserID
	if deployAppReq.ID != 0 {
		appDeployedRes, err := uc.repo.GetDeployApp(ctx, deployAppReq.ID)
		if err != nil {
			return nil, err
		}
		appDeployed.ReleaseName = appDeployedRes.ReleaseName
	}
	deployAppErr := uc.appConstruct.DeployingApp(ctx, appDeployed)
	err = uc.repo.SaveDeployApp(ctx, appDeployed)
	if err != nil {
		return nil, err
	}
	return appDeployed, deployAppErr
}

func (uc *AppUsecase) DeleteDeployedApp(ctx context.Context, id int64) error {
	appDeployed, err := uc.repo.GetDeployApp(ctx, id)
	if err != nil {
		return err
	}
	if appDeployed == nil {
		return nil
	}
	err = uc.appConstruct.UnDeployingApp(ctx, appDeployed)
	if err != nil {
		return err
	}
	err = uc.repo.DeleteDeployApp(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

func (uc *AppUsecase) StopApp(ctx context.Context, id int64) error {
	appDeployed, err := uc.repo.GetDeployApp(ctx, id)
	if err != nil {
		return err
	}
	if appDeployed == nil {
		return errors.New("app deployed not found")
	}
	unDeployAppErr := uc.appConstruct.UnDeployingApp(ctx, appDeployed)
	err = uc.repo.SaveDeployApp(ctx, appDeployed)
	if err != nil {
		return err
	}
	return unDeployAppErr
}

// 保存repo
func (uc *AppUsecase) SaveRepo(ctx context.Context, helmRepo *AppHelmRepo) error {
	repoList, err := uc.repo.ListRepo(ctx)
	if err != nil {
		return err
	}
	for _, v := range repoList {
		if v.Name == helmRepo.Name {
			helmRepo.ID = v.ID
		}
	}
	err = uc.appConstruct.AddAppRepo(ctx, helmRepo)
	if err != nil {
		return err
	}
	return uc.repo.SaveRepo(ctx, helmRepo)
}

// repo列表
func (uc *AppUsecase) ListRepo(ctx context.Context) ([]*AppHelmRepo, error) {
	return uc.repo.ListRepo(ctx)
}

// 删除repo
func (uc *AppUsecase) DeleteRepo(ctx context.Context, helmRepoID int64) error {
	return uc.repo.DeleteRepo(ctx, helmRepoID)
}

// 根据repo获取app列表
func (uc *AppUsecase) GetAppsByRepo(ctx context.Context, helmRepoID int64) ([]*App, error) {
	helmRepo, err := uc.repo.GetRepo(ctx, helmRepoID)
	if err != nil {
		return nil, err
	}
	return uc.appConstruct.GetAppsByRepo(ctx, helmRepo)
}

// 根据repo获取app详情包含app version
func (uc *AppUsecase) GetAppDetailByRepo(ctx context.Context, helmRepoID int64, appName, version string) (*App, error) {
	helmRepos, err := uc.repo.ListRepo(ctx)
	if err != nil {
		return nil, err
	}
	var helmRepo *AppHelmRepo
	for _, v := range helmRepos {
		if v.ID == helmRepoID {
			helmRepo = v
			break
		}
	}
	if helmRepo == nil {
		return nil, errors.New("helm repo not found")
	}
	return uc.appConstruct.GetAppDetailByRepo(ctx, helmRepo, appName, version)
}

type AppDeployedResource struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Events    []string `json:"events"`
	StartedAt string   `json:"started_at"`
	Status    []string `json:"status"`
}

func (uc *AppUsecase) GetDeployedResources(ctx context.Context, appDeployID int64) ([]*AppDeployedResource, error) {
	appDeployed, err := uc.repo.GetDeployApp(ctx, appDeployID)
	if err != nil {
		return nil, err
	}
	resources := make([]*AppDeployedResource, 0)
	resourcesFunc := []func(ctx context.Context, appDeployed *DeployApp) ([]*AppDeployedResource, error){
		uc.appRuntime.GetPodResources,
		uc.appRuntime.GetNetResouces,
		uc.appRuntime.GetAppsReouces,
	}
	for _, f := range resourcesFunc {
		res, err := f(ctx, appDeployed)
		if err != nil {
			return nil, err
		}
		resources = append(resources, res...)
	}
	return resources, nil
}

// 默认app安装
func (uc *AppUsecase) BaseInstallation(ctx context.Context, cluster *Cluster, project *Project) error {
	configMaps := make([]map[string]interface{}, 0)
	conf := reflect.ValueOf(uc.c)
	for i := 0; i < conf.NumField(); i++ {
		filed := conf.Field(i)
		if filed.Kind() != reflect.Map {
			continue
		}
		if configMap, ok := filed.Interface().(map[string]interface{}); ok {
			configMaps = append(configMaps, configMap)
		}
	}
	for _, configMap := range configMaps {
		enable, enableOk := utils.GetValueFromNestedMap(configMap, "base.enable")
		if !enableOk || !cast.ToBool(enable) {
			continue
		}
		repoUrl, repoUrlOk := utils.GetValueFromNestedMap(configMap, "base.repo_url")
		if !repoUrlOk {
			continue
		}
		repoName, repoNameOK := utils.GetValueFromNestedMap(configMap, "base.repo_name")
		if !repoNameOK {
			continue
		}
		appVersion, appVersionOK := utils.GetValueFromNestedMap(configMap, "base.version")
		if !appVersionOK {
			continue
		}
		chartName, chartNameOK := utils.GetValueFromNestedMap(configMap, "base.chart_name")
		if !chartNameOK {
			continue
		}
		namespace, namespaceOK := utils.GetValueFromNestedMap(configMap, "base.namespace")
		if !namespaceOK {
			namespace = "default"
		}
		repo, err := uc.repo.GetRepoByName(ctx, cast.ToString(repoName))
		if err != nil {
			return err
		}
		if repo == nil || repo.ID == 0 {
			repo = &AppHelmRepo{Name: cast.ToString(repoName), Url: cast.ToString(repoUrl)}
			err = uc.SaveRepo(ctx, repo)
			if err != nil {
				return err
			}
		}
		app, err := uc.GetAppByName(ctx, cast.ToString(chartName))
		if err != nil {
			return err
		}
		if app != nil && app.ID > 0 {
			continue
		}
		delete(configMap, "base")
		appConfigYamlByte, err := yaml.Marshal(configMap)
		if err != nil {
			return err
		}
		deployApp := &DeployApp{
			ClusterID: cluster.ID,
			AppName:   cast.ToString(chartName),
			AppTypeID: AppTypeRepo,
			RepoID:    repo.ID,
			Version:   cast.ToString(appVersion),
			Config:    string(appConfigYamlByte),
			Namespace: cast.ToString(namespace),
		}
		if project != nil {
			deployApp.ProjectID = project.ID
			deployApp.Namespace = project.Namespace
		}
		_, err = uc.DeployApp(ctx, deployApp)
		if err != nil {
			return err
		}
		err = uc.AppOperation(ctx, deployApp)
		if err != nil {
			return err
		}
	}
	return nil
}
