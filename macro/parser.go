package macro

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// BreakString mirrors kCLMacros_BreakString from the classic client.
const BreakString = " \t\r\n"

// Kind enumerates macro and command kinds.
type Kind int

const (
	// Macro types
	MacroEmpty Kind = iota
	MacroExpression
	MacroReplacement
	MacroFunction
	MacroKey
	MacroIncludeFile
	MacroVariable

	// Command types (subset of the original ones)
	CmdPause
	CmdMove
	CmdSetVariable
	CmdSetGlobalVariable
	CmdCallFunction
	CmdEnd
	CmdIf
	CmdElse
	CmdElseIf
	CmdEndIf
	CmdRandom
	CmdOr
	CmdEndRandom
	CmdLabel
	CmdGoto
	CmdText
	CmdMessage
	CmdParameter
	CmdNotCaseSensitive
	CmdStart
	CmdFinish
)

// Macro represents a parsed macro definition.
type Macro struct {
	Kind       Kind
	Name       string
	Attributes uint
	Triggers   []*Command
}

// Command represents a parsed command with parameters.
type Command struct {
	Kind   Kind
	Params []string
}

// Parser replicates the macro parsing logic of the classic client.
type Parser struct {
	FileName    string
	CmdLevel    int
	CurrentLine int
	lastMacro   *Macro
	lastCommand *Command
	Macros      []*Macro
	included    map[string]struct{}
}

// New creates a new parser.
func New() *Parser { return &Parser{included: make(map[string]struct{})} }

// ParseFile parses a macro file at the given path.
func (p *Parser) ParseFile(fname string) error {
	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer f.Close()

	p.FileName = fname
	p.CurrentLine = 1

	r := bufio.NewReader(f)
	var line strings.Builder
	for {
		ch, err := r.ReadByte()
		if errors.Is(err, io.EOF) {
			if line.Len() > 0 {
				if err2 := p.ParseLine(line.String()); err == nil {
					err = err2
				}
			}
			break
		}
		if err != nil {
			return err
		}

		switch ch {
		case '\r', '\n':
			if err := p.ParseLine(line.String()); err != nil {
				return err
			}
			line.Reset()
			p.CurrentLine++
		case '/':
			nxt, err := r.Peek(1)
			if err == nil && nxt[0] == '*' {
				r.ReadByte() // consume '*'
				p.ignoreComment(r)
			} else {
				line.WriteByte(ch)
			}
		case 0:
			// ignore nulls but keep warning behaviour minimal
			continue
		default:
			line.WriteByte(ch)
		}
	}
	if p.CmdLevel > 0 {
		return fmt.Errorf("file is missing end brackets '}'")
	}
	return nil
}

// ignoreComment consumes characters until the closing '*/'.
func (p *Parser) ignoreComment(r *bufio.Reader) {
	gotStar := false
	for {
		ch, err := r.ReadByte()
		if err != nil {
			return
		}
		if ch == '*' {
			gotStar = true
			continue
		}
		if ch == '/' && gotStar {
			return
		}
		gotStar = false
		if ch == '\n' || ch == '\r' {
			p.CurrentLine++
		}
	}
}

// ParseLine parses a single line of text.
func (p *Parser) ParseLine(line string) error {
	word, rest := p.newWord(line)
	if word == "" || strings.HasPrefix(word, "//") {
		return nil
	}

	switch strings.ToLower(word) {
	case "start":
		p.CmdLevel++
		return nil
	case "finish":
		p.CmdLevel--
		if p.CmdLevel < 0 {
			p.CmdLevel = 0
		}
		return nil
	}

	if p.CmdLevel == 0 {
		p.lastMacro = nil
	}

	if word != "" && p.lastMacro == nil && p.CmdLevel == 0 {
		var err error
		word, rest, err = p.newMacro(word, rest)
		if err != nil {
			return err
		}
	}

	if p.lastMacro == nil {
		return nil
	}

	if word != "" {
		if err := p.newCommand(word, &rest); err != nil {
			return err
		}
	}

	for {
		word, rest = p.newWord(rest)
		if word == "" {
			break
		}
		if err := p.newParameter(word); err != nil {
			return err
		}
	}
	return nil
}

// newWord returns the next word and remaining text using GetWord semantics.
func (p *Parser) newWord(line string) (word, rest string) {
	rest, word = GetWord(line, 1024, "\"'", "")
	return word, rest
}

