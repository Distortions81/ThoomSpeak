//go:build !test

package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"gothoom/eui"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

var playersWin *eui.WindowData
var playersList *eui.ItemData
var playersDirty bool
var playersRowRefs = map[*eui.ItemData]string{}
var playersCtxWin *eui.WindowData
var playersOfflineCollapsed = true

// defaultMobilePictID returns a fallback CL_Images mobile pict ID for the
// given gender when a player's specific PictID is unknown. Values are chosen
// to match classic client defaults (peasant male/female). For neutral/other,
// we fall back to the male peasant.
func defaultMobilePictID(g genderIcon) uint16 {
	switch g {
	case genderMale:
		return 447
	case genderFemale:
		return 456
	default:
		return 22
	}
}

func updatePlayersWindow() {
	if playersWin == nil || playersList == nil {
		return
	}

	// Gather current players and filter to non-NPCs with names.
	ps := getPlayers()
	// Sort: online (recently seen and not explicitly offline) first, by name; offline last, by name.
	sort.Slice(ps, func(i, j int) bool {
		staleI := time.Since(ps[i].LastSeen) > 5*time.Minute
		staleJ := time.Since(ps[j].LastSeen) > 5*time.Minute
		offI := ps[i].Offline || staleI
		offJ := ps[j].Offline || staleJ
		if offI != offJ {
			return !offI && offJ
		}
		// Both same offline status: sort by name
		return ps[i].Name < ps[j].Name
	})
	exiles := make([]Player, 0, len(ps))
	shareCount, shareeCount := 0, 0
	onlineCount := 0
	for _, p := range ps {
		if p.Name == "" || p.IsNPC {
			continue
		}
		if p.Sharing {
			shareCount++
		}
		if p.Sharee {
			shareeCount++
		}
		exiles = append(exiles, p)
		if !(p.Offline || time.Since(p.LastSeen) > 5*time.Minute) {
			onlineCount++
		}
	}

	myClan := ""
	if playerName != "" {
		playersMu.RLock()
		if me, ok := players[playerName]; ok {
			myClan = me.Clan
		}
		playersMu.RUnlock()
	}

	// Compute client area for sizing children similar to updateTextWindow.
	clientW := playersWin.GetSize().X
	clientH := playersWin.GetSize().Y - playersWin.GetTitleSize()
	s := eui.UIScale()
	if playersWin.NoScale {
		s = 1
	}
	pad := (playersWin.Padding + playersWin.BorderPad) * s
	clientWAvail := clientW - 2*pad
	if clientWAvail < 0 {
		clientWAvail = 0
	}
	clientHAvail := clientH - 2*pad
	if clientHAvail < 0 {
		clientHAvail = 0
	}

	// Determine row height from font metrics (ascent+descent).
	fontSize := gs.PlayersFontSize
	if fontSize <= 0 {
		fontSize = gs.ConsoleFontSize
	}
	ui := eui.UIScale()
	facePx := float64(float32(fontSize) * ui)
	var goFace *text.GoTextFace
	if src := eui.FontSource(); src != nil {
		goFace = &text.GoTextFace{Source: src, Size: facePx}
	} else {
		goFace = &text.GoTextFace{Size: facePx}
	}
	metrics := goFace.Metrics()
	linePx := math.Ceil(metrics.HAscent + metrics.HDescent + 2) // +2 px padding
	rowUnits := float32(linePx) / ui

	// Rebuild contents: header + one row per player
	// Layout per row: [avatar (or default/blank)] [profession (or blank)] [name]
	playersList.Contents = nil
	playersRowRefs = map[*eui.ItemData]string{}

	header := fmt.Sprintf("Players Online: %d", onlineCount)
	// Include simple share summary when relevant.
	if shareCount > 0 || shareeCount > 0 {
		parts := make([]string, 0, 2)
		if shareCount > 0 {
			parts = append(parts, fmt.Sprintf("sharing %d", shareCount))
		}
		if shareeCount > 0 {
			parts = append(parts, fmt.Sprintf("sharees %d", shareeCount))
		}
		header = fmt.Sprintf("%s — %s", header, strings.Join(parts, ", "))
	}
	ht, _ := eui.NewText()
	ht.Text = header
	ht.FontSize = float32(fontSize)
	ht.Size = eui.Point{X: clientWAvail, Y: rowUnits}
	playersList.AddItem(ht)

	onlinePlayers := make([]Player, 0, onlineCount)
	offlinePlayers := make([]Player, 0, len(exiles)-onlineCount)
	for _, p := range exiles {
		if p.Offline || time.Since(p.LastSeen) > 5*time.Minute {
			offlinePlayers = append(offlinePlayers, p)
		} else {
			onlinePlayers = append(onlinePlayers, p)
		}
	}

	addRow := func(p Player, offline bool, parent *eui.ItemData) {
		name := p.Name
		tags := make([]string, 0, 3)
		if p.Sharee {
			tags = append(tags, "<")
		}
		if p.Sharing {
			tags = append(tags, ">")
		}
		if myClan != "" && p.SameClan {
			tags = append(tags, "*")
		}
		if len(tags) > 0 {
			name = fmt.Sprintf("%s %s", name, strings.Join(tags, "--"))
		}

		row := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}

		iconSize := int(rowUnits + 0.5)

		{
			profItem, _ := eui.NewImageItem(iconSize, iconSize)
			profItem.Margin = 4
			profItem.Border = 0
			profItem.Filled = false
			profItem.Disabled = offline
			if pid := professionPictID(p.Class); pid != 0 {
				if img := loadImage(pid); img != nil {
					profItem.Image = img
					profItem.ImageName = "prof:cl:" + fmt.Sprint(pid)
				}
			}
			row.AddItem(profItem)
		}

		{
			avItem, _ := eui.NewImageItem(iconSize, iconSize)
			avItem.Margin = 4
			avItem.Border = 0
			avItem.Filled = false
			avItem.Disabled = offline
			var img *ebiten.Image
			state := uint8(0)
			if p.Dead {
				state = 32
			}
			if p.PictID != 0 {
				if m := loadMobileFrame(p.PictID, state, p.Colors); m != nil {
					img = m
				} else if im := loadImage(p.PictID); im != nil {
					img = im
				}
			}
			if img == nil {
				gid := defaultMobilePictID(genderFromString(p.Gender))
				if gid != 0 {
					if m := loadMobileFrame(gid, state, nil); m != nil {
						img = m
					} else if im := loadImage(gid); im != nil {
						img = im
					}
				}
			}
			if img != nil {
				avItem.Image = img
			}
			row.AddItem(avItem)
		}

		t, _ := eui.NewText()
		t.Text = name
		t.FontSize = float32(fontSize)
		face := mainFont
		if p.Sharing && p.Sharee {
			face = mainFontBoldItalic
		} else if p.Sharing {
			face = mainFontBold
		} else if p.Sharee {
			face = mainFontItalic
		}
		t.Face = face
		if p.Dead || offline {
			t.TextColor = eui.ColorVeryDarkGray
			t.ForceTextColor = true
		}
		t.Size = eui.Point{X: clientWAvail - float32(iconSize*2) - 8, Y: rowUnits}
		row.AddItem(t)

		row.Size.Y = rowUnits
		parent.AddItem(row)
	}

	for _, p := range onlinePlayers {
		addRow(p, false, playersList)
	}

	if len(offlinePlayers) > 0 {
		arrow := "\u25B6" // ▶
		if !playersOfflineCollapsed {
			arrow = "\u25BC" // ▼
		}
		foldBtn, events := eui.NewButton()
		foldBtn.Text = fmt.Sprintf("%s Offline: %d", arrow, len(offlinePlayers))
		foldBtn.FontSize = float32(fontSize)
		foldBtn.Size = eui.Point{X: clientWAvail, Y: rowUnits}
		foldBtn.Filled = false
		foldBtn.Outlined = false
		foldBtn.Border = 0
		foldBtn.BorderPad = 0
		foldBtn.Margin = 0
		foldBtn.Padding = 0
		foldBtn.Fixed = true
		events.Handle = func(ev eui.UIEvent) {
			if ev.Type == eui.EventClick {
				playersOfflineCollapsed = !playersOfflineCollapsed
				playersDirty = true
			}
		}
		playersList.AddItem(foldBtn)

		offFlow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Fixed: true}
		if playersOfflineCollapsed {
			offFlow.Invisible = true
		}
		playersList.AddItem(offFlow)
		for _, p := range offlinePlayers {
			addRow(p, true, offFlow)
		}
	}

	// Size flows to client area like other text windows.
	if playersList.Parent != nil {
		playersList.Parent.Size.X = clientWAvail
		playersList.Parent.Size.Y = clientHAvail
	}
	playersList.Size.X = clientWAvail
	playersList.Size.Y = clientHAvail
	playersWin.Refresh()
}
