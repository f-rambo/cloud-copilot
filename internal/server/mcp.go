package server

import (
	"context"
	"net/http"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/sync/errgroup"
)

type McpServer struct {
	srv        *http.Server
	mcpServers []*server.MCPServer
	sseServers []*server.SSEServer
}

func NewMcpServer(conf *conf.Bootstrap) *McpServer {
	return &McpServer{
		srv: &http.Server{
			Addr: conf.Server.Mcp.Addr,
		},
		mcpServers: make([]*server.MCPServer, 0),
		sseServers: make([]*server.SSEServer, 0),
	}
}

// register mcp server
func (s *McpServer) RegisterMcpServer(mcpServer *server.MCPServer) {
	s.mcpServers = append(s.mcpServers, mcpServer)
}

// register sse server
func (s *McpServer) RegisterSseServer(sseServer *server.SSEServer) {
	s.sseServers = append(s.sseServers, sseServer)
}

func (s *McpServer) Start(ctx context.Context) error {
	if len(s.mcpServers) == 0 {
		return nil
	}
	eg, _ := errgroup.WithContext(ctx)
	for _, sseServer := range s.sseServers {
		eg.Go(func() error {
			err := sseServer.Start(s.srv.Addr)
			if err != nil {
				return err
			}
			return nil
		})
	}
	return eg.Wait()
}

func (s *McpServer) Stop(ctx context.Context) error {
	if len(s.mcpServers) == 0 {
		return nil
	}
	eg, ctx := errgroup.WithContext(ctx)
	for _, sseServer := range s.sseServers {
		eg.Go(func() error {
			err := sseServer.Shutdown(ctx)
			if err != nil {
				return err
			}
			return nil
		})
	}
	return eg.Wait()
}
