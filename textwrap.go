package main

import (
	"math"
	"strings"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// wrapText splits s into lines that do not exceed maxWidth when rendered
// with the provided face. Words are kept intact when possible; if a single
// word exceeds maxWidth it will be broken across lines. Unlike strings.Fields,
// it preserves runs of spaces so user input doesn't lose spacing.
func wrapText(s string, face text.Face, maxWidth float64) (int, []string) {
	var (
		lines   []string
		maxUsed float64
	)
	for _, para := range strings.Split(s, "\n") {
		tokens := strings.SplitAfter(para, " ")
		var builder strings.Builder
		curWidth := 0.0
		for _, tok := range tokens {
			if tok == "" {
				continue
			}
			w, _ := text.Measure(tok, face, 0)
			if curWidth+w <= maxWidth {
				builder.WriteString(tok)
				curWidth += w
				continue
			}
			if builder.Len() > 0 {
				if curWidth > maxUsed {
					maxUsed = curWidth
				}
				lines = append(lines, builder.String())
				builder.Reset()
				curWidth = 0
			}
			if w <= maxWidth {
				builder.WriteString(tok)
				curWidth = w
				continue
			}
			for _, r := range tok {
				rw, _ := text.Measure(string(r), face, 0)
				if curWidth+rw > maxWidth && builder.Len() > 0 {
					if curWidth > maxUsed {
						maxUsed = curWidth
					}
					lines = append(lines, builder.String())
					builder.Reset()
					curWidth = 0
				}
				builder.WriteRune(r)
				curWidth += rw
			}
		}
		if curWidth > maxUsed {
			maxUsed = curWidth
		}
		lines = append(lines, builder.String())
	}
	return int(math.Ceil(maxUsed)), lines
}
