goThoom Plugins
================

This folder contains example plugin scripts for goThoom. Plugins are plain
Go files interpreted at runtime using Yaegi. The client exposes a small API
under the import path "gt".

Getting Started
---------------
- Copy or edit the example_ponder.go file to customize hotkeys.
- Each plugin must define an `Init()` function. The client discovers and calls
  this function after loading the script.

Imports
-------
The interpreter allows only the following packages:

- `gt` (client API)
- `bytes`
- `encoding/json`
- `errors`
- `fmt`
- `math`
- `math/big`
- `math/rand`
- `regexp`
- `sort`
- `strconv`
- `strings`
- `time`
- `unicode/utf8`

Other system packages like `os`, `io`, `net`, etc. are not available to prevent
file or network access.

API
---
- gt.Console(msg)
  Writes a message to the in-client console.
- gt.AddHotkey(combo, command)
  Registers a hotkey combo (e.g., "Digit1", "Shift-A", "Mouse Middle") that
  runs a slash-command like "/ponder hello world".
- gt.AddHotkeyFunc(combo, funcName)
  Binds a hotkey to a named plugin function registered via RegisterFunc.
- gt.Hotkeys()
  Returns the hotkeys registered by the calling plugin.
- gt.RemoveHotkey(combo)
  Removes a previously registered hotkey owned by the plugin.
- gt.RegisterCommand(name, func(args string))
  Handles a local slash command like "/name" and receives the rest as args.
- gt.RegisterFunc(name, func())
  Registers a callable function invokable from hotkeys via AddHotkeyFunc or by
  using a hotkey command string "plugin:name".
- gt.RunCommand(cmd)
  Echoes to the console and queues a command to send immediately to the server.
- gt.EnqueueCommand(cmd)
  Queues a command silently for the next tick.
- gt.PlayerName()
  Returns the name of the currently logged-in character.
- gt.Players()
  Returns a slice of known players with basic info (name, race, etc.).
- gt.RegisterChatHandler(func(msg string))
  Registers a callback invoked for each incoming chat message.
- gt.RegisterPlayerHandler(func(p gt.Player))
  Registers a callback invoked whenever player info changes.
- gt.Inventory()
  Returns a slice of inventory items with ID, name, equipped state and quantity.
- gt.ToggleEquip(id)
  Toggles the equipped state of the first matching item by ID.


Notes
-----
- The plugin directory lives under your data directory (e.g., data/plugins/)
  and is created automatically on first run with the example file.
- You can also place `.go` plugin files in a `plugins/` folder next to the
  gothoom binary; both locations are scanned.
- Hotkeys added by plugins appear in a separate "Plugin Hotkeys" section of
  the hotkeys window and can be enabled or disabled there.
