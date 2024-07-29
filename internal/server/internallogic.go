package server

import (
	"context"

	"github.com/f-rambo/ocean/internal/interfaces"
)

type OtherServer struct {
	servers []f
}

type f struct {
	start func(context.Context) error
	stop  func(context.Context) error
}

func NewOtherServer(cluster *interfaces.ClusterInterface) *OtherServer {
	s := &OtherServer{}
	s.Register(cluster.StartReconcile, cluster.StopReconcile)
	return s
}

func (s *OtherServer) Start(ctx context.Context) error {
	for _, v := range s.servers {
		if err := v.start(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *OtherServer) Stop(ctx context.Context) error {
	for _, v := range s.servers {
		if err := v.stop(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *OtherServer) Register(start func(context.Context) error, stop func(context.Context) error) {
	s.servers = append(s.servers, f{start: start, stop: stop})
}
