package main

import "time"

func runBackgroundTasks() {

	go func() {
		var x uint64
		for {
			x++
			time.Sleep(time.Millisecond * 10)

			switch x {
			case 1:
				if time.Since(lastDebugStats) > time.Second {
					lastDebugStats = time.Now()
					if debugWin != nil && debugWin.IsOpen() {
						updateDebugStats()
					}
				}

			case 2:
				if inventoryDirty {
					updateInventoryWindow()
					updateHandsWindow()
					inventoryDirty = false
				}

			case 3:
				if playersDirty {
					updatePlayersWindow()
					playersDirty = false
				}

			case 4:
				if syncWindowSettings() {
					settingsDirty = true
				}

			case 5:
				if settingsDirty && qualityPresetDD != nil {
					qualityPresetDD.Selected = detectQualityPreset()
				}

			case 6:
				if time.Since(lastSettingsSave) >= 1*time.Second {
					if settingsDirty {
						saveSettings()
						settingsDirty = false
					}
					lastSettingsSave = time.Now()
				}

			case 7:
				// Periodically persist players if there were changes.
				if time.Since(lastPlayersSave) >= 10*time.Second {
					if playersDirty || playersPersistDirty {
						savePlayersPersist()
						playersPersistDirty = false
					}
					lastPlayersSave = time.Now()
				}

			case 8:
				// Ensure the movie controller window repaints at least once per second
				// while open, even without other UI events.
				if movieWin != nil && movieWin.IsOpen() {
					if time.Since(lastMovieWinTick) >= time.Second {
						lastMovieWinTick = time.Now()
						movieWin.Refresh()
					}
				}

			default:
				x = 0
			}
		}
	}()
}
