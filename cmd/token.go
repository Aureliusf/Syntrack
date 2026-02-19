package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var saveToken bool

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage authentication tokens",
	Long:  `Generate and manage authentication tokens for secure API access.`,
}

var tokenGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new authentication token",
	Long: `Generate a secure random token for authentication.
	
The token will be printed to stdout. Use --save to store it in ~/.syntrack/tokens.
Generated tokens are 32 bytes (64 hex characters) for strong security.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := generateSecureToken()
		if err != nil {
			return fmt.Errorf("generating token: %w", err)
		}

		fmt.Println("Generated token:")
		fmt.Println(token)
		fmt.Println()

		if saveToken {
			if err := saveTokenToFile(token); err != nil {
				return fmt.Errorf("saving token: %w", err)
			}
			fmt.Println("Token saved to ~/.syntrack/tokens")
		}

		fmt.Println("Usage:")
		fmt.Println("  - Set environment variable: export SYNTRACK_AUTH_TOKENS=" + token)
		fmt.Println("  - Or use the saved token file at ~/.syntrack/tokens")
		fmt.Println("  - Include in requests: X-Auth-Token: <token>")
		fmt.Println()
		fmt.Println("Note: Localhost requests (127.0.0.1, ::1) are always allowed without token.")

		return nil
	},
}

func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "syntrack_token_" + hex.EncodeToString(bytes), nil
}

func saveTokenToFile(token string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	syntrackDir := filepath.Join(homeDir, ".syntrack")
	if err := os.MkdirAll(syntrackDir, 0700); err != nil {
		return fmt.Errorf("creating .syntrack directory: %w", err)
	}

	tokenFile := filepath.Join(syntrackDir, "tokens")

	var content string
	if existing, err := os.ReadFile(tokenFile); err == nil && len(existing) > 0 {
		content = string(existing) + "\n" + token + "\n"
	} else {
		content = token + "\n"
	}

	if err := os.WriteFile(tokenFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing token file: %w", err)
	}

	return nil
}

func init() {
	tokenGenerateCmd.Flags().BoolVarP(&saveToken, "save", "s", false, "Save token to ~/.syntrack/tokens")
	tokenCmd.AddCommand(tokenGenerateCmd)
	rootCmd.AddCommand(tokenCmd)
}
