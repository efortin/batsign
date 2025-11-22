package server

import (
	"context"
	"log"
	"strings"

	"github.com/efortin/batsign/internal/apikey"
	envoy_api_v3_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_service_auth_v3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	envoy_type_v3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
)

// AuthorizationServer implements the Envoy ext_authz gRPC service
type AuthorizationServer struct {
	store *APIKeyStore
}

// NewAuthorizationServer creates a new authorization server
func NewAuthorizationServer(store *APIKeyStore) *AuthorizationServer {
	return &AuthorizationServer{
		store: store,
	}
}

// Check implements the ext_authz Check method
func (a *AuthorizationServer) Check(ctx context.Context, req *envoy_service_auth_v3.CheckRequest) (*envoy_service_auth_v3.CheckResponse, error) {
	// Extract headers
	headers := req.GetAttributes().GetRequest().GetHttp().GetHeaders()

	// Try to get API key from headers
	apiKey := extractAPIKey(headers)
	if apiKey == "" {
		log.Printf("Denied: No API key provided")
		return denyResponse("Missing API key"), nil
	}

	// Hash the provided API key
	keyHash := apikey.HashAPIKey(apiKey)

	// Validate against store
	if !a.store.ValidateKey(keyHash) {
		hint := apikey.GenerateHint(apiKey)
		log.Printf("Denied: Invalid or disabled API key (hint: %s)", hint)
		return denyResponse("Invalid or disabled API key"), nil
	}

	log.Printf("Allowed: Valid API key (hash: %s...)", keyHash[:12])
	return allowResponse(), nil
}

// extractAPIKey extracts the API key from request headers
// Supports both "Authorization: Bearer <key>" and "x-api-key: <key>"
func extractAPIKey(headers map[string]string) string {
	// Try Authorization header first
	if auth, ok := headers["authorization"]; ok {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}

	// Try x-api-key header
	if key, ok := headers["x-api-key"]; ok {
		return key
	}

	return ""
}

// allowResponse returns a response that allows the request
func allowResponse() *envoy_service_auth_v3.CheckResponse {
	return &envoy_service_auth_v3.CheckResponse{
		Status: &status.Status{
			Code: int32(codes.OK),
		},
		HttpResponse: &envoy_service_auth_v3.CheckResponse_OkResponse{
			OkResponse: &envoy_service_auth_v3.OkHttpResponse{},
		},
	}
}

// denyResponse returns a response that denies the request
func denyResponse(message string) *envoy_service_auth_v3.CheckResponse {
	return &envoy_service_auth_v3.CheckResponse{
		Status: &status.Status{
			Code:    int32(codes.PermissionDenied),
			Message: message,
		},
		HttpResponse: &envoy_service_auth_v3.CheckResponse_DeniedResponse{
			DeniedResponse: &envoy_service_auth_v3.DeniedHttpResponse{
				Status: &envoy_type_v3.HttpStatus{
					Code: envoy_type_v3.StatusCode_Forbidden,
				},
				Body: message,
				Headers: []*envoy_api_v3_core.HeaderValueOption{
					{
						Header: &envoy_api_v3_core.HeaderValue{
							Key:   "content-type",
							Value: "text/plain",
						},
					},
				},
			},
		},
	}
}
