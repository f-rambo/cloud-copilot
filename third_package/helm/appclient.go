package helm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	releasePkg "helm.sh/helm/v3/pkg/release"
	helmrepo "helm.sh/helm/v3/pkg/repo"
)

type AppConstructRepo struct {
	log *log.Helper
	c   *conf.Bootstrap
}

func NewAppConstructRepo(c *conf.Bootstrap, logger log.Logger) biz.AppConstruct {
	return &AppConstructRepo{
		log: log.NewHelper(logger),
		c:   c,
	}
}

func (r *AppConstructRepo) GetAppAndVersionInfo(ctx context.Context, app *biz.App, appVersion *biz.AppVersion) error {
	charInfo, err := GetLocalChartInfo(appVersion.Chart)
	if err != nil {
		return err
	}
	charInfoMetadata, err := json.Marshal(charInfo.Metadata)
	if err != nil {
		return err
	}
	app.Name = charInfo.Name
	app.Readme = charInfo.Readme
	app.Description = charInfo.Description
	app.Metadata = charInfoMetadata

	appVersion.DefaultConfig = charInfo.Config
	appVersion.Version = charInfo.Version
	appVersion.Chart = charInfo.Chart
	return nil
}

func (r *AppConstructRepo) AppRelease(ctx context.Context, app *biz.App, appVersion *biz.AppVersion, appRelease *biz.AppRelease, appRepo *biz.AppRepo) error {
	helmPkg, err := NewHelmPkg(r.log, appRelease.Namespace)
	if err != nil {
		return err
	}
	install, err := helmPkg.NewInstall()
	if err != nil {
		return err
	}
	if appRepo != nil && appVersion.Chart == "" {
		appPath, err := utils.GetServerStorePathByNames(utils.AppPackage)
		if err != nil {
			return err
		}
		pullClient, err := helmPkg.NewPull()
		if err != nil {
			return err
		}
		err = helmPkg.RunPull(pullClient, appPath, fmt.Sprintf("%s/%s", appRepo.Name, app.Name))
		if err != nil {
			return err
		}
		appVersion.Chart = filepath.Join(appPath, fmt.Sprintf("%s-%s.tgz", app.Name, appVersion.Version))
	}
	install.ReleaseName = appRelease.ReleaseName
	install.Namespace = appRelease.Namespace
	install.CreateNamespace = true
	install.GenerateName = true
	install.Version = appVersion.Version
	install.DryRun = appRelease.Dryrun
	install.Atomic = appRelease.Atomic
	install.Wait = appRelease.Wait
	release, err := helmPkg.RunInstall(ctx, install, appVersion.Chart, appRelease.Config)
	appRelease.Logs = helmPkg.GetLogs()
	if err != nil {
		return err
	}
	if release != nil {
		appRelease.ReleaseName = release.Name
		appRelease.Manifest = strings.TrimSpace(release.Manifest)
		if release.Info != nil {
			appRelease.Status = biz.AppReleaseSatus(release.Info.Status)
			appRelease.Notes = release.Info.Notes
		}
		return nil
	}
	appRelease.Status = biz.AppReleaseSatus(releasePkg.StatusUnknown)
	return nil
}

func (r *AppConstructRepo) DeleteAppRelease(ctx context.Context, appRelease *biz.AppRelease) error {
	helmPkg, err := NewHelmPkg(r.log, appRelease.Namespace)
	if err != nil {
		return err
	}
	uninstall, err := helmPkg.NewUninstall()
	if err != nil {
		return err
	}
	uninstall.KeepHistory = false
	uninstall.DryRun = appRelease.Dryrun
	uninstall.Wait = appRelease.Wait
	resp, err := helmPkg.RunUninstall(uninstall, appRelease.ReleaseName)
	appRelease.Logs = helmPkg.GetLogs()
	if err != nil {
		return errors.WithMessage(err, "uninstall fail")
	}
	if resp != nil && resp.Release != nil && resp.Release.Info != nil {
		appRelease.Status = biz.AppReleaseSatus(resp.Release.Info.Status)
	}
	appRelease.Notes = resp.Info
	return nil
}

