package biz

import (
	"context"
	"sync"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
)

const (
	AppPoolNumber = 100
)

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
	GetRepo(context.Context, int64) (*AppRepo, error)
	GetRepoByName(context.Context, string) (*AppRepo, error)
	DeleteRepo(context.Context, int64) error
	GetAppReleaseResourceByProject(ctx context.Context, projectId int64, alreadyResource *AlreadyResource) error
}

type AppRuntime interface {
	CheckCluster(context.Context) bool
	GetAppReleaseResources(context.Context, *AppRelease) error
	DeleteApp(ctx context.Context, app *App) error
	DeleteAppVersion(ctx context.Context, app *App, appVersion *AppVersion) error
	GetAppAndVersionInfo(context.Context, *App, *AppVersion) error
	AppRelease(context.Context, *App, *AppVersion, *AppRelease, *AppRepo) error
	DeleteAppRelease(context.Context, *AppRelease) error
	AddAppRepo(context.Context, *AppRepo) error
	GetAppsByRepo(context.Context, *AppRepo) ([]*App, error)
	GetAppDetailByRepo(ctx context.Context, apprepo *AppRepo, appName, version string) (*App, error)
}

type AppAgent interface {
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
	for index, v := range a.Versions {
		if v.Version == version {
			a.Versions = append(a.Versions[:index], a.Versions[index+1:]...)
			return
		}
	}
}

func (uc *AppUsecase) GetAppVersionInfoByLocalFile(ctx context.Context, app *App, appVersion *AppVersion) error {
	return uc.appRuntime.GetAppAndVersionInfo(ctx, app, appVersion)
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
	repoList, err := uc.appData.ListRepo(ctx)
	if err != nil {
		return err
	}
	for _, v := range repoList {
		if v.Name == repo.Name {
			repo.Id = v.Id
		}
	}
	err = uc.appRuntime.AddAppRepo(ctx, repo)
	if err != nil {
		return err
	}
	err = uc.appData.SaveRepo(ctx, repo)
	if err != nil {
		return err
	}
	apps, err := uc.GetAppsByRepo(ctx, repo.Id)
	if err != nil {
		return err
	}
	for _, app := range apps {
		appInfo, err := uc.appData.GetByName(ctx, app.Name)
		if err != nil {
			return err
		}
		if appInfo != nil {
			app.Id = appInfo.Id
		}
		app.AppRepoId = repo.Id
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
	return uc.appRuntime.GetAppsByRepo(ctx, repo)
}

func (uc *AppUsecase) GetAppDetailByRepo(ctx context.Context, repoID int64, appName, version string) (*App, error) {
	repos, err := uc.appData.ListRepo(ctx)
	if err != nil {
		return nil, err
	}
	var repo *AppRepo
	for _, v := range repos {
		if v.Id == repoID {
			repo = v
			break
		}
	}
	if repo == nil {
		return nil, errors.New("app repo not found")
	}
	return uc.appRuntime.GetAppDetailByRepo(ctx, repo, appName, version)
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
	lock := uc.getLock(appRelease.Id)
	lock.Lock()
	defer func() {
		uc.appData.SaveAppRelease(ctx, appRelease)
		lock.Unlock()
	}()
	if appRelease.DeletedAt.Valid {
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
	err = uc.appRuntime.GetAppAndVersionInfo(ctx, app, appVersion)
	if err != nil {
		return err
	}
	var appRepo *AppRepo
	if app.AppRepoId > 0 {
		appRepo, err = uc.appData.GetRepo(ctx, app.AppRepoId)
		if err != nil {
			return err
		}
	}
	err = uc.appRuntime.AppRelease(ctx, app, appVersion, appRelease, appRepo)
	if err != nil {
		return err
	}
	return nil
}
