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
	}{
		{
			name:      "new style mixed case",
			cmd:       "/Nt 50 /sA 42 /cL 1",
			baseLevel: 50,
			level:     50,
			azimuth:   42,
			cloudy:    true,
		},
		{
			name:      "new style uppercase",
			cmd:       "/NT 51 /SA -1 /CL 0",
			baseLevel: 51,
			level:     51,
			azimuth:   -1,
			cloudy:    false,
		},
		{
			name:      "legacy long mixed case",
			cmd:       "/nT 10 20 30 40",
			baseLevel: 10,
			level:     10,
			azimuth:   30,
		},
		{
			name:      "legacy short uppercase",
			cmd:       "/NT 25",
			baseLevel: 25,
			level:     25,
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
			gNight.mu.Unlock()
			if gotBase != tt.baseLevel || gotLevel != tt.level || gotAzimuth != tt.azimuth || gotCloudy != tt.cloudy {
				t.Fatalf("parseNightCommand(%q) = {BaseLevel:%d Level:%d Azimuth:%d Cloudy:%v}, want {BaseLevel:%d Level:%d Azimuth:%d Cloudy:%v}", tt.cmd, gotBase, gotLevel, gotAzimuth, gotCloudy, tt.baseLevel, tt.level, tt.azimuth, tt.cloudy)
			}
		})
	}
}
