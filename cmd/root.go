package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tjmiller/mux/internal/config"
	"github.com/tjmiller/mux/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:   "mux",
	Short: "macOS audio preset manager",
	Long:  "Manage audio device presets — switch input/output devices and volumes with a single command.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		model := tui.NewPickerModel(cfg.Presets)
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		selected := finalModel.(tui.PickerModel).Selected()
		if selected == nil {
			return nil
		}

		return applyPreset(selected.Name)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

