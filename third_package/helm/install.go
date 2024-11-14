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

func (h *HelmPkg) NewInstall() (*action.Install, error) {
	client := action.NewInstall(h.actionConfig)
	registryClient, err := newRegistryClient(client.CertFile, client.KeyFile, client.CaFile,
		client.InsecureSkipTLSverify, client.PlainHTTP)
	if err != nil {
		return nil, err
	}
	client.SetRegistryClient(registryClient)
	return client, nil
}

func (h *HelmPkg) checkIfInstall(client *action.Install) (bool, error) {
	var isExist bool = false
	list, err := h.NewList()
	if err != nil {
		return isExist, err
	}
	data, err := h.RunList(list)
	if err != nil {
		return isExist, err
	}
	for _, r := range data {
		if r.Name == client.ReleaseName {
			isExist = true
		}
	}
	return isExist, nil
}

func (h *HelmPkg) RunInstall(ctx context.Context, client *action.Install, chart, value string) (*release.Release, error) {
	ok, err := h.checkIfInstall(client)
	if err != nil {
		return nil, err
	}
	if ok {
		upgrade, err := h.NewUpGrade()
		if err != nil {
			return nil, err
		}
		upgrade.Version = client.Version
		upgrade.ChartPathOptions = client.ChartPathOptions
		upgrade.Force = client.Force
		upgrade.DryRun = client.DryRun
		upgrade.DryRunOption = client.DryRunOption
		upgrade.DisableHooks = client.DisableHooks
		upgrade.SkipCRDs = client.SkipCRDs
		upgrade.Timeout = client.Timeout
		upgrade.Wait = client.Wait
		upgrade.WaitForJobs = client.WaitForJobs
		upgrade.Devel = client.Devel
		upgrade.Namespace = client.Namespace
		upgrade.Atomic = client.Atomic
		upgrade.PostRenderer = client.PostRenderer
		upgrade.DisableOpenAPIValidation = client.DisableOpenAPIValidation
		upgrade.SubNotes = client.SubNotes
		upgrade.Description = client.Description
		upgrade.DependencyUpdate = client.DependencyUpdate
		upgrade.Labels = client.Labels
		upgrade.EnableDNS = client.EnableDNS
		return h.RunUpgrade(ctx, upgrade, client.ReleaseName, chart, value)
	}
	if client.Timeout == 0 {
		client.Timeout = 300 * time.Second
	}
	if client.Version == "" && client.Devel {
		client.Version = ">0.0.0-0"
	}
	if client.DryRunOption == "" {
		client.DryRunOption = "none"
	}
	p := getter.All(settings)
	chartPath, err := client.ChartPathOptions.LocateChart(chart, settings)
	if err != nil {
		return nil, err
	}
	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}
	if err := checkIfInstallable(chartRequested); err != nil {
		return nil, err
	}
	if chartRequested.Metadata.Deprecated {
		h.Logf("This chart is deprecated")
	}
	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chartRequested, req); err != nil {
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
					RegistryClient:   client.GetRegistryClient(),
				}
				if err := man.Update(); err != nil {
					return nil, err
				}
				// Reload the chart with the updated Chart.lock file.
				if chartRequested, err = loader.Load(chartPath); err != nil {
					return nil, errors.Wrap(err, "failed reloading chart after repo update")
				}
			} else {
				return nil, err
			}
		}
	}
	// Validate DryRunOption member is one of the allowed values
	if err := validateDryRunOptionFlag(client.DryRunOption); err != nil {
		return nil, err
	}
	values := make(map[string]interface{})
	if value != "" {
		err = yaml.Unmarshal([]byte(value), &values)
		if err != nil {
			return nil, err
		}
	}
	return client.RunWithContext(ctx, chartRequested, values)
}
