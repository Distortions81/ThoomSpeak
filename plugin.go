package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

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
		"ShowNotification":      reflect.ValueOf(pluginShowNotification),
		"ClientVersion":         reflect.ValueOf(&clientVersion).Elem(),
		"PlayerName":            reflect.ValueOf(pluginPlayerName),
		"Players":               reflect.ValueOf(pluginPlayers),
		"Player":                reflect.ValueOf((*Player)(nil)),
		"Inventory":             reflect.ValueOf(pluginInventory),
		"InventoryItem":         reflect.ValueOf((*InventoryItem)(nil)),
		"ToggleEquip":           reflect.ValueOf(pluginToggleEquip),
		"Equip":                 reflect.ValueOf(pluginEquip),
		"Unequip":               reflect.ValueOf(pluginUnequip),
		"PlaySound":             reflect.ValueOf(pluginPlaySound),
		"RegisterInputHandler":  reflect.ValueOf(pluginRegisterInputHandler),
		"RegisterChatHandler":   reflect.ValueOf(pluginRegisterChatHandler),
		"InputText":             reflect.ValueOf(pluginInputText),
		"SetInputText":          reflect.ValueOf(pluginSetInputText),
		"PlayerStats":           reflect.ValueOf(pluginPlayerStats),
		"Stats":                 reflect.ValueOf((*Stats)(nil)),
		"RegisterPlayerHandler": reflect.ValueOf(pluginRegisterPlayerHandler),
		"KeyPressed":            reflect.ValueOf(pluginKeyPressed),
		"KeyJustPressed":        reflect.ValueOf(pluginKeyJustPressed),
		"MousePressed":          reflect.ValueOf(pluginMousePressed),
		"MouseJustPressed":      reflect.ValueOf(pluginMouseJustPressed),
		"MouseWheel":            reflect.ValueOf(pluginMouseWheel),
		"LastClick":             reflect.ValueOf(pluginLastClick),
		"ClickInfo":             reflect.ValueOf((*ClickInfo)(nil)),
		"Mobile":                reflect.ValueOf((*Mobile)(nil)),
		"EquippedItems":         reflect.ValueOf(pluginEquippedItems),
		"HasItem":               reflect.ValueOf(pluginHasItem),
		"FrameNumber":           reflect.ValueOf(pluginFrameNumber),
		"IgnoreCase":            reflect.ValueOf(pluginIgnoreCase),
		"StartsWith":            reflect.ValueOf(pluginStartsWith),
		"EndsWith":              reflect.ValueOf(pluginEndsWith),
		"Includes":              reflect.ValueOf(pluginIncludes),
		"Lower":                 reflect.ValueOf(pluginLower),
		"Upper":                 reflect.ValueOf(pluginUpper),
		"Trim":                  reflect.ValueOf(pluginTrim),
		"TrimStart":             reflect.ValueOf(pluginTrimStart),
		"TrimEnd":               reflect.ValueOf(pluginTrimEnd),
		"Words":                 reflect.ValueOf(pluginWords),
		"Join":                  reflect.ValueOf(pluginJoin),
		"Replace":               reflect.ValueOf(pluginReplace),
		"Split":                 reflect.ValueOf(pluginSplit),
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
		m["Hotkeys"] = reflect.ValueOf(func() []Hotkey { return pluginHotkeys(owner) })
		m["RemoveHotkey"] = reflect.ValueOf(func(combo string) { pluginRemoveHotkey(owner, combo) })
		m["RegisterCommand"] = reflect.ValueOf(func(name string, handler PluginCommandHandler) {
			pluginRegisterCommand(owner, name, handler)
		})
		m["AddMacro"] = reflect.ValueOf(func(short, full string) { pluginAddMacro(owner, short, full) })
		m["AddMacros"] = reflect.ValueOf(func(macros map[string]string) { pluginAddMacros(owner, macros) })
		m["AutoReply"] = reflect.ValueOf(func(trigger, cmd string) { pluginAutoReply(owner, trigger, cmd) })
		m["RunCommand"] = reflect.ValueOf(func(cmd string) { pluginRunCommand(owner, cmd) })
		m["EnqueueCommand"] = reflect.ValueOf(func(cmd string) { pluginEnqueueCommand(owner, cmd) })
		ex[pkg] = m
	}
	return ex
}

//go:embed example_plugins
var pluginExamples embed.FS

func userPluginsDir() string {
	return filepath.Join(dataDirPath, "plugins")
}

