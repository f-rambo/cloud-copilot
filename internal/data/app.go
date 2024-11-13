package data

import (
	"context"
	"sort"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type appRepo struct {
	data *Data
	log  *log.Helper
}

func NewAppRepo(data *Data, logger log.Logger) biz.AppRepo {
	return &appRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (a *appRepo) Save(ctx context.Context, app *biz.App) error {
	appVersions := make([]*biz.AppVersion, 0)
	if app.ID != 0 {
		err := a.data.db.Model(&biz.AppVersion{}).Where("app_id = ?", app.ID).Find(&appVersions).Error
		if err != nil {
			return err
		}
	}
	err := a.data.db.Save(app).Error
	if err != nil {
		return err
	}
	// 保存新增加的删除不存在的版本
	for _, v := range app.Versions {
		v.AppID = app.ID
		if v.ID == 0 {
			err = a.data.db.Model(&biz.AppVersion{}).Create(v).Error
		} else {
			err = a.data.db.Model(&biz.AppVersion{}).Where("id = ?", v.ID).Save(v).Error
		}
		if err != nil {
			return err
		}
	}
	// 删除不存在的版本
	for _, version := range appVersions {
		isExist := false
		for _, v := range app.Versions {
			if v.ID == version.ID {
				isExist = true
			}
		}
		if !isExist {
			err = a.data.db.Model(&biz.AppVersion{}).Where("id = ?", version.ID).Delete(version).Error
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
	// app versions
	appIDs := make([]int64, 0)
	appVersions := make([]*biz.AppVersion, 0)
	for _, v := range appItems {
		appIDs = append(appIDs, v.ID)
	}
	err = a.data.db.Model(&biz.AppVersion{}).Where("app_id in (?)", appIDs).Find(&appVersions).Error
	if err != nil {
		return nil, 0, err
	}
	for _, v := range appItems {
		for _, version := range appVersions {
			if v.ID == version.AppID {
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
	if app.ID == 0 {
		return nil, nil
	}
	appVersions := make([]*biz.AppVersion, 0)
	err = a.data.db.Model(&biz.AppVersion{}).Where("app_id = ?", app.ID).Find(&appVersions).Error
	if err != nil {
		return nil, err
	}
	app.Versions = appVersions
	return app, nil
}

func (a *appRepo) Delete(ctx context.Context, appID, versionID int64) error {
	if versionID == 0 {
		err := a.data.db.Delete(&biz.App{}, appID).Error
		if err != nil {
			return err
		}
		return a.data.db.Delete(&biz.AppVersion{}, "app_id = ?", appID).Error
	}
	return a.data.db.Delete(&biz.AppVersion{}, "id = ?", versionID).Error
}

func (a *appRepo) CreateAppType(ctx context.Context, appType *biz.AppType) error {
	return a.data.db.Create(appType).Error
}

func (a *appRepo) ListAppType(ctx context.Context) ([]*biz.AppType, error) {
	defaultAppTypes := biz.DefaultAppType()
	appTypes := make([]*biz.AppType, 0)
	err := a.data.db.Find(&appTypes).Error
	if err != nil {
		return nil, err
	}
	appTypes = append(appTypes, defaultAppTypes...)
	// 按照ID排序
	sort.Slice(appTypes, func(i, j int) bool {
		return appTypes[i].ID < appTypes[j].ID
	})
	// 把ID为0的放到最前面
	for i, v := range appTypes {
		if v.ID == 0 {
			appTypes[0], appTypes[i] = appTypes[i], appTypes[0]
		}
	}
	return appTypes, nil
}

func (a *appRepo) DeleteAppType(ctx context.Context, appTypeID int64) error {
	return a.data.db.Delete(&biz.AppType{}, appTypeID).Error
}

func (a *appRepo) SaveDeployApp(ctx context.Context, appDeployed *biz.AppRelease) error {
	return a.data.db.Save(appDeployed).Error
}

func (a *appRepo) DeployAppList(ctx context.Context, appDeployedReq biz.AppRelease, page, pageSize int32) ([]*biz.AppRelease, int32, error) {
	appDeployedReq.Chart = ""
	appDeployedReq.Config = ""
	appDeployedReq.Manifest = ""
	appDeployedReq.Notes = ""
	appDeployedReq.Logs = ""
	appDeployeds := make([]*biz.AppRelease, 0)
	appDeployedOrmDb := a.data.db.Model(&biz.AppRelease{})
	if appDeployedReq.ReleaseName != "" {
		appDeployedOrmDb = appDeployedOrmDb.Where("release_name like ?", "%"+appDeployedReq.ReleaseName+"%")
		appDeployedReq.ReleaseName = ""
	}
	err := appDeployedOrmDb.Where(appDeployedReq).Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&appDeployeds).Error
	if err != nil {
		return nil, 0, err
	}

	var appDeployedCount int64 = 0
	err = appDeployedOrmDb.Where(appDeployedReq).Count(&appDeployedCount).Error
	if err != nil {
		return nil, 0, err
	}

	return appDeployeds, int32(appDeployedCount), nil
}

func (a *appRepo) GetDeployApp(ctx context.Context, id int64) (*biz.AppRelease, error) {
	deployApp := &biz.AppRelease{}
	err := a.data.db.First(deployApp, id).Error
	if err != nil && err.Error() != "record not found" {
		return nil, err
	}
	return deployApp, nil
}

func (a *appRepo) DeleteDeployApp(ctx context.Context, id int64) error {
	return a.data.db.Delete(&biz.AppRelease{}, id).Error
}

func (a *appRepo) SaveRepo(ctx context.Context, helmRepo *biz.AppHelmRepo) error {
	return a.data.db.Save(helmRepo).Error
}

func (a *appRepo) ListRepo(ctx context.Context) ([]*biz.AppHelmRepo, error) {
	helmRepos := make([]*biz.AppHelmRepo, 0)
	err := a.data.db.Model(&biz.AppHelmRepo{}).Find(&helmRepos).Error
	if err != nil {
		return nil, err
	}
	return helmRepos, nil
}

func (a *appRepo) GetRepo(ctx context.Context, helmRepoID int64) (*biz.AppHelmRepo, error) {
	helmRepo := &biz.AppHelmRepo{}
	err := a.data.db.First(helmRepo, helmRepoID).Error
	if err != nil {
		return nil, err
	}
	return helmRepo, nil
}

func (a *appRepo) DeleteRepo(ctx context.Context, helmRepoID int64) error {
	return a.data.db.Delete(&biz.AppHelmRepo{}, helmRepoID).Error
}

func (a *appRepo) GetRepoByName(ctx context.Context, repoName string) (*biz.AppHelmRepo, error) {
	helmRepo := &biz.AppHelmRepo{}
	err := a.data.db.Where("name = ?", repoName).First(helmRepo).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return helmRepo, nil
}
