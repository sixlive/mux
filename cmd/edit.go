package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tjmiller/mux/internal/audio"
	"github.com/tjmiller/mux/internal/config"
	"github.com/tjmiller/mux/internal/tui"
)

var editCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit an existing preset",
	Args:  cobra.ExactArgs(1),
	RunE:  runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	preset := cfg.FindPreset(name)
	if preset == nil {
		return fmt.Errorf("preset %q not found", name)
	}

	devices, err := audio.ListDevices()
	if err != nil {
		return fmt.Errorf("failed to list audio devices: %w", err)
	}

	var existingNames []string
	for _, p := range cfg.Presets {
		existingNames = append(existingNames, p.Name)
	}

	model := tui.NewEditModel(devices, *preset, existingNames)
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	result := finalModel.(tui.CreateModel).Result()
	if result == nil {
		return nil
	}

	if err := cfg.UpdatePreset(name, *result); err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("✓ Updated preset '%s': %s\n", result.DisplayName, config.PresetSummary(result))
	return nil
}
