package biz

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	releasePkg "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Dependency describes a chart upon which another chart depends.
//
// Dependencies can be used to express developer intent, or to capture the state
// of a chart.
type Dependency struct {
	// Name is the name of the dependency.
	//
	// This must mach the name in the dependency's Chart.yaml.
	Name string `json:"name"`
	// Version is the version (range) of this chart.
	//
	// A lock file will always produce a single version, while a dependency
	// may contain a semantic version range.
	Version string `json:"version,omitempty"`
	// The URL to the repository.
	//
	// Appending `index.yaml` to this string should result in a URL that can be
	// used to fetch the repository index.
	Repository string `json:"repository"`
	// A yaml path that resolves to a boolean, used for enabling/disabling charts (e.g. subchart1.enabled )
	Condition string `json:"condition,omitempty"`
	// Tags can be used to group charts for enabling/disabling together
	Tags []string `json:"tags,omitempty"`
	// Enabled bool determines if chart should be loaded
	Enabled bool `json:"enabled,omitempty"`
	// ImportValues holds the mapping of source values to parent key to be imported. Each item can be a
	// string or pair of child/parent sublist items.
	ImportValues []interface{} `json:"import-values,omitempty"`
	// Alias usable alias to be used for the chart
	Alias string `json:"alias,omitempty"`
}

// Maintainer describes a Chart maintainer.
type Maintainer struct {
	// Name is a user name or organization name
	Name string `json:"name,omitempty"`
	// Email is an optional email address to contact the named maintainer
	Email string `json:"email,omitempty"`
	// URL is an optional URL to an address for the named maintainer
	URL string `json:"url,omitempty"`
}

// Metadata for a Chart file. This models the structure of a Chart.yaml file.
type Metadata struct {
	// The name of the chart. Required.
	Name string `json:"name,omitempty"`
	// The URL to a relevant project page, git repo, or contact person
	Home string `json:"home,omitempty"`
	// Source is the URL to the source code of this chart
	Sources []string `json:"sources,omitempty"`
	// A SemVer 2 conformant version string of the chart. Required.
	Version string `json:"version,omitempty"`
	// A one-sentence description of the chart
	Description string `json:"description,omitempty"`
	// A list of string keywords
	Keywords []string `json:"keywords,omitempty"`
	// A list of name and URL/email address combinations for the maintainer(s)
	Maintainers []*Maintainer `json:"maintainers,omitempty"`
	// The URL to an icon file.
	Icon string `json:"icon,omitempty"`
	// The API Version of this chart. Required.
	APIVersion string `json:"apiVersion,omitempty"`
	// The condition to check to enable chart
	Condition string `json:"condition,omitempty"`
	// The tags to check to enable chart
	Tags string `json:"tags,omitempty"`
	// The version of the application enclosed inside of this chart.
	AppVersion string `json:"appVersion,omitempty"`
	// Whether or not this chart is deprecated
	Deprecated bool `json:"deprecated,omitempty"`
	// Annotations are additional mappings uninterpreted by Helm,
	// made available for inspection by other applications.
	Annotations map[string]string `json:"annotations,omitempty"`
	// KubeVersion is a SemVer constraint specifying the version of Kubernetes required.
	KubeVersion string `json:"kubeVersion,omitempty"`
	// Dependencies are a list of dependencies for a chart.
	Dependencies []*Dependency `json:"dependencies,omitempty"`
	// Specifies the chart type: application or library
	Type string `json:"type,omitempty"`
}

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
	ID          int64    `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	AppID       int64    `json:"app_id" gorm:"column:app_id; default:0; NOT NULL; index"`
	AppName     string   `json:"app_name,omitempty" gorm:"column:app_name; default:''; NOT NULL"`
	Name        string   `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Chart       string   `json:"chart,omitempty" gorm:"column:chart; default:''; NOT NULL"`
	Version     string   `json:"version,omitempty" gorm:"column:version; default:''; NOT NULL; index"`
	Config      string   `json:"config,omitempty" gorm:"column:config; default:''; NOT NULL"`
	Readme      string   `json:"readme,omitempty" gorm:"-"`
	State       string   `json:"state,omitempty" gorm:"column:state; default:''; NOT NULL"`
	TestResult  string   `json:"test_result,omitempty" gorm:"column:test_result; default:''; NOT NULL"` // 哪些资源部署成功，哪些失败
	Description string   `json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`
	Metadata    Metadata `json:"metadata,omitempty" gorm:"-"`
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
	Manifest    string `json:"manifest,omitempty" gorm:"column:manifest; default:''; NOT NULL"`
	Notes       string `json:"notes,omitempty" gorm:"column:notes; default:''; NOT NULL"`
	Logs        string `json:"logs,omitempty" gorm:"column:logs; default:''; NOT NULL"`
	gorm.Model
	log *log.Helper `json:"-" gorm:"-"`
}

