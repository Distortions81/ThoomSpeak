goThoom Plugins
================

This folder contains example plugin scripts for goThoom. Plugins are plain
Go files interpreted at runtime using Yaegi. The client exposes a small API
under the import path "pluginapi".

Getting Started
---------------
- Copy or edit the example_ponder.go file to customize hotkeys.
- Each plugin must define an `Init()` function. The client discovers and calls
  this function after loading the script.

API
---
- pluginapi.AddHotkey(combo, command)
  Registers a hotkey combo (e.g., "Digit1", "Shift-A", "Mouse Middle") that
  runs a slash-command like "/ponder hello world".

Notes
-----
- The plugin directory lives under your data directory (e.g., data/plugins/)
  and is created automatically on first run with the example file.
- You can also place `.go` plugin files in a `plugins/` folder next to the
  gothoom binary; both locations are scanned.

