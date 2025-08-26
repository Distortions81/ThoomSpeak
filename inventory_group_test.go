package main

import "testing"

func TestInventorySeparateNames(t *testing.T) {
	resetInventory()
	addInventoryItem(100, -1, "First", false)
	addInventoryItem(100, -1, "Second", false)
	items := getInventory()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestToggleInventoryEquipAt(t *testing.T) {
	resetInventory()
	addInventoryItem(100, 0, "Ring A", false)
	addInventoryItem(100, 1, "Ring B", false)
	toggleInventoryEquipAt(100, 1)
	items := getInventory()
	if !items[1].Equipped {
		t.Fatalf("expected second item equipped")
	}
}
