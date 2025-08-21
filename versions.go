package main

import (
	"archive/zip"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed data/versions.json
var versionsJSON []byte

//go:embed data/changelog/*
var changelogFS embed.FS

var (
	appVersion int
	clVersion  = baseVersion
	changelog  string

	changelogVersions   []int
	changelogVersionIdx int
)

const versionsURL = "https://m45sci.xyz/u/dist/goThoom/versions.json"
const binaryURLFmt = "https://m45sci.xyz/u/dist/goThoom/gothoom-%d.zip"

var uiReady bool

// versionEntry mirrors the structure of the entries in data/versions.json.
// The JSON file uses capitalized field names, so the tags here must match
// those exactly in order for decoding to succeed.
type versionEntry struct {
	Version   int `json:"Version"`
	CLVersion int `json:"CLVersion"`
}

type versionFile struct {
	Versions []versionEntry `json:"versions"`
}

func init() {
	var vf versionFile
	if err := json.Unmarshal(versionsJSON, &vf); err != nil {
		log.Printf("parse versions.json: %v", err)
		return
	}
	if len(vf.Versions) == 0 {
		return
	}
	latest := vf.Versions[0]
	for _, v := range vf.Versions[1:] {
		if v.Version > latest.Version {
			latest = v
		}
	}
	appVersion = latest.Version
	if latest.CLVersion != 0 {
		clVersion = latest.CLVersion
	}

	// Discover available changelog versions.
	entries, err := changelogFS.ReadDir("data/changelog")
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".txt")
			if v, err := strconv.Atoi(name); err == nil {
				changelogVersions = append(changelogVersions, v)
			}
		}
		sort.Ints(changelogVersions)
		for i, v := range changelogVersions {
			if v == appVersion {
				changelogVersionIdx = i
				break
			}
		}
	}

	loadChangelogAt(changelogVersionIdx)
	if changelog == "" {
		b, err := changelogFS.ReadFile(fmt.Sprintf("data/changelog/%d.txt", appVersion))
		if err != nil {
			log.Printf("read changelog: %v", err)
		} else {
			changelog = string(b)
		}
	}
}

func loadChangelogAt(idx int) bool {
	if idx < 0 || idx >= len(changelogVersions) {
		return false
	}
	v := changelogVersions[idx]
	b, err := changelogFS.ReadFile(fmt.Sprintf("data/changelog/%d.txt", v))
	if err != nil {
		log.Printf("read changelog: %v", err)
		return false
	}
	changelog = string(b)
	changelogVersionIdx = idx
	return true
}

func checkForNewVersion() {
	gs.LastUpdateCheck = time.Now()
	settingsDirty = true

	resp, err := http.Get(versionsURL)
	if err != nil {
		log.Printf("check new version: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("check new version: %v", resp.Status)
		return
	}
	var vf versionFile
	if err := json.NewDecoder(resp.Body).Decode(&vf); err != nil {
		log.Printf("check new version: %v", err)
		return
	}
	if len(vf.Versions) == 0 {
		return
	}
	latest := vf.Versions[0]
	for _, v := range vf.Versions[1:] {
		if v.Version > latest.Version {
			latest = v
		}
	}
	if latest.Version > appVersion {
		consoleMessage(fmt.Sprintf("New goThoom version %d available", latest.Version))
		if tcpConn != nil {
			if gs.NotifiedVersion >= latest.Version {
				return
			}
			gs.NotifiedVersion = latest.Version
			settingsDirty = true
			go func(ver int) {
				for !uiReady {
					time.Sleep(100 * time.Millisecond)
				}
				showNotification(fmt.Sprintf("goThoom version %d is available!", ver))
			}(latest.Version)
			return
		}
		gs.NotifiedVersion = latest.Version
		settingsDirty = true
		go func(ver int) {
			for !uiReady {
				time.Sleep(100 * time.Millisecond)
			}
			showPopup(
				"Update Available",
				fmt.Sprintf("goThoom version %d is available!", ver),
				[]popupButton{
					{Text: "Cancel"},
					{Text: "Download", Action: func() {
						go func(v int) {
							if err := downloadAndInstall(v); err != nil {
								log.Printf("download update: %v", err)
							}
						}(ver)
					}},
				},
			)
		}(latest.Version)
		return
	}
	consoleMessage("This version of goThoom is the latest version!")
}

func versionCheckLoop() {
	for {
		wait := 3*time.Hour - time.Since(gs.LastUpdateCheck)
		if wait > 0 {
			time.Sleep(wait)
		}
		checkForNewVersion()
	}
}

func downloadAndInstall(ver int) error {
	url := fmt.Sprintf(binaryURLFmt, ver)
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exeDir := filepath.Dir(exePath)
	tmpDir, err := os.MkdirTemp(exeDir, "update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	zipPath := filepath.Join(tmpDir, "gothoom.zip")
	if err := downloadFile(url, zipPath); err != nil {
		return err
	}
	newExe, err := unzipExecutable(zipPath, tmpDir)
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		bat := filepath.Join(tmpDir, "update.bat")
		script := fmt.Sprintf(`@echo off
ping 127.0.0.1 -n 2 > nul
move /y "%s" "%s"
start "" "%s"
`, filepath.ToSlash(newExe), filepath.ToSlash(exePath), filepath.ToSlash(exePath))
		if err := os.WriteFile(bat, []byte(script), 0644); err != nil {
			return err
		}
		cmd := exec.Command("cmd", "/C", "start", "", bat)
		cmd.Dir = tmpDir
		if err := cmd.Start(); err != nil {
			return err
		}
		os.Exit(0)
		return nil
	}
	if err := os.Chmod(newExe, 0755); err != nil {
		return err
	}
	if err := os.Rename(newExe, exePath); err != nil {
		return err
	}
	if err := exec.Command(exePath).Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func unzipExecutable(zipPath, dir string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := filepath.Base(f.Name)
		if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(name), ".exe") {
			continue
		}
		outPath := filepath.Join(dir, name)
		if err := extractZipFile(f, outPath); err != nil {
			return "", err
		}
		return outPath, nil
	}
	return "", fmt.Errorf("executable not found in zip")
}

func extractZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}
