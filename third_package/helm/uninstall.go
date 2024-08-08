package helm

import (
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
)

func (h *HelmPkg) NewUninstall() (*action.Uninstall, error) {
	client := action.NewUninstall(h.actionConfig)
	return client, nil
}

func (h *HelmPkg) RunUninstall(client *action.Uninstall, name string) (*release.UninstallReleaseResponse, error) {
	list, err := h.NewList()
	if err != nil {
		return nil, err
	}
	data, err := h.RunList(list)
	if err != nil {
		return nil, err
	}
	var isExist bool = false
	for _, r := range data {
		if r.Name == name {
			isExist = true
		}
	}
	if !isExist {
		return &release.UninstallReleaseResponse{
			Release: &release.Release{
				Info: &release.Info{
					Status: release.StatusUninstalled,
				},
			},
		}, nil
	}
	if client.Timeout == 0 {
		client.Timeout = 300 * time.Second
	}
	return client.Run(name)
}
