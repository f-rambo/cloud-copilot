package biz

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type AppReleaseSatus string

const (
	AppPoolNumber = 100

	AppReleaseSatusPending AppReleaseSatus = "pending"
	AppReleaseSatusRunning AppReleaseSatus = "running"
	AppReleaseSatusFailed  AppReleaseSatus = "failed"
)

type AppType struct {
	ID   int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name string `json:"name" gorm:"column:name; default:''; NOT NULL"`
	gorm.Model
}

type AppRepo struct {
	ID          int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Url         string `json:"url" gorm:"column:url; default:''; NOT NULL"`
	IndexPath   string `json:"index_path" gorm:"column:index_path; default:''; NOT NULL"`
	Description string `json:"description" gorm:"column:description; default:''; NOT NULL"`
	gorm.Model
}

type App struct {
	ID          int64         `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string        `json:"name" gorm:"column:name; default:''; NOT NULL; index"`
	Icon        string        `json:"icon,omitempty" gorm:"column:icon; default:''; NOT NULL"`
	AppTypeID   int64         `json:"app_type_id,omitempty" gorm:"column:app_type_id; default:0; NOT NULL"`
	AppRepoID   int64         `json:"app_repo_id,omitempty" gorm:"column:app_repo_id; default:0; NOT NULL"`
	Description string        `json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`
	Versions    []*AppVersion `json:"versions,omitempty" gorm:"-"`
	Readme      string        `json:"readme,omitempty" gorm:"-"`
	Metadata    []byte        `json:"metadata,omitempty" gorm:"-"`
	gorm.Model
}

type AppVersion struct {
	ID            int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	AppID         int64  `json:"app_id" gorm:"column:app_id; default:0; NOT NULL; index"`
	Name          string `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Chart         string `json:"chart,omitempty" gorm:"column:chart; default:''; NOT NULL"` // as file path
	Version       string `json:"version,omitempty" gorm:"column:version; default:''; NOT NULL; index"`
	DefaultConfig string `json:"default_config,omitempty" gorm:"column:default_config; default:''; NOT NULL"`
	gorm.Model
}

type AppRelease struct {
	ID                  int64                 `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	ReleaseName         string                `json:"release_name,omitempty" gorm:"column:release_name; default:''; NOT NULL"`
	AppID               int64                 `json:"app_id" gorm:"column:app_id; default:0; NOT NULL; index"`
	VersionID           int64                 `json:"version_id" gorm:"column:version_id; default:0; NOT NULL; index"`
	ClusterID           int64                 `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL; index"`
	ProjectID           int64                 `json:"project_id" gorm:"column:project_id; default:0; NOT NULL; index"`
	UserID              int64                 `json:"user_id" gorm:"column:user_id; default:0; NOT NULL; index"`
	Namespace           string                `json:"namespace,omitempty" gorm:"column:namespace; default:''; NOT NULL"`
	Config              string                `json:"config,omitempty" gorm:"column:config; default:''; NOT NULL"`
	Status              AppReleaseSatus       `json:"status,omitempty" gorm:"column:status; default:''; NOT NULL"`
	Manifest            string                `json:"manifest,omitempty" gorm:"column:manifest; default:''; NOT NULL"`
	Notes               string                `json:"notes,omitempty" gorm:"column:notes; default:''; NOT NULL"`
	Logs                string                `json:"logs,omitempty" gorm:"column:logs; default:''; NOT NULL"`
	Dryrun              bool                  `json:"dryrun,omitempty" gorm:"column:dryrun; default:false; NOT NULL"`
	Atomic              bool                  `json:"atomic,omitempty" gorm:"column:atomic; default:false; NOT NULL"`
	Wait                bool                  `json:"wait,omitempty" gorm:"column:wait; default:false; NOT NULL"`
	AppReleaseResources []*AppReleaseResource `json:"resources,omitempty" gorm:"-"`
	gorm.Model
}

