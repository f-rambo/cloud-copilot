package utils

import (
	"encoding/json"

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
