package server

import (
	"context"

	"github.com/f-rambo/ocean/internal/interfaces"
)

type InternalLogic struct {
	servers []f
}

type f struct {
	start func(context.Context) error
	stop  func(context.Context) error
}

func NewInternalLogic(cluster *interfaces.ClusterInterface) *InternalLogic {
	s := &InternalLogic{}
	s.Register(cluster.StartReconcile, cluster.StopReconcile)
	return s
}

func (s *InternalLogic) Start(ctx context.Context) error {
	for _, v := range s.servers {
		go v.start(ctx)
	}
	return nil
}

func (s *InternalLogic) Stop(ctx context.Context) error {
	for _, v := range s.servers {
		go v.stop(ctx)
	}
	return nil
}

func (s *InternalLogic) Register(start func(context.Context) error, stop func(context.Context) error) {
	s.servers = append(s.servers, f{start: start, stop: stop})
}
