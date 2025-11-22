package apikey

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// failingReader is a reader that always returns an error
type failingReader struct{}

func (f *failingReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated random read failure")
}

func TestGenerateAPIKey(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"Generate API key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateAPIKey()
			if err != nil {
				t.Errorf("GenerateAPIKey() error = %v", err)
				return
			}

			// Check that key starts with "sk-"
			if !strings.HasPrefix(key, "sk-") {
				t.Errorf("GenerateAPIKey() key should start with 'sk-', got %s", key)
			}

			// Check that the key has the right length (sk- + 43 chars from base64 of 32 bytes)
			if len(key) != 46 { // "sk-" (3) + base64 URL encoded 32 bytes without padding (43)
				t.Errorf("GenerateAPIKey() key length = %d, want 46", len(key))
			}

			// Check that the base64 part is valid
			keyPart := key[3:]
			_, err = base64.RawURLEncoding.DecodeString(keyPart)
			if err != nil {
				t.Errorf("GenerateAPIKey() key part is not valid base64: %v", err)
			}
		})
	}
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	// Generate multiple keys and ensure they're unique
	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		key, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey() error = %v", err)
		}
		if keys[key] {
			t.Errorf("GenerateAPIKey() generated duplicate key: %s", key)
		}
		keys[key] = true
	}
}

func TestGenerateAPIKey_Error(t *testing.T) {
	// Test error handling when random reader fails
	_, err := GenerateAPIKeyWithReader(&failingReader{})
	if err == nil {
		t.Error("GenerateAPIKeyWithReader() with failing reader should return error")
	}
	if !strings.Contains(err.Error(), "error generating random key") {
		t.Errorf("GenerateAPIKeyWithReader() error should contain 'error generating random key', got: %v", err)
	}
}

func TestHashAPIKey(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   string
	}{
		{
			name:   "Hash known key",
			apiKey: "sk-test123",
			want:   fmt.Sprintf("%x", sha256.Sum256([]byte("sk-test123"))),
		},
		{
			name:   "Hash empty key",
			apiKey: "",
			want:   fmt.Sprintf("%x", sha256.Sum256([]byte(""))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashAPIKey(tt.apiKey)
			if got != tt.want {
				t.Errorf("HashAPIKey() = %v, want %v", got, tt.want)
			}

			// Verify it's a valid hex string of length 64
			if len(got) != 64 {
				t.Errorf("HashAPIKey() length = %d, want 64", len(got))
			}
		})
	}
}

func TestGenerateHint(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   string
	}{
		{
			name:   "Normal API key",
			apiKey: "sk-abcdefghijklmnopqrstuvwxyz12345678",
			want:   "sk-abc*************78",
		},
		{
			name:   "Short key (less than 8 chars)",
			apiKey: "sk-abc",
			want:   "sk-abc",
		},
		{
			name:   "Exactly 8 chars",
			apiKey: "sk-12345",
			want:   "sk-123*************45",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateHint(tt.apiKey)
			if got != tt.want {
				t.Errorf("GenerateHint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "Simple email",
			email: "user@example.com",
			want:  "user-at-example-com",
		},
		{
			name:  "Email with dots",
			email: "first.last@company.co.uk",
			want:  "first-last-at-company-co-uk",
		},
		{
			name:  "Email with plus",
			email: "user+tag@domain.com",
			want:  "user+tag-at-domain-com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeEmail(tt.email)
			if got != tt.want {
				t.Errorf("SanitizeEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "Valid email",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "Valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "Valid email with plus",
			email:   "user+tag@example.com",
			wantErr: false,
		},
		{
			name:    "Invalid email - no @",
			email:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "Invalid email - no domain",
			email:   "user@",
			wantErr: true,
		},
		{
			name:    "Invalid email - no user",
			email:   "@example.com",
			wantErr: true,
		},
		{
			name:    "Invalid email - no TLD",
			email:   "user@example",
			wantErr: true,
		},
		{
			name:    "Invalid email - spaces",
			email:   "user name@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateYAML(t *testing.T) {
	tests := []struct {
		name string
		spec APIKeySpec
		want string
	}{
		{
			name: "Complete spec",
			spec: APIKeySpec{
				Email:       "user@example.com",
				KeyHash:     "abc123",
				KeyHint:     "sk-abc*************de",
				Description: "Test key",
				Enabled:     true,
			},
			want: `---
apiVersion: auth.kgateway.dev/v1alpha1
kind: APIKey
metadata:
  name: user-at-example-com
spec:
  description: Test key
  email: user@example.com
  enabled: true
  keyHash: abc123
  keyHint: sk-abc*************de
`,
		},
		{
			name: "Disabled key",
			spec: APIKeySpec{
				Email:       "admin@test.org",
				KeyHash:     "xyz789",
				KeyHint:     "sk-xyz*************89",
				Description: "Admin key",
				Enabled:     false,
			},
			want: `---
apiVersion: auth.kgateway.dev/v1alpha1
kind: APIKey
metadata:
  name: admin-at-test-org
spec:
  description: Admin key
  email: admin@test.org
  enabled: false
  keyHash: xyz789
  keyHint: sk-xyz*************89
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateYAML(tt.spec)
			if err != nil {
				t.Errorf("GenerateYAML() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GenerateYAML() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateYAML_EmailSanitization(t *testing.T) {
	spec := APIKeySpec{
		Email:       "first.last@company.co.uk",
		KeyHash:     "hash123",
		KeyHint:     "sk-abc*************de",
		Description: "Test",
		Enabled:     true,
	}

	got, err := GenerateYAML(spec)
	if err != nil {
		t.Errorf("GenerateYAML() error = %v", err)
		return
	}

	// Check that the metadata.name is properly sanitized
	if !strings.Contains(got, "name: first-last-at-company-co-uk") {
		t.Errorf("GenerateYAML() should contain sanitized name, got %s", got)
	}

	// Check that the spec.email is unchanged
	if !strings.Contains(got, "email: first.last@company.co.uk") {
		t.Errorf("GenerateYAML() should contain original email, got %s", got)
	}
}
