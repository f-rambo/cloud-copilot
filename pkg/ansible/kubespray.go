package ansible

import (
	"context"
	"fmt"
	"regexp"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
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
		return nil, fmt.Errorf("kubespray version not found")
	}
	// 重命名kubespray
	err = utils.RenameFile(c.GetClusterPath()+"kubespray-"+version, k.packagePath)
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (k *Kubespray) GetDefaultClusterConfig(ctx context.Context) (string, error) {
	defaultClusterConfig := k.packagePath + "/inventory/sample/group_vars/all/all.yml"
	fileData, err := utils.ReadFile(defaultClusterConfig)
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

func (k *Kubespray) GetDefaultClusterAddons(ctx context.Context) (string, error) {
	defaultClusterAddons := k.packagePath + "/inventory/sample/group_vars/k8s_cluster/addons.yml"
	fileData, err := utils.ReadFile(defaultClusterAddons)
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

func (k *Kubespray) GetDefaultClusterAddonsConfig(ctx context.Context) (string, error) {
	defaultClusterAddonsConfig := k.packagePath + "/inventory/sample/group_vars/k8s_cluster/k8s-cluster.yml"
	fileData, err := utils.ReadFile(defaultClusterAddonsConfig)
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

/*
容器内执行
在原油的docker file上增加一层容器
git checkout v2.24.1
docker pull quay.io/kubespray/kubespray:v2.24.1
docker run --rm -it --mount type=bind,source="$(pwd)"/inventory/sample,dst=/inventory \
  --mount type=bind,source="${HOME}"/.ssh/id_rsa,dst=/root/.ssh/id_rsa \
  quay.io/kubespray/kubespray:v2.24.1 bash
# Inside the container you may now run the kubespray playbooks:
ansible-playbook -i /inventory/inventory.ini --private-key /root/.ssh/id_rsa cluster.yml
*/

func (k *Kubespray) GetResetPath() string {
	return k.packagePath + "/reset.yml"
}

func (k *Kubespray) GetClusterPath() string {
	return k.packagePath + "/cluster.yml"
}

func (k *Kubespray) GetUpgradePath() string {
	return k.packagePath + "/upgrade-cluster.yml"
}

func (k *Kubespray) GetRemoveNodePath() string {
	return k.packagePath + "/remove-node.yml"
}

func (k *Kubespray) GetScalePath() string {
	return k.packagePath + "/scale.yml"
}

func (k *Kubespray) GetPackagePath() string {
	return k.packagePath
}
