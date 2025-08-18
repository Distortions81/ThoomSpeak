package main

import (
	"context"
	"time"
)

// pnTag wraps a name with -pn markers for name parsing.
func pnTag(name string) []byte {
	b := []byte{0xC2, 'p', 'n'}
	b = append(b, []byte(name)...)
	b = append(b, 0xC2, 'p', 'n')
	return b
}

// runFakeMode injects sample share and fallen messages using real server
// formats captured from PCAP data. It allows testing client behavior without
// connecting to the live server.
func runFakeMode(ctx context.Context) {
	go func() {
		select {
		case <-gameStarted:
		case <-ctx.Done():
			return
		}

		playerName = "Hero"

		// Populate a simple player descriptor and mobile record using
		// values captured from live traffic. This allows the player to
		// appear in the player list and be rendered on screen without a
		// server connection.
		updatePlayerAppearance("Hero", 447, nil, false)
		stateMu.Lock()
		state.descriptors[0] = frameDescriptor{Index: 0, Type: kDescPlayer, PictID: 447, Name: "Hero"}
		state.mobiles[0] = frameMobile{Index: 0, H: 0, V: 0}
		prepareRenderCacheLocked()
		stateMu.Unlock()
		playersDirty = true

		// Hero shares Bob
		msg := append(pnTag("Hero"), []byte(" is sharing experiences with ")...)
		msg = append(msg, pnTag("Bob")...)
		msg = append(msg, '.')
		handleInfoText(append(msg, '\r'))

		// Bob falls
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return
		}
		msg = append(pnTag("Bob"), []byte(" has fallen")...)
		handleInfoText(append(msg, '\r'))

		// Bob recovers
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return
		}
		msg = append(pnTag("Bob"), []byte(" is no longer fallen")...)
		handleInfoText(append(msg, '\r'))

		// Hero unshares Bob
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return
		}
		msg = append(pnTag("Hero"), []byte(" is no longer sharing experiences with ")...)
		msg = append(msg, pnTag("Bob")...)
		msg = append(msg, '.')
		handleInfoText(append(msg, '\r'))
	}()
}
