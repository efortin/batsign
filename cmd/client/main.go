package main

import (
	"fmt"
	"os"

	"github.com/efortin/batsign/internal/apikey"
	"github.com/efortin/batsign/internal/models"
	"github.com/spf13/cobra"
)

var (
	email       string
	description string
	enabled     bool
)

var rootCmd = &cobra.Command{
	Use:   "apikey-manager-client",
	Short: "Generate secure API keys for kgateway",
	Long: `Generate cryptographically secure API keys with SHA-256 hashing
for use with the kgateway API authentication system.

The tool generates a random API key, hashes it, creates a visual hint,
and outputs Kubernetes-ready YAML for the APIKey custom resource.`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVarP(&email, "email", "e", "", "Email address of the API key owner (required)")
	rootCmd.Flags().StringVarP(&description, "description", "d", "", "Description of the API key purpose")
	rootCmd.Flags().BoolVar(&enabled, "enabled", true, "Whether the API key is enabled")

	// Mark email as required
	if err := rootCmd.MarkFlagRequired("email"); err != nil {
		// This should never happen unless there's a programming error
		panic(fmt.Sprintf("Failed to mark email flag as required: %v", err))
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Validate email format
	if err := apikey.ValidateEmail(email); err != nil {
		return err
	}

	// Set default description if not provided
	if description == "" {
		description = fmt.Sprintf("API key for %s", email)
	}

	// Generate a random API key
	key, err := apikey.GenerateAPIKey()
	if err != nil {
		return err
	}

	// Generate hash and hint
	keyHash := apikey.HashAPIKey(key)
	keyHint := apikey.GenerateHint(key)

	// Create the spec
	spec := models.APIKeySpec{
		Email:       email,
		KeyHash:     keyHash,
		KeyHint:     keyHint,
		Description: description,
		Enabled:     enabled,
	}

	// Generate and output the YAML
	yaml, err := apikey.GenerateYAML(spec)
	if err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}
	fmt.Print(yaml)

	// Print the actual API key to stderr so user can save it
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "╔════════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(os.Stderr, "║  IMPORTANT: Save this API key - it will not be shown again!   ║")
	fmt.Fprintln(os.Stderr, "╚════════════════════════════════════════════════════════════════╝")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "  API Key: %s\n", key)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "To apply this APIKey resource, run:")
	fmt.Fprintf(os.Stderr, "  kubectl apply -f - <<EOF\n%sEOF\n", yaml)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Or pipe directly:")
	fmt.Fprintf(os.Stderr, "  apikey-manager-client -e %s -d \"%s\" 2>/dev/null | kubectl apply -f -\n", email, description)
	fmt.Fprintln(os.Stderr, "")

	return nil
}
