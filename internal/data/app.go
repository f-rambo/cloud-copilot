package data

import (
	"context"

	"github.com/f-rambo/ocean/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
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
	err := a.data.db.Save(&app).Error
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

func (a *appRepo) DeleteApp(ctx context.Context, appId int) error {
	// 删除app
	err := a.k8s()
	if err != nil {
		return err
	}
	err = a.data.db.Where("id = ?", appId).Delete(&biz.App{}).Error
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
	k8sClient := a.data.k8sClient
	// 创建configmap
	app.ConfigMap, err = k8sClient.CoreV1().ConfigMaps(app.Namespace).Create(ctx, app.ConfigMap, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	// 创建secret
	app.Secret, err = k8sClient.CoreV1().Secrets(app.Namespace).Create(ctx, app.Secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	// 创建app
	// appObj := &operatoroceaniov1alpha1.App{}
	// k8sClient.RESTClient().Post().Namespace(app.Namespace).Resource("app").Body(app).Do(ctx).Into(appObj)
	return nil
}

func (a *appRepo) k8s() error {
	if a.data.k8sClient != nil {
		return nil
	}
	return a.data.newKubernetes()
}
