package helm

import (
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
)

func (h *HelmPkg) NewList() (*action.List, error) {
	client := action.NewList(h.actionConfig)
	if client.AllNamespaces {
		if err := h.actionConfig.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), h.Logf); err != nil {
			return nil, err
		}
	}
	client.SetStateMask()
	return client, nil
}

func (h *HelmPkg) RunList(client *action.List) ([]*release.Release, error) {
	results, err := client.Run()
	if err != nil {
		return nil, err
	}
	return results, nil
}
