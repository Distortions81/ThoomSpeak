//go:build !test

package main

import (
	"bytes"
	"fmt"
	"gothoom/eui"
	"math"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var inventoryWin *eui.WindowData
var inventoryList *eui.ItemData
var inventoryDirty bool

var TitleCaser = cases.Title(language.AmericanEnglish)

var (
	invBoldSrc   *text.GoTextFaceSource
	invItalicSrc *text.GoTextFaceSource
)

// slotNames maps item slot constants to display strings.
var slotNames = []string{
	"invalid", // kItemSlotNotInventory
	"unknown", // kItemSlotNotWearable
	"forehead",
	"neck",
	"shoulder",
	"arms",
	"gloves",
	"finger",
	"coat",
	"cloak",
	"torso",
	"waist",
	"legs",
	"feet",
	"right",
	"left",
	"hands",
	"head",
}

func makeInventoryWindow() {
	if inventoryWin != nil {
		return
	}
	inventoryWin, inventoryList, _ = makeTextWindow("Inventory", eui.HZoneLeft, eui.VZoneMiddleTop, true)
	// Ensure layout updates immediately on resize to avoid gaps.
	inventoryWin.OnResize = func() { updateInventoryWindow() }
	updateInventoryWindow()
}

func updateInventoryWindow() {
	if inventoryWin == nil || inventoryList == nil {
		return
	}

	// Build a unique list of items while counting duplicates and tracking
	// whether any instance of a given key is equipped. Items with explicit
	// per-ID indices are never grouped to preserve distinct properties.
	type invGroupKey struct {
		id   uint16
		name string
		idx  int
	}
	items := getInventory()
	counts := make(map[invGroupKey]int)
	first := make(map[invGroupKey]InventoryItem)
	anyEquipped := make(map[invGroupKey]bool)
	order := make([]invGroupKey, 0, len(items))
	for _, it := range items {
		key := invGroupKey{id: it.ID, idx: it.IDIndex}
		if it.IDIndex < 0 {
			key.idx = -1
			key.name = it.Name
		}
		if _, seen := counts[key]; !seen {
			order = append(order, key)
			first[key] = it
		}
		counts[key]++
		if it.Equipped {
			anyEquipped[key] = true
		}
	}

	// Clear prior contents and rebuild rows as [icon][name (xN)].
	inventoryList.Contents = nil

	// Compute row height from actual font metrics (ascent+descent) at the
	// exact point size used when rendering (+2px fudge for Ebiten).
	fontSize := gs.InventoryFontSize
	if fontSize <= 0 {
		fontSize = gs.ConsoleFontSize
	}
	uiScale := eui.UIScale()
	facePx := float64(float32(fontSize)*uiScale) + 2
	var goFace *text.GoTextFace
	if src := eui.FontSource(); src != nil {
		goFace = &text.GoTextFace{Source: src, Size: facePx}
	} else {
		goFace = &text.GoTextFace{Size: facePx}
	}
	metrics := goFace.Metrics()
	if invBoldSrc == nil {
		invBoldSrc, _ = text.NewGoTextFaceSource(bytes.NewReader(notoSansBold))
	}
	if invItalicSrc == nil {
		invItalicSrc, _ = text.NewGoTextFaceSource(bytes.NewReader(notoSansItalic))
	}
	// Metrics already include the rendering fudge so no extra padding is
	// needed here.
	rowPx := float32(math.Ceil(metrics.HAscent + metrics.HDescent))
	rowUnits := rowPx / uiScale
	iconSize := int(rowUnits + 0.5)

	// Compute available client width/height similar to updateTextWindow so rows
	// don't extend into the window padding and get clipped.
	clientW := inventoryWin.GetSize().X
	clientH := inventoryWin.GetSize().Y - inventoryWin.GetTitleSize()
	s := eui.UIScale()
	if inventoryWin.NoScale {
		s = 1
	}
	pad := (inventoryWin.Padding + inventoryWin.BorderPad) * s
	clientWAvail := clientW - 2*pad
	if clientWAvail < 0 {
		clientWAvail = 0
	}
	clientHAvail := clientH - 2*pad
	if clientHAvail < 0 {
		clientHAvail = 0
	}

	for _, key := range order {
		it := first[key]
		qty := counts[key]
		id := key.id

		// Row container for icon + text
		row := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
		row.Size.X = clientWAvail

		// Icon
		icon, _ := eui.NewImageItem(iconSize, iconSize)
		icon.Filled = false
		icon.Border = 0

		// Choose a pict ID for the item sprite and determine equipped slot.
		var pict uint32
		slot := -1
		if clImages != nil {
			// Inventory list usually uses the worn pict for display.
			if p := clImages.ItemWornPict(uint32(id)); p != 0 {
				pict = p
			}
			slot = clImages.ItemSlot(uint32(id))
		}
		if pict != 0 {
			if img := loadImage(uint16(pict)); img != nil {
				icon.Image = img
				icon.ImageName = fmt.Sprintf("item:%d", id)
			}
		}
		// Add a small right margin after the icon
		icon.Margin = 4
		row.AddItem(icon)

		// Text label with quantity suffix when >1
		label := it.Name
		if label == "" && clImages != nil {
			label = clImages.ItemName(uint32(id))
		}
		if label == "" {
			label = fmt.Sprintf("Item %d", id)
		}
		if qty > 1 {
			label = fmt.Sprintf("(%v) %v", qty, label)
		}

		t, _ := eui.NewText()
		t.Text = TitleCaser.String(label)
		t.FontSize = float32(fontSize)

		face := goFace
		if anyEquipped[key] {
			switch slot {
			case kItemSlotRightHand, kItemSlotLeftHand, kItemSlotBothHands:
				if invBoldSrc != nil {
					face = &text.GoTextFace{Source: invBoldSrc, Size: facePx}
					t.Face = face
				}
			default:
				if invItalicSrc != nil {
					face = &text.GoTextFace{Source: invItalicSrc, Size: facePx}
					t.Face = face
				}
			}
		}

		t.Size.Y = rowUnits

		availName := row.Size.X - float32(iconSize) - icon.Margin
		var lt *eui.ItemData
		if anyEquipped[key] && slot >= 0 && slot < len(slotNames) {
			loc := fmt.Sprintf("[%v]", TitleCaser.String(slotNames[slot]))
			locW, _ := text.Measure(loc, face, 0)
			locWU := float32(math.Ceil(locW / float64(uiScale)))
			if availName > locWU {
				availName -= locWU
				lt, _ = eui.NewText()
				lt.Text = loc
				lt.FontSize = float32(fontSize)
				lt.Face = face
				lt.Size.Y = rowUnits
				lt.Size.X = locWU
				lt.Fixed = true
				lt.Position.X = row.Size.X - locWU
			}
		}

		if availName < 0 {
			availName = 0
		}
		t.Size.X = availName
		row.AddItem(t)
		if lt != nil {
			row.AddItem(lt)
		}

		idCopy := id
		idxCopy := it.IDIndex
		click := func() { toggleInventoryEquipAt(idCopy, idxCopy) }
		icon.Action = click
		t.Action = click
		if lt != nil {
			lt.Action = click
		}

		// Row height matches the icon/text height with minimal padding.
		row.Size.Y = rowUnits

		inventoryList.AddItem(row)
	}

	// Add a trailing spacer equal to one row height so the last item is never
	// clipped at the bottom when fully scrolled.
	spacer, _ := eui.NewText()
	spacer.Text = ""
	spacer.Size = eui.Point{X: 1, Y: rowUnits}
	spacer.FontSize = float32(fontSize)
	inventoryList.AddItem(spacer)

	// Size the list and refresh window similar to updateTextWindow behavior.
	if inventoryWin != nil {
		if inventoryList.Parent != nil {
			inventoryList.Parent.Size.X = clientWAvail
			inventoryList.Parent.Size.Y = clientHAvail
		}
		inventoryList.Size.X = clientWAvail
		inventoryList.Size.Y = clientHAvail
		inventoryWin.Refresh()
	}
}
