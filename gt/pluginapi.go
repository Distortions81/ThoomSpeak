// Package gt provides a tiny, editor-only stub for the
// interpreted plugin API exposed by the client at runtime via Yaegi.
//
// This package exists solely to satisfy linters and IDEs for files in
// the plugins/ directory (e.g., healer.go) that do `import "gt"`.
// The real implementations are injected into the Yaegi interpreter at
// runtime; these no-op stubs are never called by the compiled client.
package gt

import "time"

// ClientVersion mirrors the client version value exported to plugins.
var ClientVersion int

// Logf is a no-op printf-style logger for editor/linter happiness.
func Logf(format string, args ...interface{}) {}

// Console writes a message to the in-client console.
func Console(msg string) {}

// AddHotkey binds a key combo to a slash command.
func AddHotkey(combo, command string) {}

// AddHotkeyFunc binds a key combo to a registered plugin function.
func AddHotkeyFunc(combo, funcName string) {}

// HotkeyCommand mirrors the command or function bound to a hotkey.
type HotkeyCommand struct {
	Command  string
	Function string
}

// Hotkey represents a single key binding and its metadata.
type Hotkey struct {
	Name     string
	Combo    string
	Commands []HotkeyCommand
	Plugin   string
	Disabled bool
}

// Hotkeys returns the plugin's registered hotkeys.
func Hotkeys() []Hotkey { return nil }

// RemoveHotkey removes a plugin-owned hotkey by combo.
func RemoveHotkey(combo string) {}

// RegisterCommand handles a local slash command like "/example".
func RegisterCommand(command string, handler func(args string)) {}

// RegisterFunc registers a callable function invokable via AddHotkeyFunc.
func RegisterFunc(command string, handler func()) {}

// RunCommand queues a command to send immediately to the server.
func RunCommand(cmd string) {}

// EnqueueCommand queues a command for the next tick without echoing.
func EnqueueCommand(cmd string) {}

// PlayerName returns the current player's name.
func PlayerName() string { return "" }

// Player mirrors the player's state exposed to plugins.
type Player struct {
	Name       string
	Race       string
	Gender     string
	Class      string
	Clan       string
	PictID     uint16
	Colors     []byte
	IsNPC      bool
	Sharee     bool
	Sharing    bool
	GMLevel    int
	Friend     bool
	Dead       bool
	FellWhere  string
	KillerName string
	Bard       bool
	LastSeen   time.Time
	Offline    bool
}

// Players returns the list of known players.
func Players() []Player { return nil }

// RegisterChatHandler registers a callback for incoming chat messages.
func RegisterChatHandler(fn func(msg string)) {}

// RegisterPlayerHandler registers a callback for player info updates.
func RegisterPlayerHandler(fn func(Player)) {}
// InventoryItem mirrors the client's inventory item structure.
type InventoryItem struct {
	ID       uint16
	Name     string
	Equipped bool
	Index    int
	IDIndex  int
	Quantity int
}

// Inventory returns the player's inventory.
func Inventory() []InventoryItem { return nil }

// ToggleEquip toggles the equipped state of an item by ID.
func ToggleEquip(id uint16) {}

// InputText returns the current text in the input bar.
func InputText() string { return "" }

// SetInputText replaces the text in the input bar.
func SetInputText(text string) {}

// Stats mirrors the player's HP, SP, and balance values.
type Stats struct {
	HP, HPMax           int
	SP, SPMax           int
	Balance, BalanceMax int
}

// PlayerStats returns the player's current stat values.
func PlayerStats() Stats { return Stats{} }
// Equip equips the specified item by ID if it isn't already equipped.
func Equip(id uint16) {}

// Unequip removes the specified item by ID if it is currently equipped.
func Unequip(id uint16) {}
