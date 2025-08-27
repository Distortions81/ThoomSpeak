package main

import "sync"

// inventoryShortcuts maps item indices to assigned shortcut key runes.
var (
	inventoryShortcuts  = map[int]rune{}
	inventoryShortcutMu sync.RWMutex
)

// getInventoryShortcut returns the shortcut key for a given inventory index.
func getInventoryShortcut(idx int) (rune, bool) {
	inventoryShortcutMu.RLock()
	r, ok := inventoryShortcuts[idx]
	inventoryShortcutMu.RUnlock()
	return r, ok
}
