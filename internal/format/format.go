package format

import "strings"

func CenterText(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-len(text)-padding)
}

func PadRight(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	return text + strings.Repeat(" ", width-len(text))
}

// PadRightWithEmoji pads text accounting for emoji display width (2 columns)
func PadRightWithEmoji(text string, width int) string {
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
