package helm

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
)

func (h *HelmPkg) NewUpGrade() (*action.Upgrade, error) {
	client := action.NewUpgrade(h.actionConfig)
	client.Namespace = settings.Namespace()
	registryClient, err := newRegistryClient(client.CertFile, client.KeyFile, client.CaFile,
		client.InsecureSkipTLSverify, client.PlainHTTP)
	if err != nil {
		return nil, err
	}
	client.SetRegistryClient(registryClient)
	return client, nil
}

func (h *HelmPkg) RunUpgrade(ctx context.Context, client *action.Upgrade, name, chart, value string) (*release.Release, error) {
	if client.Timeout == 0 {
		client.Timeout = 300 * time.Second
	}
	if client.Version == "" && client.Devel {
		client.Version = ">0.0.0-0"
	}
	if client.DryRunOption == "" {
		client.DryRunOption = "none"
	}
	chartPath, err := client.ChartPathOptions.LocateChart(chart, settings)
	if err != nil {
		return nil, err
	}
	// Validate dry-run flag value is one of the allowed values
	if err := validateDryRunOptionFlag(client.DryRunOption); err != nil {
		return nil, err
	}

	p := getter.All(settings)

	// Check chart dependencies to make sure all are present in /charts
	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}
	if req := ch.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(ch, req); err != nil {
			err = errors.Wrap(err, "An error occurred while checking for chart dependencies. You may need to run `helm dependency build` to fetch missing dependencies")
			if client.DependencyUpdate {
				man := &downloader.Manager{
					Out:              h,
					ChartPath:        chartPath,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
					Debug:            settings.Debug,
				}
				if err := man.Update(); err != nil {
					return nil, err
				}
				// Reload the chart with the updated Chart.lock file.
				if ch, err = loader.Load(chartPath); err != nil {
					return nil, errors.Wrap(err, "failed reloading chart after repo update")
				}
			} else {
				return nil, err
			}
		}
	}

	if ch.Metadata.Deprecated {
		h.Logf("This chart is deprecated")
	}
	values := make(map[string]interface{})
	if value != "" {
		err = yaml.Unmarshal([]byte(value), &values)
		if err != nil {
			return nil, err
		}
	}
	return client.RunWithContext(ctx, name, ch, values)
}
