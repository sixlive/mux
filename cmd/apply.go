package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tjmiller/mux/internal/audio"
	"github.com/tjmiller/mux/internal/config"
)

var applyCmd = &cobra.Command{
	Use:   "apply <name>",
	Short: "Apply a preset",
	Args:  cobra.ExactArgs(1),
	RunE:  runApply,
}

func init() {
	rootCmd.AddCommand(applyCmd)
}

func runApply(cmd *cobra.Command, args []string) error {
	return applyPreset(args[0])
}

func applyPreset(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	preset := cfg.FindPreset(name)
	if preset == nil {
		return fmt.Errorf("preset %q not found", name)
	}

	var switched bool

	if preset.Output != nil {
		currentUID, _ := audio.GetDefaultOutputUID()
		if currentUID != preset.Output.UID {
			if err := audio.SetDefaultOutputDevice(preset.Output.UID); err != nil {
				return fmt.Errorf("output device %q (UID: %s) is not connected or failed to set: %w",
					preset.Output.Name, preset.Output.UID, err)
			}
			switched = true
		}
	}

	if preset.Input != nil {
		currentUID, _ := audio.GetDefaultInputUID()
		if currentUID != preset.Input.UID {
			if err := audio.SetDefaultInputDevice(preset.Input.UID); err != nil {
				return fmt.Errorf("input device %q (UID: %s) is not connected or failed to set: %w",
					preset.Input.Name, preset.Input.UID, err)
			}
			switched = true
		}
	}

	if switched {
		time.Sleep(500 * time.Millisecond)
	}

	if preset.Output != nil && preset.Output.Volume >= 0 {
		audio.SetVolume(preset.Output.UID, audio.ScopeOutput, preset.Output.Volume)
	}

	if preset.Input != nil && preset.Input.Volume >= 0 {
		audio.SetVolume(preset.Input.UID, audio.ScopeInput, preset.Input.Volume)
	}

	fmt.Printf("✓ Applied '%s': %s\n", preset.DisplayName, config.PresetSummary(preset))
	return nil
}
