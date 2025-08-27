//go:build !test

package main

import (
	"bytes"
	"fmt"
	"gothoom/eui"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

var inventoryWin *eui.WindowData
var inventoryList *eui.ItemData
var inventoryDirty bool

type invRef struct {
	id     uint16
	idx    int
	global int
}

var inventoryRowRefs = map[*eui.ItemData]invRef{}
var inventoryCtxWin *eui.WindowData
var invShortcutWin *eui.WindowData
var invShortcutDD *eui.ItemData
var invShortcutTarget int

var selectedInvID uint16
var selectedInvIdx int = -1
var lastInvClickID uint16
var lastInvClickIdx int
var lastInvClickTime time.Time

var TitleCaser = cases.Title(language.AmericanEnglish)
var foldCaser = cases.Fold()

var (
	invBoldSrc   *text.GoTextFaceSource
	invItalicSrc *text.GoTextFaceSource
)

type invGroupKey struct {
	id   uint16
	name string
	idx  int
}

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

	prevScroll := inventoryList.Scroll

	// Build a unique list of items while counting duplicates and tracking
	// whether any instance of a given key is equipped. Non-clothing items are
	// grouped by ID and name so identical items appear once with a quantity,
	// while clothing items are listed individually to allow swapping similar
	// pieces (e.g. different pairs of shoes).
	items := getInventory()
	counts := make(map[invGroupKey]int)
	first := make(map[invGroupKey]InventoryItem)
	anyEquipped := make(map[invGroupKey]bool)
	hasShortcut := make(map[invGroupKey]bool)
	order := make([]invGroupKey, 0, len(items))
	for _, it := range items {
		key := invGroupKey{id: it.ID, name: it.Name}
		if it.IDIndex >= 0 {
			// Template-data items must remain unique by their per-ID index
			key.idx = it.IDIndex
			key.name = ""
		}
		if _, seen := counts[key]; !seen {
			order = append(order, key)
			first[key] = it
		}
		counts[key] += it.Quantity
		if it.Equipped {
			anyEquipped[key] = true
		}
		if r, ok := getInventoryShortcut(it.Index); ok && r != 0 {
			hasShortcut[key] = true
		}
	}

	sort.SliceStable(order, func(i, j int) bool {
		ai := order[i]
		aj := order[j]
		hi := hasShortcut[ai]
		hj := hasShortcut[aj]
		if hi != hj {
			return hi
		}
		nameI := officialName(ai, first[ai])
		nameJ := officialName(aj, first[aj])
		if nameI != nameJ {
			return nameI < nameJ
		}
		return first[ai].Index < first[aj].Index
	})

	// Clear prior contents and rebuild rows as [icon][name (xN)].
	inventoryList.Contents = nil
	inventoryRowRefs = map[*eui.ItemData]invRef{}

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

		// Text label with quantity suffix after the name when >1
		label := it.Name
		if label == "" && clImages != nil {
			label = clImages.ItemName(uint32(id))
		}
		if label == "" {
			label = fmt.Sprintf("Item %d", id)
		}
		if r, ok := getInventoryShortcut(it.Index); ok && r != 0 {
			label = fmt.Sprintf("[%c] %v", unicode.ToUpper(r), label)
		}
		if qty > 1 {
			label = fmt.Sprintf("%v (%v)", label, qty)
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
		if qty > 1 {
			idxCopy = -1
		}
		if idCopy == selectedInvID && idxCopy == selectedInvIdx {
			row.Filled = true
			if inventoryWin != nil && inventoryWin.Theme != nil {
				row.Color = inventoryWin.Theme.Button.SelectedColor
			}
		}
		click := func() { handleInventoryClick(idCopy, idxCopy) }
		icon.Action = click
		t.Action = click
		if lt != nil {
			lt.Action = click
		}

		// Row height matches the icon/text height with minimal padding.
		row.Size.Y = rowUnits

		inventoryList.AddItem(row)
		inventoryRowRefs[row] = invRef{id: idCopy, idx: idxCopy, global: it.Index}
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
		inventoryList.Scroll = prevScroll
		inventoryWin.Refresh()
	}
}

func handleInventoryClick(id uint16, idx int) {
	now := time.Now()
	if id == lastInvClickID && idx == lastInvClickIdx && now.Sub(lastInvClickTime) < 500*time.Millisecond {
		if ebiten.IsKeyPressed(ebiten.KeyShift) || ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
			enqueueCommand(fmt.Sprintf("/useitem %d", id))
			nextCommand()
		} else {
			toggleInventoryEquipAt(id, idx)
		}
		lastInvClickTime = time.Time{}
	} else {
		selectInventoryItem(id, idx)
		lastInvClickID = id
		lastInvClickIdx = idx
		lastInvClickTime = now
	}
}

