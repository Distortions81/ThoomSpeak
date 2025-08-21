package main

import (
	"bytes"
	"strconv"
	"strings"
	"time"
)

// parseWhoText parses a plain-text /who line with embedded BEPP player tags.
// Returns true if handled and should be suppressed from console.
func parseWhoText(raw []byte, s string) bool {
	if strings.HasPrefix(s, "You are the only one in the lands.") {
		// Nothing to add
		return true
	}
	if !strings.HasPrefix(s, "In the world are ") {
		return false
	}
	// Find first -pn tag segment and extract all names.
	// The format is: In the world are …: -pn <name> -pn , realname , <gm> \t ...
	off := bytes.Index(raw, []byte{0xC2, 'p', 'n'})
	if off < 0 {
		return true // handled, but no names
	}
	names := parseNames(raw[off:])
	if len(names) == 0 {
		return true
	}
	for _, name := range names {
		p := getPlayer(name)
		playersMu.Lock()
		p.LastSeen = time.Now()
		p.Offline = false
		playersMu.Unlock()
	}
	playersDirty = true
	return true
}

// parseShareText parses plain share/unshare lines with embedded -pn tags.
// Returns true if the line was recognized and handled.
func parseShareText(raw []byte, s string) bool {
	switch {
	case strings.HasPrefix(s, "You are not sharing experiences with anyone.") ||
		strings.HasPrefix(s, "You are no longer sharing experiences with anyone."):
		// Clear sharees
		playersMu.Lock()
		changed := make([]string, 0, len(players))
		for _, p := range players {
			if p.Sharee {
				p.Sharee = false
				changed = append(changed, p.Name)
			}
		}
		playersMu.Unlock()
		for _, n := range changed {
			killNameTagCacheFor(n)
		}
		playersDirty = true
		return true
	case strings.HasPrefix(s, "You are no longer sharing experiences with "):
		// a single sharee removed
		// name will be in -pn tags
		off := bytes.Index(raw, []byte{0xC2, 'p', 'n'})
		if off >= 0 {
			names := parseNames(raw[off:])
			playersMu.Lock()
			changed := make([]string, 0, len(names))
			for _, name := range names {
				if p, ok := players[name]; ok && p.Sharee {
					p.Sharee = false
					changed = append(changed, name)
				}
			}
			playersMu.Unlock()
			for _, n := range changed {
				killNameTagCacheFor(n)
			}
			playersDirty = true
		}
		return true
	case strings.HasPrefix(s, "You are sharing experiences with ") || strings.HasPrefix(s, "You begin sharing your experiences with "):
		// Self -> sharees
		// Clear any existing sharees first
		playersMu.Lock()
		changed := make([]string, 0, len(players))
		for _, p := range players {
			if p.Sharee {
				p.Sharee = false
				changed = append(changed, p.Name)
			}
		}
		playersMu.Unlock()
		for _, n := range changed {
			killNameTagCacheFor(n)
		}
		off := bytes.Index(raw, []byte{0xC2, 'p', 'n'})
		if off >= 0 {
			names := parseNames(raw[off:])
			changed = changed[:0]
			for _, name := range names {
				p := getPlayer(name)
				if !p.Sharee {
					p.Sharee = true
					changed = append(changed, name)
				}
			}
			for _, n := range changed {
				killNameTagCacheFor(n)
			}
			playersDirty = true
		}
		return true
	case playerName != "" && (strings.HasPrefix(s, playerName+" is sharing experiences with ") || strings.HasPrefix(s, playerName+" begins sharing experiences with ")):
		// Hero (you) sharing others in third-person form
		playersMu.Lock()
		changed := make([]string, 0, len(players))
		for _, p := range players {
			if p.Sharee {
				p.Sharee = false
				changed = append(changed, p.Name)
			}
		}
		playersMu.Unlock()
		for _, n := range changed {
			killNameTagCacheFor(n)
		}
		off := bytes.Index(raw, []byte{0xC2, 'p', 'n'})
		if off >= 0 {
			names := parseNames(raw[off:])
			playersMu.Lock()
			changed = changed[:0]
			for _, name := range names {
				if name == playerName {
					continue
				}
				p := getPlayer(name)
				if !p.Sharee {
					p.Sharee = true
					changed = append(changed, name)
				}
			}
			playersMu.Unlock()
			for _, n := range changed {
				killNameTagCacheFor(n)
			}
			playersDirty = true
		}
		return true
	case playerName != "" && strings.HasPrefix(s, playerName+" is no longer sharing experiences with "):
		// Hero (you) unsharing others in third-person form
		off := bytes.Index(raw, []byte{0xC2, 'p', 'n'})
		if off >= 0 {
			names := parseNames(raw[off:])
			playersMu.Lock()
			changed := make([]string, 0, len(names))
			for _, name := range names {
				if name == playerName {
					continue
				}
				if p, ok := players[name]; ok && p.Sharee {
					p.Sharee = false
					changed = append(changed, name)
				}
			}
			playersMu.Unlock()
			for _, n := range changed {
				killNameTagCacheFor(n)
			}
			playersDirty = true
		}
		return true
	case strings.HasSuffix(s, " is sharing experiences with you."):
		name := utfFold(firstTagContent(raw, 'p', 'n'))
		if name != "" {
			p := getPlayer(name)
			playersMu.Lock()
			changed := !p.Sharing
			p.Sharing = true
			playersMu.Unlock()
			if changed {
				killNameTagCacheFor(name)
			}
			playersDirty = true
			showNotification(name + " is sharing with you")
		}
		return true
	case strings.Contains(s, " is no longer sharing experiences with you"):
		name := utfFold(firstTagContent(raw, 'p', 'n'))
		if name != "" {
			playersMu.Lock()
			changed := false
			if p, ok := players[name]; ok {
				if p.Sharing {
					p.Sharing = false
					changed = true
				}
			}
			playersMu.Unlock()
			if changed {
				killNameTagCacheFor(name)
			}
			playersDirty = true
		}
		return true
	case strings.HasPrefix(s, "Currently sharing their experiences with you"):
		// Upstream sharers
		off := bytes.Index(raw, []byte{0xC2, 'p', 'n'})
		if off >= 0 {
			names := parseNames(raw[off:])
			playersMu.Lock()
			changed := make([]string, 0, len(names))
			for _, name := range names {
				p := getPlayer(name)
				if !p.Sharing {
					p.Sharing = true
					changed = append(changed, name)
				}
			}
			playersMu.Unlock()
			for _, n := range changed {
				killNameTagCacheFor(n)
			}
			playersDirty = true
		}
		return true
	}
	return false
}

