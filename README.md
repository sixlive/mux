# mux

macOS audio preset manager — switch input/output devices and volumes with a single command.

## Install

```sh
brew install tjmiller/tap/mux
```

Or download a binary from the [releases page](https://github.com/tjmiller/mux/releases).

### Build from source

Requires Go 1.26+ and macOS (CoreAudio).

```sh
go install github.com/tjmiller/mux@latest
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

## License

MIT
