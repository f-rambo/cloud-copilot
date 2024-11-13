package biz

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

const (
	AppPoolNumber = 100

	AppPackageName     = "app"
	AppPackageRepoName = "repo"
	AppPathckageIcon   = "icon"

	AppUntested   = "untested"
	AppTested     = "tested"
	AppTestFailed = "test_failed"
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
	TestResult  string `json:"test_result,omitempty" gorm:"column:test_result; default:''; NOT NULL"`
	Description string `json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`
	Metadata    []byte `json:"metadata,omitempty" gorm:"-"`
	gorm.Model
}

type AppRelease struct {
	ID                  int64                 `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	ReleaseName         string                `json:"release_name,omitempty" gorm:"column:release_name; default:''; NOT NULL"`
	AppID               int64                 `json:"app_id" gorm:"column:app_id; default:0; NOT NULL; index"`
	VersionID           int64                 `json:"version_id" gorm:"column:version_id; default:0; NOT NULL; index"`
	Version             string                `json:"version,omitempty" gorm:"column:version; default:''; NOT NULL"`
	RepoID              int64                 `json:"repo_id,omitempty" gorm:"column:repo_id; default:0; NOT NULL"`
	AppName             string                `json:"app_name,omitempty" gorm:"column:app_name; default:''; NOT NULL"`
	AppTypeID           int64                 `json:"app_type_id,omitempty" gorm:"column:app_type_id; default:0; NOT NULL"`
	Chart               string                `json:"chart,omitempty" gorm:"column:chart; default:''; NOT NULL"`
	ClusterID           int64                 `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL; index"`
	ProjectID           int64                 `json:"project_id" gorm:"column:project_id; default:0; NOT NULL; index"`
	UserID              int64                 `json:"user_id" gorm:"column:user_id; default:0; NOT NULL; index"`
	Namespace           string                `json:"namespace,omitempty" gorm:"column:namespace; default:''; NOT NULL"`
	Config              string                `json:"config,omitempty" gorm:"column:config; default:''; NOT NULL"`
	State               string                `json:"state,omitempty" gorm:"column:state; default:''; NOT NULL"`
	IsTest              bool                  `json:"is_test,omitempty" gorm:"column:is_test; default:false; NOT NULL"`
	Manifest            string                `json:"manifest,omitempty" gorm:"column:manifest; default:''; NOT NULL"`
	Notes               string                `json:"notes,omitempty" gorm:"column:notes; default:''; NOT NULL"`
	Logs                string                `json:"logs,omitempty" gorm:"column:logs; default:''; NOT NULL"`
	AppReleaseResources []*AppReleaseResource `json:"resources,omitempty" gorm:"-"`
	gorm.Model
}

type AppReleaseResource struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Events    []string `json:"events"`
	StartedAt string   `json:"started_at"`
	Status    []string `json:"status"`
}

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

