package biz

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/pkg/helm"
	"github.com/f-rambo/ocean/pkg/kubeclient"
	"github.com/f-rambo/ocean/pkg/sailor"
	"github.com/f-rambo/ocean/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"

	sailorV1alpha1 "github.com/f-rambo/sailor/api/v1alpha1"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	pkgChart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	releasePkg "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppType struct {
	ID   int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name string `json:"name" gorm:"column:name; default:''; NOT NULL"`
	gorm.Model
}

const (
	AppTypeAll        = 0
	AppTypeAppPackage = -1
	AppTypeRepo       = -2
)

func DefaultAppType() []*AppType {
	return []*AppType{
		{Name: "All", ID: AppTypeAll},
		{Name: "App Package", ID: AppTypeAppPackage},
		{Name: "Repo", ID: AppTypeRepo},
	}
}

type AppHelmRepo struct {
	ID          int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Url         string `json:"url" gorm:"column:url; default:''; NOT NULL"`
	IndexPath   string `json:"index_path" gorm:"column:index_path; default:''; NOT NULL"`
	Description string `json:"description" gorm:"column:description; default:''; NOT NULL"`
	gorm.Model
}

func (a *AppHelmRepo) SetIndexPath(path string) {
	a.IndexPath = path
}

type App struct {
	ID            int64         `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name          string        `json:"name" gorm:"column:name; default:''; NOT NULL; index"`
	Icon          string        `json:"icon,omitempty" gorm:"column:icon; default:''; NOT NULL"`
	AppTypeID     int64         `json:"app_type_id,omitempty" gorm:"column:app_type_id; default:0; NOT NULL"`
	AppHelmRepoID int64         `json:"app_helm_repo_id,omitempty" gorm:"column:app_helm_repo_id; default:0; NOT NULL"`
	Versions      []*AppVersion `json:"versions,omitempty" gorm:"-"`
	gorm.Model
}

type AppVersion struct {
	ID          int64             `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	AppID       int64             `json:"app_id" gorm:"column:app_id; default:0; NOT NULL; index"`
	AppName     string            `json:"app_name,omitempty" gorm:"column:app_name; default:''; NOT NULL"`
	Name        string            `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Chart       string            `json:"chart,omitempty" gorm:"column:chart; default:''; NOT NULL"`
	Version     string            `json:"version,omitempty" gorm:"column:version; default:''; NOT NULL; index"`
	Config      string            `json:"config,omitempty" gorm:"column:config; default:''; NOT NULL"`
	Readme      string            `json:"readme,omitempty" gorm:"-"`
	State       string            `json:"state,omitempty" gorm:"column:state; default:''; NOT NULL"`
	TestResult  string            `json:"test_result,omitempty" gorm:"column:test_result; default:''; NOT NULL"` // 哪些资源部署成功，哪些失败
	Description string            `json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`
	Metadata    pkgChart.Metadata `json:"metadata,omitempty" gorm:"-"`
	gorm.Model
}

type DeployApp struct {
	ID          int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	ReleaseName string `json:"release_name,omitempty" gorm:"column:release_name; default:''; NOT NULL"`
	AppID       int64  `json:"app_id" gorm:"column:app_id; default:0; NOT NULL; index"`
	VersionID   int64  `json:"version_id" gorm:"column:version_id; default:0; NOT NULL; index"`
	Version     string `json:"version,omitempty" gorm:"column:version; default:''; NOT NULL"`
	RepoID      int64  `json:"repo_id,omitempty" gorm:"column:repo_id; default:0; NOT NULL"`
	AppName     string `json:"app_name,omitempty" gorm:"column:app_name; default:''; NOT NULL"`
	AppTypeID   int64  `json:"app_type_id,omitempty" gorm:"column:app_type_id; default:0; NOT NULL"`
	Chart       string `json:"chart,omitempty" gorm:"column:chart; default:''; NOT NULL"`
	ClusterID   int64  `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL; index"`
	ProjectID   int64  `json:"project_id" gorm:"column:project_id; default:0; NOT NULL; index"`
	UserID      int64  `json:"user_id" gorm:"column:user_id; default:0; NOT NULL; index"`
	Namespace   string `json:"namespace,omitempty" gorm:"column:namespace; default:''; NOT NULL"`
	Config      string `json:"config,omitempty" gorm:"column:config; default:''; NOT NULL"`
	State       string `json:"state,omitempty" gorm:"column:state; default:''; NOT NULL"`
	IsTest      bool   `json:"is_test,omitempty" gorm:"column:is_test; default:false; NOT NULL"`
	Manifest    string `json:"manifest,omitempty" gorm:"column:manifest; default:''; NOT NULL"` // also template | yaml
	Notes       string `json:"notes,omitempty" gorm:"column:notes; default:''; NOT NULL"`
	Logs        string `json:"logs,omitempty" gorm:"column:logs; default:''; NOT NULL"`
	gorm.Model
}

