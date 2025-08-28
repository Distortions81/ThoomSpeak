package main

import "testing"

func TestCurrentNightLevel(t *testing.T) {
	cases := []struct {
		name   string
		force  int
		max    int
		flags  uint
		server int
		want   int
	}{
		{"AutoWithinLimit", -1, 100, 0, 60, 60},
		{"AutoClampedToMax", -1, 25, 0, 80, 25},
		{"ForcedNightClamped", 100, 25, 0, 0, 25},
		{"ForcedDayOverrides", 0, 100, 0, 80, 0},
		{"Force100FlagOverridesMax", -1, 25, kLightForce100Pct, 80, 80},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gNight.mu.Lock()
			oldLvl, oldFlags := gNight.Level, gNight.Flags
			gNight.Level = tc.server
			gNight.Flags = tc.flags
			gNight.mu.Unlock()
			defer func() {
				gNight.mu.Lock()
				gNight.Level, gNight.Flags = oldLvl, oldFlags
				gNight.mu.Unlock()
			}()

			oldForce, oldMax := gs.ForceNightLevel, gs.MaxNightLevel
			gs.ForceNightLevel, gs.MaxNightLevel = tc.force, tc.max
			defer func() {
				gs.ForceNightLevel, gs.MaxNightLevel = oldForce, oldMax
			}()

			if got := currentNightLevel(); got != tc.want {
				t.Fatalf("currentNightLevel() = %d; want %d", got, tc.want)
			}
		})
	}
}
