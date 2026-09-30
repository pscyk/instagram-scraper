// Package tui provides an offline, read-only terminal view of raw Instagram snapshots.
package tui

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// SafeText removes escape sequences, controls and bidi formatting from upstream strings.
func SafeText(value string, limit int) string {
	value = ansi.Strip(value)
	var clean strings.Builder
	count := 0
	for _, character := range value {
		if count >= limit {
			break
		}
		if unicode.IsControl(character) || unicode.Is(unicode.Cf, character) {
			if character == '\n' || character == '\t' {
				clean.WriteByte(' ')
				count++
			}
			continue
		}
		clean.WriteRune(character)
		count++
	}
	return strings.TrimSpace(clean.String())
}

func number(value *int64) string {
	if value == nil {
		return "—"
	}
	digits := strconv.FormatInt(*value, 10)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	return digits
}

func clipped(value string, width int) string {
	return ansi.Truncate(SafeText(value, 4096), max(0, width), "…")
}
