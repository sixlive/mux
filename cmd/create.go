package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tjmiller/mux/internal/audio"
	"github.com/tjmiller/mux/internal/config"
	"github.com/tjmiller/mux/internal/tui"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new preset",
	RunE:  runCreate,
}

func init() {
	rootCmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	devices, err := audio.ListDevices()
	if err != nil {
		return fmt.Errorf("failed to list audio devices: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var existingNames []string
	for _, p := range cfg.Presets {
		existingNames = append(existingNames, p.Name)
	}

	model := tui.NewCreateModel(devices, existingNames)
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	result := finalModel.(tui.CreateModel).Result()
	if result == nil {
		return nil
	}

	if err := cfg.AddPreset(*result); err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return err
	}

	summary := strings.ReplaceAll(config.PresetSummary(result), "\n", "\n  ")
	fmt.Printf("\nCreated preset '%s'\n  %s\n", result.DisplayName, summary)
	return nil
}
