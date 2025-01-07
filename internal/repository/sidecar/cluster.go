package sidecar

import (
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

const (
	ServiceNameSidecar = "sidecar"
)

type SidecarCluster struct {
	conf *conf.Bootstrap
	log  *log.Helper
}

func NewSidecarCluster(conf *conf.Bootstrap, logger log.Logger) *SidecarCluster {
	return &SidecarCluster{
		conf: conf,
		log:  log.NewHelper(logger),
	}
}
