# mux

macOS audio preset manager, switch input/output devices and volumes with a single command.

## Install

Download a binary from the [releases page](https://github.com/sixlive/mux/releases).

### Build from source

Requires Go 1.26+ and macOS (CoreAudio).

```sh
go install github.com/sixlive/mux@latest
```

## Usage

Run `mux` with no arguments to launch the interactive preset picker with fuzzy search.

### Commands

```
mux              Interactive preset picker
mux create       Create a new preset (guided wizard)
mux apply NAME   Apply a preset by name
mux edit NAME    Edit an existing preset
mux delete NAME  Delete a preset
mux list         List all presets
mux devices      List all audio devices
```

### Example

```sh
# Create a preset for your desk setup
mux create

# Switch to it later
mux apply desk-speakers

# Or pick interactively
mux
```

## Configuration

Presets are stored in `~/.config/mux/config.json`.

Each preset can configure:
- Output device and volume
- Input device and volume

Devices are matched by name, so presets survive USB reconnections that change device UIDs.

## Raycast

A [Raycast](https://raycast.com/) extension lives in [`raycast/`](raycast/) — pick and apply presets with fuzzy search, without opening a terminal. It reads the same `~/.config/mux/config.json`, so presets you create with the CLI show up automatically.

See [`raycast/README.md`](raycast/README.md) for setup.

## License

MIT
