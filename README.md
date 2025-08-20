# goThoom Client

A open source (MIT) client for the Clan Lord MMORPG.

### Requirements

- Go 1.24 or newer
- OpenGL and X11 development libraries

On Debian/Ubuntu:

```bash
sudo apt-get update
sudo apt-get install -y golang-go build-essential libgl1-mesa-dev libglu1-mesa-dev xorg-dev
```

### Build

From the repository root run:

```bash
go build
```

### Run

Launch the client with:

```bash
go build
./gothoom
```

To exercise parsing and the GUI without a server, replay a captured
network trace:

```bash
go run . -pcap reference-client.pcapng
```

To build release binaries for Linux and Windows, use:

```bash
scripts/build_binaries.sh
```

The script optionally self-signs Windows executables and macOS `.app` bundles.
On Ubuntu, it attempts to install missing tools like `zip` and `osslsigncode` automatically.
Provide certificate paths via environment variables before running:

```bash
export WINDOWS_CERT_FILE=certs/fullchain.pem
export WINDOWS_KEY_FILE=certs/privkey.pem   # optional WINDOWS_KEY_PASS, WINDOWS_CERT_NAME, WINDOWS_TIMESTAMP_URL
export MAC_SIGN_IDENTITY="-"                # ad-hoc by default; set to your certificate name to sign
scripts/build_binaries.sh
```

## Command-line Flags

The Go client accepts the following flags:

- `-clmov` – play back a `.clMov` movie file instead of connecting to a server
- `-pcap` – replay network frames from a `.pcap/.pcapng` file
- `-pgo` – create `default.pgo` by playing `test.clMov` at 30 fps for 30 seconds
- `-client-version` – client version number (`kVersionNumber`, default `1445`)
- `-debug` – enable debug logging (default `true`)
- `-dumpMusic` – save played music as a WAV file

## Setup

- Missing `CL_Images` or `CL_Sounds` archives in `data` are fetched automatically
- Custom splash and background images, just place background.png and/or splash.png into the data directory.

