package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/internal/interfaces"
	"github.com/mark3labs/mcp-go/server"
)

type McpServer struct {
	conf               *conf.Bootstrap
	server             *http.Server
	ClusterInterface   *interfaces.ClusterInterface
	AppInterface       *interfaces.AppInterface
	ServicesInterface  *interfaces.ServicesInterface
	UserInterface      *interfaces.UserInterface
	WorkspaceInterface *interfaces.WorkspaceInterface
	ProjectInterface   *interfaces.ProjectInterface
	mcpServers         map[string]*server.MCPServer
}

func NewMcpServer(ctx context.Context, conf *conf.Bootstrap, cluster *interfaces.ClusterInterface, app *interfaces.AppInterface, services *interfaces.ServicesInterface, user *interfaces.UserInterface, workspace *interfaces.WorkspaceInterface, project *interfaces.ProjectInterface) (*McpServer, error) {
	s := &McpServer{
		conf:               conf,
		ClusterInterface:   cluster,
		AppInterface:       app,
		ServicesInterface:  services,
		UserInterface:      user,
		WorkspaceInterface: workspace,
		ProjectInterface:   project,
		mcpServers:         make(map[string]*server.MCPServer),
	}
	clusterMcpService := interfaces.NewClusterInterfaceMcpService(s.ClusterInterface)
	clusterMcpServer, err := clusterMcpService.ClusterMcp()
	if err != nil {
		return nil, err
	}
	s.RegisterServer(biz.ClusterKey.String(), clusterMcpServer)
	return s, nil
}

func (s *McpServer) RegisterServer(serverName string, mcpServer *server.MCPServer) {
	s.mcpServers[serverName] = mcpServer

}

func (s *McpServer) getMcpServerPath(serverName string) string {
	return fmt.Sprintf("/mcp/%s/", serverName)
}

func (s *McpServer) getMcpSseServerPath(serverName string) string {
	return fmt.Sprintf("/mcp/%s/sse", serverName)
}

func (s *McpServer) getMcpMessageServerPath(serverName string) string {
	return fmt.Sprintf("/mcp/%s/message", serverName)
}

func (s *McpServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	for serverName, mcpServer := range s.mcpServers {
		sseServer := server.NewSSEServer(
			mcpServer,
			server.WithDynamicBasePath(func(r *http.Request, sessionID string) string {
				return s.getMcpServerPath(serverName)
			}),
		)
		mux.Handle(s.getMcpSseServerPath(serverName), sseServer.SSEHandler())
		mux.Handle(s.getMcpMessageServerPath(serverName), sseServer.MessageHandler())
	}
	s.server = &http.Server{
		Addr:    s.conf.Server.Mcp.Addr,
		Handler: mux,
	}
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start MCP server: %v", err)
	}
	return nil
}

func (s *McpServer) Stop(ctx context.Context) error {
	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to stop MCP server: %v", err)
		}
	}
	return nil
}
