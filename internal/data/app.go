package data

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type appRepo struct {
	data *Data
	log  *log.Helper
}

func NewAppRepo(data *Data, logger log.Logger) biz.AppData {
	return &appRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (a *appRepo) Save(ctx context.Context, app *biz.App) error {
	appVersions := make([]*biz.AppVersion, 0)
	if app.Id != 0 {
		err := a.data.db.Model(&biz.AppVersion{}).Where("app_id = ?", app.Id).Find(&appVersions).Error
		if err != nil {
			return err
		}
	}
	err := a.data.db.Save(app).Error
	if err != nil {
		return err
	}
	for _, v := range app.Versions {
		v.AppId = app.Id
		if v.Id == 0 {
			err = a.data.db.Model(&biz.AppVersion{}).Create(v).Error
		} else {
			err = a.data.db.Model(&biz.AppVersion{}).Where("id = ?", v.Id).Save(v).Error
		}
		if err != nil {
			return err
		}
	}
	for _, version := range appVersions {
		isExist := false
		for _, v := range app.Versions {
			if v.Id == version.Id {
				isExist = true
			}
		}
		if !isExist {
			err = a.data.db.Model(&biz.AppVersion{}).Where("id = ?", version.Id).Delete(version).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *appRepo) List(ctx context.Context, appReq *biz.App, page, pageSize int32) ([]*biz.App, int32, error) {
	appItems := make([]*biz.App, 0)
	appOrmDb := a.data.db.Model(&biz.App{})
	if appReq.Name != "" {
		appOrmDb.Where("name like ?", "%"+appReq.Name+"%")
		appReq.Name = ""
	}
	err := appOrmDb.Where(appReq).Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&appItems).Error
	if err != nil {
		return nil, 0, err
	}
	var appCount int64 = 0
	err = appOrmDb.Where(appReq).Count(&appCount).Error
	if err != nil {
		return nil, 0, err
	}
	appIDs := make([]int64, 0)
	appVersions := make([]*biz.AppVersion, 0)
	for _, v := range appItems {
		appIDs = append(appIDs, v.Id)
	}
	err = a.data.db.Model(&biz.AppVersion{}).Where("app_id in (?)", appIDs).Find(&appVersions).Error
	if err != nil {
		return nil, 0, err
	}
	for _, v := range appItems {
		for _, version := range appVersions {
			if v.Id == version.AppId {
				v.Versions = append(v.Versions, version)
			}
		}
	}
	return appItems, int32(appCount), nil
}
func (a *appRepo) Get(ctx context.Context, appID int64) (*biz.App, error) {
	app := &biz.App{}
	err := a.data.db.First(app, appID).Error
	if err != nil {
		return nil, err
	}
	appVersions := make([]*biz.AppVersion, 0)
	err = a.data.db.Model(&biz.AppVersion{}).Where("app_id = ?", appID).Find(&appVersions).Error
	if err != nil {
		return nil, err
	}
	app.Versions = appVersions
	return app, nil
}

func (a *appRepo) GetByName(ctx context.Context, appName string) (*biz.App, error) {
	app := &biz.App{}
	err := a.data.db.Where("name = ?", appName).First(app).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if app.Id == 0 {
		return nil, nil
	}
	appVersions := make([]*biz.AppVersion, 0)
	err = a.data.db.Model(&biz.AppVersion{}).Where("app_id = ?", app.Id).Find(&appVersions).Error
	if err != nil {
		return nil, err
	}
	app.Versions = appVersions
	return app, nil
}

func (a *appRepo) Delete(ctx context.Context, appID int64) error {
	err := a.data.db.Delete(&biz.App{}, appID).Error
	if err != nil {
		return err
	}
	return a.data.db.Delete(&biz.AppVersion{}, "app_id = ?", appID).Error
}

func (a *appRepo) CreateAppType(ctx context.Context, appType *biz.AppType) error {
	return a.data.db.Create(appType).Error
}

func (a *appRepo) ListAppType(ctx context.Context) ([]*biz.AppType, error) {
	appTypes := make([]*biz.AppType, 0)
	err := a.data.db.Find(&appTypes).Error
	if err != nil {
		return nil, err
	}
	return appTypes, nil
}

func (a *appRepo) DeleteAppType(ctx context.Context, appTypeID int64) error {
	return a.data.db.Delete(&biz.AppType{}, appTypeID).Error
}

func (a *appRepo) SaveAppRelease(ctx context.Context, appDeployed *biz.AppRelease) error {
	return a.data.db.Save(appDeployed).Error
}

func (a *appRepo) AppReleaseList(ctx context.Context, appReleaseReq map[string]string, page, pageSize int32) ([]*biz.AppRelease, int32, error) {
	appDeployeds := make([]*biz.AppRelease, 0)
	appDeployedOrmDb := a.data.db.Model(&biz.AppRelease{})
	if releaseName, ok := appReleaseReq["release_name"]; ok {
		appDeployedOrmDb = appDeployedOrmDb.Where("release_name like ?", "%"+releaseName+"%")
	}
	err := appDeployedOrmDb.Where(appReleaseReq).Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&appDeployeds).Error
	if err != nil {
		return nil, 0, err
	}
	var appDeployedCount int64 = 0
	err = appDeployedOrmDb.Where(appReleaseReq).Count(&appDeployedCount).Error
	if err != nil {
		return nil, 0, err
	}
	return appDeployeds, int32(appDeployedCount), nil
}

func (a *appRepo) GetAppRelease(ctx context.Context, id int64) (*biz.AppRelease, error) {
	deployApp := &biz.AppRelease{}
	err := a.data.db.First(deployApp, id).Error
	if err != nil && err.Error() != "record not found" {
		return nil, err
	}
	return deployApp, nil
}

func (a *appRepo) DeleteAppRelease(ctx context.Context, id int64) error {
	return a.data.db.Delete(&biz.AppRelease{}, id).Error
}

func (a *appRepo) SaveRepo(ctx context.Context, repo *biz.AppRepo) error {
	err := a.data.db.Save(repo).Error
	if err != nil {
		return err
	}
	// todo save app and app version
	return nil
}

func (a *appRepo) ListRepo(ctx context.Context) ([]*biz.AppRepo, error) {
	repos := make([]*biz.AppRepo, 0)
	err := a.data.db.Model(&biz.AppRepo{}).Find(&repos).Error
	if err != nil {
		return nil, err
	}
	return repos, nil
}

func (a *appRepo) GetRepo(ctx context.Context, repoID int64) (*biz.AppRepo, error) {
	repo := &biz.AppRepo{}
	err := a.data.db.First(repo, repoID).Error
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (a *appRepo) DeleteRepo(ctx context.Context, repoID int64) error {
	return a.data.db.Delete(&biz.AppRepo{}, repoID).Error
}

func (a *appRepo) GetAppVersionDetailByRepo(ctx context.Context, repoId int64, appName, version string) (*biz.App, error) {
	return nil, nil
}

func (a *appRepo) GetAppsByRepo(ctx context.Context, repoId int64) ([]*biz.App, error) {
	return nil, nil
}

func (a *appRepo) GetRepoByName(ctx context.Context, repoName string) (*biz.AppRepo, error) {
	repo := &biz.AppRepo{}
	err := a.data.db.Where("name = ?", repoName).First(repo).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return repo, nil
}
