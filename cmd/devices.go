package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tjmiller/mux/internal/audio"
)

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List all audio devices",
	RunE:  runDevices,
}

func init() {
	rootCmd.AddCommand(devicesCmd)
}

func runDevices(cmd *cobra.Command, args []string) error {
	devices, err := audio.ListDevices()
	if err != nil {
		return err
	}

	defaultOutUID, _ := audio.GetDefaultOutputUID()
	defaultInUID, _ := audio.GetDefaultInputUID()

	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	marker := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	var outputDevices, inputDevices []audio.Device
	for _, d := range devices {
		if d.HasOutput {
			outputDevices = append(outputDevices, d)
		}
		if d.HasInput {
			inputDevices = append(inputDevices, d)
		}
	}

	printSection := func(title string, devs []audio.Device, defaultUID string) {
		fmt.Fprintln(os.Stdout, header.Render(title))
		if len(devs) == 0 {
			fmt.Fprintln(os.Stdout, "  (none)")
			return
		}

		nameWidth := 0
		for _, d := range devs {
			if len(d.Name) > nameWidth {
				nameWidth = len(d.Name)
			}
		}

		for _, d := range devs {
			prefix := "  "
			if d.UID == defaultUID {
				prefix = marker.Render("● ")
			}
			transport := dim.Render(fmt.Sprintf("(%s)", d.TransportType))
			padding := strings.Repeat(" ", nameWidth-len(d.Name)+2)
			uid := dim.Render(fmt.Sprintf("UID: %s", d.UID))
			fmt.Fprintf(os.Stdout, "  %s%s%s%-14s %s\n", prefix, d.Name, padding, transport, uid)
		}
	}

	printSection("OUTPUT DEVICES", outputDevices, defaultOutUID)
	fmt.Fprintln(os.Stdout)
	printSection("INPUT DEVICES", inputDevices, defaultInUID)

	return nil
}
