package biz

import (
	"context"
	"slices"
	"sort"
	"sync"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
)

const (
	AppsDir       = "apps"
	AppPoolNumber = 100
)

type AppReleaseSatus int32

const (
	AppReleaseSatus_PENDING AppReleaseSatus = 1
	AppReleaseSatus_RUNNING AppReleaseSatus = 2
	AppReleaseSatus_FAILED  AppReleaseSatus = 3
)

type AppReleaseResourceStatus int32

const (
	AppReleaseResourceStatus_SUCCESSFUL AppReleaseResourceStatus = 1
	AppReleaseResourceStatus_UNHEALTHY  AppReleaseResourceStatus = 2
)

type AppType struct {
	Id          int64  `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Description string `json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`
}

type AppRepo struct {
	Id          int64  `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Url         string `json:"url,omitempty" gorm:"column:url; default:''; NOT NULL"`
	IndexPath   string `json:"index_path,omitempty" gorm:"column:index_path; default:''; NOT NULL"`
	Apps        []*App `json:"apps,omitempty" gorm:"-"`
	Description string `json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`
}

type AppVersion struct {
	Id            int64  `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	AppId         int64  `json:"app_id,omitempty" gorm:"column:app_id; default:0; NOT NULL; index"`
	Name          string `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Chart         string `json:"chart,omitempty" gorm:"column:chart; default:''; NOT NULL"` // as file path
	Version       string `json:"version,omitempty" gorm:"column:version; default:''; NOT NULL; index"`
	DefaultConfig string `json:"default_config,omitempty" gorm:"column:default_config; default:''; NOT NULL"`
}

type App struct {
	Id          int64         `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string        `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL; index"`
	Icon        string        `json:"icon,omitempty" gorm:"column:icon; default:''; NOT NULL"`
	AppTypeId   int64         `json:"app_type_id,omitempty" gorm:"column:app_type_id; default:0; NOT NULL"`
	AppRepoId   int64         `json:"app_repo_id,omitempty" gorm:"column:app_repo_id; default:0; NOT NULL"`
	Description string        `json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`
	Versions    []*AppVersion `json:"versions,omitempty" gorm:"-"`
	Readme      string        `json:"readme,omitempty" gorm:"-"`
	Metadata    []byte        `json:"metadata,omitempty" gorm:"-"`
}

type AppReleaseResource struct {
	Id        string                   `json:"id,omitempty" gorm:"column:id;primaryKey;NOT NULL"`
	ReleaseId int64                    `json:"release_id,omitempty" gorm:"column:release_id;default:0;NOT NULL;index"`
	Name      string                   `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Namespace string                   `json:"namespace,omitempty" gorm:"column:namespace;default:'';NOT NULL"`
	Kind      string                   `json:"kind,omitempty" gorm:"column:kind;default:'';NOT NULL"`
	Lables    string                   `json:"lables,omitempty" gorm:"column:lables;default:'';NOT NULL"`
	Manifest  string                   `json:"manifest,omitempty" gorm:"column:manifest;default:'';NOT NULL"`
	StartedAt string                   `json:"started_at,omitempty" gorm:"column:started_at;default:'';NOT NULL"`
	Status    AppReleaseResourceStatus `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	Events    string                   `json:"events,omitempty" gorm:"column:events;default:'';NOT NULL"`
}

