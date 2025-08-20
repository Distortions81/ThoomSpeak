package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
)

//go:embed data/versions.json
var versionsJSON []byte

//go:embed data/changelog/*
var changelogFS embed.FS

var (
	appVersion int
	clVersion  = baseVersion
	changelog  string
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
	b, err := changelogFS.ReadFile(fmt.Sprintf("data/changelog/%d.txt", latest.Version))
	if err != nil {
		log.Printf("read changelog: %v", err)
	} else {
		changelog = string(b)
	}
}
