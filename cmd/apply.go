package cmd

import (
	"fmt"
	"strings"
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

// resolveDeviceUID finds the current UID for a device config. If the stored UID
// is still valid, it returns it as-is. Otherwise it falls back to matching by
// device name, which handles the case where macOS assigns a new UID after
// a device is disconnected and reconnected.
func resolveDeviceUID(dc *config.DeviceConfig, devices []audio.Device) (string, error) {
	for _, d := range devices {
		if d.UID == dc.UID {
			return dc.UID, nil
		}
	}
	for _, d := range devices {
		if strings.EqualFold(d.Name, dc.Name) {
			return d.UID, nil
		}
	}
	return "", fmt.Errorf("device %q is not connected", dc.Name)
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

	devices, err := audio.ListDevices()
	if err != nil {
		return fmt.Errorf("failed to list audio devices: %w", err)
	}

	configChanged := false
	var switched bool

	if preset.Output != nil {
		uid, err := resolveDeviceUID(preset.Output, devices)
		if err != nil {
			return fmt.Errorf("output: %w", err)
		}
		if uid != preset.Output.UID {
			preset.Output.UID = uid
			configChanged = true
		}

		currentUID, _ := audio.GetDefaultOutputUID()
		if currentUID != uid {
			if err := audio.SetDefaultOutputDevice(uid); err != nil {
				return fmt.Errorf("failed to set output device %q: %w", preset.Output.Name, err)
			}
			switched = true
		}
	}

	if preset.Input != nil {
		uid, err := resolveDeviceUID(preset.Input, devices)
		if err != nil {
			return fmt.Errorf("input: %w", err)
		}
		if uid != preset.Input.UID {
			preset.Input.UID = uid
			configChanged = true
		}

		currentUID, _ := audio.GetDefaultInputUID()
		if currentUID != uid {
			if err := audio.SetDefaultInputDevice(uid); err != nil {
				return fmt.Errorf("failed to set input device %q: %w", preset.Input.Name, err)
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

	if configChanged {
		config.Save(cfg)
	}

	fmt.Printf("✓ Applied '%s': %s\n", preset.DisplayName, config.PresetSummary(preset))
	return nil
}
