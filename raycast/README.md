# mux Audio Presets — Raycast Extension

Switch macOS audio input/output presets from Raycast. The extension reads your
existing mux config at `~/.config/mux/config.json` and runs `mux apply <name>`
when you pick a preset — so the list always reflects whatever presets you've
created with the CLI.

## Prerequisites

- The [`mux`](https://github.com/sixlive/mux) CLI installed (`go install github.com/sixlive/mux@latest`).
- [Node.js](https://nodejs.org/) 18+ and [Raycast](https://raycast.com/).

## Develop / install locally

```sh
cd raycast
npm install
npm run dev      # builds, installs into Raycast, and hot-reloads
```

While `npm run dev` is running, the **Apply Audio Preset** command appears in
Raycast. Stop the dev process and the command stays installed until you remove
it from Raycast's extension list.

## Usage

1. Open Raycast and run **Apply Audio Preset** (or type a preset's display name).
2. Pick a preset and press `↵` to apply it.

## Binary path

Raycast runs commands with a minimal `PATH`, so the extension resolves the `mux`
binary by absolute path. It auto-detects `~/go/bin/mux`, `/opt/homebrew/bin/mux`,
and `/usr/local/bin/mux`. If yours lives elsewhere, set **mux Binary Path** in the
extension preferences (`⌘,` while the command is selected).