func selectInventoryItem(id uint16, idx int) {
	if id == selectedInvID && idx == selectedInvIdx {
		return
	}
	selectedInvID = id
	selectedInvIdx = idx
	serverIdx := idx
	if serverIdx < 0 {
		serverIdx = 0
	}
	enqueueCommand(fmt.Sprintf("\\BE-SELECT %d %d", id, serverIdx))
	nextCommand()
	updateInventoryWindow()
}

// handleInventoryContextClick opens the inventory context menu if the mouse
// position is over an inventory row. Returns true if a menu was opened.
func handleInventoryContextClick(mx, my int) bool {
	if inventoryWin == nil || inventoryList == nil || !inventoryWin.IsOpen() {
		return false
	}
	pos := eui.Point{X: float32(mx), Y: float32(my)}
	for _, row := range inventoryList.Contents {
		if !row.Hovered {
			continue
		}
		if ref, ok := inventoryRowRefs[row]; ok {
			openInventoryContextMenu(ref, pos)
			return true
		}
	}
	return false
}

func openInventoryContextMenu(ref invRef, pos eui.Point) {
	if inventoryCtxWin == nil {
		inventoryCtxWin = eui.NewWindow()
		inventoryCtxWin.Closable = true
		inventoryCtxWin.Movable = false
		inventoryCtxWin.Resizable = false
		inventoryCtxWin.NoScroll = true
		inventoryCtxWin.AutoSize = true
	}
	inventoryCtxWin.Contents = nil
	flow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Fixed: true}
	add := func(name string, fn func()) {
		b, _ := eui.NewButton()
		b.Text = name
		b.FontSize = 12
		b.Action = func() {
			fn()
			inventoryCtxWin.Close()
		}
		flow.AddItem(b)
	}
	add("Use", func() {
		selectInventoryItem(ref.id, ref.idx)
		enqueueCommand(fmt.Sprintf("/useitem %d", ref.id))
		nextCommand()
	})
	add("Drop", func() {
		selectInventoryItem(ref.id, ref.idx)
		enqueueCommand(fmt.Sprintf("/drop %d", ref.id))
		nextCommand()
	})
	add("Equip", func() { queueEquipCommand(ref.id, ref.idx) })
	add("Unequip", func() {
		enqueueCommand(fmt.Sprintf("/unequip %d", ref.id))
		nextCommand()
		equipInventoryItem(ref.id, ref.idx, false)
	})
	add("Examine", func() {
		selectInventoryItem(ref.id, ref.idx)
		enqueueCommand(fmt.Sprintf("/examine %d", ref.id))
		nextCommand()
	})
	add("Sell", func() {
		selectInventoryItem(ref.id, ref.idx)
		enqueueCommand(fmt.Sprintf("/sell %d", ref.id))
		nextCommand()
	})
	add("Show", func() {
		selectInventoryItem(ref.id, ref.idx)
		enqueueCommand(fmt.Sprintf("/show %d", ref.id))
		nextCommand()
	})
	add("Assign Shortcut", func() { promptInventoryShortcut(ref.global) })
	inventoryCtxWin.AddItem(flow)
	inventoryCtxWin.Position = pos
	inventoryCtxWin.MarkOpen()
	inventoryCtxWin.Refresh()
}

func promptInventoryShortcut(idx int) {
	invShortcutTarget = idx
	if invShortcutWin == nil {
		invShortcutWin = eui.NewWindow()
		invShortcutWin.Title = "Shortcut"
		invShortcutWin.AutoSize = true
		invShortcutWin.Closable = true
		invShortcutWin.Movable = false
		invShortcutWin.Resizable = false
		invShortcutWin.NoScroll = true
	}
	invShortcutWin.Contents = nil
	opts := []string{"None"}
	for r := '0'; r <= '9'; r++ {
		opts = append(opts, string(r))
	}
	for r := 'A'; r <= 'Z'; r++ {
		opts = append(opts, string(r))
	}
	dd, _ := eui.NewDropdown()
	dd.Options = opts
	dd.OnSelect = func(n int) {
		if n > 0 {
			setInventoryShortcut(idx, rune(opts[n][0]))
		} else {
			setInventoryShortcut(idx, 0)
		}
		inventoryDirty = true
		invShortcutWin.Close()
	}
	invShortcutWin.AddItem(dd)
	invShortcutWin.MarkOpen()
	invShortcutWin.Refresh()
}

func officialName(k invGroupKey, it InventoryItem) string {
	name := it.Name
	if name == "" && clImages != nil {
		name = clImages.ItemName(uint32(k.id))
	}
	if name == "" {
		name = fmt.Sprintf("Item %d", k.id)
	}
	name = norm.NFD.String(name)
	name = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, name)
	return foldCaser.String(name)
}