type AppReleaseResource struct {
	ID        int64    `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	ReleaseID int64    `json:"release_id" gorm:"column:release_id; default:0; NOT NULL; index"`
	Name      string   `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Kind      string   `json:"kind,omitempty" gorm:"column:kind; default:''; NOT NULL"`
	Manifest  string   `json:"manifest,omitempty" gorm:"column:manifest; default:''; NOT NULL"`
	StartedAt string   `json:"started_at,omitempty" gorm:"column:started_at; default:''; NOT NULL"`
	Events    []string `json:"events,omitempty" gorm:"column:events; default:''; NOT NULL"`
	Status    []string `json:"status,omitempty" gorm:"column:status; default:''; NOT NULL"`
	gorm.Model
}

func (a *AppRepo) SetIndexPath(path string) {
	a.IndexPath = path
}

func (a *App) AddVersion(version *AppVersion) {
	if a.Versions == nil {
		a.Versions = make([]*AppVersion, 0)
	}
	a.Versions = append(a.Versions, version)
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

type AppData interface {
	Save(context.Context, *App) error
	List(context.Context, *App, int32, int32) ([]*App, int32, error)
	Get(ctx context.Context, appID int64) (*App, error)
	Delete(ctx context.Context, appID int64) error
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
	DeleteApp(ctx context.Context, app *App) error
	DeleteAppVersion(ctx context.Context, app *App, appVersion *AppVersion) error
	GetAppAndVersionInfo(context.Context, *App, *AppVersion) error
	AppRelease(context.Context, *App, *AppVersion, *AppRelease, *AppRepo) error
	DeleteAppRelease(context.Context, *AppRelease) error
	AddAppRepo(context.Context, *AppRepo) error
	GetAppsByRepo(context.Context, *AppRepo) ([]*App, error)
	GetAppDetailByRepo(ctx context.Context, apprepo *AppRepo, appName, version string) (*App, error)
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
	return &AppUsecase{
		appData:      appData,
		appRuntime:   appRuntime,
		appConstruct: appConstruct,
		conf:         conf,
		log:          log.NewHelper(logger),
		locks:        make(map[int64]*sync.Mutex),
		eventChan:    make(chan *AppRelease, AppPoolNumber),
	}
}

func (uc *AppUsecase) Init(ctx context.Context) error {
	if !uc.appRuntime.CheckCluster(ctx) {
		return nil
	}
	appPath, err := utils.GetServerStorePathByNames(utils.AppPackage)
	if err != nil {
		return err
	}
	configPath, err := utils.GetServerStorePathByNames(utils.ConfigPackage)
	if err != nil {
		return err
	}
	for _, v := range uc.conf.App {
		uc.log.Info(v.Name, v.Version)
		appchart := fmt.Sprintf("%s/%s-%s.tgz", appPath, v.Name, v.Version)
		if !utils.IsFileExist(appchart) {
			return fmt.Errorf("appchart not found: %s", appchart)
		}
		app := &App{Name: v.Name}
		appVersion := &AppVersion{Chart: appchart, Version: v.Version}
		err = uc.GetAppVersionInfoByLocalFile(ctx, app, appVersion)
		if err != nil {
			return err
		}
		app.AddVersion(appVersion)
		err = uc.appData.Save(ctx, app)
		if err != nil {
			return err
		}
		appConfigPath := fmt.Sprintf("%s/%s-%s.yaml", configPath, v.Name, v.Version)
		if utils.IsFileExist(appConfigPath) {
			appConfig, err := os.ReadFile(appConfigPath)
			if err != nil {
				return err
			}
			appVersion.DefaultConfig = string(appConfig)
		}
		uc.apply(&AppRelease{
			ReleaseName: fmt.Sprintf("%s-%s", v.Name, v.Version),
			AppID:       app.ID,
			VersionID:   appVersion.ID,
			Namespace:   v.Namespace,
			Config:      appVersion.DefaultConfig,
			Status:      AppReleaseSatusPending,
		})
	}
	return nil
}

func (uc *AppUsecase) GetAppVersionInfoByLocalFile(ctx context.Context, app *App, appVersion *AppVersion) error {
	return uc.appConstruct.GetAppAndVersionInfo(ctx, app, appVersion)
}

func (uc *AppUsecase) GetAppByName(ctx context.Context, name string) (app *App, err error) {
	return uc.appData.GetByName(ctx, name)
}

func (uc *AppUsecase) List(ctx context.Context, appReq *App, page, pageSize int32) ([]*App, int32, error) {
	return uc.appData.List(ctx, appReq, page, pageSize)
}

func (uc *AppUsecase) Get(ctx context.Context, id int64) (*App, error) {
	return uc.appData.Get(ctx, id)
}

func (uc *AppUsecase) GetAppVersion(ctx context.Context, appID, versionID int64) (*AppVersion, error) {
	app, err := uc.appData.Get(ctx, appID)
	if err != nil {
		return nil, err
	}
	return app.GetVersionById(versionID), nil
}

func (uc *AppUsecase) Save(ctx context.Context, app *App) error {
	return uc.appData.Save(ctx, app)
}

func (uc *AppUsecase) Delete(ctx context.Context, id int64) error {
	app, err := uc.appData.Get(ctx, id)
	if err != nil {
		return err
	}
	err = uc.appData.Delete(ctx, id)
	if err != nil {
		return err
	}
	return uc.appConstruct.DeleteApp(ctx, app)
}

func (uc *AppUsecase) DeleteAppVersion(ctx context.Context, app *App, appVersion *AppVersion) error {
	app, err := uc.appData.Get(ctx, app.ID)
	if err != nil {
		return err
	}
	app.DeleteVersion(appVersion.Version)
	err = uc.appData.Save(ctx, app)
	if err != nil {
		return err
	}
	return uc.appConstruct.DeleteAppVersion(ctx, app, appVersion)
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
	err = uc.appData.SaveRepo(ctx, repo)
	if err != nil {
		return err
	}
	apps, err := uc.GetAppsByRepo(ctx, repo.ID)
	if err != nil {
		return err
	}
	for _, app := range apps {
		appInfo, err := uc.appData.GetByName(ctx, app.Name)
		if err != nil {
			return err
		}
		if appInfo != nil {
			app.ID = appInfo.ID
		}
		app.AppRepoID = repo.ID
		err = uc.appData.Save(ctx, app)
		if err != nil {
			return err
		}
	}
	return nil
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

func (uc *AppUsecase) CreateAppRelease(ctx context.Context, appRelease *AppRelease) error {
	err := uc.appData.SaveAppRelease(ctx, appRelease)
	if err != nil {
		return err
	}
	appRelease.Status = AppReleaseSatusPending
	uc.apply(appRelease)
	return nil
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

func (uc *AppUsecase) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case data, ok := <-uc.eventChan:
			if !ok {
				return nil
			}
			err := uc.handleEvent(ctx, data)
			if err != nil {
				return err
			}
			return nil
		}
	}
}

func (uc *AppUsecase) Stop(ctx context.Context) error {
	close(uc.eventChan)
	return nil
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
	app, err := uc.appData.Get(ctx, appRelease.AppID)
	if err != nil {
		return err
	}
	appVersion := app.GetVersionById(appRelease.VersionID)
	if appVersion == nil {
		return errors.New("app version not found")
	}
	err = uc.appConstruct.GetAppAndVersionInfo(ctx, app, appVersion)
	if err != nil {
		return err
	}
	var appRepo *AppRepo
	if app.AppRepoID > 0 {
		appRepo, err = uc.appData.GetRepo(ctx, app.AppRepoID)
		if err != nil {
			return err
		}
	}
	err = uc.appConstruct.AppRelease(ctx, app, appVersion, appRelease, appRepo)
	if err != nil {
		return err
	}
	return nil
}
