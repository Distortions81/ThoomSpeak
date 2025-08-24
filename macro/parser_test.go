package macro

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParserExpression(t *testing.T) {
	p := New()
	if err := p.ParseLine("\"foo\" pause 10"); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(p.Macros) != 1 {
		t.Fatalf("expected 1 macro, got %d", len(p.Macros))
	}
	m := p.Macros[0]
	if m.Kind != MacroExpression || m.Name != "foo" {
		t.Fatalf("unexpected macro: %#v", m)
	}
	if len(m.Triggers) != 1 {
		t.Fatalf("expected 1 command, got %d", len(m.Triggers))
	}
	cmd := m.Triggers[0]
	if cmd.Kind != CmdPause {
		t.Fatalf("expected pause command, got %v", cmd.Kind)
	}
	if len(cmd.Params) != 1 || cmd.Params[0] != "10" {
		t.Fatalf("unexpected params: %#v", cmd.Params)
	}
}

func TestParserIgnoresComments(t *testing.T) {
	p := New()
	if err := p.ParseLine("// nothing"); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(p.Macros) != 0 {
		t.Fatalf("expected no macros, got %d", len(p.Macros))
	}
}

func TestParserReplacementText(t *testing.T) {
	p := New()
	if err := p.ParseLine("'bar' hello world"); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(p.Macros) != 1 {
		t.Fatalf("expected 1 macro, got %d", len(p.Macros))
	}
	m := p.Macros[0]
	if m.Kind != MacroReplacement || m.Name != "bar" {
		t.Fatalf("unexpected macro: %#v", m)
	}
	if len(m.Triggers) != 1 {
		t.Fatalf("expected 1 command, got %d", len(m.Triggers))
	}
	cmd := m.Triggers[0]
	if cmd.Kind != CmdText {
		t.Fatalf("expected text command, got %v", cmd.Kind)
	}
	if len(cmd.Params) != 2 || cmd.Params[0] != "hello" || cmd.Params[1] != "world" {
		t.Fatalf("unexpected params: %#v", cmd.Params)
	}
}

func TestSetVariableProducesNoMacro(t *testing.T) {
	p := New()
	if err := p.ParseLine("set @foo 1"); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(p.Macros) != 0 {
		t.Fatalf("expected no macros, got %d", len(p.Macros))
	}
}

func TestParserInclude(t *testing.T) {
	dir := t.TempDir()
	inc := filepath.Join(dir, "inc.txt")
	if err := os.WriteFile(inc, []byte("'bar' hello"), 0o644); err != nil {
		t.Fatalf("write include: %v", err)
	}
	main := filepath.Join(dir, "main.txt")
	content := "include \"inc.txt\""
	if err := os.WriteFile(main, []byte(content), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}
	p := New()
	if err := p.ParseFile(main); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(p.Macros) != 1 {
		t.Fatalf("expected 1 macro from include, got %d", len(p.Macros))
	}
	if p.Macros[0].Name != "bar" {
		t.Fatalf("unexpected macro name: %s", p.Macros[0].Name)
	}
}
