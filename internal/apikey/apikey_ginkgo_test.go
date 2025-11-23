package apikey_test

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/efortin/batsign/internal/apikey"
	"github.com/efortin/batsign/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("APIKey", func() {
	Describe("GenerateAPIKey", func() {
		Context("when generating API keys", func() {
			It("should start with sk- prefix", func() {
				key, err := apikey.GenerateAPIKey()
				Expect(err).ToNot(HaveOccurred())
				Expect(key).To(HavePrefix("sk-"))
			})

			It("should have correct length", func() {
				key, err := apikey.GenerateAPIKey()
				Expect(err).ToNot(HaveOccurred())
				Expect(key).To(HaveLen(46)) // "sk-" (3) + base64 URL encoded 32 bytes (43)
			})

			It("should contain valid base64", func() {
				key, err := apikey.GenerateAPIKey()
				Expect(err).ToNot(HaveOccurred())

				keyPart := key[3:]
				_, err = base64.RawURLEncoding.DecodeString(keyPart)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should generate unique keys", func() {
				keys := make(map[string]bool)
				for i := 0; i < 100; i++ {
					key, err := apikey.GenerateAPIKey()
					Expect(err).ToNot(HaveOccurred())
					Expect(keys[key]).To(BeFalse(), "Generated duplicate key: %s", key)
					keys[key] = true
				}
			})
		})

		Context("when random reader fails", func() {
			It("should return an error", func() {
				_, err := apikey.GenerateAPIKeyWithReader(&failingReader{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error generating random key"))
			})
		})
	})

	Describe("HashAPIKey", func() {
		Context("with known input", func() {
			It("should generate correct SHA-256 hash", func() {
				apiKey := "sk-test123"
				expected := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))

				result := apikey.HashAPIKey(apiKey)
				Expect(result).To(Equal(expected))
			})

			It("should have length of 64 characters", func() {
				hash := apikey.HashAPIKey("sk-test123")
				Expect(hash).To(HaveLen(64))
			})
		})

		Context("with empty input", func() {
			It("should still generate valid hash", func() {
				expected := fmt.Sprintf("%x", sha256.Sum256([]byte("")))

				result := apikey.HashAPIKey("")
				Expect(result).To(Equal(expected))
			})
		})
	})

	Describe("GenerateHint", func() {
		Context("with normal API key", func() {
			It("should show first 6 and last 2 chars", func() {
				apiKey := "sk-abcdefghijklmnopqrstuvwxyz12345678"
				hint := apikey.GenerateHint(apiKey)

				Expect(hint).To(Equal("sk-abc*************78"))
			})
		})

		Context("with short key (less than 8 chars)", func() {
			It("should return the key as-is", func() {
				apiKey := "sk-abc"
				hint := apikey.GenerateHint(apiKey)

				Expect(hint).To(Equal("sk-abc"))
			})
		})

		Context("with exactly 8 chars", func() {
			It("should still generate hint", func() {
				apiKey := "sk-12345"
				hint := apikey.GenerateHint(apiKey)

				Expect(hint).To(Equal("sk-123*************45"))
			})
		})
	})

	Describe("SanitizeEmail", func() {
		DescribeTable("sanitizing email addresses",
			func(email, expected string) {
				result := apikey.SanitizeEmail(email)
				Expect(result).To(Equal(expected))
			},
			Entry("simple email", "user@example.com", "user-at-example-com"),
			Entry("email with dots", "first.last@company.co.uk", "first-last-at-company-co-uk"),
			Entry("email with plus", "user+tag@domain.com", "user+tag-at-domain-com"),
		)
	})

	Describe("ValidateEmail", func() {
		Context("with valid emails", func() {
			DescribeTable("should not return error",
				func(email string) {
					err := apikey.ValidateEmail(email)
					Expect(err).ToNot(HaveOccurred())
				},
				Entry("standard email", "user@example.com"),
				Entry("with subdomain", "user@mail.example.com"),
				Entry("with plus sign", "user+tag@example.com"),
			)
		})

		Context("with invalid emails", func() {
			DescribeTable("should return error",
				func(email string) {
					err := apikey.ValidateEmail(email)
					Expect(err).To(HaveOccurred())
				},
				Entry("missing @", "userexample.com"),
				Entry("missing domain", "user@"),
				Entry("missing user", "@example.com"),
				Entry("missing TLD", "user@example"),
				Entry("with spaces", "user name@example.com"),
			)
		})
	})

	Describe("GenerateYAML", func() {
		Context("with complete spec", func() {
			It("should generate valid YAML", func() {
				spec := models.APIKeySpec{
					Email:       "user@example.com",
					KeyHash:     "abc123",
					KeyHint:     "sk-abc*************de",
					Description: "Test key",
					Enabled:     true,
				}

				yaml, err := apikey.GenerateYAML(spec)
				Expect(err).ToNot(HaveOccurred())

				Expect(yaml).To(ContainSubstring("apiVersion: auth.kgateway.dev/v1alpha1"))
				Expect(yaml).To(ContainSubstring("kind: APIKey"))
				Expect(yaml).To(ContainSubstring("name: user-at-example-com"))
				Expect(yaml).To(ContainSubstring("email: user@example.com"))
				Expect(yaml).To(ContainSubstring("keyHash: abc123"))
				Expect(yaml).To(ContainSubstring("keyHint: sk-abc*************de"))
				Expect(yaml).To(ContainSubstring("description: Test key"))
				Expect(yaml).To(ContainSubstring("enabled: true"))
			})
		})

		Context("with disabled key", func() {
			It("should show enabled: false", func() {
				spec := models.APIKeySpec{
					Email:       "admin@test.org",
					KeyHash:     "xyz789",
					KeyHint:     "sk-xyz*************89",
					Description: "Admin key",
					Enabled:     false,
				}

				yaml, err := apikey.GenerateYAML(spec)
				Expect(err).ToNot(HaveOccurred())

				Expect(yaml).To(ContainSubstring("enabled: false"))
			})
		})

		Context("email sanitization", func() {
			It("should sanitize in metadata.name but keep original in spec.email", func() {
				spec := models.APIKeySpec{
					Email:       "first.last@company.co.uk",
					KeyHash:     "hash123",
					KeyHint:     "sk-abc*************de",
					Description: "Test",
					Enabled:     true,
				}

				yaml, err := apikey.GenerateYAML(spec)
				Expect(err).ToNot(HaveOccurred())

				Expect(yaml).To(ContainSubstring("name: first-last-at-company-co-uk"))
				Expect(yaml).To(ContainSubstring("email: first.last@company.co.uk"))
			})
		})
	})
})

// failingReader is a reader that always returns an error
type failingReader struct{}

func (f *failingReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated random read failure")
}
