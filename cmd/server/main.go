package main

import (
	"fmt"
	"os"

	"github.com/efortin/batsign/internal/server"
	"github.com/spf13/cobra"
)

var (
	grpcPort   int
	httpPort   int
	namespace  string
	kubeconfig string
	logLevel   string
)

var rootCmd = &cobra.Command{
	Use:   "apikey-manager-server",
	Short: "Authorization server for kgateway (replaces OPA)",
	Long: `Run an Envoy ext_authz compatible gRPC server that validates API keys.

The server watches APIKey CRDs in Kubernetes and validates incoming requests
by comparing SHA-256 hashes of provided API keys against stored hashes.

This server replaces OPA for API key-based authentication.`,
	RunE: run,
}

func init() {
	rootCmd.Flags().IntVarP(&grpcPort, "grpc-port", "g", 9191, "gRPC port for Envoy ext_authz")
	rootCmd.Flags().IntVarP(&httpPort, "http-port", "p", 8080, "HTTP port for health checks")
	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace to watch (empty = all namespaces)")
	rootCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (empty = in-cluster config)")
	rootCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	config := &server.Config{
		GRPCPort:   grpcPort,
		HTTPPort:   httpPort,
		Namespace:  namespace,
		Kubeconfig: kubeconfig,
		LogLevel:   logLevel,
	}

	srv, err := server.New(config)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	return srv.Run()
}