func (d *DeployApp) newLog(log *log.Helper) *DeployApp {
	d.log = log
	return d
}

func (d *DeployApp) deployAppLogf(format string, v ...interface{}) {
	d.log.Debugf(format, v...)
	d.Logs = fmt.Sprintf("%s%s\n", d.Logs, fmt.Sprintf(format, v...))
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
	if v.Chart == "" {
		return nil
	}
	if utils.IsHttpUrl(v.Chart) {
		appPath = fmt.Sprintf("%s%s/", appPath, v.AppName)
		fileName := utils.GetFileNameByUrl(v.Chart)
		err := utils.DownloadFile(v.Chart, appPath, fileName)
		if err != nil {
			return errors.WithMessage(err, "download chart fail")
		}
		v.Chart = fileName
	}
	chartPath := appPath + v.Chart
	readme, err := action.NewShow(action.ShowReadme).Run(chartPath)
	if err != nil {
		return errors.WithMessage(err, "show readme fail")
	}
	chartYaml, err := action.NewShow(action.ShowChart).Run(chartPath)
	if err != nil {
		return errors.WithMessage(err, "show chart fail")
	}
	valuesYaml, err := action.NewShow(action.ShowValues).Run(chartPath)
	if err != nil {
		return errors.WithMessage(err, "show values fail")
	}
	chartMateData := &Metadata{}
	err = yaml.Unmarshal([]byte(chartYaml), chartMateData)
	if err != nil {
		return errors.WithMessage(err, "unmarshal chart yaml fail")
	}
	v.Name = chartMateData.Name
	v.Config = valuesYaml
	v.Readme = readme
	v.Description = chartMateData.Description
	v.Metadata = *chartMateData
	v.Version = chartMateData.Version
	v.AppName = chartMateData.Name
	return nil
}

func (v *AppVersion) GetAppDeployed() *DeployApp {
	return &DeployApp{
		ReleaseName: v.AppName,
		AppID:       v.AppID,
		VersionID:   v.ID,
		Version:     v.Version,
		Chart:       v.Chart,
		AppName:     v.AppName,
		Namespace:   "default",
		Config:      v.Config,
		State:       releasePkg.StatusUnknown.String(),
	}
}

type AppRepo interface {
	Save(context.Context, *App) error
	List(ctx context.Context, appReq *App, page, pageSize int32) ([]*App, int32, error)
	Get(ctx context.Context, appID int64) (*App, error)
	Delete(ctx context.Context, appID, versionID int64) error
	CreateAppType(ctx context.Context, appType *AppType) error
	ListAppType(ctx context.Context) ([]*AppType, error)
	DeleteAppType(ctx context.Context, appTypeID int64) error
	SaveDeployApp(ctx context.Context, appDeployed *DeployApp) error
	DeleteDeployApp(ctx context.Context, id int64) error
	DeployAppList(ctx context.Context, appDeployedReq *DeployApp, page, pageSuze int32) ([]*DeployApp, int32, error)
	GetDeployApp(ctx context.Context, id int64) (*DeployApp, error)
	SaveRepo(ctx context.Context, helmRepo *AppHelmRepo) error
	ListRepo(ctx context.Context) ([]*AppHelmRepo, error)
	DeleteRepo(ctx context.Context, helmRepoID int64) error
}

type AppUsecase struct {
	repo        AppRepo
	log         *log.Helper
	resConf     *conf.Resource
	clusterRepo ClusterRepo
	projectRepo ProjectRepo
}

func NewAppUsecase(repo AppRepo, logger log.Logger, resConf *conf.Resource, clusterRepo ClusterRepo, projectRepo ProjectRepo) *AppUsecase {
	return &AppUsecase{repo, log.NewHelper(logger), resConf, clusterRepo, projectRepo}
}

