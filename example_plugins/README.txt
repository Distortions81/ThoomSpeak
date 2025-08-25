goThoom Plugins
================

This folder contains example plugin scripts for goThoom. Plugins are plain Go
files interpreted at runtime using Yaegi. The client exposes a small API under
the import path "gt".

Getting Started
---------------
- Make sure each plugin starts with `//go:build plugin`.
- Copy or edit any of the example `.go` files to get started.
- Each plugin must define an `Init()` function. The client discovers and calls
  this function after loading the script.
- Each plugin must define a unique `PluginName` string. Plugins with duplicate
  names are ignored.
- Place `.go` files in the `plugins/` directory under your data directory
  (e.g., `data/plugins/`) or next to the `gothoom` program. Both locations are
  scanned.
- Hotkeys added by plugins appear in a "Plugin Hotkeys" section of the hotkeys
  window where you can enable or disable them.

Key and Mouse Names
-------------------
Hotkeys and input functions refer to keys and mouse buttons by specific names.
Combine modifiers with `-` like `Ctrl-Shift-A`. Names are case-insensitive.

Modifiers: `Ctrl`, `Alt`, `Shift`

Mouse buttons for hotkeys: `LeftClick`, `RightClick`, `MiddleClick`, `Mouse 3`,
`Mouse 4`, …

Mouse buttons for `MousePressed` and `MouseJustPressed`: `right`, `middle`,
`mouse1`, `mouse2`, `mouse3`, …

Mouse wheel: `WheelUp`, `WheelDown`, `WheelLeft`, `WheelRight`

Key names:

A, Alt, AltLeft, AltRight, ArrowDown, ArrowLeft, ArrowRight, ArrowUp, B,
Backquote, Backslash, Backspace, BracketLeft, BracketRight, C, CapsLock, Comma,
ContextMenu, Control, ControlLeft, ControlRight, D, Delete, Digit0, Digit1,
Digit2, Digit3, Digit4, Digit5, Digit6, Digit7, Digit8, Digit9, E, End, Enter,
Equal, Escape, F, F1, F10, F11, F12, F13, F14, F15, F16, F17, F18, F19, F2,
F20, F21, F22, F23, F24, F3, F4, F5, F6, F7, F8, F9, G, H, Home, I, Insert,
IntlBackslash, J, K, L, M, Meta, MetaLeft, MetaRight, Minus, N, NumLock,
Numpad0, Numpad1, Numpad2, Numpad3, Numpad4, Numpad5, Numpad6, Numpad7,
Numpad8, Numpad9, NumpadAdd, NumpadDecimal, NumpadDivide, NumpadEnter,
NumpadEqual, NumpadMultiply, NumpadSubtract, O, P, PageDown, PageUp, Pause,
Period, PrintScreen, Q, Quote, R, S, ScrollLock, Semicolon, Shift, ShiftLeft,
ShiftRight, Slash, Space, T, Tab, U, V, W, X, Y, Z

API
---
The interpreter allows only these packages: `gt`, `bytes`, `encoding/json`,
`errors`, `fmt`, `math`, `math/big`, `math/rand`, `regexp`, `sort`, `strconv`,
`strings`, `time`, `unicode/utf8`.

Common API calls:
- `gt.Console(msg)` – write a message to the in-game console.
- `gt.Logf(format, args...)` – log a formatted message to console and log.
- `gt.ShowNotification(msg)` – pop up a notification on screen.
- `gt.AddHotkey(combo, command)` – register a hotkey combo that runs a slash
  command.
- `gt.Hotkeys()` – list hotkeys registered by this plugin.
- `gt.RemoveHotkey(combo)` – remove a hotkey this plugin owns.
- `gt.RegisterCommand(name, func(args string))` – define a slash command that
  runs locally.
- `gt.RunCommand(cmd)` – send a command to the server immediately.
- `gt.EnqueueCommand(cmd)` – queue a command to send on the next tick.
- `gt.AddMacro(short, full)` – expand a short prefix into a full command.
- `gt.AddMacros(map[string]string)` – register many macros at once.
- `gt.AutoReply(trigger, cmd)` – run a command when chat starts with trigger.
- `gt.RegisterInputHandler(func(text string) string)` – inspect or change chat
  text before it is sent.
- `gt.RegisterChatHandler(func(msg string))` – react to every chat message.
- `gt.RegisterPlayerHandler(func(p gt.Player))` – react when player info
  changes.
- `gt.PlayerName()` – name of your current character.
- `gt.Players()` – slice of known players with basic info.
- `gt.PlayerStats()` – current HP and SP.
- `gt.Inventory()` – slice of inventory items.
- `gt.ToggleEquip(id)`, `gt.Equip(id)`, `gt.Unequip(id)` – change equipment by
  item ID.
