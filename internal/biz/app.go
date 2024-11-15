package biz

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	AppPoolNumber = 100

	AppUntested   = "untested"
	AppTested     = "tested"
	AppTestFailed = "test_failed"

	AppTypeAll        = 0
	AppTypeAppPackage = -1 // package
	AppTypeRepo       = -2 // url
)

type AppType struct {
	ID   int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name string `json:"name" gorm:"column:name; default:''; NOT NULL"`
	gorm.Model
}

func DefaultAppType() []*AppType {
	return []*AppType{
		{Name: "All", ID: AppTypeAll},
		{Name: "App Package", ID: AppTypeAppPackage},
		{Name: "Repo", ID: AppTypeRepo},
	}
}

type AppRepo struct {
	ID          int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Url         string `json:"url" gorm:"column:url; default:''; NOT NULL"`
	IndexPath   string `json:"index_path" gorm:"column:index_path; default:''; NOT NULL"`
	Description string `json:"description" gorm:"column:description; default:''; NOT NULL"`
	gorm.Model
}

func (a *AppRepo) SetIndexPath(path string) {
	a.IndexPath = path
}

type App struct {
	ID        int64         `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name      string        `json:"name" gorm:"column:name; default:''; NOT NULL; index"`
	Icon      string        `json:"icon,omitempty" gorm:"column:icon; default:''; NOT NULL"`
	AppTypeID int64         `json:"app_type_id,omitempty" gorm:"column:app_type_id; default:0; NOT NULL"`
	AppRepoID int64         `json:"app_repo_id,omitempty" gorm:"column:app_repo_id; default:0; NOT NULL"`
	Versions  []*AppVersion `json:"versions,omitempty" gorm:"-"`
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
	Status      string `json:"status,omitempty" gorm:"column:status; default:''; NOT NULL"`
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
	Status              string                `json:"status,omitempty" gorm:"column:status; default:''; NOT NULL"`
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

func (v *AppVersion) AppVersionToAppRelease() *AppRelease {
	releaseName := fmt.Sprintf("%s-%s", v.AppName, strings.ReplaceAll(v.Version, ".", "-"))
	return &AppRelease{
		AppID:       v.AppID,
		VersionID:   v.ID,
		Version:     v.Version,
		Chart:       v.Chart,
		AppName:     v.AppName,
		Config:      v.Config,
		ReleaseName: releaseName,
	}
}

type AppData interface {
	Save(context.Context, *App) error
	List(context.Context, *App, int32, int32) ([]*App, int32, error)
	Get(ctx context.Context, appID int64) (*App, error)
	Delete(ctx context.Context, appID, versionID int64) error
	GetByName(context.Context, string) (*App, error)
	CreateAppType(context.Context, *AppType) error
	ListAppType(ctx context.Context) ([]*AppType, error)
	DeleteAppType(context.Context, int64) error
	SaveAppRelease(context.Context, *AppRelease) error
	DeleteAppRelease(context.Context, int64) error
	AppReleaseList(context.Context, AppRelease, int32, int32) ([]*AppRelease, int32, error)
	GetAppRelease(ctx context.Context, id int64) (*AppRelease, error)
	SaveRepo(context.Context, *AppRepo) error
	ListRepo(context.Context) ([]*AppRepo, error)
	GetRepo(context.Context, int64) (*AppRepo, error)
	GetRepoByName(context.Context, string) (*AppRepo, error)
	DeleteRepo(context.Context, int64) error
}

type AppRuntime interface {
	CheckCluster(context.Context) bool
	GetPodResources(context.Context, *AppRelease) ([]*AppReleaseResource, error)
	GetNetResouces(context.Context, *AppRelease) ([]*AppReleaseResource, error)
	GetAppsReouces(context.Context, *AppRelease) ([]*AppReleaseResource, error)
}

type AppConstruct interface {
	GetAppVersionChartInfomation(context.Context, *AppVersion) error
	AppRelease(context.Context, *AppRelease) error
	DeleteAppRelease(context.Context, *AppRelease) error
	AddAppRepo(context.Context, *AppRepo) error
	GetAppDetailByRepo(ctx context.Context, apprepo *AppRepo, appName, version string) (*App, error)
	GetAppsByRepo(context.Context, *AppRepo) ([]*App, error)
	DeleteAppChart(ctx context.Context, app *App, versionId int64) (err error)
}

type AppUsecase struct {
	appData      AppData
	appRuntime   AppRuntime
	appConstruct AppConstruct
	locks        map[int64]*sync.Mutex
	locksMux     sync.Mutex
	eventChan    chan *AppRelease
	conf         *conf.Bootstrap
	log          *log.Helper
}

func NewAppUsecase(appData AppData, appRuntime AppRuntime, appConstruct AppConstruct, logger log.Logger, conf *conf.Bootstrap) *AppUsecase {
	appuc := &AppUsecase{
		appData:      appData,
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

func (uc *AppUsecase) GetAppByName(ctx context.Context, name string) (app *App, err error) {
	return uc.appData.GetByName(ctx, name)
}

func (uc *AppUsecase) List(ctx context.Context, appReq *App, page, pageSize int32) ([]*App, int32, error) {
	return uc.appData.List(ctx, appReq, page, pageSize)
}

func (uc *AppUsecase) Get(ctx context.Context, id, versionId int64) (*App, error) {
	app, err := uc.appData.Get(ctx, id)
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
	return uc.appData.Save(ctx, app)
}

func (uc *AppUsecase) Delete(ctx context.Context, id, versionId int64) error {
	app, err := uc.Get(ctx, id, versionId)
	if err != nil {
		return err
	}
	err = uc.appData.Delete(ctx, id, versionId)
	if err != nil {
		return err
	}
	return uc.appConstruct.DeleteAppChart(ctx, app, versionId)
}

func (uc *AppUsecase) CreateAppType(ctx context.Context, appType *AppType) error {
	return uc.appData.CreateAppType(ctx, appType)
}

func (uc *AppUsecase) ListAppType(ctx context.Context) ([]*AppType, error) {
	return uc.appData.ListAppType(ctx)
}

func (uc *AppUsecase) DeleteAppType(ctx context.Context, appTypeID int64) error {
	return uc.appData.DeleteAppType(ctx, appTypeID)
}

func (uc *AppUsecase) SaveRepo(ctx context.Context, repo *AppRepo) error {
	repoList, err := uc.appData.ListRepo(ctx)
	if err != nil {
		return err
	}
	for _, v := range repoList {
		if v.Name == repo.Name {
			repo.ID = v.ID
		}
	}
	err = uc.appConstruct.AddAppRepo(ctx, repo)
	if err != nil {
		return err
	}
	return uc.appData.SaveRepo(ctx, repo)
}

func (uc *AppUsecase) ListRepo(ctx context.Context) ([]*AppRepo, error) {
	return uc.appData.ListRepo(ctx)
}

func (uc *AppUsecase) DeleteRepo(ctx context.Context, repoID int64) error {
	return uc.appData.DeleteRepo(ctx, repoID)
}

func (uc *AppUsecase) GetAppsByRepo(ctx context.Context, repoID int64) ([]*App, error) {
	repo, err := uc.appData.GetRepo(ctx, repoID)
	if err != nil {
		return nil, err
	}
	return uc.appConstruct.GetAppsByRepo(ctx, repo)
}

func (uc *AppUsecase) GetAppDetailByRepo(ctx context.Context, repoID int64, appName, version string) (*App, error) {
	repos, err := uc.appData.ListRepo(ctx)
	if err != nil {
		return nil, err
	}
	var repo *AppRepo
	for _, v := range repos {
		if v.ID == repoID {
			repo = v
			break
		}
	}
	if repo == nil {
		return nil, errors.New("app repo not found")
	}
	return uc.appConstruct.GetAppDetailByRepo(ctx, repo, appName, version)
}

func (uc *AppUsecase) GetAppRelease(ctx context.Context, id int64) (*AppRelease, error) {
	return uc.appData.GetAppRelease(ctx, id)
}

func (uc *AppUsecase) AppReleaseList(ctx context.Context, appReleaseReq AppRelease, page, pageSize int32) ([]*AppRelease, int32, error) {
	return uc.appData.AppReleaseList(ctx, appReleaseReq, page, pageSize)
}

func (uc *AppUsecase) GetAppVersionChartInfomation(ctx context.Context, appVersion *AppVersion) error {
	return uc.appConstruct.GetAppVersionChartInfomation(ctx, appVersion)
}

func (uc *AppUsecase) GetAppReleaseResourcesInCluster(ctx context.Context, appReleaseID int64) ([]*AppReleaseResource, error) {
	appRelease, err := uc.appData.GetAppRelease(ctx, appReleaseID)
	if err != nil {
		return nil, err
	}
	resources := make([]*AppReleaseResource, 0)
	resourcesFunc := []func(ctx context.Context, appRelease *AppRelease) ([]*AppReleaseResource, error){
		uc.appRuntime.GetPodResources,
		uc.appRuntime.GetNetResouces,
		uc.appRuntime.GetAppsReouces,
	}
	for _, f := range resourcesFunc {
		res, err := f(ctx, appRelease)
		if err != nil {
			return nil, err
		}
		resources = append(resources, res...)
	}
	return resources, nil
}

func (uc *AppUsecase) SaveAppRelease(ctx context.Context, appReleaseReq *AppRelease) (appRelease *AppRelease, err error) {
	var appVersion *AppVersion
	if appReleaseReq.AppTypeID == AppTypeRepo {
		app, err := uc.GetAppDetailByRepo(ctx, appReleaseReq.RepoID, appReleaseReq.AppName, appReleaseReq.Version)
		if err != nil {
			return nil, err
		}
		appVersion = app.GetVersion(appReleaseReq.Version)
	}
	if appReleaseReq.AppTypeID != AppTypeRepo {
		app, err := uc.Get(ctx, appReleaseReq.AppID, appReleaseReq.VersionID)
		if err != nil {
			return nil, err
		}
		appVersion = app.GetVersionById(appReleaseReq.VersionID)
	}
	if appVersion == nil {
		return nil, errors.New("app version not found or not support app type")
	}
	if appReleaseReq.ID != 0 {
		appRelease, err = uc.appData.GetAppRelease(ctx, appReleaseReq.ID)
		if err != nil {
			return nil, err
		}
		appRelease.Config = appReleaseReq.Config
		err = uc.appData.SaveAppRelease(ctx, appRelease)
		if err != nil {
			return nil, err
		}
		uc.apply(appRelease)
		return appRelease, nil
	}
	appReleaseRes := appVersion.AppVersionToAppRelease()
	appRelease = appReleaseReq
	appRelease.ReleaseName = appReleaseRes.ReleaseName
	appRelease.Chart = appReleaseRes.Chart
	appRelease.AppID = appReleaseRes.AppID
	appRelease.VersionID = appReleaseRes.VersionID
	if appRelease.Config == "" {
		appRelease.Config = appReleaseRes.Config
	}
	err = uc.appData.SaveAppRelease(ctx, appRelease)
	if err != nil {
		return nil, err
	}
	uc.apply(appRelease)
	return appRelease, nil
}

func (uc *AppUsecase) DeleteAppRelease(ctx context.Context, id int64) error {
	err := uc.appData.DeleteAppRelease(ctx, id)
	if err != nil {
		return err
	}
	appRelease, err := uc.appData.GetAppRelease(ctx, id)
	if err != nil {
		return err
	}
	uc.apply(appRelease)
	return nil
}

func (uc *AppUsecase) apply(appRelease *AppRelease) {
	uc.eventChan <- appRelease
}

func (uc *AppUsecase) appReleaseRunner() {
	ctx := context.Background()
	if !uc.appRuntime.CheckCluster(ctx) {
		return
	}
	// todo default app check
	for event := range uc.eventChan {
		go func(event *AppRelease) {
			err := uc.handleEvent(ctx, event)
			if err != nil {
				uc.log.Errorf("Failed to app release handle event: %v", err)
			}
		}(event)
	}
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

func (uc *AppUsecase) handleEvent(ctx context.Context, appRelease *AppRelease) (err error) {
	lock := uc.getLock(appRelease.ID)
	lock.Lock()
	defer func() {
		uc.appData.SaveAppRelease(ctx, appRelease)
		lock.Unlock()
	}()
	if appRelease.DeletedAt.Valid {
		err = uc.appConstruct.DeleteAppRelease(ctx, appRelease)
		if err != nil {
			return err
		}
	}
	err = uc.appConstruct.AppRelease(ctx, appRelease)
	if err != nil {
		return err
	}
	return nil
}
