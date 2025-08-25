package main

import (
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
    "embed"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Expose the plugin API under both a short and a module-qualified path so
// Yaegi can resolve imports regardless of how the script refers to it.
var pluginExports = interp.Exports{
    // Short path used by simple plugin scripts: import "pluginapi"
    // Yaegi expects keys as "importPath/pkgName".
    "pluginapi/pluginapi": {
        "Logf":          reflect.ValueOf(pluginLogf),
        "AddHotkey":     reflect.ValueOf(pluginAddHotkey),
        "ClientVersion": reflect.ValueOf(&clientVersion).Elem(),
    },
    // Module-qualified path alternative: import "gothoom/pluginapi"
    "gothoom/pluginapi/pluginapi": {
        "Logf":          reflect.ValueOf(pluginLogf),
        "AddHotkey":     reflect.ValueOf(pluginAddHotkey),
        "ClientVersion": reflect.ValueOf(&clientVersion).Elem(),
    },
}

//go:embed embedded_plugins/*
var embeddedPlugins embed.FS

func userPluginsDir() string {
    return filepath.Join(dataDirPath, "plugins")
}

// ensureDefaultPlugins creates the user plugins directory and populates it
// with an example plugin when it is empty.
func ensureDefaultPlugins() {
    dir := userPluginsDir()
    if err := os.MkdirAll(dir, 0o755); err != nil {
        log.Printf("create plugins dir: %v", err)
        return
    }
    // Check if directory already has any .go plugin files
    hasGo := false
    if entries, err := os.ReadDir(dir); err == nil {
        for _, e := range entries {
            if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
                hasGo = true
                break
            }
        }
    }
    if hasGo {
        return
    }
    // Write example plugin files
    files := []string{
        "embedded_plugins/example_ponder.go",
        "embedded_plugins/README.txt",
    }
    for _, src := range files {
        data, err := embeddedPlugins.ReadFile(src)
        if err != nil {
            log.Printf("read embedded %s: %v", src, err)
            continue
        }
        base := filepath.Base(src)
        dst := filepath.Join(dir, base)
        if err := os.WriteFile(dst, data, 0o644); err != nil {
            log.Printf("write %s: %v", dst, err)
            continue
        }
    }
}

func pluginLogf(format string, args ...interface{}) {
	log.Printf("[plugin] "+format, args...)
}

func pluginAddHotkey(combo, command string) {
    hk := Hotkey{Combo: combo, Commands: []HotkeyCommand{{Command: command}}}
    hotkeys = append(hotkeys, hk)
    refreshHotkeysList()
    saveHotkeys()
    msg := "[plugin] hotkey added: " + combo + " -> " + command
    consoleMessage(msg)
    log.Printf(msg)
}

func loadPlugins() {
    // Ensure user plugins directory and example exist
    ensureDefaultPlugins()

    pluginDirs := []string{
        userPluginsDir(), // per-user/app data directory
        "plugins",        // legacy: relative to current working directory
    }
    for _, dir := range pluginDirs {
        entries, err := os.ReadDir(dir)
        if err != nil {
            if !os.IsNotExist(err) {
                log.Printf("read plugin dir %s: %v", dir, err)
            }
            continue
        }
        for _, e := range entries {
            if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
                continue
            }
            path := filepath.Join(dir, e.Name())
            src, err := os.ReadFile(path)
            if err != nil {
                log.Printf("read plugin %s: %v", path, err)
                continue
            }
            i := interp.New(interp.Options{})
            i.Use(stdlib.Symbols)
            i.Use(pluginExports)
            if _, err := i.Eval(string(src)); err != nil {
                log.Printf("plugin %s: %v", e.Name(), err)
                consoleMessage("[plugin] load error for " + path + ": " + err.Error())
                continue
            }
            if v, err := i.Eval("Init"); err == nil {
                if fn, ok := v.Interface().(func()); ok {
                    fn()
                }
            }
            log.Printf("loaded plugin %s", path)
            consoleMessage("[plugin] loaded: " + path)
        }
    }
}
