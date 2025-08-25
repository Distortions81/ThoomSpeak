package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Expose the plugin API under both a short and a module-qualified path so
// Yaegi can resolve imports regardless of how the script refers to it.
var basePluginExports = interp.Exports{
	// Short path used by simple plugin scripts: import "gt"
	// Yaegi expects keys as "importPath/pkgName".
	"gt/gt": {
		"Logf":                  reflect.ValueOf(pluginLogf),
		"Console":               reflect.ValueOf(pluginConsole),
		"RegisterCommand":       reflect.ValueOf(pluginRegisterCommand),
		"RegisterFunc":          reflect.ValueOf(pluginRegisterFunc),
		"RunCommand":            reflect.ValueOf(pluginRunCommand),
		"EnqueueCommand":        reflect.ValueOf(pluginEnqueueCommand),
		"ClientVersion":         reflect.ValueOf(&clientVersion).Elem(),
		"PlayerName":            reflect.ValueOf(pluginPlayerName),
		"Players":               reflect.ValueOf(pluginPlayers),
		"Player":                reflect.ValueOf((*Player)(nil)),
		"Inventory":             reflect.ValueOf(pluginInventory),
		"InventoryItem":         reflect.ValueOf((*InventoryItem)(nil)),
		"ToggleEquip":           reflect.ValueOf(pluginToggleEquip),
		"Equip":                 reflect.ValueOf(pluginEquip),
		"Unequip":               reflect.ValueOf(pluginUnequip),
		"RegisterChatHandler":   reflect.ValueOf(pluginRegisterChatHandler),
		"InputText":             reflect.ValueOf(pluginInputText),
		"SetInputText":          reflect.ValueOf(pluginSetInputText),
		"PlayerStats":           reflect.ValueOf(pluginPlayerStats),
		"Stats":                 reflect.ValueOf((*Stats)(nil)),
		"RegisterPlayerHandler": reflect.ValueOf(pluginRegisterPlayerHandler),
	},
}

func exportsForPlugin(owner string) interp.Exports {
	ex := make(interp.Exports)
	for pkg, symbols := range basePluginExports {
		m := map[string]reflect.Value{}
		for k, v := range symbols {
			m[k] = v
		}
		m["AddHotkey"] = reflect.ValueOf(func(combo, command string) { pluginAddHotkey(owner, combo, command) })
		m["AddHotkeyFunc"] = reflect.ValueOf(func(combo, funcName string) { pluginAddHotkeyFunc(owner, combo, funcName) })
		m["RegisterFunc"] = reflect.ValueOf(func(name string, fn PluginFunc) { pluginRegisterFunc(owner, name, fn) })
		m["Hotkeys"] = reflect.ValueOf(func() []Hotkey { return pluginHotkeys(owner) })
		m["RemoveHotkey"] = reflect.ValueOf(func(combo string) { pluginRemoveHotkey(owner, combo) })
		ex[pkg] = m
	}
	return ex
}

//go:embed example_plugins/example_ponder.go example_plugins/README.txt
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
			if !e.IsDir() {
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
	msg := fmt.Sprintf("[plugin] "+format, args...)
	pluginConsole(msg)
	log.Print(msg)
}

func pluginConsole(msg string) {
	consoleMessage(msg)
}

func pluginAddHotkey(owner, combo, command string) {
	hk := Hotkey{Name: command, Combo: combo, Commands: []HotkeyCommand{{Command: command}}, Plugin: owner}
	hotkeysMu.Lock()
	for _, existing := range hotkeys {
		if existing.Plugin == owner && existing.Combo == combo {
			hotkeysMu.Unlock()
			return
		}
	}
	hotkeys = append(hotkeys, hk)
	hotkeysMu.Unlock()
	refreshHotkeysList()
	saveHotkeys()
	name := pluginDisplayNames[owner]
	if name == "" {
		name = owner
	}
	msg := fmt.Sprintf("[plugin:%s] hotkey added: %s -> %s", name, combo, command)
	consoleMessage(msg)
	log.Print(msg)
}

// pluginAddHotkeyFunc registers a hotkey that invokes a named plugin function
// registered via RegisterFunc.
func pluginAddHotkeyFunc(owner, combo, funcName string) {
	hk := Hotkey{Name: funcName, Combo: combo, Commands: []HotkeyCommand{{Function: funcName, Plugin: owner}}, Plugin: owner}
	hotkeysMu.Lock()
	for _, existing := range hotkeys {
		if existing.Plugin == owner && existing.Combo == combo {
			hotkeysMu.Unlock()
			return
		}
	}
	hotkeys = append(hotkeys, hk)
	hotkeysMu.Unlock()
	refreshHotkeysList()
	saveHotkeys()
	name := pluginDisplayNames[owner]
	if name == "" {
		name = owner
	}
	msg := fmt.Sprintf("[plugin:%s] hotkey added: %s -> plugin:%s", name, combo, funcName)
	consoleMessage(msg)
	log.Print(msg)
}

