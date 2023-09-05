package data

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

// 读取文件的函数
func readFile(filename string) ([]byte, error) {
	// 打开文件
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// 读取文件内容
	return os.ReadFile(filename)
}

// 写入文件的函数
func writeFile(filename string, data []byte) error {
	// 检查文件是否存在
	if !isFileExist(filename) {
		err := createFile(filename)
		if err != nil {
			return err
		}
	}
	// 打开文件
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	// 写入文件内容
	_, err = file.Write(data)
	return err
}

// 判断文件是否存在的函数
func isFileExist(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func createFile(filename string) error {
	// 创建文件
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 设置文件权限
	err = os.Chmod(filename, 0666)
	if err != nil {
		return err
	}

	return nil
}

func getCopyKubeConfigPath() string {
	return os.Getenv("INFRA_PATH") + "/copy_auth_config.sh"
}

func getRemoveNodesPath() string {
	return os.Getenv("INFRA_PATH") + "/remove_node_cluster.sh"
}

func getAddNodesPath() string {
	return os.Getenv("INFRA_PATH") + "/add_node_cluster.sh"
}

func getSetServerLognPath() string {
	return os.Getenv("INFRA_PATH") + "/set_server_login.sh"
}

func getDestroyClusterPath() string {
	return os.Getenv("INFRA_PATH") + "/reset_cluster.sh"
}

func getSetUpClusterPath() string {
	return os.Getenv("INFRA_PATH") + "/setup_cluster.sh"
}

func getSetUpKubersparayPath() string {
	return os.Getenv("INFRA_PATH") + "/setup_kubesparay.sh"
}

func getInfraPath() string {
	return os.Getenv("INFRA_PATH")
}

func getConfigPath() string {
	return os.Getenv("CONFIG_PATH")
}

func getInfraConfigPath() string {
	return os.Getenv("CONFIG_PATH") + "/infra.yaml"
}

func getClusterConfigPath() string {
	return os.Getenv("CONFIG_PATH") + "/servers.yaml"
}

func getAPPConfigPath() string {
	return os.Getenv("CONFIG_PATH") + "/apps.yaml"
}

func getAPPValuesPath(appName string) string {
	return fmt.Sprintf("%s/app-%s", os.Getenv("CONFIG_PATH"), appName)
}

func getKubersparayPath(kubesparyVersion string) string {
	// v2.1.2 替换 2.1.2 删除字母v
	newKubesparyVersion := strings.Replace(kubesparyVersion, "v", "", 1)
	return "kubespray-" + newKubesparyVersion
}

func containsValue(arr []string, value string) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}

func getUUID() string {
	// uuidgen
	uuid := uuid.New()
	return uuid.String()
}
