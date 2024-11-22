package sidecar

import (
	"github.com/expr-lang/expr/conf"
	"github.com/go-kratos/kratos/v2/log"
)

const (
	ServiceNameSidecar = "sidecar"
)

type SidecarCluster struct {
	conf *conf.Config
	log  *log.Helper
}

func NewSidecarCluster(conf *conf.Config, logger log.Logger) *SidecarCluster {
	return &SidecarCluster{
		conf: conf,
		log:  log.NewHelper(logger),
	}
}