type AppRelease struct {
	Id          int64                 `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	ReleaseName string                `json:"release_name,omitempty" gorm:"column:release_name;default:'';NOT NULL"`
	Namespace   string                `json:"namespace,omitempty" gorm:"column:namespace;default:'';NOT NULL"`
	Config      string                `json:"config,omitempty" gorm:"column:config;default:'';NOT NULL"`
	ConfigFile  string                `json:"config_file,omitempty" gorm:"column:config_file;default:'';NOT NULL"`
	Status      AppReleaseSatus       `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	Notes       string                `json:"notes,omitempty" gorm:"column:notes;default:'';NOT NULL"`
	Logs        string                `json:"logs,omitempty" gorm:"column:logs;default:'';NOT NULL"`
	Dryrun      bool                  `json:"dryrun,omitempty" gorm:"column:dryrun;default:false;NOT NULL"`
	Atomic      bool                  `json:"atomic,omitempty" gorm:"column:atomic;default:false;NOT NULL"`
	Wait        bool                  `json:"wait,omitempty" gorm:"column:wait;default:false;NOT NULL"`
	Replicas    int32                 `json:"replicas,omitempty" gorm:"column:replicas;default:0;NOT NULL"`
	Cpu         int32                 `json:"cpu,omitempty" gorm:"column:cpu;default:0;NOT NULL"`
	LimitCpu    int32                 `json:"limit_cpu,omitempty" gorm:"column:limit_cpu;default:0;NOT NULL"`
	Memory      int32                 `json:"memory,omitempty" gorm:"column:memory;default:0;NOT NULL"`
	LimitMemory int32                 `json:"limit_memory,omitempty" gorm:"column:limit_memory;default:0;NOT NULL"`
	Gpu         int32                 `json:"gpu,omitempty" gorm:"column:gpu;default:0;NOT NULL"`
	LimitGpu    int32                 `json:"limit_gpu,omitempty" gorm:"column:limit_gpu;default:0;NOT NULL"`
	Storage     int32                 `json:"storage,omitempty" gorm:"column:storage;default:0;NOT NULL"`
	Resources   []*AppReleaseResource `json:"resources,omitempty" gorm:"-"`
	AppId       int64                 `json:"app_id,omitempty" gorm:"column:app_id;default:0;NOT NULL;index"`
	VersionId   int64                 `json:"version_id,omitempty" gorm:"column:version_id;default:0;NOT NULL;index"`
	ClusterId   int64                 `json:"cluster_id,omitempty" gorm:"column:cluster_id;default:0;NOT NULL;index"`
	ProjectId   int64                 `json:"project_id,omitempty" gorm:"column:project_id;default:0;NOT NULL;index"`
	UserId      int64                 `json:"user_id,omitempty" gorm:"column:user_id;default:0;NOT NULL;index"`
	WorkspaceId int64                 `json:"workspace_id,omitempty" gorm:"column:workspace_id;default:0;NOT NULL;index"`
	Chart       string                `json:"chart,omitempty" gorm:"column:chart;default:'';NOT NULL"`
	RepoName    string                `json:"repo_name,omitempty" gorm:"column:repo_name;default:'';NOT NULL"`
	AppVersion  string                `json:"app_version,omitempty" gorm:"column:app_version;default:'';NOT NULL"`
	AppName     string                `json:"app_name,omitempty" gorm:"column:app_name;default:'';NOT NULL"`
}

func (a *AppRelease) IsDeleted() bool {
	return false
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
	AppReleaseList(context.Context, map[string]string, int32, int32) ([]*AppRelease, int32, error)
	GetAppRelease(ctx context.Context, id int64) (*AppRelease, error)
	SaveRepo(context.Context, *AppRepo) error
	ListRepo(context.Context) ([]*AppRepo, error)
	GetAppVersionDetailByRepo(ctx context.Context, repoId int64, appName, version string) (*App, error)
	GetAppsByRepo(ctx context.Context, repoId int64) ([]*App, error)
	GetRepo(context.Context, int64) (*AppRepo, error)
	GetRepoByName(context.Context, string) (*AppRepo, error)
	DeleteRepo(context.Context, int64) error
	GetAppReleaseResourceByProject(ctx context.Context, projectId int64, alreadyResource *AlreadyResource) error
}

type AppRuntime interface {
	GetAppReleaseResources(context.Context, *AppRelease) error
	DeleteApp(ctx context.Context, app *App) error
	DeleteAppVersion(ctx context.Context, app *App, appVersion *AppVersion) error
	GetAppAndVersionInfo(context.Context, *App) error
	AppRelease(context.Context, *AppRelease) error
	DeleteAppRelease(context.Context, *AppRelease) error
	ReloadAppRepo(context.Context, *AppRepo) error
}

type AppUsecase struct {
	appData    AppData
	appRuntime AppRuntime
	locks      map[int64]*sync.Mutex
	locksMux   sync.Mutex
	eventChan  chan *AppRelease
	conf       *conf.Bootstrap
	log        *log.Helper
}

func NewAppUsecase(appData AppData, appRuntime AppRuntime, logger log.Logger, conf *conf.Bootstrap) *AppUsecase {
	return &AppUsecase{
		appData:    appData,
		appRuntime: appRuntime,
		conf:       conf,
		log:        log.NewHelper(logger),
		locks:      make(map[int64]*sync.Mutex),
		eventChan:  make(chan *AppRelease, AppPoolNumber),
	}
}

func GetAppById(apps []*App, id int64) *App {
	for _, v := range apps {
		if v.Id == id {
			return v
		}
	}
	return nil
}

func (a *App) UpdateApp(app *App) {
	a.Name = app.Name
	a.Icon = app.Icon
	a.AppTypeId = app.AppTypeId
	a.Description = app.Description
	a.AppRepoId = app.AppRepoId
	a.AppTypeId = app.AppTypeId
}

