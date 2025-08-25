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
- pluginapi.AddHotkeyFunc(combo, funcName)
  Binds a hotkey to a named plugin function registered via RegisterFunc.
- pluginapi.RegisterCommand(name, func(args string))
  Handles a local slash command like "/name" and receives the rest as args.
- pluginapi.RegisterFunc(name, func())
  Registers a callable function invokable from hotkeys via AddHotkeyFunc or by
  using a hotkey command string "plugin:name".
- pluginapi.RunCommand(cmd)
  Echoes to the console and queues a command to send immediately to the server.
- pluginapi.EnqueueCommand(cmd)
  Queues a command silently for the next tick.

Notes
-----
- The plugin directory lives under your data directory (e.g., data/plugins/)
  and is created automatically on first run with the example file.
- You can also place `.go` plugin files in a `plugins/` folder next to the
  gothoom binary; both locations are scanned.