func (r *AppConstructRepo) AddAppRepo(ctx context.Context, repo *biz.AppRepo) (err error) {
	settings := cli.New()
	res, err := helmrepo.NewChartRepository(&helmrepo.Entry{
		Name: repo.Name,
		URL:  repo.Url,
	}, getter.All(settings))
	if err != nil {
		return err
	}
	res.CachePath, err = utils.GetServerStorePathByNames(utils.AppPackage)
	if err != nil {
		return err
	}
	indexFile, err := res.DownloadIndexFile()
	if err != nil {
		return err
	}
	repo.SetIndexPath(indexFile)
	return nil
}

func (r *AppConstructRepo) GetAppsByRepo(ctx context.Context, repo *biz.AppRepo) ([]*biz.App, error) {
	index, err := helmrepo.LoadIndexFile(repo.IndexPath)
	if err != nil {
		return nil, err
	}
	apps := make([]*biz.App, 0)
	for chartName, chartVersions := range index.Entries {
		app := &biz.App{Name: chartName, AppRepoID: repo.ID, Versions: make([]*biz.AppVersion, 0)}
		app.CreatedAt = repo.CreatedAt
		app.UpdatedAt = repo.UpdatedAt
		for _, chartMatedata := range chartVersions {
			if len(chartMatedata.URLs) == 0 {
				return nil, errors.New("chart urls is empty")
			}
			app.Icon = chartMatedata.Icon
			app.Description = chartMatedata.Description
			appVersion := &biz.AppVersion{Name: chartMatedata.Name, Chart: chartMatedata.URLs[0], Version: chartMatedata.Version}
			app.AddVersion(appVersion)
		}
		apps = append(apps, app)
	}
	return apps, nil
}

func (r *AppConstructRepo) GetAppDetailByRepo(ctx context.Context, repo *biz.AppRepo, appName, version string) (*biz.App, error) {
	index, err := helmrepo.LoadIndexFile(repo.IndexPath)
	if err != nil {
		return nil, err
	}
	app := &biz.App{Name: appName, AppRepoID: repo.ID, Versions: make([]*biz.AppVersion, 0)}
	for chartName, chartVersions := range index.Entries {
		if chartName != appName {
			continue
		}
		for i, chartMatedata := range chartVersions {
			if len(chartMatedata.URLs) == 0 {
				return nil, errors.New("chart urls is empty")
			}
			app.Icon = chartMatedata.Icon
			app.Name = chartName
			app.Description = chartMatedata.Description
			appVersion := &biz.AppVersion{Name: chartMatedata.Name, Chart: chartMatedata.URLs[0], Version: chartMatedata.Version}
			if (version == "" && i == 0) || (version != "" && version == chartMatedata.Version) {
				err = r.GetAppAndVersionInfo(ctx, app, appVersion)
				if err != nil {
					return nil, err
				}
			}
			app.AddVersion(appVersion)
		}
	}
	return app, nil
}

func (r *AppConstructRepo) DeleteApp(ctx context.Context, app *biz.App) error {
	appPath, err := utils.GetServerStorePathByNames(utils.AppPackage)
	if err != nil {
		return err
	}
	err = os.Remove(appPath)
	if err != nil {
		return err
	}
	return nil
}

func (r *AppConstructRepo) DeleteAppVersion(ctx context.Context, app *biz.App, appVersion *biz.AppVersion) (err error) {
	appPath, err := utils.GetServerStorePathByNames(utils.AppPackage)
	if err != nil {
		return err
	}
	for _, v := range app.Versions {
		if v.Chart == "" || appVersion.ID != v.ID {
			continue
		}
		chartPath := filepath.Join(appPath, v.Chart)
		if utils.IsFileExist(chartPath) {
			err = os.Remove(chartPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
