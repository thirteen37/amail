package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize amail in current directory",
	Long: `Initialize amail in the current directory.

Creates .amail/ directory with:
  - mail.db: SQLite database for messages
  - config.toml: Project configuration

Examples:
  amail init
  amail init --agents pm,dev,qa,research`,
	RunE: runInit,
}

var initAgents string

func init() {
	initCmd.Flags().StringVar(&initAgents, "agents", "", "Comma-separated list of agent roles (e.g., pm,dev,qa)")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	amailDir := filepath.Join(cwd, ".amail")

	// Check if already initialized
	if _, err := os.Stat(amailDir); err == nil {
		return fmt.Errorf("amail already initialized in this directory")
	}

	// Create .amail directory
	if err := os.MkdirAll(amailDir, 0755); err != nil {
		return fmt.Errorf("failed to create .amail directory: %w", err)
	}

	// Parse agent roles
	var roles []string
	if initAgents != "" {
		for _, role := range strings.Split(initAgents, ",") {
			role = strings.TrimSpace(role)
			if role != "" && role != "user" { // user is reserved
				roles = append(roles, role)
			}
		}
	}

	// Create config file
	configPath := config.ConfigPath(cwd)
	configContent := config.GenerateDefaultConfigContent(roles)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	// Create and initialize database
	dbPath := db.DBPath(cwd)
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer database.Close()

	if err := database.Init(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	fmt.Println("✓ Initialized amail in", cwd)
	fmt.Println("  Created .amail/mail.db")
	fmt.Println("  Created .amail/config.toml")

	if len(roles) > 0 {
		fmt.Printf("  Agent roles: %s\n", strings.Join(roles, ", "))
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit .amail/config.toml to customize settings")
	fmt.Println("  2. Set your identity: source <(amail use <role>)")
	fmt.Println("  3. Send a message: amail send <to> \"subject\" \"body\"")

	return nil
}
