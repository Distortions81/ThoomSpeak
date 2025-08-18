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

		// Populate simple player descriptors and mobiles so Hero and Bob
		// appear in the player list and on screen without a server
		// connection.
		updatePlayerAppearance("Hero", 447, nil, false)
		updatePlayerAppearance("Bob", 447, nil, false)
		stateMu.Lock()
		playerIndex = 0
		state.descriptors[0] = frameDescriptor{Index: 0, Type: kDescPlayer, PictID: 447, Name: "Hero"}
		state.descriptors[1] = frameDescriptor{Index: 1, Type: kDescPlayer, PictID: 447, Name: "Bob"}
		state.mobiles[0] = frameMobile{Index: 0, H: 0, V: 0}
		state.mobiles[1] = frameMobile{Index: 1, H: 32, V: 0}
		prepareRenderCacheLocked()
		stateMu.Unlock()
		playersDirty = true

		// Helper to append a bubble and show corresponding chat message.
		emitBubble := func(idx uint8, typ int, name, verb, txt string) {
			stateMu.Lock()
			state.bubbles = append(state.bubbles, bubble{Index: idx, Text: txt, Type: typ, CreatedFrame: frameCounter})
			stateMu.Unlock()
			if verb == "" {
				chatMessage(txt)
			} else {
				chatMessage(name + " " + verb + ", " + txt)
			}
		}

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		step := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			switch step {
			case 0: // You share Bob
				msg := append([]byte("You are sharing experiences with "), pnTag("Bob")...)
				msg = append(msg, '.')
				handleInfoText(append(msg, '\r'))
			case 1: // Bob shares you
				msg := append(pnTag("Bob"), []byte(" is sharing experiences with you.")...)
				handleInfoText(append(msg, '\r'))
			case 2: // Hero speaks
				emitBubble(0, kBubbleNormal, "Hero", "says", "Hello there!")
			case 3: // Bob whispers
				emitBubble(1, kBubbleWhisper, "Bob", "whispers", "psst...")
			case 4: // Hero yells
				emitBubble(0, kBubbleYell, "Hero", "yells", "Watch out!")
			case 5: // Bob thinks
				emitBubble(1, kBubbleThought, "Bob", "thinks", "I wonder...")
			case 6: // Bob thinks to you
				emitBubble(1, kBubbleThought, "Bob", "thinks to you", "Hello Hero")
			case 7: // Bob falls
				msg := append(pnTag("Bob"), []byte(" has fallen")...)
				handleInfoText(append(msg, '\r'))
			case 8: // Bob recovers
				msg := append(pnTag("Bob"), []byte(" is no longer fallen")...)
				handleInfoText(append(msg, '\r'))
			case 9: // You unshare Bob
				msg := append([]byte("You are no longer sharing experiences with "), pnTag("Bob")...)
				msg = append(msg, '.')
				handleInfoText(append(msg, '\r'))
			case 10: // Bob unshares you
				msg := append(pnTag("Bob"), []byte(" is no longer sharing experiences with you.")...)
				handleInfoText(append(msg, '\r'))
			}
			step = (step + 1) % 11
		}
	}()
}