// GetWord replicates the classic GetWord routine.
func GetWord(inLine string, maxLen int, quotes, sep string) (string, string) {
	p := inLine
	// skip breaks
	for len(p) > 0 && strings.ContainsRune(BreakString, rune(p[0])) {
		p = p[1:]
	}
	if len(p) == 0 {
		return "", ""
	}
	var out strings.Builder
	var quote rune
	if quotes != "" && strings.ContainsRune(quotes, rune(p[0])) {
		quote, _ = utf8DecodeRuneInString(p)
		out.WriteRune(quote)
		p = p[1:]
	}

	for len(p) > 0 {
		r, size := utf8DecodeRuneInString(p)
		if quote != 0 {
			if r == quote {
				out.WriteRune(r)
				p = p[size:]
				break
			}
			if r == '\\' {
				if len(p[size:]) > 0 {
					nr := p[size]
					switch nr {
					case 'r':
						out.WriteByte('\r')
						p = p[size+1:]
						continue
					case '"', '\'', '\\':
						out.WriteByte(nr)
						p = p[size+1:]
						continue
					default:
						out.WriteByte('\\')
						p = p[size:]
						continue
					}
				}
			}
			out.WriteRune(r)
			p = p[size:]
			continue
		}

		// unquoted
		if r == '/' && strings.HasPrefix(p, "//") {
			break
		}
		if r == '\\' {
			if len(p[size:]) > 0 {
				nr := p[size]
				switch nr {
				case 'r':
					out.WriteByte('\r')
					p = p[size+1:]
					continue
				case '"', '\'', '\\':
					out.WriteByte(nr)
					p = p[size+1:]
					continue
				default:
					out.WriteByte('\\')
					p = p[size:]
					continue
				}
			}
		}
		if strings.ContainsRune(sep, r) || strings.ContainsRune(BreakString, r) || r == '\r' || r == '\n' {
			break
		}
		out.WriteRune(r)
		p = p[size:]
	}

	return p, out.String()
}

func utf8DecodeRuneInString(s string) (r rune, size int) {
	if len(s) == 0 {
		return 0, 0
	}
	r, size = rune(s[0]), 1
	return
}

// newMacro defines a new macro.
func (p *Parser) newMacro(word string, line string) (string, string, error) {
	switch {
	case strings.HasPrefix(word, "\""):
		name := strings.Trim(word, "\"")
		m := &Macro{Kind: MacroExpression, Name: name}
		p.Macros = append(p.Macros, m)
		p.lastMacro = m
		word, line = p.newWord(line)
		return word, line, nil
	case strings.HasPrefix(word, "'"):
		name := strings.Trim(word, "'")
		m := &Macro{Kind: MacroReplacement, Name: name}
		p.Macros = append(p.Macros, m)
		p.lastMacro = m
		word, line = p.newWord(line)
		return word, line, nil
	case strings.EqualFold(word, "set"):
		varname, rest := p.newWord(line)
		if varname == "" {
			return "", rest, nil
		}
		value, rest2 := p.newWord(rest)
		_ = value // stub
		p.lastMacro = nil
		return "", rest2, nil
	case strings.EqualFold(word, "include"):
		fname, rest := p.newWord(line)
		if fname == "" {
			p.lastMacro = nil
			return "", rest, nil
		}
		name := strings.Trim(fname, "\"'")
		if !filepath.IsAbs(name) && p.FileName != "" {
			name = filepath.Join(filepath.Dir(p.FileName), name)
		}
		if _, ok := p.included[name]; ok {
			p.lastMacro = nil
			return "", rest, nil
		}
		p.included[name] = struct{}{}
		child := New()
		child.included = p.included
		if err := child.ParseFile(name); err != nil {
			return "", rest, err
		}
		p.Macros = append(p.Macros, child.Macros...)
		p.lastMacro = nil
		return "", rest, nil
	default:
		m := &Macro{Kind: MacroFunction, Name: word}
		p.Macros = append(p.Macros, m)
		p.lastMacro = m
		word, line = p.newWord(line)
		return word, line, nil
	}
}

// command lookup table
var cmdKinds = map[string]Kind{
	"pause":       CmdPause,
	"move":        CmdMove,
	"set":         CmdSetVariable,
	"setglobal":   CmdSetGlobalVariable,
	"call":        CmdCallFunction,
	"end":         CmdEnd,
	"if":          CmdIf,
	"else":        CmdElse,
	"random":      CmdRandom,
	"or":          CmdOr,
	"label":       CmdLabel,
	"goto":        CmdGoto,
	"message":     CmdMessage,
	"ignore_case": CmdNotCaseSensitive,
	"start":       CmdStart,
	"finish":      CmdFinish,
}

func (p *Parser) newCommand(word string, line *string) error {
	if word == "" {
		return nil
	}
	if k, ok := cmdKinds[strings.ToLower(word)]; ok {
		// handle "end random" and "end if"
		if k == CmdEnd {
			w, rest := p.newWord(*line)
			switch strings.ToLower(w) {
			case "random":
				k = CmdEndRandom
				*line = rest
				word = w
			case "if":
				k = CmdEndIf
				*line = rest
				word = w
			default:
				*line = rest
			}
		}
		// handle "else if"
		if k == CmdElse {
			w, rest := p.newWord(*line)
			if strings.ToLower(w) == "if" {
				k = CmdElseIf
				*line = rest
			} else {
				// treat w as first parameter
				word = w
				*line = rest
				cmd := &Command{Kind: k}
				p.lastMacro.Triggers = append(p.lastMacro.Triggers, cmd)
				p.lastCommand = cmd
				return p.newParameter(word)
			}
		}
		cmd := &Command{Kind: k}
		p.lastMacro.Triggers = append(p.lastMacro.Triggers, cmd)
		p.lastCommand = cmd
		return nil
	}
	// treat as text command
	cmd := &Command{Kind: CmdText}
	p.lastMacro.Triggers = append(p.lastMacro.Triggers, cmd)
	p.lastCommand = cmd
	return p.newParameter(word)
}

func (p *Parser) newParameter(word string) error {
	if p.lastCommand == nil {
		return nil
	}
	p.lastCommand.Params = append(p.lastCommand.Params, word)
	return nil
}