// ensureExamplePlugins creates the example_plugins directory next to the
// executable and populates it with the embedded example plugin files if it is
// missing.
func ensureExamplePlugins() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	dir := filepath.Join(filepath.Dir(exe), "example_plugins")
	if _, err := os.Stat(dir); err == nil {
		return
	} else if !os.IsNotExist(err) {
		log.Printf("check example_plugins dir: %v", err)
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("create example_plugins dir: %v", err)
		return
	}
	entries, err := pluginExamples.ReadDir("example_plugins")
	if err != nil {
		log.Printf("read embedded example plugins: %v", err)
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := pluginExamples.ReadFile(path.Join("example_plugins", e.Name()))
		if err != nil {
			log.Printf("read embedded %s: %v", e.Name(), err)
			continue
		}
		dst := filepath.Join(dir, e.Name())
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			log.Printf("write %s: %v", dst, err)
		}
	}
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
		"default_macros.go",
		"README.txt",
		"numpad_poser.go",
	}
	for _, src := range files {
		sPath := path.Join("example_plugins", src)
		data, err := pluginExamples.ReadFile(sPath)
		if err != nil {
			log.Printf("read embedded %s: %v", sPath, err)
			continue
		}
		base := filepath.Base(sPath)
		dst := filepath.Join(dir, base)
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			log.Printf("write %s: %v", dst, err)
			continue
		}
	}
}

var pluginAllowedPkgs = []string{
	"bytes/bytes",
	"encoding/json/json",
	"errors/errors",
	"fmt/fmt",
	"math/big/big",
	"math/math",
	"math/rand/rand",
	"regexp/regexp",
	"sort/sort",
	"strconv/strconv",
	"strings/strings",
	"time/time",
	"unicode/utf8/utf8",
}

const pluginGoroutineLimit = 1000

func init() {
	go pluginGoroutineWatchdog()
}

func pluginGoroutineWatchdog() {
	for {
		if runtime.NumGoroutine() > pluginGoroutineLimit {
			log.Printf("[plugin] goroutine limit exceeded; stopping all plugins")
			consoleMessage("[plugin] goroutine limit exceeded; stopping plugins")
			stopAllPlugins()
			return
		}
		time.Sleep(time.Second)
	}
}

func restrictedStdlib() interp.Exports {
	restricted := interp.Exports{}
	for _, key := range pluginAllowedPkgs {
		if syms, ok := stdlib.Symbols[key]; ok {
			restricted[key] = syms
		}
	}
	return restricted
}

func pluginLogf(format string, args ...interface{}) {
	msg := fmt.Sprintf("[plugin] "+format, args...)
	pluginConsole(msg)
	log.Print(msg)
}

func pluginConsole(msg string) {
	consoleMessage(msg)
	if gs.pluginOutputDebug {
		chatMessage(msg)
	}
}

func pluginShowNotification(msg string) {
	showNotification(msg)
}

func pluginAddHotkey(owner, combo, command string) {
	if pluginDisabled[owner] {
		return
	}
	hk := Hotkey{Name: command, Combo: combo, Commands: []HotkeyCommand{{Command: command}}, Plugin: owner, Disabled: true}
	if m := pluginHotkeyEnabled[owner]; m != nil {
		if m[combo] {
			hk.Disabled = false
		}
	}
	hotkeysMu.Lock()
	for _, existing := range hotkeys {
		if existing.Plugin == owner && existing.Combo == combo {
			hotkeysMu.Unlock()
			return
		}
	}
	hotkeys = append(hotkeys, hk)
	hotkeysMu.Unlock()
	if hk.Disabled {
		if m := pluginHotkeyEnabled[owner]; m != nil {
			delete(m, combo)
			if len(m) == 0 {
				delete(pluginHotkeyEnabled, owner)
			}
		}
	} else {
		m := pluginHotkeyEnabled[owner]
		if m == nil {
			m = map[string]bool{}
			pluginHotkeyEnabled[owner] = m
		}
		m[combo] = true
	}
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

// Plugin command registries.
type PluginCommandHandler func(args string)

var (
	pluginCommands      = map[string]PluginCommandHandler{}
	pluginMu            sync.RWMutex
	pluginNames         = map[string]bool{}
	pluginDisplayNames  = map[string]string{}
	pluginDisabled      = map[string]bool{}
	pluginPaths         = map[string]string{}
	pluginTerminators   = map[string]func(){}
	chatHandlers        []func(string)
	chatHandlersMu      sync.RWMutex
	inputHandlers       []func(string) string
	inputHandlersMu     sync.RWMutex
	pluginCommandOwners = map[string]string{}
	pluginSendHistory   = map[string][]time.Time{}
	pluginModTime       time.Time
	pluginModCheck      time.Time
)

// pluginRegisterCommand lets plugins handle a local slash command like
// "/example". The name should be without the leading slash and will be
// matched case-insensitively.
func pluginRegisterCommand(owner, name string, handler PluginCommandHandler) {
	if name == "" || handler == nil {
		return
	}
	if pluginDisabled[owner] {
		return
	}
	key := strings.ToLower(strings.TrimPrefix(name, "/"))
	pluginMu.Lock()
	pluginCommands[key] = handler
	pluginCommandOwners[key] = owner
	pluginMu.Unlock()
	consoleMessage("[plugin] command registered: /" + key)
	log.Printf("[plugin] command registered: /%s", key)
}

// pluginRunCommand echoes and enqueues a command for immediate sending.
func pluginRunCommand(owner, cmd string) {
	if pluginDisabled[owner] {
		return
	}
	if recordPluginSend(owner) {
		return
	}
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}
	consoleMessage("> " + cmd)
	enqueueCommand(cmd)
	nextCommand()
}

