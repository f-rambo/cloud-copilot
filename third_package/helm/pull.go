package helm

import "helm.sh/helm/v3/pkg/action"

func (h *HelmPkg) NewPull() (*action.Pull, error) {
	pullObj := action.NewPullWithOpts(action.WithConfig(h.actionConfig))
	pullObj.SetRegistryClient(h.actionConfig.RegistryClient)
	return pullObj, nil
}

func (h *HelmPkg) RunPull(client *action.Pull, dir, chartRef string) error {
	client.DestDir = dir
	_, err := client.Run(chartRef)
	if err != nil {
		return err
	}
	return nil
}
