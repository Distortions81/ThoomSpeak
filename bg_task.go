package main

import (
	"context"
	"time"
)

func runBackgroundTasks() {
	go debugStatsTask(gameCtx)
	go inventoryRefreshTask(gameCtx)
	go playersRefreshTask(gameCtx)
	go syncWindowSettingsTask(gameCtx)
	go qualityPresetTask(gameCtx)
	go settingsSaveTask(gameCtx)
	go playersSaveTask(gameCtx)
	go movieWindowRefreshTask(gameCtx)
}

func debugStatsTask(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			ebitenStopped = true
			return
		case <-ticker.C:
			if debugWin != nil && debugWin.IsOpen() {
				updateDebugStats()
			}
		}
	}
}

func inventoryRefreshTask(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			ebitenStopped = true
			return
		case <-ticker.C:
			if inventoryDirty {
				updateInventoryWindow()
				updateHandsWindow()
				inventoryDirty = false
			}
		}
	}
}

func playersRefreshTask(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			ebitenStopped = true
			return
		case <-ticker.C:
			if playersDirty {
				updatePlayersWindow()
				playersDirty = false
			}
		}
	}
}

func syncWindowSettingsTask(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			ebitenStopped = true
			return
		case <-ticker.C:
			if syncWindowSettings() {
				settingsDirty = true
			}
		}
	}
}

func qualityPresetTask(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			ebitenStopped = true
			return
		case <-ticker.C:
			if settingsDirty && qualityPresetDD != nil {
				qualityPresetDD.Selected = detectQualityPreset()
			}
		}
	}
}

func settingsSaveTask(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			ebitenStopped = true
			return
		case <-ticker.C:
			if settingsDirty {
				saveSettings()
				settingsDirty = false
			}
			lastSettingsSave = time.Now()
		}
	}
}

func playersSaveTask(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			ebitenStopped = true
			return
		case <-ticker.C:
			if playersDirty || playersPersistDirty {
				savePlayersPersist()
				playersPersistDirty = false
			}
			lastPlayersSave = time.Now()
		}
	}
}

func movieWindowRefreshTask(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			ebitenStopped = true
			return
		case <-ticker.C:
			if movieWin != nil && movieWin.IsOpen() {
				movieWin.Refresh()
			}
		}
	}
}