const (
	AppUntested   = "untested"
	AppTested     = "tested"
	AppTestFailed = "test_failed"
)

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

func (v *AppVersion) GetChartInfo(appPath string) error {
	charInfo, err := helm.GetLocalChartInfo(v.AppName, appPath, v.Chart)
	if err != nil {
		return err
	}
	v.Name = charInfo.Name
	v.Config = charInfo.Config
	v.Readme = charInfo.Readme
	v.Description = charInfo.Description
	v.Metadata = charInfo.Metadata
	v.Version = charInfo.Version
	v.AppName = charInfo.Name
	v.Chart = charInfo.Chart
	return nil
}

func (v *AppVersion) GetAppDeployed() *DeployApp {
	releaseName := fmt.Sprintf("%s-%s", v.AppName, strings.ReplaceAll(v.Version, ".", "-"))
	return &DeployApp{
		AppID:       v.AppID,
		VersionID:   v.ID,
		Version:     v.Version,
		Chart:       v.Chart,
		AppName:     v.AppName,
		Namespace:   "default",
		Config:      v.Config,
		State:       releasePkg.StatusUnknown.String(),
		ReleaseName: releaseName,
	}
}

type AppRepo interface {
	Save(context.Context, *App) error
	List(ctx context.Context, appReq *App, page, pageSize int32) ([]*App, int32, error)
	Get(ctx context.Context, appID int64) (*App, error)
	GetByName(ctx context.Context, name string) (*App, error)
	Delete(ctx context.Context, appID, versionID int64) error
	CreateAppType(ctx context.Context, appType *AppType) error
	ListAppType(ctx context.Context) ([]*AppType, error)
	DeleteAppType(ctx context.Context, appTypeID int64) error
	SaveDeployApp(ctx context.Context, appDeployed *DeployApp) error
	DeleteDeployApp(ctx context.Context, id int64) error
	DeployAppList(ctx context.Context, appDeployedReq DeployApp, page, pageSuze int32) ([]*DeployApp, int32, error)
	GetDeployApp(ctx context.Context, id int64) (*DeployApp, error)
	SaveRepo(ctx context.Context, helmRepo *AppHelmRepo) error
	ListRepo(ctx context.Context) ([]*AppHelmRepo, error)
	GetRepo(ctx context.Context, helmRepoID int64) (*AppHelmRepo, error)
	GetRepoByName(ctx context.Context, repoName string) (*AppHelmRepo, error)
	DeleteRepo(ctx context.Context, helmRepoID int64) error
}

type AppUsecase struct {
	repo        AppRepo
	log         *log.Helper
	c           *conf.Bootstrap
	clusterRepo ClusterRepo
	projectRepo ProjectRepo
}

func NewAppUsecase(repo AppRepo, logger log.Logger, c *conf.Bootstrap, clusterRepo ClusterRepo, projectRepo ProjectRepo) *AppUsecase {
	return &AppUsecase{repo, log.NewHelper(logger), c, clusterRepo, projectRepo}
}

func (uc *AppUsecase) GetAppByName(ctx context.Context, name string) (app *App, err error) {
	return uc.repo.GetByName(ctx, name)
}

func (uc *AppUsecase) List(ctx context.Context, appReq *App, page, pageSize int32) ([]*App, int32, error) {
	return uc.repo.List(ctx, appReq, page, pageSize)
}

