package data

import (
	"context"
	"ocean/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gopkg.in/yaml.v3"
)

var appKey = "app/config"

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

func (a *appRepo) GetApps(ctx context.Context) ([]*biz.App, error) {
	// get app config file content
	appData, err := readFile(getAPPConfigPath())
	if err != nil {
		return nil, err
	}
	apps := make([]*biz.App, 0)
	// yaml unmarshal
	err = yaml.Unmarshal(appData, &apps)
	if err != nil {
		return nil, err
	}
	return apps, nil
}

func (a *appRepo) SaveApp(ctx context.Context, app *biz.App) error {
	// 设置ID
	if app.ID == "" {
		app.ID = getUUID()
	}
	if a.data.db == nil {
		return a.saveAppToYamlFile(ctx, app)
	}
	return a.saveAppToDB(ctx, app)
}

func (a *appRepo) saveAppToYamlFile(ctx context.Context, app *biz.App) error {
	apps, err := a.GetApps(ctx)
	if err != nil {
		return err
	}
	appIsExist := false
	index := 0
	for i, v := range apps {
		if v.ID == app.ID {
			appIsExist = true
			index = i
		}
	}
	if appIsExist {
		apps[index] = app
	} else {
		apps = append(apps, app)
	}
	// yaml marshal
	appData, err := yaml.Marshal(apps)
	if err != nil {
		return err
	}
	// write to file
	err = writeFile(getAPPConfigPath(), appData)
	if err != nil {
		return err
	}
	return nil
}

func (a *appRepo) saveAppToDB(ctx context.Context, app *biz.App) error {
	return nil
}

func (a *appRepo) GetAppById(ctx context.Context, appId string) (*biz.App, error) {
	apps, err := a.GetApps(ctx)
	if err != nil {
		return nil, err
	}
	for _, app := range apps {
		if app.ID == appId {
			return app, nil
		}
	}
	return nil, nil
}

func (a *appRepo) DeployApp(ctx context.Context, appId string) error {
	return nil
}

func (a *appRepo) DestroyApp(ctx context.Context, appId string) error {
	return nil
}
