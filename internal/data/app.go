package data

import (
	"context"
	"fmt"
	"time"

	k8serr "k8s.io/apimachinery/pkg/api/errors"

	"github.com/f-rambo/ocean/internal/biz"

	operatoroceaniov1alpha1 "github.com/f-rambo/operatorapp/api/v1alpha1"
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

func (a *appRepo) DeleteApp(ctx context.Context, app *biz.App) error {
	// 删除app
	err := a.k8s()
	if err != nil {
		return err
	}
	k8sClient := a.data.k8sClient
	err = k8sClient.RESTClient().Delete().Namespace(app.Namespace).Resource("apps").Name(app.Name).Do(ctx).Error()
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
	k8sClient := a.data.k8sClient
	// 创建configmap
	app.ConfigMap.Name = fmt.Sprintf("%s-%s", app.ConfigMap.Name, time.Now().Format("20060102150405"))
	app.ConfigMap, err = k8sClient.CoreV1().ConfigMaps(app.Namespace).Create(ctx, app.ConfigMap, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	// 创建secret
	app.Secret.Name = fmt.Sprintf("%s-%s", app.Secret.Name, time.Now().Format("20060102150405"))
	app.Secret, err = k8sClient.CoreV1().Secrets(app.Namespace).Create(ctx, app.Secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	// 创建app
	appObj := &operatoroceaniov1alpha1.App{}
	appObj.ObjectMeta = metav1.ObjectMeta{
		Name:      app.Name,
		Namespace: app.Namespace,
	}
	appObj.Spec = operatoroceaniov1alpha1.AppSpec{
		RepoName:      app.RepoName,
		RepoURL:       app.RepoURL,
		ChartName:     app.ChartName,
		Version:       app.Version,
		ConfigMapName: app.ConfigMap.Name,
		SecretName:    app.Secret.Name,
	}
	return k8sClient.RESTClient().Post().Namespace(app.Namespace).Resource("apps").Body(appObj).Do(ctx).Into(appObj)
}

func (a *appRepo) k8s() error {
	if a.data.k8sClient != nil {
		return nil
	}
	return a.data.newKubernetes()
}
