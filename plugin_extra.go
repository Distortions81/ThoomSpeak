package main

import (
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var keyNameMap = func() map[string]ebiten.Key {
	m := make(map[string]ebiten.Key)
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		m[strings.ToLower(k.String())] = k
	}
	return m
}()

func keyFromName(name string) (ebiten.Key, bool) {
	k, ok := keyNameMap[strings.ToLower(name)]
	return k, ok
}

func mouseButtonFromName(name string) (ebiten.MouseButton, bool) {
	n := strings.ToLower(strings.TrimSpace(name))
	switch n {
	case "right", "rightclick":
		return ebiten.MouseButtonRight, true
	case "middle", "middleclick":
		return ebiten.MouseButtonMiddle, true
	}
	if strings.HasPrefix(n, "mouse") {
		numStr := strings.TrimSpace(strings.TrimPrefix(n, "mouse"))
		if numStr == "" {
			return 0, false
		}
		if num, err := strconv.Atoi(numStr); err == nil {
			b := ebiten.MouseButton(num)
			if b > ebiten.MouseButtonLeft && b <= ebiten.MouseButtonMax {
				return b, true
			}
		}
	}
	return 0, false
}

func pluginKeyPressed(name string) bool {
	if k, ok := keyFromName(name); ok {
		return ebiten.IsKeyPressed(k)
	}
	return false
}

func pluginKeyJustPressed(name string) bool {
	if k, ok := keyFromName(name); ok {
		return inpututil.IsKeyJustPressed(k)
	}
	return false
}

func pluginMousePressed(name string) bool {
	if b, ok := mouseButtonFromName(name); ok {
		return ebiten.IsMouseButtonPressed(b)
	}
	return false
}

func pluginMouseJustPressed(name string) bool {
	if b, ok := mouseButtonFromName(name); ok {
		return inpututil.IsMouseButtonJustPressed(b)
	}
	return false
}

func pluginLastClick() ClickInfo {
	lastClickMu.Lock()
	defer lastClickMu.Unlock()
	return lastClick
}

func pluginEquippedItems() []InventoryItem {
	items := getInventory()
	res := make([]InventoryItem, 0, len(items))
	for _, it := range items {
		if it.Equipped {
			res = append(res, it)
		}
	}
	return res
}

func pluginHasItem(name string) bool {
	n := strings.ToLower(name)
	for _, it := range getInventory() {
		if strings.ToLower(it.Name) == n {
			return true
		}
	}
	return false
}

func pluginFrameNumber() int {
	return frameCounter
}
