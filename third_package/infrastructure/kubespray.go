package infrastructure

import (
	"regexp"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/pkg/errors"
)

type Kubespray struct {
	packagePath string
}

func NewKubespray(c *conf.Resource) (*Kubespray, error) {
	k := &Kubespray{}
	// 检查文件是否存储
	k.packagePath = c.GetClusterPath() + "kubespray"
	if utils.IsFileExist(k.packagePath) {
		return k, nil
	}
	// 下载kubespray
	fileName := "kubespray.tar.gz"
	err := utils.DownloadFile(c.GetKubesprayUrl(), c.GetClusterPath(), fileName)
	if err != nil {
		return nil, err
	}
	// 解压kubespray
	err = utils.Decompress(c.GetClusterPath()+fileName, c.GetClusterPath())
	if err != nil {
		return nil, err
	}
	version := ""
	re := regexp.MustCompile(`v(\d+\.\d+\.\d+)`)
	match := re.FindStringSubmatch(c.GetKubesprayUrl())
	if len(match) > 1 {
		version = match[1]
	} else {
		return nil, errors.New("kubespray version not found")
	}
	// 重命名kubespray
	err = utils.RenameFile(c.GetClusterPath()+"kubespray-"+version, k.packagePath)
	if err != nil {
		return nil, err
	}
	err = k.generateConfig()
	if err != nil {
		return nil, err
	}
	return k, nil
}

// 生成配置文件
func (k *Kubespray) generateConfig() error {
	k.readClusterConfig()
	k.readClusterAddons()
	k.readClusterAddonsConfig()
	return nil
}

func (k *Kubespray) readClusterConfig() (string, error) {
	defaultClusterConfig := k.packagePath + "/inventory/sample/group_vars/all/all.yml"
	fileData, err := utils.ReadFile(defaultClusterConfig)
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

func (k *Kubespray) readClusterAddons() (string, error) {
	defaultClusterAddons := k.packagePath + "/inventory/sample/group_vars/k8s_cluster/addons.yml"
	fileData, err := utils.ReadFile(defaultClusterAddons)
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

func (k *Kubespray) readClusterAddonsConfig() (string, error) {
	defaultClusterAddonsConfig := k.packagePath + "/inventory/sample/group_vars/k8s_cluster/k8s-cluster.yml"
	fileData, err := utils.ReadFile(defaultClusterAddonsConfig)
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

func (k *Kubespray) GetPackagePath() string {
	// cmd dir
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
