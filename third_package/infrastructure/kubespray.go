package infrastructure

import (
	"path/filepath"

	"github.com/f-rambo/ocean/utils"
)

const (
	kubesprayPackageName = "kubespray"
	kubesprayUrl         = ""
)

type Kubespray struct {
	packagePath string
}

func NewKubespray() (k *Kubespray, err error) {
	k = &Kubespray{}
	err = k.autoInstallKubespray()
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (k *Kubespray) autoInstallKubespray() (err error) {
	// 检查文件是否存储
	k.packagePath, err = utils.GetPackageStorePathByNames(kubesprayPackageName)
	if err != nil {
		return err
	}
	if utils.IsFileExist(k.packagePath) {
		return nil
	}
	// 下载kubespray
	err = utils.DownloadFile(kubesprayUrl, k.packagePath)
	if err != nil {
		return err
	}
	// 解压kubespray
	err = utils.Decompress(filepath.Join(k.packagePath, utils.GetFileNameByUrl(kubesprayUrl)), k.packagePath)
	if err != nil {
		return err
	}
	return nil
}

func (k *Kubespray) ReadClusterConfig() (string, error) {
	defaultClusterConfig := k.packagePath + "/inventory/sample/group_vars/all/all.yml"
	fileData, err := utils.ReadFile(defaultClusterConfig)
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

func (k *Kubespray) ReadClusterAddons() (string, error) {
	defaultClusterAddons := k.packagePath + "/inventory/sample/group_vars/k8s_cluster/addons.yml"
	fileData, err := utils.ReadFile(defaultClusterAddons)
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

func (k *Kubespray) ReadClusterAddonsConfig() (string, error) {
	defaultClusterAddonsConfig := k.packagePath + "/inventory/sample/group_vars/k8s_cluster/k8s-cluster.yml"
	fileData, err := utils.ReadFile(defaultClusterAddonsConfig)
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

func (k *Kubespray) GetPackagePath() string {
	return k.packagePath
}

func GetResetPlaybookPath() string {
	return "reset.yml"
}

func GetClusterPlaybookPath() string {
	return "cluster.yml"
}

func GetUpgradePlaybookPath() string {
	return "upgrade-cluster.yml"
}

func GetRemoveNodePlaybookPath() string {
	return "remove-node.yml"
}

func GetScalePlaybookPath() string {
	return "scale.yml"
}
