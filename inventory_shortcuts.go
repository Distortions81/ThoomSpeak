package main

import (
        "sync"
        "unicode"
)

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

// setInventoryShortcut assigns the rune r as the shortcut for the given
// inventory index. Any existing assignment of r to another index is removed.
// Passing r==0 clears the shortcut for idx.
func setInventoryShortcut(idx int, r rune) {
        inventoryShortcutMu.Lock()
        r = unicode.ToLower(r)
        for k, v := range inventoryShortcuts {
                if v == r {
                        delete(inventoryShortcuts, k)
                }
        }
        if r == 0 {
                delete(inventoryShortcuts, idx)
        } else {
                inventoryShortcuts[idx] = r
        }
        inventoryShortcutMu.Unlock()
}

// inventoryIndexForShortcut returns the inventory index assigned to rune r or
// -1 if none is assigned.
func inventoryIndexForShortcut(r rune) int {
        inventoryShortcutMu.RLock()
        defer inventoryShortcutMu.RUnlock()
        r = unicode.ToLower(r)
        for idx, v := range inventoryShortcuts {
                if v == r {
                        return idx
                }
        }
        return -1
}
