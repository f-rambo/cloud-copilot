package helm

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
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

/*
	f.BoolVar(&client.CreateNamespace, "create-namespace", false, "create the release namespace if not present")
	// --dry-run options with expected outcome:
	// - Not set means no dry run and server is contacted.
	// - Set with no value, a value of client, or a value of true and the server is not contacted
	// - Set with a value of false, none, or false and the server is contacted
	// The true/false part is meant to reflect some legacy behavior while none is equal to "".
	f.StringVar(&client.DryRunOption, "dry-run", "", "simulate an install. If --dry-run is set with no option being specified or as '--dry-run=client', it will not attempt cluster connections. Setting '--dry-run=server' allows attempting cluster connections.")
	f.Lookup("dry-run").NoOptDefVal = "client"
	f.BoolVar(&client.Force, "force", false, "force resource updates through a replacement strategy")
	f.BoolVar(&client.DisableHooks, "no-hooks", false, "prevent hooks from running during install")
	f.BoolVar(&client.Replace, "replace", false, "re-use the given name, only if that name is a deleted release which remains in the history. This is unsafe in production")
	f.DurationVar(&client.Timeout, "timeout", 300*time.Second, "time to wait for any individual Kubernetes operation (like Jobs for hooks)")
	f.BoolVar(&client.Wait, "wait", false, "if set, will wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment, StatefulSet, or ReplicaSet are in a ready state before marking the release as successful. It will wait for as long as --timeout")
	f.BoolVar(&client.WaitForJobs, "wait-for-jobs", false, "if set and --wait enabled, will wait until all Jobs have been completed before marking the release as successful. It will wait for as long as --timeout")
	f.BoolVarP(&client.GenerateName, "generate-name", "g", false, "generate the name (and omit the NAME parameter)")
	f.StringVar(&client.NameTemplate, "name-template", "", "specify template used to name the release")
	f.StringVar(&client.Description, "description", "", "add a custom description")
	f.BoolVar(&client.Devel, "devel", false, "use development versions, too. Equivalent to version '>0.0.0-0'. If --version is set, this is ignored")
	f.BoolVar(&client.DependencyUpdate, "dependency-update", false, "update dependencies if they are missing before installing the chart")
	f.BoolVar(&client.DisableOpenAPIValidation, "disable-openapi-validation", false, "if set, the installation process will not validate rendered templates against the Kubernetes OpenAPI Schema")
	f.BoolVar(&client.Atomic, "atomic", false, "if set, the installation process deletes the installation on failure. The --wait flag will be set automatically if --atomic is used")
	f.BoolVar(&client.SkipCRDs, "skip-crds", false, "if set, no CRDs will be installed. By default, CRDs are installed if not already present")
	f.BoolVar(&client.SubNotes, "render-subchart-notes", false, "if set, render subchart notes along with the parent")
	f.StringToStringVarP(&client.Labels, "labels", "l", nil, "Labels that would be added to release metadata. Should be divided by comma.")
	f.BoolVar(&client.EnableDNS, "enable-dns", false, "enable DNS lookups when rendering templates")
*/

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