- `gt.EquippedItems()` – list of currently equipped items.
- `gt.HasItem(name)` – whether your inventory has an item by name.
- `gt.MousePressed(name)`, `gt.MouseJustPressed(name)` – check mouse buttons
  using the names above.
- `gt.MouseWheel()` – get scroll wheel movement since last frame.
- `gt.KeyPressed(name)`, `gt.KeyJustPressed(name)` – check keyboard keys.
- `gt.LastClick()` – info about the most recent click.
- `gt.FrameNumber()` – current frame count.
- `gt.SetInputText(txt)` and `gt.InputText()` – set or read the chat input box.
- Simple text helpers: `gt.Lower`, `gt.Upper`, `gt.IgnoreCase`, `gt.StartsWith`,
  `gt.EndsWith`, `gt.Includes`, `gt.Trim`, `gt.TrimStart`, `gt.TrimEnd`,
  `gt.Words`, `gt.Join`.

Plugin Tutorials
----------------
Each example file below is a complete tutorial. Copy the file into your
`plugins/` folder and restart the game to activate it.

### Examples (`example_ponder.go`)
Shows many features at once. Type `/rad` followed by a word to try a feature:
- `/rad notify` shows a popup.
- `/rad stats` prints your HP and SP.
- `/rad players` lists nearby players.
- `/rad gear` lists equipped items.
It also adds hotkeys:
- `Ctrl-D` runs a small dance routine.
- `Ctrl-N` shows a notification.

### Default Macros (`default_macros.go`)
Replaces short text in the chat box with full commands using `gt.AddMacros`.
1. Type `??` followed by text to open `/help`.
2. Try typing `pphello` and it becomes `/ponder hello`.
Edit the `Init` function to add your own shortcuts with `gt.AddMacro`.

### Healer Self-Heal (`healer_selfheal.go`)
Right-click yourself to cast a self-heal.
1. Keep a moonstone in your inventory.
2. Right-click your own character.
3. The plugin equips the moonstone and uses `/use 10` automatically.

### Chain Swap (`chain_swap.go`)
Quickly swap between your chain and the last weapon you used.
1. Have a chain and another weapon in your inventory.
2. Scroll the mouse wheel up or down or type `/swapchain`.
3. The plugin equips the chain. Scroll again to return to the previous weapon.

### Coin Lord (`coin_lord.go`)
Tracks coins you pick up.
1. Type `/cw` to start or stop counting.
2. Use `/cwnew` to reset totals.
3. Use `/cwdata` or press `Shift-C` to see your total and coins per hour.

### Sharecads (`sharecads.go`)
Automatically share when you see healing energy.
1. Type `/shcads` or press `Shift-S` to toggle the plugin.
2. When someone heals you, the plugin runs `/share <name>` once per person.

### Kudzu (`kudzu.go`)
Helps with planting and moving kudzu seeds.
1. `/zu` plants a seed. Hotkey: `Shift-K`.
2. `/zuget` adds a seed to your bag.
3. `/zustore` removes a seed from the bag.
4. `/zutrans name` transfers seeds to someone else.

### Bard Macros (`bard.go`)
Plays tunes without typing long commands.
1. Use `/playsong <instrument> <notes>` to play.
2. Press `Shift-B` to play a sample tune.
The plugin pulls the instrument from your case, plays it, then puts it back.

### Dance Macros (`dance.go`)
Adds a simple dance command using `/pose` positions.
- Type `/dance` or press `Shift-D` to run a short pose routine.

### Dice Roller (`dice_roll.go`) *(optional example)*
Roll virtual dice.
1. Type `/roll NdM` such as `/roll 2d6`.
2. The plugin rolls the dice, totals them, and announces the result.
If you have a dice item, it will try to equip it before rolling.

### Weapon Cycle (`weapon_cycle.go`) *(optional example)*
Cycle through a list of weapons with a single key.
1. Edit the `cycleItems` list in the file to match your weapons.
2. Press `F3` or type `/cycleweapon` to equip the next item in the list.

### Quick Reply (`quick_reply.go`)
Reply to the last exile who thinks to you.
1. Type `/r <message>` to respond with `/thinkto <name> <message>`.

### Auto Yes Boats (`auto_yes_boats.go`)
Automatically whisper "yes" when a boat ferryman offers a ride.
1. When the ferryman says "My fine boats", the plugin replies for you.

Notes
-----
- This directory is created automatically the first time the game runs.
- You can also place `.go` plugin files next to the game binary; both locations
  are scanned.

