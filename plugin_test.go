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
	pluginEquip(200)
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
