package utils

import (
	"encoding/json"
	"os"

	"gopkg.in/yaml.v3"
)

func YamlToJson(yamlDatas ...string) (string, error) {
	data := make(map[string]interface{})
	for _, v := range yamlDatas {
		var md map[string]interface{}
		err := yaml.Unmarshal([]byte(v), &md)
		if err != nil {
			return "", err
		}
		for k, v := range md {
			data[k] = v
		}
	}
	jsonStr, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(jsonStr), nil
}

// 判断文件是否存在
func CheckFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

// 创建一个可读可写的文件
func CreateFile(filename string) error {
	var file, err = os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
}

// 创建一个嵌套的目录
func CreateDir(path string) error {
	var err = os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}
	return nil
}
