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
