package format

import "strings"

// CenterText centers text within a given width
func CenterText(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-len(text)-padding)
}

// PadRight pads text to the right to fill width
func PadRight(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	return text + strings.Repeat(" ", width-len(text))
}

// PadRightWithEmoji pads text to the right, accounting for emoji display width
// Emojis take 2 terminal columns instead of 1
func PadRightWithEmoji(text string, width int) string {
	// Calculate display width - emojis take 2 terminal columns
	displayLen := 0
	for _, r := range text {
		if r == '🟢' || r == '🔴' || r == '🟡' {
			displayLen += 2
		} else {
			displayLen += 1
		}
	}

	if displayLen >= width {
		return text
	}
	return text + strings.Repeat(" ", width-displayLen)
}
