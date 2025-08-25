# AGENTS

This repo includes a minimal Go client under `gothoom/`. To build or run the Go program you need Go version 1.24 or later.
Do not increment JSON versions in GT_Players.json or settings.json or characters.json. They will be done manually if needed.
Any functions or variables or types exposed to the plugins need to also be put empty stubs into pluginapi so the linters do not complain for users.

## Installing dependencies

1. Install Go 1.24 or later. On Debian/Ubuntu you can run:
   ```bash
   sudo apt-get update
   sudo apt-get install -y golang-go build-essential libgl1-mesa-dev libglu1-mesa-dev xorg-dev
   ```
   `libgl1-mesa-dev`, `libglu1-mesa-dev`, and `xorg-dev` provide the OpenGL and X11 libraries required by Ebiten.
   On other distributions install the equivalent development packages.
2. Fetch Go module dependencies:
   ```bash
   cd gothoom
   go mod download
   ```

For convenience the `scripts` directory contains small helper scripts:
`scripts/build_gothoom.sh` fetches dependencies, formats the sources and
compiles the client. `scripts/run_gothoom.sh` launches the program.

Both scripts expect to be executed from the repository root.

## Build steps
1. Navigate to the `gothoom` directory if not already there:
   ```bash
   cd gothoom
   ```
2. Compile the program:
   ```bash
   go build
   ```
   This produces the executable `gothoom` in the current directory.
   You can also run `../scripts/build_gothoom.sh` from the repo root which
   runs `go mod download` and `go build ./...` in one step.
3. You can also run the program directly with:
   ```bash
   go run .
   ```
   Alternatively run `../scripts/run_gothoom.sh` from the repo root.

The module path is `gothoom` and the main package is located in this directory.

The `mac_client` directory contains a reference implementation written in C and should *never* be modified. It is only for reference!

## Session notes
The following dependencies were installed when building the Go client
in this session:

```bash
sudo apt-get update
sudo apt-get install -y golang-go build-essential libgl1-mesa-dev libglu1-mesa-dev xorg-dev
```

Example build and run commands used:

```bash
go build ./...
go run .
```

Running the client without a display (i.e. no `$DISPLAY` variable) will exit
with an X11 initialization error.

## Deprecated Ebiten calls to avoid

- `op.ColorM.Scale`
- `op.ColorM.Translate`
- `op.ColorM.Rotate`
- `op.ColorM.ChangeHSV`
- `ebiten.UncappedTPS`
- `ebiten.CurrentFPS`
- `ebiten.CurrentTPS`
- `ebiten.DeviceScaleFactor`
- `ebiten.GamepadAxis`
- `ebiten.GamepadAxisNum`
- `ebiten.GamepadButtonNum`
- `ebiten.InputChars`
- `ebiten.IsScreenFilterEnabled`
- `ebiten.IsScreenTransparent`
- `ebiten.IsWindowResizable`
- `ebiten.MaxTPS`
- `ebiten.ScheduleFrame`
- `ebiten.ScreenSizeInFullscreen`
- `ebiten.SetFPSMode`
- `ebiten.SetInitFocused`
- `ebiten.SetMaxTPS`
- `ebiten.SetScreenFilterEnabled`
- `ebiten.SetScreenTransparent`
- `ebiten.SetWindowResizable`
- `ebiten.GamepadIDs`
- `(*ebiten.Image).Dispose`
- `(*ebiten.Image).ReplacePixels`
- `(*ebiten.Image).Size`
- `(*ebiten.Shader).Dispose`
- `ebiten.TouchIDs`
