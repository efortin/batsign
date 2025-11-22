package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	envoy_service_auth_v3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server represents the authorization server
type Server struct {
	config     *Config
	store      *APIKeyStore
	grpcServer *grpc.Server
	httpServer *http.Server
	router     *gin.Engine
}

// New creates a new server instance
func New(config *Config) (*Server, error) {
	// Create API key store
	store, err := NewAPIKeyStore(config.Kubeconfig, config.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key store: %w", err)
	}

	return &Server{
		config: config,
		store:  store,
	}, nil
}

// Run starts the server
func (s *Server) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watching APIKeys
	if err := s.store.Start(ctx); err != nil {
		return fmt.Errorf("failed to start API key store: %w", err)
	}

	// Start gRPC server
	errChan := make(chan error, 2)
	go func() {
		if err := s.startGRPCServer(); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// Start HTTP server for health checks
	go func() {
		if err := s.startHTTPServer(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan:
		log.Printf("Received signal %s, shutting down...", sig)
		return s.shutdown()
	}
}

// startGRPCServer starts the gRPC server for Envoy ext_authz
func (s *Server) startGRPCServer() error {
	addr := fmt.Sprintf(":%d", s.config.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Create gRPC server
	s.grpcServer = grpc.NewServer()

	// Register authorization service
	authzServer := NewAuthorizationServer(s.store)
	envoy_service_auth_v3.RegisterAuthorizationServer(s.grpcServer, authzServer)

	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s.grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Register reflection service (useful for debugging)
	reflection.Register(s.grpcServer)

	log.Printf("gRPC server listening on %s", addr)
	return s.grpcServer.Serve(lis)
}

// startHTTPServer starts the HTTP server for health checks
func (s *Server) startHTTPServer() error {
	// Set Gin mode based on log level
	if s.config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	s.router = gin.New()
	s.router.Use(gin.Recovery())

	if s.config.LogLevel == "debug" {
		s.router.Use(gin.Logger())
	}

	// Register routes
	s.router.GET("/health", s.healthHandler)
	s.router.GET("/ready", s.readyHandler)
	s.router.GET("/stats", s.statsHandler)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.HTTPPort),
		Handler: s.router,
	}

	log.Printf("HTTP server listening on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// healthHandler handles health check requests
func (s *Server) healthHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

// readyHandler handles readiness check requests
func (s *Server) readyHandler(c *gin.Context) {
	stats := s.store.GetStats()
	if stats["total"] == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "No APIKeys loaded",
		})
		return
	}
	c.String(http.StatusOK, "Ready")
}

// statsHandler returns statistics about loaded API keys
func (s *Server) statsHandler(c *gin.Context) {
	stats := s.store.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"total":    stats["total"],
		"enabled":  stats["enabled"],
		"disabled": stats["disabled"],
	})
}

// shutdown gracefully shuts down the server
func (s *Server) shutdown() error {
	log.Println("Shutting down servers...")

	// Stop the API key store
	s.store.Stop()

	// Shutdown gRPC server
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	// Shutdown HTTP server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("HTTP server shutdown error: %w", err)
		}
	}

	log.Println("Shutdown complete")
	return nil
}
