package main

import (
	"encoding/binary"
	"testing"
)

func TestUpdateFrameCounters(t *testing.T) {
	lastAckFrame = 0
	numFrames = 0
	lostFrames = 0

	if dropped := updateFrameCounters(1); dropped != 0 {
		t.Fatalf("expected 0 dropped, got %d", dropped)
	}
	if numFrames != 1 || lostFrames != 0 || lastAckFrame != 1 {
		t.Fatalf("unexpected counters after first frame: num=%d lost=%d last=%d", numFrames, lostFrames, lastAckFrame)
	}

	if dropped := updateFrameCounters(3); dropped != 1 {
		t.Fatalf("expected 1 dropped, got %d", dropped)
	}
	if numFrames != 2 || lostFrames != 1 || lastAckFrame != 3 {
		t.Fatalf("unexpected counters after second frame: num=%d lost=%d last=%d", numFrames, lostFrames, lastAckFrame)
	}

	if dropped := updateFrameCounters(4); dropped != 0 {
		t.Fatalf("expected 0 dropped, got %d", dropped)
	}
	if numFrames != 3 || lostFrames != 1 || lastAckFrame != 4 {
		t.Fatalf("unexpected counters after third frame: num=%d lost=%d last=%d", numFrames, lostFrames, lastAckFrame)
	}
}

func TestParseDrawStateCountsDroppedFrames(t *testing.T) {
	// Simulate starting from frame 1 as handleDrawState would after
	// receiving the first frame.
	frameCounter = 1
	lastAckFrame = 1
	numFrames = 0
	lostFrames = 0

	// Minimal draw state packet with ackFrame=3 indicating one dropped frame
	// between the previous and current acknowledgements.
	data := make([]byte, 21)
	binary.BigEndian.PutUint32(data[1:5], uint32(3))
	// resendFrame remains zeroes
	// descriptor count
	data[9] = 0
	// stats (hp/hpmax/sp/spmax/balance/balmax/lighting)
	data[10] = 0
	data[11] = 0
	data[12] = 0
	data[13] = 0
	data[14] = 0
	data[15] = 0
	data[16] = 0
	// picture count
	data[17] = 0
	// mobile count
	data[18] = 0
	// state size already zero

	if err := parseDrawState(data, false); err != nil {
		t.Fatalf("parseDrawState error: %v", err)
	}
	if frameCounter != 2 {
		t.Fatalf("expected frameCounter 2, got %d", frameCounter)
	}
	if lostFrames != 1 {
		t.Fatalf("expected lostFrames 1, got %d", lostFrames)
	}
}