func (uc *AppUsecase) Get(ctx context.Context, id, versionId int64) (*App, error) {
	app, err := uc.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	appVersion := app.GetVersionById(versionId)
	if appVersion == nil {
		return app, nil
	}
	cresource := uc.c.GetOceanResource()
	err = appVersion.GetChartInfo(cresource.GetAppPath())
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (uc *AppUsecase) Save(ctx context.Context, app *App) error {
	return uc.repo.Save(ctx, app)
}

func (uc *AppUsecase) Delete(ctx context.Context, id, versionId int64) error {
	app, err := uc.Get(ctx, id, versionId)
	if err != nil {
		return err
	}
	err = uc.repo.Delete(ctx, id, versionId)
	if err != nil {
		return err
	}
	cresource := uc.c.GetOceanResource()
	if app.Icon != "" && utils.IsFileExist(cresource.GetIconPath()+app.Icon) && versionId == 0 {
		err = utils.DeleteFile(cresource.GetIconPath() + app.Icon)
		if err != nil {
			return err
		}
	}
	for _, v := range app.Versions {
		if v.Chart != "" && utils.IsFileExist(cresource.GetAppPath()+v.Chart) && versionId == 0 {
			err = utils.DeleteFile(cresource.GetAppPath() + v.Chart)
			if err != nil {
				return err
			}
		}
		if v.Chart != "" && utils.IsFileExist(cresource.GetAppPath()+v.Chart) && versionId == v.ID {
			err = utils.DeleteFile(cresource.GetAppPath() + v.Chart)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (uc *AppUsecase) CreateAppType(ctx context.Context, appType *AppType) error {
	return uc.repo.CreateAppType(ctx, appType)
}

func (uc *AppUsecase) ListAppType(ctx context.Context) ([]*AppType, error) {
	return uc.repo.ListAppType(ctx)
}

func (uc *AppUsecase) DeleteAppType(ctx context.Context, appTypeID int64) error {
	return uc.repo.DeleteAppType(ctx, appTypeID)
}

func (uc *AppUsecase) GetAppDeployed(ctx context.Context, id int64) (*DeployApp, error) {
	return uc.repo.GetDeployApp(ctx, id)
}

func (uc *AppUsecase) DeployAppList(ctx context.Context, appDeployedReq DeployApp, page, pageSize int32) ([]*DeployApp, int32, error) {
	return uc.repo.DeployAppList(ctx, appDeployedReq, page, pageSize)
}

func (uc *AppUsecase) AppOperation(ctx context.Context, deployedApp *DeployApp) error {
	restConfig, err := kubeclient.GetKubeConfig()
	if err != nil {
		return err
	}
	sailorApp, err := sailor.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	app := sailor.BuildAppResource(deployedApp.Namespace, deployedApp.ReleaseName,
		sailorV1alpha1.AppSpec{Manifest: deployedApp.Manifest})
	_, err = sailorApp.Apps(deployedApp.Namespace).Create(ctx, &app)
	if err != nil {
		return err
	}
	return nil
}

func (uc *AppUsecase) AppTest(ctx context.Context, appID, versionID int64) (*DeployApp, error) {
	app, err := uc.Get(ctx, appID, versionID)
	if err != nil {
		return nil, err
	}
	appVersion := app.GetVersionById(versionID)
	if appVersion == nil {
		return nil, errors.New("app version not found")
	}
	appDeployed := appVersion.GetAppDeployed()
	appDeployed.IsTest = true
	deployAppErr := uc.deployApp(ctx, appDeployed)
	if deployAppErr != nil {
		appVersion.State = AppTestFailed
	}
	if deployAppErr == nil {
		appVersion.State = AppTested
		appVersion.TestResult = "success"
	}
	err = uc.repo.Save(ctx, app)
	if err != nil {
		return nil, err
	}
	return appDeployed, deployAppErr
}

func (uc *AppUsecase) DeployApp(ctx context.Context, deployAppReq *DeployApp) (*DeployApp, error) {
	var app *App
	var appVersion *AppVersion
	var err error
	if deployAppReq.AppTypeID == AppTypeRepo {
		app, err = uc.GetAppDetailByRepo(ctx, deployAppReq.RepoID, deployAppReq.AppName, deployAppReq.Version)
		if err != nil {
			return nil, err
		}
		appVersion = app.GetVersion(deployAppReq.Version)
	}
	if deployAppReq.AppTypeID != AppTypeRepo {
		app, err = uc.Get(ctx, deployAppReq.AppID, deployAppReq.VersionID)
		if err != nil {
			return nil, err
		}
		appVersion = app.GetVersionById(deployAppReq.VersionID)
	}
	appDeployed := appVersion.GetAppDeployed()
	appDeployed.ID = deployAppReq.ID
	appDeployed.RepoID = deployAppReq.RepoID
	appDeployed.AppTypeID = app.AppTypeID
	appDeployed.ClusterID = deployAppReq.ClusterID
	appDeployed.ProjectID = deployAppReq.ProjectID
	appDeployed.Namespace = deployAppReq.Namespace
	appDeployed.Config = deployAppReq.Config
	appDeployed.UserID = deployAppReq.UserID
	if deployAppReq.ID != 0 {
		appDeployedRes, err := uc.repo.GetDeployApp(ctx, deployAppReq.ID)
		if err != nil {
			return nil, err
		}
		appDeployed.ReleaseName = appDeployedRes.ReleaseName
	}
	deployAppErr := uc.deployApp(ctx, appDeployed)
	err = uc.repo.SaveDeployApp(ctx, appDeployed)
	if err != nil {
		return nil, err
	}
	return appDeployed, deployAppErr
}

func (uc *AppUsecase) DeleteDeployedApp(ctx context.Context, id int64) error {
	appDeployed, err := uc.repo.GetDeployApp(ctx, id)
	if err != nil {
		return err
	}
	if appDeployed == nil {
		return nil
	}
	err = uc.unDeployApp(appDeployed)
	if err != nil {
		return err
	}
	err = uc.repo.DeleteDeployApp(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

func (uc *AppUsecase) StopApp(ctx context.Context, id int64) error {
	appDeployed, err := uc.repo.GetDeployApp(ctx, id)
	if err != nil {
		return err
	}
	if appDeployed == nil {
		return errors.New("app deployed not found")
	}
	unDeployAppErr := uc.unDeployApp(appDeployed)
	err = uc.repo.SaveDeployApp(ctx, appDeployed)
	if err != nil {
		return err
	}
	return unDeployAppErr
}

// 保存repo
func (uc *AppUsecase) SaveRepo(ctx context.Context, helmRepo *AppHelmRepo) error {
	repoList, err := uc.repo.ListRepo(ctx)
	if err != nil {
		return err
	}
	for _, v := range repoList {
		if v.Name == helmRepo.Name {
			helmRepo.ID = v.ID
		}
	}
	err = uc.addRepo(helmRepo)
	if err != nil {
		return err
	}
	return uc.repo.SaveRepo(ctx, helmRepo)
}

// repo列表
func (uc *AppUsecase) ListRepo(ctx context.Context) ([]*AppHelmRepo, error) {
	return uc.repo.ListRepo(ctx)
}

// 删除repo
func (uc *AppUsecase) DeleteRepo(ctx context.Context, helmRepoID int64) error {
	return uc.repo.DeleteRepo(ctx, helmRepoID)
}

// 根据repo获取app列表
func (uc *AppUsecase) GetAppsByRepo(ctx context.Context, helmRepoID int64) ([]*App, error) {
	helmRepo, err := uc.repo.GetRepo(ctx, helmRepoID)
	if err != nil {
		return nil, err
	}
	return uc.getAppsByRepo(helmRepo)
}

// 根据repo获取app详情包含app version
func (uc *AppUsecase) GetAppDetailByRepo(ctx context.Context, helmRepoID int64, appName, version string) (*App, error) {
	helmRepos, err := uc.repo.ListRepo(ctx)
	if err != nil {
		return nil, err
	}
	var helmRepo *AppHelmRepo
	for _, v := range helmRepos {
		if v.ID == helmRepoID {
			helmRepo = v
			break
		}
	}
	if helmRepo == nil {
		return nil, errors.New("helm repo not found")
	}
	return uc.getAppDetailByRepo(helmRepo, appName, version)
}

func (uc *AppUsecase) deployApp(ctx context.Context, appDeployed *DeployApp) error {
	helmPkg, err := helm.NewHelmPkg(uc.log, appDeployed.Namespace)
	if err != nil {
		return err
	}
	install, err := helmPkg.NewInstall()
	if err != nil {
		return err
	}
	cresource := uc.c.GetOceanResource()
	chart := fmt.Sprintf("%s%s", cresource.GetAppPath(), appDeployed.Chart)
	if appDeployed.AppTypeID == AppTypeRepo {
		chart = fmt.Sprintf("%s%s/%s", cresource.GetRepoPath(), appDeployed.AppName, appDeployed.Chart)
	}
	install.ReleaseName = appDeployed.ReleaseName
	install.Namespace = appDeployed.Namespace
	install.CreateNamespace = true
	install.GenerateName = true
	install.Version = appDeployed.Version
	install.DryRun = appDeployed.IsTest
	install.Atomic = true
	install.Wait = true
	release, err := helmPkg.RunInstall(ctx, install, chart, appDeployed.Config)
	appDeployed.Logs = helmPkg.GetLogs()
	if err != nil {
		return err
	}
	if release != nil {
		appDeployed.ReleaseName = release.Name
		appDeployed.Manifest = strings.TrimSpace(release.Manifest)
		if release.Info != nil {
			appDeployed.State = string(release.Info.Status)
			appDeployed.Notes = release.Info.Notes
		}
		return nil
	}
	appDeployed.State = releasePkg.StatusUnknown.String()
	return nil
}

func (uc *AppUsecase) unDeployApp(appDeployed *DeployApp) error {
	helmPkg, err := helm.NewHelmPkg(uc.log, appDeployed.Namespace)
	if err != nil {
		return err
	}
	uninstall, err := helmPkg.NewUninstall()
	if err != nil {
		return err
	}
	uninstall.KeepHistory = false
	uninstall.DryRun = appDeployed.IsTest
	uninstall.Wait = true
	resp, err := helmPkg.RunUninstall(uninstall, appDeployed.ReleaseName)
	appDeployed.Logs = helmPkg.GetLogs()
	if err != nil {
		return errors.WithMessage(err, "uninstall fail")
	}
	if resp != nil && resp.Release != nil && resp.Release.Info != nil {
		appDeployed.State = string(resp.Release.Info.Status)
	}
	appDeployed.Notes = resp.Info
	return nil
}

type AppDeployedResource struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Events    []string `json:"events"`
	StartedAt string   `json:"started_at"`
	Status    []string `json:"status"`
}

func (uc *AppUsecase) GetDeployedResources(ctx context.Context, appDeployID int64) ([]*AppDeployedResource, error) {
	appDeployed, err := uc.repo.GetDeployApp(ctx, appDeployID)
	if err != nil {
		return nil, err
	}
	resources := make([]*AppDeployedResource, 0)
	resourcesFunc := []func(ctx context.Context, appDeployed *DeployApp) ([]*AppDeployedResource, error){
		uc.getAppsReouces,
		uc.getNetResouces,
		uc.getPodResources,
	}
	for _, f := range resourcesFunc {
		res, err := f(ctx, appDeployed)
		if err != nil {
			return nil, err
		}
		resources = append(resources, res...)
	}
	return resources, nil
}

// 默认app安装
func (uc *AppUsecase) BaseInstallation(ctx context.Context, cluster *Cluster, project *Project) error {
	configMaps := make([]map[string]interface{}, 0)
	conf := reflect.ValueOf(uc.c)
	for i := 0; i < conf.NumField(); i++ {
		filed := conf.Field(i)
		if filed.Kind() != reflect.Map {
			continue
		}
		if configMap, ok := filed.Interface().(map[string]interface{}); ok {
			configMaps = append(configMaps, configMap)
		}
	}
	for _, configMap := range configMaps {
		enable, enableOk := utils.GetValueFromNestedMap(configMap, "base.enable")
		if !enableOk || !cast.ToBool(enable) {
			continue
		}
		repoUrl, repoUrlOk := utils.GetValueFromNestedMap(configMap, "base.repo_url")
		if !repoUrlOk {
			continue
		}
		repoName, repoNameOK := utils.GetValueFromNestedMap(configMap, "base.repo_name")
		if !repoNameOK {
			continue
		}
		appVersion, appVersionOK := utils.GetValueFromNestedMap(configMap, "base.version")
		if !appVersionOK {
			continue
		}
		chartName, chartNameOK := utils.GetValueFromNestedMap(configMap, "base.chart_name")
		if !chartNameOK {
			continue
		}
		namespace, namespaceOK := utils.GetValueFromNestedMap(configMap, "base.namespace")
		if !namespaceOK {
			namespace = "default"
		}
		repo, err := uc.repo.GetRepoByName(ctx, cast.ToString(repoName))
		if err != nil {
			return err
		}
		if repo == nil || repo.ID == 0 {
			repo = &AppHelmRepo{Name: cast.ToString(repoName), Url: cast.ToString(repoUrl)}
			err = uc.SaveRepo(ctx, repo)
			if err != nil {
				return err
			}
		}
		app, err := uc.GetAppByName(ctx, cast.ToString(chartName))
		if err != nil {
			return err
		}
		if app != nil && app.ID > 0 {
			continue
		}
		delete(configMap, "base")
		appConfigYamlByte, err := yaml.Marshal(configMap)
		if err != nil {
			return err
		}
		deployApp := &DeployApp{
			ClusterID: cluster.ID,
			AppName:   cast.ToString(chartName),
			AppTypeID: AppTypeRepo,
			RepoID:    repo.ID,
			Version:   cast.ToString(appVersion),
			UserID:    AdminID,
			Config:    string(appConfigYamlByte),
			Namespace: cast.ToString(namespace),
		}
		if project != nil {
			deployApp.ProjectID = project.ID
			deployApp.Namespace = project.Namespace
		}
		_, err = uc.DeployApp(ctx, deployApp)
		if err != nil {
			return err
		}
		err = uc.AppOperation(ctx, deployApp)
		if err != nil {
			return err
		}
	}
	return nil
}

func (uc *AppUsecase) getPodResources(ctx context.Context, appDeployed *DeployApp) (resources []*AppDeployedResource, err error) {
	resources = make([]*AppDeployedResource, 0)
	clusterClient, err := kubeclient.GetKubeClientSet()
	if err != nil {
		return nil, err
	}
	labelSelector := fmt.Sprintf("app.kubernetes.io/instance=%s", appDeployed.ReleaseName)
	podResources, _ := clusterClient.CoreV1().Pods(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if podResources != nil && len(podResources.Items) > 0 {
		for _, pod := range podResources.Items {
			resource := &AppDeployedResource{
				Name:      pod.Name,
				Kind:      "Pod",
				StartedAt: pod.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status:    []string{string(pod.Status.Phase)},
				Events:    make([]string, 0),
			}
			events, _ := clusterClient.CoreV1().Events(appDeployed.Namespace).List(ctx, metav1.ListOptions{
				FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod.Name),
			})
			if events != nil && len(events.Items) > 0 {
				for _, event := range events.Items {
					resource.Events = append(resource.Events, event.Message)
				}
			}
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func (uc *AppUsecase) getNetResouces(ctx context.Context, appDeployed *DeployApp) (resources []*AppDeployedResource, err error) {
	resources = make([]*AppDeployedResource, 0)
	clusterClient, err := kubeclient.GetKubeClientSet()
	if err != nil {
		return nil, err
	}
	labelSelector := fmt.Sprintf("app.kubernetes.io/instance=%s", appDeployed.ReleaseName)
	serviceResources, _ := clusterClient.CoreV1().Services(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if serviceResources != nil && len(serviceResources.Items) > 0 {
		for _, service := range serviceResources.Items {
			port := ""
			for _, v := range service.Spec.Ports {
				port = fmt.Sprintf("%s %d:%d/%s", port, v.Port, v.NodePort, v.Protocol)
			}
			externalIPs := ""
			for _, v := range service.Spec.ExternalIPs {
				externalIPs = fmt.Sprintf("%s,%s", externalIPs, v)
			}
			resource := &AppDeployedResource{
				Name:      service.Name,
				Kind:      "Service",
				StartedAt: service.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"Type: " + string(service.Spec.Type),
					"ClusterIP: " + service.Spec.ClusterIP,
					"ExternalIP: " + externalIPs,
					"Port: " + port,
				},
			}
			resources = append(resources, resource)
		}
	}
	ingressResources, _ := clusterClient.NetworkingV1beta1().Ingresses(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if ingressResources != nil && len(ingressResources.Items) > 0 {
		for _, ingress := range ingressResources.Items {
			class := ""
			if ingress.Spec.IngressClassName != nil {
				class = *ingress.Spec.IngressClassName
			}
			hosts := ""
			for _, v := range ingress.Spec.Rules {
				hosts = fmt.Sprintf("%s,%s", hosts, v.Host)
			}
			ports := ""
			for _, v := range ingress.Spec.TLS {
				for _, v := range v.Hosts {
					ports = fmt.Sprintf("%s,%s", ports, v)
				}
			}
			loadBalancerIP := ""
			for _, v := range ingress.Status.LoadBalancer.Ingress {
				loadBalancerIP = fmt.Sprintf("%s,%s", loadBalancerIP, v.IP)
			}
			resource := &AppDeployedResource{
				Name:      ingress.Name,
				Kind:      "Ingress",
				StartedAt: ingress.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"class: " + class,
					"hosts: " + hosts,
					"address: " + loadBalancerIP,
					"ports: " + ports,
				},
			}
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func (uc *AppUsecase) getAppsReouces(ctx context.Context, appDeployed *DeployApp) (resources []*AppDeployedResource, err error) {
	resources = make([]*AppDeployedResource, 0)
	clusterClient, err := kubeclient.GetKubeClientSet()
	if err != nil {
		return nil, err
	}
	labelSelector := fmt.Sprintf("app.kubernetes.io/instance=%s", appDeployed.ReleaseName)
	deploymentResources, _ := clusterClient.AppsV1().Deployments(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if deploymentResources != nil && len(deploymentResources.Items) > 0 {
		for _, deployment := range deploymentResources.Items {
			resource := &AppDeployedResource{
				Name:      deployment.Name,
				Kind:      "Deployment",
				StartedAt: deployment.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"Ready: " + cast.ToString(deployment.Status.ReadyReplicas),
					"Up-to-date: " + cast.ToString(deployment.Status.UpdatedReplicas),
					"Available: " + cast.ToString(deployment.Status.AvailableReplicas),
				},
			}
			resources = append(resources, resource)
		}
	}

	statefulSetResources, _ := clusterClient.AppsV1().StatefulSets(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if statefulSetResources != nil && len(statefulSetResources.Items) > 0 {
		for _, statefulSet := range statefulSetResources.Items {
			resource := &AppDeployedResource{
				Name:      statefulSet.Name,
				Kind:      "StatefulSet",
				StartedAt: statefulSet.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"Ready: " + cast.ToString(statefulSet.Status.ReadyReplicas),
					"Up-to-date: " + cast.ToString(statefulSet.Status.UpdatedReplicas),
					"Available: " + cast.ToString(statefulSet.Status.AvailableReplicas),
				},
			}
			resources = append(resources, resource)
		}
	}

	deamonsetResources, _ := clusterClient.AppsV1().DaemonSets(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if deamonsetResources != nil && len(deamonsetResources.Items) > 0 {
		for _, deamonset := range deamonsetResources.Items {
			resource := &AppDeployedResource{
				Name:      deamonset.Name,
				Kind:      "Deamonset",
				StartedAt: deamonset.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"Desired: " + cast.ToString(deamonset.Status.DesiredNumberScheduled),
					"Current: " + cast.ToString(deamonset.Status.CurrentNumberScheduled),
					"Ready: " + cast.ToString(deamonset.Status.NumberReady),
					"Up-to-date: " + cast.ToString(deamonset.Status.UpdatedNumberScheduled),
					"Available: " + cast.ToString(deamonset.Status.NumberAvailable),
				},
			}
			resources = append(resources, resource)
		}
	}

	jobResources, _ := clusterClient.BatchV1().Jobs(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if jobResources != nil && len(jobResources.Items) > 0 {
		for _, job := range jobResources.Items {
			resource := &AppDeployedResource{
				Name:      job.Name,
				Kind:      "Job",
				StartedAt: job.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"Completions: " + cast.ToString(job.Spec.Completions),
					"Parallelism: " + cast.ToString(job.Spec.Parallelism),
					"BackoffLimit: " + cast.ToString(job.Spec.BackoffLimit),
				},
			}
			resources = append(resources, resource)
		}
	}

	cronjobResources, _ := clusterClient.BatchV1beta1().CronJobs(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if cronjobResources != nil && len(cronjobResources.Items) > 0 {
		for _, cronjob := range cronjobResources.Items {
			suspend := *cronjob.Spec.Suspend // Dereference the pointer to bool
			resource := &AppDeployedResource{
				Name:      cronjob.Name,
				Kind:      "Cronjob",
				StartedAt: cronjob.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"Schedule: " + cronjob.Spec.Schedule,
					"Suspend: " + cast.ToString(suspend),
					"Active: " + cast.ToString(len(cronjob.Status.Active)),
					"Last Schedule: " + cronjob.Status.LastScheduleTime.Format("2006-01-02 15:04:05"),
				},
			}
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

// add helm repo
func (uc *AppUsecase) addRepo(helmRepo *AppHelmRepo) error {
	settings := cli.New()
	r, err := repo.NewChartRepository(&repo.Entry{
		Name: helmRepo.Name,
		URL:  helmRepo.Url,
	}, getter.All(settings))
	if err != nil {
		return err
	}
	cresource := uc.c.GetOceanResource()
	r.CachePath = cresource.GetRepoPath()
	indexFile, err := r.DownloadIndexFile()
	if err != nil {
		return err
	}
	helmRepo.SetIndexPath(indexFile)
	return nil
}

func (uc *AppUsecase) getAppsByRepo(helmRepo *AppHelmRepo) ([]*App, error) {
	index, err := repo.LoadIndexFile(helmRepo.IndexPath)
	if err != nil {
		return nil, err
	}
	apps := make([]*App, 0)
	for chartName, chartVersions := range index.Entries {
		app := &App{
			Name:          chartName,
			AppTypeID:     AppTypeRepo,
			AppHelmRepoID: helmRepo.ID,
			Versions:      make([]*AppVersion, 0),
		}
		app.CreatedAt = helmRepo.CreatedAt
		app.UpdatedAt = helmRepo.UpdatedAt
		for _, chartMatedata := range chartVersions {
			if app.Icon == "" {
				app.Icon = chartMatedata.Icon
			}
			if len(chartMatedata.URLs) == 0 {
				return nil, errors.New("chart urls is empty")
			}
			appVersion := &AppVersion{
				AppName:     chartName,
				Name:        chartMatedata.Name,
				Chart:       chartMatedata.URLs[0],
				Version:     chartMatedata.Version,
				Description: chartMatedata.Description,
				State:       AppTested,
			}
			app.AddVersion(appVersion)
		}
		apps = append(apps, app)
	}
	return apps, nil
}

func (uc *AppUsecase) getAppDetailByRepo(helmRepo *AppHelmRepo, appName, version string) (*App, error) {
	cresource := uc.c.GetOceanResource()
	index, err := repo.LoadIndexFile(helmRepo.IndexPath)
	if err != nil {
		return nil, err
	}
	app := &App{
		Name:          appName,
		AppTypeID:     AppTypeRepo,
		AppHelmRepoID: helmRepo.ID,
		Versions:      make([]*AppVersion, 0),
	}
	for chartName, chartVersions := range index.Entries {
		if chartName != appName {
			continue
		}
		for i, chartMatedata := range chartVersions {
			if len(chartMatedata.URLs) == 0 {
				return nil, errors.New("chart urls is empty")
			}
			if app.Icon == "" {
				app.Icon = chartMatedata.Icon
			}
			appVersion := &AppVersion{
				AppName:     chartName,
				Name:        chartMatedata.Name,
				Chart:       chartMatedata.URLs[0],
				Version:     chartMatedata.Version,
				Description: chartMatedata.Description,
			}
			if (version == "" && i == 0) || (version != "" && version == chartMatedata.Version) {
				err = appVersion.GetChartInfo(cresource.GetRepoPath())
				if err != nil {
					return nil, err
				}
			}
			app.AddVersion(appVersion)
		}
	}
	return app, nil
}