func (uc *AppUsecase) GetAppByName(ctx context.Context, name string) (app *App, err error) {
	apps, _, err := uc.repo.List(ctx, &App{Name: name}, 1, 1)
	if err != nil {
		return nil, err
	}
	for _, v := range apps {
		return v, nil
	}
	return nil, nil
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
	err = appVersion.GetChartInfo(uc.resConf.GetAppPath())
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
	if app.Icon != "" && utils.IsFileExist(uc.resConf.GetIconPath()+app.Icon) && versionId == 0 {
		err = utils.DeleteFile(uc.resConf.GetIconPath() + app.Icon)
		if err != nil {
			return err
		}
	}
	for _, v := range app.Versions {
		if v.Chart != "" && utils.IsFileExist(uc.resConf.GetAppPath()+v.Chart) && versionId == 0 {
			err = utils.DeleteFile(uc.resConf.GetAppPath() + v.Chart)
			if err != nil {
				return err
			}
		}
		if v.Chart != "" && utils.IsFileExist(uc.resConf.GetAppPath()+v.Chart) && versionId == v.ID {
			err = utils.DeleteFile(uc.resConf.GetAppPath() + v.Chart)
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

func (uc *AppUsecase) DeployAppList(ctx context.Context, appDeployedReq *DeployApp, page, pageSize int32) ([]*DeployApp, int32, error) {
	return uc.repo.DeployAppList(ctx, appDeployedReq, page, pageSize)
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
	deployAppData, err := uc.findOneDeployApp(ctx, &DeployApp{AppID: appID, VersionID: versionID, IsTest: true})
	if err != nil {
		return nil, err
	}
	if deployAppData != nil {
		appDeployed.ID = deployAppData.ID

	}
	deployAppErr := uc.deployApp(ctx, appDeployed)
	if deployAppErr != nil {
		appVersion.State = AppTestFailed
		appVersion.TestResult = err.Error()
	}
	if deployAppErr == nil {
		appVersion.State = AppTested
		appVersion.TestResult = "success"
	}
	err = uc.repo.Save(ctx, app)
	if err != nil {
		return nil, err
	}
	err = uc.repo.SaveDeployApp(ctx, appDeployed)
	if err != nil {
		return nil, err
	}
	return appDeployed, deployAppErr
}

func (uc *AppUsecase) DeployApp(ctx context.Context, deployAppReq *DeployApp) (*DeployApp, error) {
	// 两种app部署方式，一种是app package，一种是helm repo
	var app *App
	var appVersion *AppVersion
	var err error
	if deployAppReq.AppTypeID == AppTypeRepo {
		app, err = uc.GetAppDetailByRepo(ctx, deployAppReq.RepoID, deployAppReq.AppName, deployAppReq.Version)
		if err != nil {
			return nil, err
		}
		appVersion = app.GetVersion(deployAppReq.Version)
		if appVersion == nil {
			return nil, errors.New("app version not found")
		}
	} else {
		app, err = uc.Get(ctx, deployAppReq.AppID, deployAppReq.VersionID)
		if err != nil {
			return nil, err
		}
		appVersion = app.GetVersionById(deployAppReq.VersionID)
		if appVersion == nil {
			return nil, errors.New("app version not found")
		}
	}

	project, err := uc.projectRepo.Get(ctx, deployAppReq.ProjectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.New("project not found")
	}
	appDeployed := appVersion.GetAppDeployed()
	appDeployed.RepoID = deployAppReq.RepoID
	appDeployed.AppTypeID = app.AppTypeID
	appDeployed.ClusterID = deployAppReq.ClusterID
	appDeployed.ProjectID = deployAppReq.ProjectID
	appDeployed.Namespace = project.Namespace
	appDeployed.Config = deployAppReq.Config
	appDeployed.UserID = deployAppReq.UserID

	deployAppRes, err := uc.findOneDeployApp(ctx, &DeployApp{
		ID:        deployAppReq.ID,
		ProjectID: deployAppReq.ProjectID,
		ClusterID: deployAppReq.ClusterID,
		AppID:     deployAppReq.AppID,
		VersionID: deployAppReq.VersionID,
		AppName:   deployAppReq.AppName,
		Version:   deployAppReq.Version,
	})
	if err != nil {
		return nil, err
	}
	if deployAppRes != nil {
		appDeployed.ID = deployAppRes.ID
	}
	deployAppErr := uc.deployApp(ctx, appDeployed)
	err = uc.repo.SaveDeployApp(ctx, appDeployed)
	if err != nil {
		return nil, err
	}
	return appDeployed, deployAppErr
}

func (uc *AppUsecase) findOneDeployApp(ctx context.Context, deployAppReq *DeployApp) (*DeployApp, error) {
	appDeployeds, _, err := uc.repo.DeployAppList(ctx, deployAppReq, 1, 1)
	if err != nil {
		return nil, err
	}
	for _, v := range appDeployeds {
		return v, nil
	}
	return nil, nil

}

func (uc *AppUsecase) DeleteDeployedApp(ctx context.Context, id int64) error {
	appDeployed, err := uc.repo.GetDeployApp(ctx, id)
	if err != nil {
		return err
	}
	if appDeployed == nil {
		return nil
	}
	err = uc.unDeployApp(ctx, appDeployed)
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
	unDeployAppErr := uc.unDeployApp(ctx, appDeployed)
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
			return errors.New("repo name already exists")
		}
	}
	err = uc.addRepo(ctx, helmRepo)
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
	return uc.getAppsByRepo(ctx, helmRepo)
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
	return uc.getAppDetailByRepo(ctx, helmRepo, appName, version)
}

func (uc *AppUsecase) deployApp(ctx context.Context, appDeployed *DeployApp) error {
	// todo 需要区分不同的集群
	settings := cli.New()
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(
		settings.RESTClientGetter(),
		appDeployed.Namespace,
		os.Getenv("HELM_DRIVER"),
		appDeployed.newLog(uc.log).deployAppLogf,
	); err != nil {
		return errors.WithMessage(err, "init action fail")
	}
	install := action.NewInstall(actionConfig)
	install.Namespace = appDeployed.Namespace
	install.ReleaseName = appDeployed.ReleaseName
	install.Version = appDeployed.Version
	install.CreateNamespace = true
	install.DryRun = appDeployed.IsTest
	install.GenerateName = true
	install.Atomic = true
	install.Wait = true
	install.Timeout = 3 * time.Minute
	chartPath := fmt.Sprintf("%s%s", uc.resConf.GetAppPath(), appDeployed.Chart)
	if appDeployed.AppTypeID == AppTypeRepo {
		chartPath = fmt.Sprintf("%s%s/%s", uc.resConf.GetRepoPath(), appDeployed.AppName, appDeployed.Chart)
	}
	chart, err := loader.Load(chartPath)
	if err != nil {
		return errors.WithMessage(err, "load chart fail")
	}
	values := make(map[string]interface{})
	if appDeployed.Config != "" {
		err = yaml.Unmarshal([]byte(appDeployed.Config), &values)
		if err != nil {
			return err
		}
	}
	list := action.NewList(actionConfig)
	appList, err := list.Run()
	if err != nil {
		return err
	}
	for _, release := range appList {
		// 升级配置更新
		if release.Name == appDeployed.ReleaseName {
			upgrade := action.NewUpgrade(actionConfig)
			upgrade.Namespace = appDeployed.Namespace
			upgrade.Version = appDeployed.Version
			upgrade.DryRun = appDeployed.IsTest
			upgrade.Atomic = true
			upgrade.Wait = true
			upgrade.Timeout = 3 * time.Minute
			_, err = upgrade.Run(appDeployed.ReleaseName, chart, values)
			if err != nil {
				return errors.WithMessage(err, "upgrade fail")
			}
			return nil
		}
	}
	release, err := install.Run(chart, values)
	if err != nil {
		return errors.WithMessage(err, "install fail")
	}
	if release != nil {
		appDeployed.ReleaseName = release.Name
		appDeployed.Manifest = release.Manifest
		if release.Info != nil {
			appDeployed.State = string(release.Info.Status)
			appDeployed.Notes = release.Info.Notes
		}
	} else {
		appDeployed.State = releasePkg.StatusUnknown.String()
	}
	return nil
}

func (uc *AppUsecase) unDeployApp(ctx context.Context, appDeployed *DeployApp) error {
	settings := cli.New()
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(
		settings.RESTClientGetter(),
		appDeployed.Namespace,
		os.Getenv("HELM_DRIVER"),
		appDeployed.newLog(uc.log).deployAppLogf,
	); err != nil {
		return errors.WithMessage(err, "init action fail")
	}
	list := action.NewList(actionConfig)
	appList, err := list.Run()
	if err != nil {
		return err
	}
	isInstalled := false
	for _, release := range appList {
		if release.Name == appDeployed.ReleaseName {
			isInstalled = true
		}
	}
	if !isInstalled {
		appDeployed.State = releasePkg.StatusUninstalled.String()
		uc.log.Infof("%s not installed", appDeployed.ReleaseName)
		return nil
	}
	uninstall := action.NewUninstall(actionConfig)
	uninstall.KeepHistory = false
	resp, err := uninstall.Run(appDeployed.ReleaseName)
	if err != nil {
		return errors.WithMessage(err, "uninstall fail")
	}
	if resp != nil && resp.Release != nil && resp.Release.Info != nil {
		appDeployed.State = string(resp.Release.Info.Status)
	}
	appDeployed.Notes = resp.Info
	return nil
}

func (uc *AppUsecase) TrackingAppDeployed(ctx context.Context, appDeployedId int64) error {
	deployApp, err := uc.repo.GetDeployApp(ctx, appDeployedId)
	if err != nil {
		return err
	}
	getAppDeployedStatusErr := uc.getAppDeployedStatus(ctx, deployApp)
	if getAppDeployedStatusErr != nil && (getAppDeployedStatusErr.Error() == "timeout" || getAppDeployedStatusErr.Error() == "release info is nil") {
		deployApp.State = releasePkg.StatusUnknown.String()
		deployApp.Notes = err.Error()
	}
	err = uc.repo.SaveDeployApp(ctx, deployApp)
	if err != nil {
		return err
	}
	return getAppDeployedStatusErr
}

func (uc *AppUsecase) getAppDeployedStatus(ctx context.Context, appDeployed *DeployApp) error {
	settings := cli.New()
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(
		settings.RESTClientGetter(),
		appDeployed.Namespace,
		os.Getenv("HELM_DRIVER"),
		appDeployed.newLog(uc.log).deployAppLogf,
	); err != nil {
		return errors.WithMessage(err, "init action fail")
	}
	statucAction := action.NewStatus(actionConfig)
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	tick := ticker.C
	for {
		select {
		case <-timeout:
			return errors.New("timeout")
		case <-tick:
			release, err := statucAction.Run(appDeployed.ReleaseName)
			if err != nil {
				return err
			}
			if release == nil || release.Info == nil {
				return errors.New("release info is nil")
			}
			appDeployed.State = string(release.Info.Status)
			appDeployed.Manifest = release.Manifest
			appDeployed.Notes = release.Info.Notes
			// StatusPendingInstall：表示一个安装操作正在进行中。
			// StatusPendingUpgrade：表示一个升级操作正在进行中。
			// StatusPendingRollback：表示一个回滚操作正在进行中。
			// StatusUninstalling：表示一个卸载操作正在进行中。
			if release.Info.Status != releasePkg.StatusPendingInstall &&
				release.Info.Status != releasePkg.StatusPendingUpgrade &&
				release.Info.Status != releasePkg.StatusPendingRollback &&
				release.Info.Status != releasePkg.StatusUninstalling {
				return nil
			}
		}
	}
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

func (uc *AppUsecase) getPodResources(ctx context.Context, appDeployed *DeployApp) (resources []*AppDeployedResource, err error) {
	resources = make([]*AppDeployedResource, 0)
	clusterClient, err := uc.clusterRepo.ClusterClient(ctx, appDeployed.ClusterID)
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
	clusterClient, err := uc.clusterRepo.ClusterClient(ctx, appDeployed.ClusterID)
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
	clusterClient, err := uc.clusterRepo.ClusterClient(ctx, appDeployed.ClusterID)
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
func (uc *AppUsecase) addRepo(ctx context.Context, helmRepo *AppHelmRepo) error {
	settings := cli.New()
	r, err := repo.NewChartRepository(&repo.Entry{
		Name: helmRepo.Name,
		URL:  helmRepo.Url,
	}, getter.All(settings))
	if err != nil {
		return err
	}
	r.CachePath = uc.resConf.GetRepoPath()
	indexFile, err := r.DownloadIndexFile()
	if err != nil {
		return err
	}
	helmRepo.SetIndexPath(indexFile)
	return nil
}

func (uc *AppUsecase) getAppsByRepo(ctx context.Context, helmRepo *AppHelmRepo) ([]*App, error) {
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
			}
			app.AddVersion(appVersion)
		}
		apps = append(apps, app)
	}
	return apps, nil
}

func (uc *AppUsecase) getAppDetailByRepo(ctx context.Context, helmRepo *AppHelmRepo, appName, version string) (*App, error) {
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
				err = appVersion.GetChartInfo(uc.resConf.GetRepoPath())
				if err != nil {
					return nil, err
				}
			}
			app.AddVersion(appVersion)
		}
	}
	return app, nil
}