func (v *AppVersion) GetAppDeployed() *AppRelease {
	releaseName := fmt.Sprintf("%s-%s", v.AppName, strings.ReplaceAll(v.Version, ".", "-"))
	return &AppRelease{
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
	List(context.Context, *App, int32, int32) ([]*App, int32, error)
	Get(ctx context.Context, appID int64) (*App, error)
	Delete(ctx context.Context, appID, versionID int64) error
	GetByName(context.Context, string) (*App, error)
	CreateAppType(context.Context, *AppType) error
	ListAppType(ctx context.Context) ([]*AppType, error)
	DeleteAppType(context.Context, int64) error
	SaveDeployApp(context.Context, *AppRelease) error
	DeleteDeployApp(context.Context, int64) error
	DeployAppList(context.Context, AppRelease, int32, int32) ([]*AppRelease, int32, error)
	GetDeployApp(ctx context.Context, id int64) (*AppRelease, error)
	SaveRepo(context.Context, *AppHelmRepo) error
	ListRepo(context.Context) ([]*AppHelmRepo, error)
	GetRepo(context.Context, int64) (*AppHelmRepo, error)
	GetRepoByName(context.Context, string) (*AppHelmRepo, error)
	DeleteRepo(context.Context, int64) error
}

type AppRuntime interface {
	GetPodResources(context.Context, *AppRelease) ([]*AppReleaseResource, error)
	GetNetResouces(context.Context, *AppRelease) ([]*AppReleaseResource, error)
	GetAppsReouces(context.Context, *AppRelease) ([]*AppReleaseResource, error)
}

type AppConstruct interface {
	GetAppVersionChartInfomation(context.Context, *AppVersion) error
	DeployingApp(context.Context, *AppRelease) error
	UnDeployingApp(context.Context, *AppRelease) error
	AddAppRepo(context.Context, *AppHelmRepo) error
	GetAppDetailByRepo(ctx context.Context, apprepo *AppHelmRepo, appName, version string) (*App, error)
	GetAppsByRepo(context.Context, *AppHelmRepo) ([]*App, error)
	DeleteAppChart(ctx context.Context, app *App, versionId int64) (err error)
}

type AppUsecase struct {
	appRepo      AppRepo
	appRuntime   AppRuntime
	appConstruct AppConstruct
	locks        map[int64]*sync.Mutex
	locksMux     sync.Mutex
	eventChan    chan *AppRelease
	conf         *conf.Bootstrap
	log          *log.Helper
}

func NewAppUsecase(appRepo AppRepo, appRuntime AppRuntime, appConstruct AppConstruct, logger log.Logger, conf *conf.Bootstrap) *AppUsecase {
	appuc := &AppUsecase{
		appRepo:      appRepo,
		appRuntime:   appRuntime,
		appConstruct: appConstruct,
		conf:         conf,
		log:          log.NewHelper(logger),
		locks:        make(map[int64]*sync.Mutex),
		eventChan:    make(chan *AppRelease, AppPoolNumber),
	}
	go appuc.appReleaseRunner()
	return appuc
}

func (uc *AppUsecase) getLock(appID int64) *sync.Mutex {
	uc.locksMux.Lock()
	defer uc.locksMux.Unlock()

	if appID < 0 {
		uc.log.Errorf("Invalid appID: %d", appID)
		return &sync.Mutex{}
	}

	if _, exists := uc.locks[appID]; !exists {
		uc.locks[appID] = &sync.Mutex{}
	}
	return uc.locks[appID]
}

func (uc *AppUsecase) apply(appRelease *AppRelease) {
	uc.eventChan <- appRelease
}

func (uc *AppUsecase) appReleaseRunner() {
	for event := range uc.eventChan {
		go uc.handleEvent(event)
	}
}

func (uc *AppUsecase) handleEvent(appRelease *AppRelease) (err error) {
	lock := uc.getLock(appRelease.ID)
	lock.Lock()
	defer func() {
		if err != nil {
			uc.log.Errorf("Reconcile error: %v", err)
		}
		lock.Unlock()
	}()
	fmt.Println("Reconcile", appRelease)
	return nil
}

func (uc *AppUsecase) GetAppByName(ctx context.Context, name string) (app *App, err error) {
	return uc.appRepo.GetByName(ctx, name)
}

func (uc *AppUsecase) List(ctx context.Context, appReq *App, page, pageSize int32) ([]*App, int32, error) {
	return uc.appRepo.List(ctx, appReq, page, pageSize)
}

func (uc *AppUsecase) Get(ctx context.Context, id, versionId int64) (*App, error) {
	app, err := uc.appRepo.Get(ctx, id)
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
	return uc.appRepo.Save(ctx, app)
}

func (uc *AppUsecase) Delete(ctx context.Context, id, versionId int64) error {
	app, err := uc.Get(ctx, id, versionId)
	if err != nil {
		return err
	}
	err = uc.appRepo.Delete(ctx, id, versionId)
	if err != nil {
		return err
	}
	return uc.appConstruct.DeleteAppChart(ctx, app, versionId)
}

func (uc *AppUsecase) CreateAppType(ctx context.Context, appType *AppType) error {
	return uc.appRepo.CreateAppType(ctx, appType)
}

func (uc *AppUsecase) ListAppType(ctx context.Context) ([]*AppType, error) {
	return uc.appRepo.ListAppType(ctx)
}

func (uc *AppUsecase) DeleteAppType(ctx context.Context, appTypeID int64) error {
	return uc.appRepo.DeleteAppType(ctx, appTypeID)
}

func (uc *AppUsecase) GetAppDeployed(ctx context.Context, id int64) (*AppRelease, error) {
	return uc.appRepo.GetDeployApp(ctx, id)
}

func (uc *AppUsecase) DeployAppList(ctx context.Context, appDeployedReq AppRelease, page, pageSize int32) ([]*AppRelease, int32, error) {
	return uc.appRepo.DeployAppList(ctx, appDeployedReq, page, pageSize)
}

func (uc *AppUsecase) GetAppVersionChartInfomation(ctx context.Context, appVersion *AppVersion) error {
	return uc.appConstruct.GetAppVersionChartInfomation(ctx, appVersion)
}

func (uc *AppUsecase) AppTest(ctx context.Context, appID, versionID int64) (*AppRelease, error) {
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
	err = uc.appRepo.Save(ctx, app)
	if err != nil {
		return nil, err
	}
	return appDeployed, deployAppErr
}

func (uc *AppUsecase) DeployApp(ctx context.Context, deployAppReq *AppRelease) (*AppRelease, error) {
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
		appDeployedRes, err := uc.appRepo.GetDeployApp(ctx, deployAppReq.ID)
		if err != nil {
			return nil, err
		}
		appDeployed.ReleaseName = appDeployedRes.ReleaseName
	}
	deployAppErr := uc.appConstruct.DeployingApp(ctx, appDeployed)
	err = uc.appRepo.SaveDeployApp(ctx, appDeployed)
	if err != nil {
		return nil, err
	}
	uc.apply(appDeployed)
	return appDeployed, deployAppErr
}

