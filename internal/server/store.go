package server

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/efortin/batsign/internal/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// APIKeyStore manages the in-memory cache of API key hashes
type APIKeyStore struct {
	mu sync.RWMutex
	// keyHashes maps SHA-256 hash to APIKey metadata
	keyHashes map[string]*models.APIKeyEntry

	client    dynamic.Interface
	namespace string
	stopCh    chan struct{}
}

var apiKeyGVR = schema.GroupVersionResource{
	Group:    "auth.kgateway.dev",
	Version:  "v1alpha1",
	Resource: "apikeys",
}

// NewAPIKeyStore creates a new API key store
func NewAPIKeyStore(kubeconfig, namespace string) (*APIKeyStore, error) {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		// Use in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	} else {
		// Use kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	store := &APIKeyStore{
		keyHashes: make(map[string]*APIKeyEntry),
		client:    client,
		namespace: namespace,
		stopCh:    make(chan struct{}),
	}

	return store, nil
}

// Start begins watching APIKey resources
func (s *APIKeyStore) Start(ctx context.Context) error {
	// Initial list to populate cache
	if err := s.syncAPIKeys(ctx); err != nil {
		return fmt.Errorf("failed initial sync: %w", err)
	}

	// Start watching for changes
	go s.watchAPIKeys(ctx)

	return nil
}

// Stop stops the watcher
func (s *APIKeyStore) Stop() {
	close(s.stopCh)
}

// ValidateKey checks if the provided API key hash is valid and enabled
func (s *APIKeyStore) ValidateKey(keyHash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.keyHashes[keyHash]
	if !exists {
		return false
	}

	return entry.Enabled
}

// syncAPIKeys performs an initial list of all APIKey resources
func (s *APIKeyStore) syncAPIKeys(ctx context.Context) error {
	var list *unstructured.UnstructuredList
	var err error

	if s.namespace == "" {
		// Watch all namespaces
		list, err = s.client.Resource(apiKeyGVR).List(ctx, metav1.ListOptions{})
	} else {
		// Watch specific namespace
		list, err = s.client.Resource(apiKeyGVR).Namespace(s.namespace).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return fmt.Errorf("failed to list APIKeys: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear and repopulate
	s.keyHashes = make(map[string]*APIKeyEntry)

	for _, item := range list.Items {
		if entry := s.parseAPIKey(&item); entry != nil {
			s.keyHashes[entry.KeyHash] = entry
			log.Printf("Loaded APIKey: %s (enabled=%v, hint=%s)", entry.Email, entry.Enabled, entry.KeyHint)
		}
	}

	log.Printf("Synced %d APIKeys", len(s.keyHashes))
	return nil
}

// watchAPIKeys watches for changes to APIKey resources
func (s *APIKeyStore) watchAPIKeys(ctx context.Context) {
	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		var watcher watch.Interface
		var err error

		if s.namespace == "" {
			watcher, err = s.client.Resource(apiKeyGVR).Watch(ctx, metav1.ListOptions{})
		} else {
			watcher, err = s.client.Resource(apiKeyGVR).Namespace(s.namespace).Watch(ctx, metav1.ListOptions{})
		}

		if err != nil {
			log.Printf("Failed to start watch: %v, retrying...", err)
			continue
		}

		for event := range watcher.ResultChan() {
			s.handleWatchEvent(event)
		}

		watcher.Stop()
	}
}

// handleWatchEvent processes watch events
func (s *APIKeyStore) handleWatchEvent(event watch.Event) {
	obj, ok := event.Object.(*unstructured.Unstructured)
	if !ok {
		return
	}

	entry := s.parseAPIKey(obj)
	if entry == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch event.Type {
	case watch.Added, watch.Modified:
		s.keyHashes[entry.KeyHash] = entry
		log.Printf("APIKey %s %s (enabled=%v)", event.Type, entry.Email, entry.Enabled)

	case watch.Deleted:
		delete(s.keyHashes, entry.KeyHash)
		log.Printf("APIKey deleted: %s", entry.Email)
	}
}

// parseAPIKey extracts APIKeyEntry from unstructured object
func (s *APIKeyStore) parseAPIKey(obj *unstructured.Unstructured) *models.APIKeyEntry {
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return nil
	}

	entry := &models.APIKeyEntry{
		Name: obj.GetName(),
	}

	if email, found, _ := unstructured.NestedString(spec, "email"); found {
		entry.Email = email
	}
	if keyHash, found, _ := unstructured.NestedString(spec, "keyHash"); found {
		entry.KeyHash = keyHash
	}
	if keyHint, found, _ := unstructured.NestedString(spec, "keyHint"); found {
		entry.KeyHint = keyHint
	}
	if description, found, _ := unstructured.NestedString(spec, "description"); found {
		entry.Description = description
	}
	if enabled, found, _ := unstructured.NestedBool(spec, "enabled"); found {
		entry.Enabled = enabled
	} else {
		entry.Enabled = true // Default to enabled
	}

	return entry
}

// GetStats returns statistics about the store
func (s *APIKeyStore) GetStats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	enabled := 0
	disabled := 0

	for _, entry := range s.keyHashes {
		if entry.Enabled {
			enabled++
		} else {
			disabled++
		}
	}

	return map[string]int{
		"total":    len(s.keyHashes),
		"enabled":  enabled,
		"disabled": disabled,
	}
}