// parseFallenText detects fallen/not-fallen messages and updates state.
// Returns true if handled.
func parseFallenText(raw []byte, s string) bool {
	// Fallen: "<pn name> has fallen" (with optional -mn and -lo tags)
	if strings.Contains(s, " has fallen") {
		// Extract main player name
		name := utfFold(firstTagContent(raw, 'p', 'n'))
		if name == "" {
			if idx := strings.Index(s, " has fallen"); idx >= 0 {
				name = utfFold(strings.TrimSpace(s[:idx]))
			}
		}
		if name == "" {
			return false
		}
		killer := utfFold(firstTagContent(raw, 'm', 'n'))
		where := firstTagContent(raw, 'l', 'o')
		p := getPlayer(name)
		playersMu.Lock()
		p.Dead = true
		p.KillerName = killer
		p.FellWhere = where
		playersMu.Unlock()
		playersDirty = true
		if gs.NotifyFallen {
			showNotification(name + " has fallen")
		}
		return true
	}
	// Not fallen: "<pn name> is no longer fallen"
	if strings.Contains(s, " is no longer fallen") {
		name := utfFold(firstTagContent(raw, 'p', 'n'))
		if name == "" {
			if idx := strings.Index(s, " is no longer fallen"); idx >= 0 {
				name = utfFold(strings.TrimSpace(s[:idx]))
			}
		}
		if name == "" {
			return false
		}
		playersMu.Lock()
		if p, ok := players[name]; ok {
			p.Dead = false
			p.FellWhere = ""
			p.KillerName = ""
		}
		playersMu.Unlock()
		playersDirty = true
		if gs.NotifyNotFallen {
			showNotification(name + " is no longer fallen")
		}
		return true
	}
	return false
}

// parsePresenceText detects login/logoff/plain presence changes. Returns true if handled.
func parsePresenceText(raw []byte, s string) bool {
	// Attempt to detect common phrases. Names are provided in -pn tags.
	// We treat any recognized login as Online and any recognized logout as Offline.
	lower := strings.ToLower(s)
	name := utfFold(firstTagContent(raw, 'p', 'n'))
	if name == "" {
		return false
	}
	// Login-like phrases
	if strings.Contains(lower, "has logged on") || strings.Contains(lower, "has entered the lands") || strings.Contains(lower, "has joined the world") || strings.Contains(lower, "has arrived") {
		var friend bool
		playersMu.Lock()
		if p, ok := players[name]; ok {
			p.LastSeen = time.Now()
			p.Offline = false
			friend = p.Friend
		}
		playersMu.Unlock()
		playersDirty = true
		if friend && gs.NotifyFriendOnline {
			showNotification(name + " is online")
		}
		return true
	}
	// Logout-like phrases
	if strings.Contains(lower, "has logged off") || strings.Contains(lower, "has left the lands") || strings.Contains(lower, "has left the world") || strings.Contains(lower, "has departed") || strings.Contains(lower, "has signed off") {
		playersMu.Lock()
		if p, ok := players[name]; ok {
			p.Offline = true
		}
		playersMu.Unlock()
		playersDirty = true
		return true
	}
	return false
}

