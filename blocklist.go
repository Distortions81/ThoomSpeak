package main

import "strings"

func init() {
	pluginRegisterCommand("client", "block", handleBlockCommand)
	pluginRegisterCommand("client", "ignore", handleIgnoreCommand)
	pluginRegisterCommand("client", "forget", handleForgetCommand)
}

func handleBlockCommand(args string) {
	name := utfFold(strings.TrimSpace(args))
	if name == "" {
		return
	}
	p := getPlayer(name)
	playersMu.Lock()
	wasBlocked := p.Blocked
	if wasBlocked {
		p.Blocked = false
		if p.FriendLabel == 6 {
			p.FriendLabel = 0
		}
	} else {
		p.Blocked = true
		p.Ignored = false
		p.Friend = false
		p.FriendLabel = 6
	}
	playerCopy := *p
	playersMu.Unlock()
	playersDirty = true
	killNameTagCacheFor(p.Name)
	notifyPlayerHandlers(playerCopy)
	msg := "Blocking " + p.Name + "."
	if wasBlocked {
		msg = "No longer blocking " + p.Name + "."
	}
	consoleMessage(msg)
}

func handleIgnoreCommand(args string) {
	name := utfFold(strings.TrimSpace(args))
	if name == "" {
		return
	}
	p := getPlayer(name)
	playersMu.Lock()
	wasIgnored := p.Ignored
	if wasIgnored {
		p.Ignored = false
		if p.FriendLabel == 7 {
			p.FriendLabel = 0
		}
	} else {
		p.Ignored = true
		p.Blocked = false
		p.Friend = false
		p.FriendLabel = 7
	}
	playerCopy := *p
	playersMu.Unlock()
	playersDirty = true
	killNameTagCacheFor(p.Name)
	notifyPlayerHandlers(playerCopy)
	msg := "Ignoring " + p.Name + "."
	if wasIgnored {
		msg = "No longer ignoring " + p.Name + "."
	}
	consoleMessage(msg)
}

func handleForgetCommand(args string) {
	name := utfFold(strings.TrimSpace(args))
	if name == "" {
		return
	}
	p := getPlayer(name)
	playersMu.Lock()
	wasBlocked := p.Blocked
	wasIgnored := p.Ignored
	wasFriend := p.Friend || p.FriendLabel != 0
	p.Blocked = false
	p.Ignored = false
	p.Friend = false
	p.FriendLabel = 0
	playerCopy := *p
	playersMu.Unlock()
	playersDirty = true
	killNameTagCacheFor(p.Name)
	notifyPlayerHandlers(playerCopy)
	msg := "Forgot " + p.Name + "."
	switch {
	case wasIgnored:
		msg = "No longer ignoring " + p.Name + "."
	case wasBlocked:
		msg = "No longer blocking " + p.Name + "."
	case wasFriend:
		msg = "Removing label from " + p.Name + "."
	}
	consoleMessage(msg)
}
