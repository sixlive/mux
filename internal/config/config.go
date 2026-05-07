package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Config struct {
	Presets []Preset `json:"presets"`
}

type Preset struct {
	Name        string        `json:"name"`
	DisplayName string        `json:"display_name"`
	Output      *DeviceConfig `json:"output,omitempty"`
	Input       *DeviceConfig `json:"input,omitempty"`
}

type DeviceConfig struct {
	UID    string `json:"uid"`
	Name   string `json:"name"`
	Volume int    `json:"volume"` // -1 means device controls its own volume, don't set
}

var namePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("name too long (max 64 characters)")
	}
	if !namePattern.MatchString(name) {
		return fmt.Errorf("name must be kebab-case (lowercase letters, numbers, hyphens)")
	}
	return nil
}

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "mux")
}

func Path() string {
	return filepath.Join(Dir(), "config.json")
}

func Load() (*Config, error) {
	path := Path()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config at %s: %w", path, err)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, "config-*.json")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, Path()); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("saving config: %w", err)
	}
	return nil
}

func (c *Config) FindPreset(name string) *Preset {
	lower := strings.ToLower(name)
	for i := range c.Presets {
		if strings.ToLower(c.Presets[i].Name) == lower {
			return &c.Presets[i]
		}
	}
	return nil
}

func (c *Config) AddPreset(p Preset) error {
	if err := ValidateName(p.Name); err != nil {
		return err
	}
	if c.FindPreset(p.Name) != nil {
		return fmt.Errorf("preset %q already exists (use 'mux edit %s' to modify)", p.Name, p.Name)
	}
	c.Presets = append(c.Presets, p)
	return nil
}

func (c *Config) UpdatePreset(name string, p Preset) error {
	lower := strings.ToLower(name)
	for i := range c.Presets {
		if strings.ToLower(c.Presets[i].Name) == lower {
			c.Presets[i] = p
			return nil
		}
	}
	return fmt.Errorf("preset %q not found", name)
}

func (c *Config) DeletePreset(name string) error {
	lower := strings.ToLower(name)
	for i := range c.Presets {
		if strings.ToLower(c.Presets[i].Name) == lower {
			c.Presets = append(c.Presets[:i], c.Presets[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("preset %q not found", name)
}

func deviceSummary(emoji, name string, volume int) string {
	if volume < 0 {
		return fmt.Sprintf("%s %s", emoji, name)
	}
	return fmt.Sprintf("%s %s @ %d%%", emoji, name, volume)
}

func PresetSummary(p *Preset) string {
	var parts []string
	if p.Output != nil {
		parts = append(parts, deviceSummary("\U0001f50a", p.Output.Name, p.Output.Volume))
	}
	if p.Input != nil {
		parts = append(parts, deviceSummary("\U0001f3a4", p.Input.Name, p.Input.Volume))
	}
	if len(parts) == 0 {
		return "(no devices configured)"
	}
	return strings.Join(parts, "  ·  ")
}
