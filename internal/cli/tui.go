package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
	"github.com/thirteen37/amail/internal/identity"
	"github.com/thirteen37/amail/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive terminal UI",
	Long: `Launch the interactive terminal UI for managing messages.

The TUI provides:
  - Inbox view with message list
  - Message reading pane
  - Compose new messages
  - Switch between mailboxes (admin mode)

Examples:
  amail tui`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Open project
	database, root, err := db.OpenProject()
	if err != nil {
		return err
	}
	defer database.Close()

	// Load config
	cfg, err := config.LoadProject(root)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Resolve identity (or use first available role)
	var currentIdentity string
	res, err := identity.Resolve(cfg)
	if err == nil && res != nil {
		currentIdentity = res.Identity
	} else {
		// Default to first role or "user"
		if len(cfg.Agents.Roles) > 0 {
			currentIdentity = cfg.Agents.Roles[0]
		} else {
			currentIdentity = "user"
		}
	}

	// Create and run TUI
	model := tui.NewModel(database, cfg, currentIdentity)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
