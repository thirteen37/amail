package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
)

// ListOutput is the JSON output structure for the list command
type ListOutput struct {
	Roles         []string          `json:"roles"`
	Groups        map[string]GroupJSON `json:"groups,omitempty"`
	BuiltinGroups []string          `json:"builtin_groups"`
}

// GroupJSON is the JSON representation of a group
type GroupJSON struct {
	Members []string `json:"members"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all mailboxes/roles",
	Long: `List all defined mailboxes/roles for the project.

Examples:
  amail list`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	// Find project root
	root, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	// Load config
	cfg, err := config.LoadProject(root)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Build roles list (including reserved "user")
	roles := append(cfg.Agents.Roles, "user")

	// JSON output
	if IsJSONOutput() {
		output := ListOutput{
			Roles:         roles,
			BuiltinGroups: []string{"@all", "@agents", "@others"},
		}
		if len(cfg.Groups) > 0 {
			output.Groups = make(map[string]GroupJSON)
			for name, members := range cfg.Groups {
				output.Groups[name] = GroupJSON{Members: members}
			}
		}
		return PrintJSON(output)
	}

	// Text output
	fmt.Println("Roles:")
	for _, role := range cfg.Agents.Roles {
		fmt.Printf("  %s\n", role)
	}
	fmt.Println("  user (reserved)")

	if len(cfg.Groups) > 0 {
		fmt.Println()
		fmt.Println("Groups:")
		for name, members := range cfg.Groups {
			fmt.Printf("  @%s: %v\n", name, members)
		}
	}

	fmt.Println()
	fmt.Println("Built-in groups:")
	fmt.Println("  @all: all roles + user")
	fmt.Println("  @agents: all roles (excludes user)")
	fmt.Println("  @others: all except sender")

	return nil
}
