package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Expose the plugin API under both a short and a module-qualified path so
// Yaegi can resolve imports regardless of how the script refers to it.
var pluginExports = interp.Exports{
	// Short path used by simple plugin scripts: import "gt"
	// Yaegi expects keys as "importPath/pkgName".
	"gt/gt": {
		"Logf":            reflect.ValueOf(pluginLogf),
		"AddHotkey":       reflect.ValueOf(pluginAddHotkey),
		"AddHotkeyFunc":   reflect.ValueOf(pluginAddHotkeyFunc),
		"RegisterCommand": reflect.ValueOf(pluginRegisterCommand),
		"RegisterFunc":    reflect.ValueOf(pluginRegisterFunc),
		"RunCommand":      reflect.ValueOf(pluginRunCommand),
		"EnqueueCommand":  reflect.ValueOf(pluginEnqueueCommand),
		"ClientVersion":   reflect.ValueOf(&clientVersion).Elem(),
	},
}

//go:embed plugins/example_ponder.go plugins/README.txt
var pluginExamples embed.FS

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
		"plugins/example_ponder.go",
		"plugins/README.txt",
	}
	for _, src := range files {
		data, err := pluginExamples.ReadFile(src)
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
	hk := Hotkey{Name: command, Combo: combo, Commands: []HotkeyCommand{{Command: command}}}
	hotkeysMu.Lock()
	hotkeys = append(hotkeys, hk)
	hotkeysMu.Unlock()
	refreshHotkeysList()
	saveHotkeys()
	msg := "[plugin] hotkey added: " + combo + " -> " + command
	consoleMessage(msg)
	log.Print(msg)
}

// pluginAddHotkeyFunc registers a hotkey that invokes a named plugin function
// registered via RegisterFunc.
func pluginAddHotkeyFunc(combo, funcName string) {
	hk := Hotkey{Name: funcName, Combo: combo, Commands: []HotkeyCommand{{Command: "plugin:" + funcName}}}
	hotkeysMu.Lock()
	hotkeys = append(hotkeys, hk)
	hotkeysMu.Unlock()
	refreshHotkeysList()
	saveHotkeys()
	msg := "[plugin] hotkey added: " + combo + " -> plugin:" + funcName
	consoleMessage(msg)
	log.Print(msg)
}

// Plugin command and function registries.
type PluginCommandHandler func(args string)
type PluginFunc func()

var (
	pluginCommands = map[string]PluginCommandHandler{}
	pluginFuncs    = map[string]PluginFunc{}
	pluginMu       sync.RWMutex
)

// pluginRegisterCommand lets plugins handle a local slash command like
// "/example". The name should be without the leading slash and will be
// matched case-insensitively.
func pluginRegisterCommand(name string, handler PluginCommandHandler) {
	if name == "" || handler == nil {
		return
	}
	key := strings.ToLower(strings.TrimPrefix(name, "/"))
	pluginMu.Lock()
	pluginCommands[key] = handler
	pluginMu.Unlock()
	consoleMessage("[plugin] command registered: /" + key)
	log.Printf("[plugin] command registered: /%s", key)
}

// pluginRegisterFunc registers a named function that can be called from
// hotkeys using the special command string "plugin:<name>".
func pluginRegisterFunc(name string, fn PluginFunc) {
	if name == "" || fn == nil {
		return
	}
	key := strings.ToLower(name)
	pluginMu.Lock()
	pluginFuncs[key] = fn
	pluginMu.Unlock()
	consoleMessage("[plugin] function registered: " + key)
	log.Printf("[plugin] function registered: %s", key)
}

// pluginRunCommand echoes and enqueues a command for immediate sending.
func pluginRunCommand(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}
	consoleMessage("> " + cmd)
	enqueueCommand(cmd)
	nextCommand()
}

// pluginEnqueueCommand enqueues a command to be sent on the next tick without echoing.
func pluginEnqueueCommand(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}
	enqueueCommand(cmd)
}

func loadPlugins() {
	// Ensure user plugins directory and example exist
	ensureDefaultPlugins()

	pluginDirs := []string{
		userPluginsDir(), // per-user/app data directory
		"plugins",        // legacy: relative to current working directory
	}
	// Build restricted stdlib symbol map containing safe stdlib packages
	allowedPkgs := []string{
		"bytes/bytes",
		"fmt/fmt",
		"math/math",
		"math/rand",
		"sort/sort",
		"strconv/strconv",
		"strings/strings",
		"time/time",
		"unicode/utf8",
	}
	restricted := interp.Exports{}
	for _, key := range allowedPkgs {
		if syms, ok := stdlib.Symbols[key]; ok {
			restricted[key] = syms
		}
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
			// IMPORTANT: only allow a restricted subset of stdlib (possibly empty)
			if len(restricted) > 0 {
				i.Use(restricted)
			}
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