// parseBardText detects bard guild messages and updates bard status.
// It also handles bard tune messages. Returns true if the message was fully
// handled and should not be displayed.
func parseBardText(_ []byte, s string) bool {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "* ") {
		s = strings.TrimSpace(s[2:])
	}
	if strings.HasPrefix(s, "¥ ") {
		s = strings.TrimSpace(s[2:])
	}

	if parseMusicCommand(s) {
		return true
	}

	phrases := []struct {
		suffix string
		bard   bool
	}{
		{" is a Bard Crafter", true},
		{" is a Bard Master", true},
		{" is a Bard Trustee", true},
		{" is a Bard Quester", true},
		{" is a Bard Guest", true},
		{" is a Bard", true},
		{" is not in the Bards' Guild", false},
		{" is not a Bard", false},
	}
	for _, ph := range phrases {
		if strings.HasSuffix(s, ph.suffix) {
			name := strings.TrimSpace(strings.TrimSuffix(s, ph.suffix))
			if name == "" {
				return false
			}
			p := getPlayer(name)
			playersMu.Lock()
			p.Bard = ph.bard
			p.LastSeen = time.Now()
			p.Offline = false
			playersMu.Unlock()
			playersDirty = true
			playersPersistDirty = true
			return false
		}
	}
	return false
}

// parseMusicCommand handles bard /music or /play commands and plays the tune.
// It supports both the simple "/play <inst> <notes>" form and the slash-delimited
// backend messages like "/music/.../play/inst<inst>/notes<notes>".
func parseMusicCommand(s string) bool {
	if strings.HasPrefix(s, "/music/") {
		s = s[len("/music/"):]
	}

	switch {
	case strings.HasPrefix(s, "/play"):
		s = s[len("/play"):]
	case strings.HasPrefix(s, "play"):
		s = s[len("play"):]
	case len(s) > 0 && (s[0] == 'P' || s[0] == 'p'):
		s = s[1:]
	default:
		return false
	}

	s = strings.TrimSpace(s)

	inst := defaultInstrument
	if idx := strings.Index(s, "/inst"); idx >= 0 {
		v := s[idx+len("/inst"):]
		v = strings.TrimPrefix(v, "/")
		if j := strings.IndexByte(v, '/'); j >= 0 {
			v = v[:j]
		}
		if n, err := strconv.Atoi(v); err == nil {
			inst = n
		}
	} else if idx := strings.Index(s, "/I"); idx >= 0 {
		v := s[idx+len("/I"):]
		if len(v) > 0 && v[0] == '/' {
			v = v[1:]
		}
		if j := strings.IndexByte(v, '/'); j >= 0 {
			v = v[:j]
		}
		if n, err := strconv.Atoi(v); err == nil {
			inst = n
		}

	}

	notes := ""
	if idx := strings.Index(s, "/notes"); idx >= 0 {
		notes = s[idx+len("/notes"):]
	} else if idx := strings.Index(s, "/N"); idx >= 0 {
		notes = s[idx+len("/N"):]
	} else {
		notes = s
	}
	notes = strings.Trim(notes, "/")
	if notes == "" {
		return false
	}
	go playClanLordTune(strconv.Itoa(inst) + " " + strings.TrimSpace(notes))

	if musicDebug {
		msg := "/play " + strconv.Itoa(inst) + " " + strings.TrimSpace(notes)
		consoleMessage(msg)
		if !gs.MessagesToConsole {
			chatMessage(msg)
		}
	}
	return true
}

// firstTagContent extracts the first bracketed content for a given 2-letter BEPP tag.
func firstTagContent(b []byte, a, b2 byte) string {
	i := bytes.Index(b, []byte{0xC2, a, b2})
	if i < 0 {
		return ""
	}
	rest := b[i+3:]
	j := bytes.Index(rest, []byte{0xC2, a, b2})
	if j < 0 {
		return ""
	}
	return strings.TrimSpace(decodeMacRoman(rest[:j]))
}