// pluginEnqueueCommand enqueues a command to be sent on the next tick without echoing.
func pluginEnqueueCommand(owner, cmd string) {
	if pluginDisabled[owner] {
		return
	}
	if recordPluginSend(owner) {
		return
	}
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}
	enqueueCommand(cmd)
}

func loadPluginSource(owner, name, path string, src []byte, restricted interp.Exports) {
	i := interp.New(interp.Options{})
	if len(restricted) > 0 {
		i.Use(restricted)
	}
	i.Use(exportsForPlugin(owner))
	pluginMu.Lock()
	pluginDisabled[owner] = false
	pluginMu.Unlock()
	if _, err := i.Eval(string(src)); err != nil {
		log.Printf("plugin %s: %v", path, err)
		consoleMessage("[plugin] load error for " + path + ": " + err.Error())
		disablePlugin(owner, "load error")
		return
	}
	if v, err := i.Eval("Terminate"); err == nil {
		if fn, ok := v.Interface().(func()); ok {
			pluginMu.Lock()
			pluginTerminators[owner] = fn
			pluginMu.Unlock()
		}
	}
	if v, err := i.Eval("Init"); err == nil {
		if fn, ok := v.Interface().(func()); ok {
			go fn()
		}
	}
	log.Printf("loaded plugin %s", path)
	consoleMessage("[plugin] loaded: " + name)
}

func enablePlugin(owner string) {
	pluginMu.RLock()
	path := pluginPaths[owner]
	name := pluginDisplayNames[owner]
	pluginMu.RUnlock()
	if path == "" {
		return
	}
	src, err := os.ReadFile(path)
	if err != nil {
		log.Printf("read plugin %s: %v", path, err)
		consoleMessage("[plugin] read error for " + path + ": " + err.Error())
		return
	}
	loadPluginSource(owner, name, path, src, restrictedStdlib())
	settingsDirty = true
	refreshPluginsWindow()
}

func recordPluginSend(owner string) bool {
	if !gs.PluginSpamKill {
		return false
	}
	now := time.Now()
	cutoff := now.Add(-5 * time.Second)
	pluginMu.Lock()
	times := pluginSendHistory[owner]
	n := 0
	for _, t := range times {
		if t.After(cutoff) {
			times[n] = t
			n++
		}
	}
	times = times[:n]
	times = append(times, now)
	pluginSendHistory[owner] = times
	count := len(times)
	pluginMu.Unlock()
	if count > 30 {
		disablePlugin(owner, "sent too many lines")
		return true
	}
	return false
}

func disablePlugin(owner, reason string) {
	pluginMu.Lock()
	pluginDisabled[owner] = true
	term := pluginTerminators[owner]
	delete(pluginTerminators, owner)
	pluginMu.Unlock()
	if term != nil {
		go term()
	}
	for _, hk := range pluginHotkeys(owner) {
		pluginRemoveHotkey(owner, hk.Combo)
	}
	pluginRemoveMacros(owner)
	pluginMu.Lock()
	for cmd, o := range pluginCommandOwners {
		if o == owner {
			delete(pluginCommands, cmd)
			delete(pluginCommandOwners, cmd)
		}
	}
	delete(pluginSendHistory, owner)
	disp := pluginDisplayNames[owner]
	pluginMu.Unlock()
	if disp == "" {
		disp = owner
	}
	consoleMessage("[plugin:" + disp + "] stopped: " + reason)
	settingsDirty = true
	refreshPluginsWindow()
}

