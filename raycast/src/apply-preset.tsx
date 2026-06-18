import { useEffect, useState } from "react";
import {
  Action,
  ActionPanel,
  Icon,
  List,
  Toast,
  getPreferenceValues,
  showHUD,
  showToast,
} from "@raycast/api";
import { execFile } from "child_process";
import { promisify } from "util";
import { existsSync, readFileSync } from "fs";
import { homedir } from "os";
import { join } from "path";

const execFileAsync = promisify(execFile);

interface DeviceConfig {
  uid: string;
  name: string;
  volume: number; // -1 means the device controls its own volume
}

interface Preset {
  name: string;
  display_name: string;
  output?: DeviceConfig;
  input?: DeviceConfig;
}

interface Config {
  presets: Preset[];
}

interface Preferences {
  muxPath?: string;
}

const CONFIG_PATH = join(homedir(), ".config", "mux", "config.json");

// Raycast spawns commands with a minimal PATH, so the binary must be resolved
// to an absolute path. Honor the user override first, then probe common spots.
function resolveMuxPath(): string | null {
  const { muxPath } = getPreferenceValues<Preferences>();
  const candidates = [
    muxPath,
    join(homedir(), "go", "bin", "mux"),
    "/opt/homebrew/bin/mux",
    "/usr/local/bin/mux",
  ].filter((p): p is string => Boolean(p));
  return candidates.find((p) => existsSync(p)) ?? null;
}

function deviceSummary(d?: DeviceConfig): string | null {
  if (!d) return null;
  return d.volume >= 0 ? `${d.name} @ ${d.volume}%` : d.name;
}

function loadPresets(): Preset[] {
  const cfg = JSON.parse(readFileSync(CONFIG_PATH, "utf8")) as Config;
  return cfg.presets ?? [];
}

export default function Command() {
  const [presets, setPresets] = useState<Preset[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    try {
      setPresets(loadPresets());
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsLoading(false);
    }
  }, []);

  async function apply(preset: Preset) {
    const bin = resolveMuxPath();
    if (!bin) {
      await showToast({
        style: Toast.Style.Failure,
        title: "mux binary not found",
        message: "Set its absolute path in the extension preferences (⌘,).",
      });
      return;
    }

    try {
      await showToast({ style: Toast.Style.Animated, title: `Applying ${preset.display_name}…` });
      await execFileAsync(bin, ["apply", preset.name]);
      await showHUD(`🎧 ${preset.display_name}`);
    } catch (e) {
      const err = e as { stderr?: string; message?: string };
      await showToast({
        style: Toast.Style.Failure,
        title: `Failed to apply ${preset.display_name}`,
        message: (err.stderr || err.message || String(e)).trim(),
      });
    }
  }

  if (error) {
    return (
      <List>
        <List.EmptyView
          icon={Icon.Warning}
          title="No mux presets found"
          description={`Could not read ${CONFIG_PATH}\n\n${error}`}
        />
      </List>
    );
  }

  return (
    <List isLoading={isLoading} searchBarPlaceholder="Search audio presets…">
      {presets.map((preset) => {
        const out = deviceSummary(preset.output);
        const inp = deviceSummary(preset.input);
        const accessories: List.Item.Accessory[] = [];
        if (out) accessories.push({ icon: Icon.Speaker, text: out, tooltip: "Output" });
        if (inp) accessories.push({ icon: Icon.Microphone, text: inp, tooltip: "Input" });

        return (
          <List.Item
            key={preset.name}
            icon={Icon.Music}
            title={preset.display_name}
            subtitle={preset.name}
            accessories={accessories}
            keywords={[preset.name]}
            actions={
              <ActionPanel>
                <Action title="Apply Preset" icon={Icon.Check} onAction={() => apply(preset)} />
                <Action.CopyToClipboard title="Copy Preset Name" content={preset.name} />
                <Action.ShowInFinder title="Reveal Config File" path={CONFIG_PATH} />
              </ActionPanel>
            }
          />
        );
      })}
    </List>
  );
}
