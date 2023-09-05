package data

import (
	"context"
	"ocean/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gopkg.in/yaml.v2"
)

type infraRepo struct {
	data *Data
	log  *log.Helper
}

func NewInfraRepo(data *Data, logger log.Logger) biz.GetInfraRepo {
	return &infraRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *infraRepo) GetInfra(ctx context.Context) (*biz.Infra, error) {
	infra := &biz.Infra{}
	// 读取 YAML 文件
	infraData, err := readFile(getInfraConfigPath())
	if err != nil {
		return nil, err
	}
	// 解析 YAML 文件
	if err := yaml.Unmarshal(infraData, infra); err != nil {
		return nil, err
	}
	return infra, nil
}

func (r *infraRepo) SaveInfra(ctx context.Context, infra *biz.Infra) error {
	// 将修改后的结构体编码为 YAML 格式
	newData, err := yaml.Marshal(infra)
	if err != nil {
		return err
	}
	// 将编码后的 YAML 写入文件
	return writeFile(getInfraConfigPath(), newData)
}
