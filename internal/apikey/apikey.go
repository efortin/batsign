package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/efortin/batsign/internal/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// randReader is the default random reader (crypto/rand.Reader)
var randReader io.Reader = rand.Reader

// GenerateAPIKey generates a secure random API key with format sk-<base64>
func GenerateAPIKey() (string, error) {
	return GenerateAPIKeyWithReader(randReader)
}

// GenerateAPIKeyWithReader generates an API key using the provided reader (exported for testing)
func GenerateAPIKeyWithReader(reader io.Reader) (string, error) {
	// Generate 32 random bytes
	b := make([]byte, 32)
	if _, err := reader.Read(b); err != nil {
		return "", fmt.Errorf("error generating random key: %w", err)
	}

	// Encode to base64 URL-safe without padding
	encoded := base64.RawURLEncoding.EncodeToString(b)

	return "sk-" + encoded, nil
}

// HashAPIKey generates a SHA-256 hash of the API key
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("%x", hash)
}

// GenerateHint creates a hint showing first 6 and last 2 characters
func GenerateHint(apiKey string) string {
	if len(apiKey) < 8 {
		return apiKey
	}
	first6 := apiKey[:6]
	last2 := apiKey[len(apiKey)-2:]
	stars := strings.Repeat("*", 13)
	return first6 + stars + last2
}

// SanitizeEmail converts email to a valid Kubernetes resource name
func SanitizeEmail(email string) string {
	// Replace @ with -at- and dots with dashes
	name := strings.ReplaceAll(email, "@", "-at-")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format: %s", email)
	}
	return nil
}

// GenerateYAML generates the Kubernetes YAML for an APIKey resource
func GenerateYAML(spec models.APIKeySpec) (string, error) {
	resourceName := SanitizeEmail(spec.Email)

	apiKey := &models.APIKey{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "auth.kgateway.dev/v1alpha1",
			Kind:       "APIKey",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
		},
		Spec: spec,
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(apiKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal APIKey to YAML: %w", err)
	}

	// Add document separator
	return "---\n" + string(yamlBytes), nil
}
