package main

import (
	"context"
	"time"
)

// mobile pose helpers for orienting characters in fake mode.
const (
	poseStand uint8 = iota
	poseWalk1
	poseWalk2
	posePunch
	poseMax
)

const (
	facingEast uint8 = iota
	facingSE
	facingSouth
	facingSW
	facingWest
	facingNW
	facingNorth
	facingNE
)

func mobilePose(action, facing uint8) uint8 {
	return facing*poseMax + action
}

// pnTag wraps a name with -pn markers for name parsing.
func pnTag(name string) []byte {
	b := []byte{0xC2, 'p', 'n'}
	b = append(b, []byte(name)...)
	b = append(b, 0xC2, 'p', 'n')
	return b
}

// bepp wraps plain text with a BEPP prefix and NUL terminator.
func bepp(prefix string, msg []byte) []byte {
	b := []byte{0xC2}
	b = append(b, prefix[0], prefix[1])
	b = append(b, msg...)
	b = append(b, 0)
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
		state.mobiles[0] = frameMobile{Index: 0, State: mobilePose(poseStand, facingEast), H: -16, V: 0}
		state.mobiles[1] = frameMobile{Index: 1, State: mobilePose(poseStand, facingWest), H: 16, V: 0}

		// Lay out a tiny square room for the pair to inhabit.
		state.pictures = []framePicture{
			{PictID: 3038, H: -32, V: -32, Plane: -1},
			{PictID: 3038, H: 0, V: -32, Plane: -1},
			{PictID: 3038, H: -32, V: 0, Plane: -1},
			{PictID: 3038, H: 0, V: 0, Plane: -1},
			{PictID: 3039, H: -32, V: -64, Plane: 0}, // north wall
			{PictID: 3039, H: 0, V: -64, Plane: 0},
			{PictID: 3039, H: -64, V: -32, Plane: 0}, // west wall
			{PictID: 3039, H: -64, V: 0, Plane: 0},
			{PictID: 3039, H: 32, V: -32, Plane: 0}, // east wall
			{PictID: 3039, H: 32, V: 0, Plane: 0},
			{PictID: 3039, H: -32, V: 32, Plane: 0}, // south wall
			{PictID: 3039, H: 0, V: 32, Plane: 0},
			{PictID: 3040, H: -16, V: -16, Plane: 1}, // decorative rug
		}

		prepareRenderCacheLocked()
		stateMu.Unlock()
		playersDirty = true

		// Helper to append a bubble and show corresponding chat message.
		emitBubble := func(idx uint8, typ int, name, verb, txt string) {
			stateMu.Lock()
			state.bubbles = append(state.bubbles, bubble{Index: idx, Text: txt, Type: typ, CreatedFrame: frameCounter})
			stateMu.Unlock()
			switch verb {
			case "", bubbleVerbVerbatim:
				chatMessage(txt)
			case bubbleVerbParentheses:
				chatMessage(name + " " + txt)
			default:
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
				handleInfoText(append(bepp("sh", msg), '\r'))
			case 1: // Bob shares you
				msg := append(pnTag("Bob"), []byte(" is sharing experiences with you.")...)
				handleInfoText(append(bepp("sh", msg), '\r'))
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
			case 7: // Bob acts
				emitBubble(1, kBubblePlayerAction, "Bob", bubbleVerbParentheses, "waves excitedly")
			case 8: // Bob falls
				msg := append(pnTag("Bob"), []byte(" has fallen")...)
				handleInfoText(append(msg, '\r'))
			case 9: // Bob recovers
				msg := append(pnTag("Bob"), []byte(" is no longer fallen")...)
				handleInfoText(append(msg, '\r'))
			case 10: // You unshare Bob
				msg := append([]byte("You are no longer sharing experiences with "), pnTag("Bob")...)
				msg = append(msg, '.')
				handleInfoText(append(bepp("su", msg), '\r'))
			case 11: // Bob unshares you
				msg := append(pnTag("Bob"), []byte(" is no longer sharing experiences with you.")...)
				handleInfoText(append(bepp("su", msg), '\r'))
			}
			step = (step + 1) % 12
		}
	}()
}
