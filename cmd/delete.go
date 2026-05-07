package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tjmiller/mux/internal/config"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a preset",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	preset := cfg.FindPreset(name)
	if preset == nil {
		return fmt.Errorf("preset %q not found", name)
	}

	fmt.Printf("Delete preset '%s' (%s)? [y/N] ", preset.Name, preset.DisplayName)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" && answer != "yes" {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := cfg.DeletePreset(name); err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Deleted preset '%s'.\n", preset.Name)
	return nil
}
