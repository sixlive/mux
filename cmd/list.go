package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tjmiller/mux/internal/config"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved presets",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.Presets) == 0 {
		fmt.Fprintln(os.Stdout, "No presets configured. Run 'mux create' to create one.")
		return nil
	}

	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	fmt.Fprintln(os.Stdout, header.Render("PRESETS"))

	nameWidth := 0
	for _, p := range cfg.Presets {
		if len(p.Name) > nameWidth {
			nameWidth = len(p.Name)
		}
	}

	for _, p := range cfg.Presets {
		namePad := strings.Repeat(" ", nameWidth-len(p.Name)+2)
		summary := strings.ReplaceAll(config.PresetSummary(&p), "\n", "\n    ")
		fmt.Fprintf(os.Stdout, "  %s%s%s\n    %s\n",
			p.Name, namePad,
			dim.Render(p.DisplayName),
			summary,
		)
	}

	return nil
}
