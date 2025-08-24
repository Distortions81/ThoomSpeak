# goThoom

An open-source (MIT) client for the classic **Clan Lord** MMORPG

> Status: actively developed, cross-platform builds provided in Releases.

---

## Why this exists

- The original client is old and finicky.
- Single binary, fast rendering (OpenGL), and fewer weird dependencies.
- Higher framerates (linear interpolation), de-dithering of old graphics, multi-platform.

---

## Download

**Easiest:** grab the latest build from **Releases** on this repo (Windows, macOS, Linux).

If you build from source, you'll need Go **1.24+** and OpenGL/X11 dev libs on Linux. See "Build from source."

---

## Quick start (players)

1. **Download** a release for your OS and unzip it.  
2. **Run** the app:
   - **Windows/macOS:** launch the executable/app bundle.
   - **Linux:** `./gothoom` (you may need `chmod +x gothoom`).
3. On first run, the client **auto-fetches missing game assets** (images, sounds) into `data/`. No manual wrangling.

### Optional extras
- Drop a `background.png` and/or `splash.png` into `data/` for a custom look.

---

## Power-user tricks

You can run the client with flags:

- `-clmov` - play a recorded `.clMov` movie file
- `-pcap`  - replay network frames from a `.pcap/.pcapng` (good for testing UI/parse)  
- `-pgo`   - create `default.pgo` by playing `test.clMov` at 30fps for 30s  
- `-debug` - verbose logging
- `-dumpMusic` - save played music as WAV
- `-imgDump` - dump loaded images as PNG to `dump/img`
- `-sndDump` - dump loaded sounds as WAV to `dump/snd`

Examples:
```bash
# Replay a capture to kick the tires
go run . -pcap reference-client.pcapng
```

---

## Build from source (devs)

### Linux (Debian/Ubuntu)
```bash
sudo apt-get update
sudo apt-get install -y golang-go build-essential libgl1-mesa-dev libglu1-mesa-dev xorg-dev
go build
./gothoom
```
Requirements: Go **1.24+**, OpenGL + X11 development libraries.

### Cross-platform release builds
A helper script builds **Linux + Windows** binaries (and can sign Windows EXEs and macOS `.app` bundles). On Ubuntu it will install missing tools like `zip`/`osslsigncode` automatically. Set cert env vars, then run:
```bash
export WINDOWS_CERT_FILE=certs/fullchain.pem
export WINDOWS_KEY_FILE=certs/privkey.pem   # optional: WINDOWS_KEY_PASS, WINDOWS_CERT_NAME, WINDOWS_TIMESTAMP_URL
export MAC_SIGN_IDENTITY="-"                # ad-hoc by default; set to your certificate name to sign
scripts/build_binaries.sh
```

---

## Troubleshooting

- **Missing assets**: the client will fetch `CL_Images` / `CL_Sounds` archives on first run. If you interrupted it, delete partial files in `data/` and relaunch.
- **Linux can’t start**: ensure OpenGL and X11 dev libs are installed (see commands above).
- **Weird graphics/audio**: try `-debug` to see logs, and file an issue with your OS/GPU/driver info.

---

## Contributing

PRs welcome. Keep changes focused and testable. If you’re adding protocol or UI tweaks, include a small `.pcap` or `.clMov` so others can reproduce quickly. The repo includes tests for text parsing, sound, synthesis, and more—use them.

---

## License

MIT. Game assets and “Clan Lord” are property of their respective owners; this project ships **a client**, not server content.

---

## Credits

Built in Go with a sprinkle of pragmatism and a lot of late-night packet spelunking. If you enjoy this, star the repo or link it.
