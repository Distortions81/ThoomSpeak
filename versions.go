package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
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

type versionEntry struct {
	Version   int `json:"version"`
	CLVersion int `json:"cl_version"`
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
