package main

import "testing"

func TestParseNightCommand(t *testing.T) {
	tests := []struct {
		name      string
		cmd       string
		baseLevel int
		level     int
		azimuth   int
		cloudy    bool
		shadows   int
	}{
		{
			name:      "new style mixed case",
			cmd:       "/Nt 50 /sA 42 /cL 1",
			baseLevel: 50,
			level:     50,
			azimuth:   42,
			cloudy:    true,
			shadows:   0,
		},
		{
			name:      "new style uppercase",
			cmd:       "/NT 51 /SA -1 /CL 0",
			baseLevel: 51,
			level:     51,
			azimuth:   -1,
			cloudy:    false,
			shadows:   0,
		},
		{
			name:      "new style odd whitespace",
			cmd:       "/Nt\u00a052\t/SA\t40\u00a0/CL\u00a01",
			baseLevel: 52,
			level:     52,
			azimuth:   40,
			cloudy:    true,
			shadows:   0,
		},
		{
			name:      "legacy long mixed case",
			cmd:       "/nT 10 20 30 40",
			baseLevel: 10,
			level:     10,
			azimuth:   30,
			cloudy:    false,
			shadows:   20,
		},
		{
			name:      "legacy short uppercase",
			cmd:       "/NT 25",
			baseLevel: 25,
			level:     25,
			azimuth:   0,
			cloudy:    false,
			shadows:   25,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gNight = NightInfo{}
			if !parseNightCommand(tt.cmd) {
				t.Fatalf("parseNightCommand(%q) = false, want true", tt.cmd)
			}
			gNight.mu.Lock()
			gotBase := gNight.BaseLevel
			gotLevel := gNight.Level
			gotAzimuth := gNight.Azimuth
			gotCloudy := gNight.Cloudy
			gotShadows := gNight.Shadows
			gNight.mu.Unlock()
			if gotBase != tt.baseLevel || gotLevel != tt.level || gotAzimuth != tt.azimuth || gotCloudy != tt.cloudy || gotShadows != tt.shadows {
				t.Fatalf("parseNightCommand(%q) = {BaseLevel:%d Level:%d Azimuth:%d Cloudy:%v Shadows:%d}, want {BaseLevel:%d Level:%d Azimuth:%d Cloudy:%v Shadows:%d}",
					tt.cmd, gotBase, gotLevel, gotAzimuth, gotCloudy, gotShadows, tt.baseLevel, tt.level, tt.azimuth, tt.cloudy, tt.shadows)
			}
		})
	}
}

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
