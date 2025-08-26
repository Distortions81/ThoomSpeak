package main

import (
	"sync"
	"testing"
	"time"
)

// Test that a single macro expands input text and matches case-insensitively.
func TestPluginAddMacroExpandsInput(t *testing.T) {
	// Reset shared state.
	macroMu = sync.RWMutex{}
	macroMaps = map[string]map[string]string{}
	inputHandlersMu = sync.RWMutex{}
	inputHandlers = nil

	pluginAddMacro("tester", "pp", "/ponder ")

	if got, want := runInputHandlers("pp"), "/ponder "; got != want {
		t.Fatalf("bare macro failed: got %q, want %q", got, want)
	}

	if got, want := runInputHandlers("pp hello"), "/ponder hello"; got != want {
		t.Fatalf("lowercase macro with space failed: got %q, want %q", got, want)
	}

	if got, want := runInputHandlers("PP Hello"), "/ponder Hello"; got != want {
		t.Fatalf("uppercase macro failed: got %q, want %q", got, want)
	}

	if got, want := runInputHandlers("pphi"), "pphi"; got != want {
		t.Fatalf("macro should not expand within word: got %q, want %q", got, want)
	}
}

// Test that multiple macros can be registered at once.
func TestPluginAddMacros(t *testing.T) {
	macroMu = sync.RWMutex{}
	macroMaps = map[string]map[string]string{}
	inputHandlersMu = sync.RWMutex{}
	inputHandlers = nil

	pluginAddMacros("bulk", map[string]string{"pp": "/ponder ", "hi": "/hello "})

	if got, want := runInputHandlers("pp there"), "/ponder there"; got != want {
		t.Fatalf("pp macro failed: got %q, want %q", got, want)
	}
	if got, want := runInputHandlers("hi you"), "/hello you"; got != want {
		t.Fatalf("hi macro failed: got %q, want %q", got, want)
	}
}

// Test that AutoReply triggers the specified command when the message starts with the trigger.
func TestPluginAutoReplyRunsCommand(t *testing.T) {
	macroMu = sync.RWMutex{}
	macroMaps = map[string]map[string]string{}
	inputHandlersMu = sync.RWMutex{}
	inputHandlers = nil
	chatHandlersMu = sync.RWMutex{}
	pluginChatHandlers = map[string][]func(string){}
	consoleLog = messageLog{max: maxMessages}
	commandQueue = nil
	pendingCommand = ""
	pluginSendHistory = map[string][]time.Time{}

	pluginAutoReply("bot", "hi", "/wave")

	chatHandlersMu.RLock()
	handlers := append([]func(string){}, pluginChatHandlers["bot"]...)
	chatHandlersMu.RUnlock()
	if len(handlers) != 1 {
		t.Fatalf("unexpected handler count: %d", len(handlers))
	}

	handlers[0]("Hi there")

	msgs := getConsoleMessages()
	if len(msgs) == 0 || msgs[len(msgs)-1] != "> /wave" {
		t.Fatalf("unexpected console messages %v", msgs)
	}
	if pendingCommand != "/wave" {
		t.Fatalf("pending command %q, want %q", pendingCommand, "/wave")
	}
}

// Test that disabling a plugin removes any macros it registered.
func TestPluginRemoveMacrosOnDisable(t *testing.T) {
	// Reset shared state.
	macroMu = sync.RWMutex{}
	macroMaps = map[string]map[string]string{}
	inputHandlersMu = sync.RWMutex{}
	inputHandlers = nil
	pluginMu = sync.RWMutex{}
	pluginDisabled = map[string]bool{}
	pluginDisplayNames = map[string]string{}
	pluginTerminators = map[string]func(){}
	pluginCommandOwners = map[string]string{}
	pluginCommands = map[string]PluginCommandHandler{}
	pluginSendHistory = map[string][]time.Time{}
	hotkeysMu = sync.RWMutex{}
	hotkeys = nil
	pluginHotkeyEnabled = map[string]map[string]bool{}
	consoleLog = messageLog{max: maxMessages}

	owner := "plug"
	pluginAddMacro(owner, "pp", "/ponder ")
	if got, want := runInputHandlers("pp hello"), "/ponder hello"; got != want {
		t.Fatalf("macro not added: got %q, want %q", got, want)
	}

	disablePlugin(owner, "testing")

	if got, want := runInputHandlers("pp hello"), "pp hello"; got != want {
		t.Fatalf("macro not removed: got %q, want %q", got, want)
	}
}
