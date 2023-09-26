package data

import (
	"context"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/pkg/operatorapp"
	operatoroceaniov1alpha1 "github.com/f-rambo/operatorapp/api/v1alpha1"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	err := a.data.db.Omit("created_at").Save(&app).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

func (a *appRepo) GetApps(ctx context.Context, clusterID int) ([]*biz.App, error) {
	apps := make([]*biz.App, 0)
	err := a.data.db.Where("cluster_id = ?", clusterID).Find(&apps).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return apps, nil
}

func (a *appRepo) GetApp(ctx context.Context, appId int) (*biz.App, error) {
	app := &biz.App{}
	err := a.data.db.Where("id = ?", appId).First(&app).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return app, nil
}

func (a *appRepo) DeleteApp(ctx context.Context, app *biz.App) error {
	// 删除app
	err := a.k8s()
	if err != nil {
		return err
	}
	err = a.data.operatorappClient.Apps(app.Namespace).Delete(ctx, app.Name)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}
	err = a.data.db.Where("id = ?", app.ID).Delete(&biz.App{}).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

func (a *appRepo) Apply(ctx context.Context, app *biz.App) error {
	err := a.k8s()
	if err != nil {
		return err
	}
	// 创建app
	appObj := &operatoroceaniov1alpha1.App{}
	appObj.ObjectMeta = metav1.ObjectMeta{
		Name:      app.Name,
		Namespace: app.Namespace,
	}
	appObj.TypeMeta = metav1.TypeMeta{
		APIVersion: operatorapp.GetApiVersion(),
		Kind:       operatorapp.GetKind(),
	}
	appObj.Spec = operatoroceaniov1alpha1.AppSpec{
		AppChart: operatoroceaniov1alpha1.AppChart{
			Enable:    true,
			RepoName:  app.RepoName,
			RepoURL:   app.RepoURL,
			ChartName: app.ChartName,
			Version:   app.Version,
			Config:    app.Config,
			Secret:    app.Secret,
		},
	}
	resAppData, err := a.data.operatorappClient.Apps(appObj.Namespace).Get(ctx, appObj.Name, metav1.GetOptions{})
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}
	if resAppData == nil || k8serr.IsNotFound(err) {
		_, err = a.data.operatorappClient.Apps(appObj.Namespace).Create(ctx, appObj)
		if err != nil {
			return err
		}
		return nil
	}
	appObj.ResourceVersion = resAppData.ResourceVersion
	_, err = a.data.operatorappClient.Apps(appObj.Namespace).Update(ctx, appObj)
	if err != nil {
		return err
	}
	return nil
}

func (a *appRepo) k8s() error {
	if a.data.k8sClient != nil {
		return nil
	}
	return a.data.newKubernetes()
}
