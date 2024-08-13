package server

import (
	"context"

	"github.com/f-rambo/ocean/internal/interfaces"
	"golang.org/x/sync/errgroup"
)

type OtherServer struct {
	servers []f
}

type f struct {
	start func(context.Context) error
	stop  func(context.Context) error
}

func NewInternalLogic(cluster *interfaces.ClusterInterface) *OtherServer {
	s := &OtherServer{}
	s.Register(cluster.StartReconcile, cluster.StopReconcile)
	s.Register(cluster.StartMock, cluster.StopMock)
	return s
}

func (s *OtherServer) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, v := range s.servers {
		g.Go(func() error {
			if err := v.start(ctx); err != nil {
				return err
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func (s *OtherServer) Stop(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, v := range s.servers {
		g.Go(func() error {
			if err := v.stop(ctx); err != nil {
				return err
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func (s *OtherServer) Register(start func(context.Context) error, stop func(context.Context) error) {
	s.servers = append(s.servers, f{start: start, stop: stop})
}
