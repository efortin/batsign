package models

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// APIKey represents an API key custom resource
// +kubebuilder:object:root=true
type APIKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec APIKeySpec `json:"spec"`
}

// APIKeySpec defines the desired state of APIKey
type APIKeySpec struct {
	Email       string `json:"email"`
	KeyHash     string `json:"keyHash"`
	KeyHint     string `json:"keyHint"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// APIKeyEntry holds metadata about an API key in memory
type APIKeyEntry struct {
	Name        string
	Email       string
	KeyHash     string
	KeyHint     string
	Description string
	Enabled     bool
}
