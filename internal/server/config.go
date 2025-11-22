package server

// Config holds the server configuration
type Config struct {
	// GRPCPort is the port for the gRPC server (Envoy ext_authz)
	GRPCPort int

	// HTTPPort is the port for HTTP health checks
	HTTPPort int

	// Namespace to watch for APIKey resources (empty = all namespaces)
	Namespace string

	// Kubeconfig path (empty = in-cluster config)
	Kubeconfig string

	// LogLevel for the server (debug, info, warn, error)
	LogLevel string
}
