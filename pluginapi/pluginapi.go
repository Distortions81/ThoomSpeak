// Package pluginapi provides a tiny, editor-only stub for the
// interpreted plugin API exposed by the client at runtime via Yaegi.
//
// This package exists solely to satisfy linters and IDEs for files in
// the plugins/ directory (e.g., healer.go) that do `import "pluginapi"`.
// The real implementations are injected into the Yaegi interpreter at
// runtime; these no-op stubs are never called by the compiled client.
package pluginapi

// ClientVersion mirrors the client version value exported to plugins.
var ClientVersion int

// Logf is a no-op printf-style logger for editor/linter happiness.
func Logf(format string, args ...interface{}) {}

// AddHotkey is a no-op stub matching the runtime API signature.
func AddHotkey(combo, command string) {}

func RegisterCommand(command string, handler func(args string)) {}
func RegisterFunc(command string, handler func())               {}