func (a *App) SortVersions() {
	if len(a.Versions) == 0 {
		return
	}
	sort.Slice(a.Versions, func(i, j int) bool {
		return a.Versions[i].Version < a.Versions[j].Version
	})
}

func (a *App) GetLastVersion() *AppVersion {
	if len(a.Versions) == 0 {
		return nil
	}
	a.SortVersions()
	return a.Versions[len(a.Versions)-1]
}

func (a *App) IsEmpty() bool {
	return a.Id == 0
}

func (a *App) GetVersion(version string) *AppVersion {
	for _, v := range a.Versions {
		if v.Version == version {
			return v
		}
	}
	return nil
}

func (a *App) GetLabels() map[string]string {
	return map[string]string{
		"app": a.Name,
	}
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
		if v.Id == id {
			return v
		}
	}
	return nil
}

func (a *App) DeleteVersion(version string) {
	for i, v := range a.Versions {
		if v.Version == version {
			a.Versions = slices.Delete(a.Versions, i, i+1)
			return
		}
	}
}

func (uc *AppUsecase) GetAppAndVersionInfo(ctx context.Context, app *App) error {
	return uc.appRuntime.GetAppAndVersionInfo(ctx, app)
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
	return uc.appRuntime.DeleteApp(ctx, app)
}

func (uc *AppUsecase) DeleteAppVersion(ctx context.Context, app *App, appVersion *AppVersion) error {
	app, err := uc.appData.Get(ctx, app.Id)
	if err != nil {
		return err
	}
	app.DeleteVersion(appVersion.Version)
	err = uc.appData.Save(ctx, app)
	if err != nil {
		return err
	}
	return uc.appRuntime.DeleteAppVersion(ctx, app, appVersion)
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
	repoData, err := uc.appData.GetRepoByName(ctx, repo.Name)
	if err != nil {
		return err
	}
	repo.Id = repoData.Id
	err = uc.appRuntime.ReloadAppRepo(ctx, repo)
	if err != nil {
		return err
	}
	err = uc.appData.SaveRepo(ctx, repo)
	if err != nil {
		return err
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
	return uc.appData.GetAppsByRepo(ctx, repoID)
}

func (uc *AppUsecase) GetAppDetailByRepo(ctx context.Context, repoID int64, appName, version string) (*App, error) {
	return uc.appData.GetAppVersionDetailByRepo(ctx, repoID, appName, version)
}

func (uc *AppUsecase) GetAppRelease(ctx context.Context, id int64) (*AppRelease, error) {
	return uc.appData.GetAppRelease(ctx, id)
}

func (uc *AppUsecase) AppReleaseList(ctx context.Context, appReleaseReq map[string]string, page, pageSize int32) ([]*AppRelease, int32, error) {
	appReleases, count, err := uc.appData.AppReleaseList(ctx, appReleaseReq, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return appReleases, count, nil
}

func (uc *AppUsecase) GetAppReleaseResourcesInCluster(ctx context.Context, appReleaseID int64) ([]*AppReleaseResource, error) {
	appRelease, err := uc.appData.GetAppRelease(ctx, appReleaseID)
	if err != nil {
		return nil, err
	}
	err = uc.appRuntime.GetAppReleaseResources(ctx, appRelease)
	if err != nil {
		return nil, err
	}
	return appRelease.Resources, nil
}

func (uc *AppUsecase) CreateAppRelease(ctx context.Context, appRelease *AppRelease) error {
	err := uc.appData.SaveAppRelease(ctx, appRelease)
	if err != nil {
		return err
	}
	appRelease.Status = AppReleaseSatus_PENDING
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

func (uc *AppUsecase) GetAppReleaseResourceByProject(ctx context.Context, projectId int64) (*AlreadyResource, error) {
	return nil, nil
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
	lock := uc.getLock(appRelease.Id)
	lock.Lock()
	defer func() {
		uc.appData.SaveAppRelease(ctx, appRelease)
		lock.Unlock()
	}()
	if appRelease.IsDeleted() {
		err = uc.appRuntime.DeleteAppRelease(ctx, appRelease)
		if err != nil {
			return err
		}
	}
	app, err := uc.appData.Get(ctx, appRelease.AppId)
	if err != nil {
		return err
	}
	appVersion := app.GetVersionById(appRelease.VersionId)
	if appVersion == nil {
		return errors.New("app version not found")
	}
	err = uc.appRuntime.AppRelease(ctx, appRelease)
	if err != nil {
		return err
	}
	return nil
}