// Plugin command and function registries.
type PluginCommandHandler func(args string)
type PluginFunc func()

var (
	pluginCommands     = map[string]PluginCommandHandler{}
	pluginFuncs        = map[string]map[string]PluginFunc{}
	pluginMu           sync.RWMutex
	pluginNames        = map[string]bool{}
	pluginDisplayNames = map[string]string{}
	chatHandlers       []func(string)
	chatHandlersMu     sync.RWMutex
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
// hotkeys using AddHotkeyFunc or a command string "plugin:<name>".
func pluginRegisterFunc(owner, name string, fn PluginFunc) {
	if name == "" || fn == nil {
		return
	}
	key := strings.ToLower(name)
	pluginMu.Lock()
	if pluginFuncs[owner] == nil {
		pluginFuncs[owner] = map[string]PluginFunc{}
	}
	pluginFuncs[owner][key] = fn
	pluginMu.Unlock()
	disp := pluginDisplayNames[owner]
	if disp == "" {
		disp = owner
	}
	consoleMessage("[plugin:" + disp + "] function registered: " + key)
	log.Printf("[plugin] function registered: %s (%s)", key, owner)
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

func pluginPlayerName() string {
	return playerName
}

func pluginPlayers() []Player {
	ps := getPlayers()
	out := make([]Player, len(ps))
	copy(out, ps)
	return out
}

func pluginInventory() []InventoryItem {
	return getInventory()
}

func pluginToggleEquip(id uint16) {
	toggleInventoryEquip(id)
}

type Stats struct {
	HP, HPMax           int
	SP, SPMax           int
	Balance, BalanceMax int
}

func pluginPlayerStats() Stats {
	stateMu.Lock()
	s := Stats{
		HP:         state.hp,
		HPMax:      state.hpMax,
		SP:         state.sp,
		SPMax:      state.spMax,
		Balance:    state.balance,
		BalanceMax: state.balanceMax,
	}
	stateMu.Unlock()
	return s
}

func pluginInputText() string {
	inputMu.Lock()
	txt := string(inputText)
	inputMu.Unlock()
	return txt
}

func pluginSetInputText(text string) {
	inputMu.Lock()
	inputText = []rune(text)
	inputActive = true
	inputMu.Unlock()
}

func pluginEquip(id uint16) {
	items := getInventory()
	idx := -1
	for _, it := range items {
		if it.ID != id {
			continue
		}
		if it.Equipped {
			return
		}
		if idx < 0 {
			idx = it.IDIndex
		}
	}
	if idx < 0 {
		return
	}
	if idx >= 0 {
		pendingCommand = fmt.Sprintf("/equip %d %d", id, idx+1)
	} else {
		pendingCommand = fmt.Sprintf("/equip %d", id)
	}
	equipInventoryItem(id, idx, true)
}

func pluginUnequip(id uint16) {
	items := getInventory()
	equipped := false
	for _, it := range items {
		if it.ID == id && it.Equipped {
			equipped = true
			break
		}
	}
	if !equipped {
		return
	}
	pendingCommand = fmt.Sprintf("/unequip %d", id)
	equipInventoryItem(id, -1, false)
}

func pluginRegisterChatHandler(fn func(string)) {
	if fn == nil {
		return
	}
	chatHandlersMu.Lock()
	chatHandlers = append(chatHandlers, fn)
	chatHandlersMu.Unlock()
}

func pluginRegisterPlayerHandler(fn func(Player)) {
	if fn == nil {
		return
	}
	playerHandlersMu.Lock()
	playerHandlers = append(playerHandlers, fn)
	playerHandlersMu.Unlock()
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
		"encoding/json/json",
		"errors/errors",
		"fmt/fmt",
		"math/big/big",
		"math/math",
		"math/rand",
		"regexp/regexp",
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

	nameRE := regexp.MustCompile(`(?m)^\s*(?:var|const)\s+PluginName\s*=\s*"([^"]+)"`)
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
			match := nameRE.FindSubmatch(src)
			if len(match) < 2 {
				log.Printf("plugin %s missing PluginName", path)
				consoleMessage("[plugin] missing name: " + path)
				continue
			}
			name := strings.TrimSpace(string(match[1]))
			if name == "" {
				log.Printf("plugin %s empty PluginName", path)
				consoleMessage("[plugin] empty name: " + path)
				continue
			}
			lower := strings.ToLower(name)
			if pluginNames[lower] {
				log.Printf("plugin %s duplicate name %s", path, name)
				consoleMessage("[plugin] duplicate name: " + name)
				continue
			}
			pluginNames[lower] = true
			base := strings.TrimSuffix(e.Name(), ".go")
			owner := name + "_" + base
			pluginDisplayNames[owner] = name
			i := interp.New(interp.Options{})
			if len(restricted) > 0 {
				i.Use(restricted)
			}
			i.Use(exportsForPlugin(owner))
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
			consoleMessage("[plugin] loaded: " + name)
		}
	}
}