func (uc *AppUsecase) DeleteDeployedApp(ctx context.Context, id int64) error {
	appDeployed, err := uc.appRepo.GetDeployApp(ctx, id)
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
	err = uc.appRepo.DeleteDeployApp(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

func (uc *AppUsecase) StopApp(ctx context.Context, id int64) error {
	appDeployed, err := uc.appRepo.GetDeployApp(ctx, id)
	if err != nil {
		return err
	}
	if appDeployed == nil {
		return errors.New("app deployed not found")
	}
	unDeployAppErr := uc.appConstruct.UnDeployingApp(ctx, appDeployed)
	err = uc.appRepo.SaveDeployApp(ctx, appDeployed)
	if err != nil {
		return err
	}
	return unDeployAppErr
}

// 保存repo
func (uc *AppUsecase) SaveRepo(ctx context.Context, helmRepo *AppHelmRepo) error {
	repoList, err := uc.appRepo.ListRepo(ctx)
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
	return uc.appRepo.SaveRepo(ctx, helmRepo)
}

func (uc *AppUsecase) ListRepo(ctx context.Context) ([]*AppHelmRepo, error) {
	return uc.appRepo.ListRepo(ctx)
}

func (uc *AppUsecase) DeleteRepo(ctx context.Context, helmRepoID int64) error {
	return uc.appRepo.DeleteRepo(ctx, helmRepoID)
}

func (uc *AppUsecase) GetAppsByRepo(ctx context.Context, helmRepoID int64) ([]*App, error) {
	helmRepo, err := uc.appRepo.GetRepo(ctx, helmRepoID)
	if err != nil {
		return nil, err
	}
	return uc.appConstruct.GetAppsByRepo(ctx, helmRepo)
}

func (uc *AppUsecase) GetAppDetailByRepo(ctx context.Context, helmRepoID int64, appName, version string) (*App, error) {
	helmRepos, err := uc.appRepo.ListRepo(ctx)
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

func (uc *AppUsecase) GetReleaseResources(ctx context.Context, appDeployID int64) ([]*AppReleaseResource, error) {
	appDeployed, err := uc.appRepo.GetDeployApp(ctx, appDeployID)
	if err != nil {
		return nil, err
	}
	resources := make([]*AppReleaseResource, 0)
	resourcesFunc := []func(ctx context.Context, appDeployed *AppRelease) ([]*AppReleaseResource, error){
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

func (uc *AppUsecase) BaseInstallation(ctx context.Context, cluster *Cluster, project *Project) error {
	configMaps := make([]map[string]interface{}, 0)
	conf := reflect.ValueOf(uc.conf)
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
		repo, err := uc.appRepo.GetRepoByName(ctx, cast.ToString(repoName))
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
		deployApp := &AppRelease{
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
	}
	return nil
}
