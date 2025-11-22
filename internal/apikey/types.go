package apikey

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// APIKey represents an API key custom resource
// +kubebuilder:object:root=true
type APIKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec APIKeySpecKube `json:"spec"`
}

// APIKeySpecKube defines the desired state of APIKey
type APIKeySpecKube struct {
	Email       string `json:"email"`
	KeyHash     string `json:"keyHash"`
	KeyHint     string `json:"keyHint"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}
