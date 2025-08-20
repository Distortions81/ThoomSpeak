package main

import (
	"encoding/json"
	"log"

	_ "embed"
)

//go:embed data/versions.json
var versionsJSON []byte

var (
	appVersion int
	clVersion  = baseVersion
	changelog  string
)

type versionEntry struct {
	Version   int    `json:"version"`
	CLVersion int    `json:"cl_version"`
	Changelog string `json:"changelog"`
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
	changelog = latest.Changelog
}