func stopAllPlugins() {
	pluginMu.RLock()
	owners := make([]string, 0, len(pluginDisplayNames))
	for o := range pluginDisplayNames {
		if !pluginDisabled[o] {
			owners = append(owners, o)
		}
	}
	pluginMu.RUnlock()
	for _, o := range owners {
		disablePlugin(o, "stopped by user")
	}
	if len(owners) > 0 {
		commandQueue = nil
		pendingCommand = ""
		consoleMessage("[plugin] all plugins stopped")
	}
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
			name := it.Name
			if name == "" {
				name = fmt.Sprintf("%d", id)
			}
			consoleMessage(name + " already equipped, skipping")
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

func pluginRegisterInputHandler(fn func(string) string) {
	if fn == nil {
		return
	}
	inputHandlersMu.Lock()
	inputHandlers = append(inputHandlers, fn)
	inputHandlersMu.Unlock()
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

func runInputHandlers(txt string) string {
	inputHandlersMu.RLock()
	handlers := append([]func(string) string{}, inputHandlers...)
	inputHandlersMu.RUnlock()
	for _, h := range handlers {
		if h != nil {
			txt = h(txt)
		}
	}
	return txt
}

func pluginPlaySound(ids []uint16) {
	playSound(ids)
}

func pluginCommandsFor(owner string) []string {
	pluginMu.RLock()
	defer pluginMu.RUnlock()
	var list []string
	for cmd, o := range pluginCommandOwners {
		if o == owner {
			list = append(list, cmd)
		}
	}
	return list
}

func pluginSource(owner string) string {
	pluginMu.RLock()
	path := pluginPaths[owner]
	pluginMu.RUnlock()
	if path == "" {
		return "plugin source not found"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("error reading %s: %v", path, err)
	}
	return string(data)
}

func refreshPluginMod() {
	dirs := []string{userPluginsDir(), "plugins"}
	latest := time.Time{}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
				continue
			}
			if info, err := e.Info(); err == nil {
				if info.ModTime().After(latest) {
					latest = info.ModTime()
				}
			}
		}
	}
	pluginModTime = latest
}

func rescanPlugins() {
	pluginDirs := []string{
		userPluginsDir(),
		"plugins",
	}
	nameRE := regexp.MustCompile(`(?m)^\s*(?:var|const)\s+PluginName\s*=\s*"([^"]+)"`)
	newDisplay := map[string]string{}
	newPaths := map[string]string{}
	seenNames := map[string]bool{}
	for _, dir := range pluginDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
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
				continue
			}
			name := strings.TrimSpace(string(match[1]))
			if name == "" {
				continue
			}
			lower := strings.ToLower(name)
			if seenNames[lower] {
				continue
			}
			seenNames[lower] = true
			base := strings.TrimSuffix(e.Name(), ".go")
			owner := name + "_" + base
			newDisplay[owner] = name
			newPaths[owner] = path
		}
	}

	pluginMu.RLock()
	oldDisabled := make(map[string]bool, len(pluginDisabled))
	for o, d := range pluginDisabled {
		oldDisabled[o] = d
	}
	oldOwners := make(map[string]struct{}, len(pluginDisplayNames))
	for o := range pluginDisplayNames {
		oldOwners[o] = struct{}{}
	}
	pluginMu.RUnlock()

	for o := range oldOwners {
		if _, ok := newDisplay[o]; !ok {
			disablePlugin(o, "removed")
		}
	}

	pluginMu.Lock()
	pluginDisplayNames = newDisplay
	pluginPaths = newPaths
	pluginDisabled = make(map[string]bool, len(newDisplay))
	for o := range newDisplay {
		if d, ok := oldDisabled[o]; ok {
			pluginDisabled[o] = d
		} else {
			pluginDisabled[o] = true
		}
	}
	pluginNames = make(map[string]bool, len(newDisplay))
	for _, n := range newDisplay {
		pluginNames[strings.ToLower(n)] = true
	}
	pluginMu.Unlock()

	refreshPluginsWindow()
	settingsDirty = true
}

func checkPluginMods() {
	if time.Since(pluginModCheck) < 500*time.Millisecond {
		return
	}
	pluginModCheck = time.Now()
	old := pluginModTime
	refreshPluginMod()
	if pluginModTime.After(old) {
		rescanPlugins()
	}
}

func loadPlugins() {
	ensureExamplePlugins()
	ensureDefaultPlugins()

	pluginDirs := []string{
		userPluginsDir(),
		"plugins",
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
			disabled := true
			if gs.EnabledPlugins != nil {
				if en, ok := gs.EnabledPlugins[owner]; ok {
					disabled = !en
				}
			}
			pluginMu.Lock()
			pluginDisplayNames[owner] = name
			pluginPaths[owner] = path
			pluginDisabled[owner] = disabled
			pluginMu.Unlock()
			if !disabled {
				loadPluginSource(owner, name, path, src, restrictedStdlib())
			}
		}
	}
	hotkeysMu.Lock()
	for i := range hotkeys {
		if hotkeys[i].Plugin != "" {
			hotkeys[i].Disabled = true
		}
	}
	hotkeysMu.Unlock()
	refreshHotkeysList()
	refreshPluginsWindow()
	refreshPluginMod()
}
