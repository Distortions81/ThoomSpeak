package main

import (
	"math"
	"strings"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// wrapText splits s into lines that do not exceed maxWidth when rendered
// with the provided face. Words are kept intact when possible; if a single
// word exceeds maxWidth it will be broken across lines.
func wrapText(s string, face text.Face, maxWidth float64) (int, []string) {
	var (
		lines         []string
		maxUsed       float64
		runesBuffer   []rune
		spaceWidth, _ = text.Measure(" ", face, 0)
	)
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}

		var builder strings.Builder
		curWidth := 0.0

		for _, w := range words {
			wWidth, _ := text.Measure(w, face, 0)

			if curWidth == 0 {
				if wWidth <= maxWidth {
					builder.WriteString(w)
					curWidth = wWidth
					continue
				}
				runesBuffer = runesBuffer[:0]
				partWidth := 0.0
				for _, r := range w {
					rw, _ := text.Measure(string(r), face, 0)
					if partWidth+rw > maxWidth && len(runesBuffer) > 0 {
						part := string(runesBuffer)
						if partWidth > maxUsed {
							maxUsed = partWidth
						}
						lines = append(lines, part)
						runesBuffer = runesBuffer[:0]
						partWidth = 0
					}
					runesBuffer = append(runesBuffer, r)
					partWidth += rw
				}
				builder.WriteString(string(runesBuffer))
				curWidth = partWidth
				continue
			}

			candWidth := curWidth + spaceWidth + wWidth
			if candWidth <= maxWidth {
				builder.WriteByte(' ')
				builder.WriteString(w)
				curWidth = candWidth
				continue
			}

			if curWidth > maxUsed {
				maxUsed = curWidth
			}
			lines = append(lines, builder.String())
			builder.Reset()
			curWidth = 0

			if wWidth <= maxWidth {
				builder.WriteString(w)
				curWidth = wWidth
			} else {
				runesBuffer = runesBuffer[:0]
				partWidth := 0.0
				for _, r := range w {
					rw, _ := text.Measure(string(r), face, 0)
					if partWidth+rw > maxWidth && len(runesBuffer) > 0 {
						part := string(runesBuffer)
						if partWidth > maxUsed {
							maxUsed = partWidth
						}
						lines = append(lines, part)
						runesBuffer = runesBuffer[:0]
						partWidth = 0
					}
					runesBuffer = append(runesBuffer, r)
					partWidth += rw
				}
				builder.WriteString(string(runesBuffer))
				curWidth = partWidth
			}
		}

		if curWidth > maxUsed {
			maxUsed = curWidth
		}
		lines = append(lines, builder.String())
	}

	return int(math.Ceil(maxUsed)), lines
}
