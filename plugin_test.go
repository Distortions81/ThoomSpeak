package main

import (
	"sync"
	"testing"
	"time"
)

// Test that plugin equip command skips already equipped items.
func TestPluginEquipAlreadyEquipped(t *testing.T) {
	resetInventory()
	addInventoryItem(200, -1, "Shield", true)
	consoleLog = messageLog{max: maxMessages}
	pendingCommand = ""
	pluginEquip("tester", 200)
	msgs := getConsoleMessages()
	if len(msgs) == 0 || msgs[len(msgs)-1] != "Shield already equipped, skipping" {
		t.Fatalf("unexpected console messages %v", msgs)
	}
	if pendingCommand != "" {
		t.Fatalf("pending command queued: %q", pendingCommand)
	}
}

// getQueuedCommands returns the pending command followed by any queued commands.
func getQueuedCommands() []string {
	cmds := append([]string{}, commandQueue...)
	if pendingCommand != "" {
		cmds = append([]string{pendingCommand}, cmds...)
	}
	return cmds
}

// Test registering and running a mixed-case command and ensuring disabled plugins
// cannot run commands.
func TestPluginRegisterAndDisableCommand(t *testing.T) {
	// Reset shared state.
	pluginMu = sync.RWMutex{}
	pluginCommands = map[string]PluginCommandHandler{}
	pluginCommandOwners = map[string]string{}
	pluginDisabled = map[string]bool{}
	pluginSendHistory = map[string][]time.Time{}
	consoleLog = messageLog{max: maxMessages}
	commandQueue = nil
	pendingCommand = ""

	owner := "tester"
	pluginRegisterCommand(owner, "MiXeD", func(args string) {
		consoleMessage("handled " + args)
	})

	if _, ok := pluginCommands["mixed"]; !ok {
		t.Fatalf("command not registered: %v", pluginCommands)
	}

	handler := pluginCommands["mixed"]
	handler("input")

	msgs := getConsoleMessages()
	if len(msgs) == 0 || msgs[len(msgs)-1] != "handled input" {
		t.Fatalf("unexpected console messages %v", msgs)
	}

	// Disable plugin and ensure pluginRunCommand does nothing.
	pluginDisabled[owner] = true
	consoleLog = messageLog{max: maxMessages}
	commandQueue = nil
	pendingCommand = ""

	pluginRunCommand(owner, "/wave")

	if msgs := getConsoleMessages(); len(msgs) != 0 {
		t.Fatalf("console output when plugin disabled: %v", msgs)
	}
	if cmds := getQueuedCommands(); len(cmds) != 0 {
		t.Fatalf("commands queued when plugin disabled: %v", cmds)
	}
}

// Test that registering a command twice logs a conflict and keeps the original handler.
func TestPluginRegisterCommandConflict(t *testing.T) {
	// Reset shared state.
	pluginMu = sync.RWMutex{}
	pluginCommands = map[string]PluginCommandHandler{}
	pluginCommandOwners = map[string]string{}
	consoleLog = messageLog{max: maxMessages}

	owner1 := "one"
	owner2 := "two"

	ran := false
	pluginRegisterCommand(owner1, "cmd", func(args string) { ran = true })

	// Clear console messages before second registration attempt.
	consoleLog = messageLog{max: maxMessages}

	pluginRegisterCommand(owner2, "cmd", func(args string) {})

	msgs := getConsoleMessages()
	want := "[plugin] command conflict: /cmd already registered"
	if len(msgs) == 0 || msgs[len(msgs)-1] != want {
		t.Fatalf("unexpected console messages %v", msgs)
	}

	// Ensure original handler remains registered.
	if h, ok := pluginCommands["cmd"]; ok {
		h("")
	}
	if !ran {
		t.Fatalf("original handler overwritten")
	}
}

// Test that trigger handlers registered by plugins receive messages.
func TestPluginTriggers(t *testing.T) {
	pluginTriggers = map[string][]triggerHandler{}
	pluginConsoleTriggers = map[string][]triggerHandler{}
	triggerHandlersMu = sync.RWMutex{}
	pluginDisabled = map[string]bool{}
	var got string
	var wg sync.WaitGroup
	wg.Add(1)
	pluginRegisterTriggers("test", "", []string{"hello"}, func(msg string) {
		got = msg
		wg.Done()
	})
	runChatTriggers("say hello")
	wg.Wait()
	if got != "say hello" {
		t.Fatalf("handler did not run, got %q", got)
	}
}

// Test that disabling a plugin removes any trigger handlers it registered.
func TestPluginRemoveTriggersOnDisable(t *testing.T) {
	pluginTriggers = map[string][]triggerHandler{}
	pluginConsoleTriggers = map[string][]triggerHandler{}
	triggerHandlersMu = sync.RWMutex{}
	pluginInputHandlers = nil
	inputHandlersMu = sync.RWMutex{}
	pluginMu = sync.RWMutex{}
	pluginDisabled = map[string]bool{}
	pluginDisplayNames = map[string]string{}
	pluginTerminators = map[string]func(){}
	pluginCommandOwners = map[string]string{}
	pluginCommands = map[string]PluginCommandHandler{}
	pluginSendHistory = map[string][]time.Time{}

	ran := false
	pluginRegisterTriggers("plug", "", []string{"hi"}, func(msg string) { ran = true })
	pluginRegisterConsoleTriggers("plug", []string{"hi"}, func(msg string) { ran = true })
	disablePlugin("plug", "test")
	runChatTriggers("hi there")
	runConsoleTriggers("hi there")
	if ran {
		t.Fatalf("trigger ran after plugin disabled")
	}
}
