package format

import "testing"

func TestCenterText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		{"short text", "Hi", 10, "    Hi    "},
		{"exact width", "Hello", 5, "Hello"},
		{"longer than width", "Hello World", 5, "Hello"},
		{"odd padding", "Hi", 9, "   Hi    "},
		{"empty text", "", 10, "          "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CenterText(tt.text, tt.width)
			if result != tt.expected {
				t.Errorf("CenterText(%q, %d) = %q, want %q", tt.text, tt.width, result, tt.expected)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		{"short text", "Hi", 10, "Hi        "},
		{"exact width", "Hello", 5, "Hello"},
		{"longer than width", "Hello World", 5, "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadRight(tt.text, tt.width)
			if result != tt.expected {
				t.Errorf("PadRight(%q, %d) = %q, want %q", tt.text, tt.width, result, tt.expected)
			}
		})
	}
}

func TestPadRightWithEmoji(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		// "🟢 running" = emoji(2) + space(1) + "running"(7) = 10 display width
		// width 15 - 10 = 5 spaces padding
		{"green emoji", "🟢 running", 15, "🟢 running     "},
		{"red emoji", "🔴 stopped", 15, "🔴 stopped     "},
		// "🟡 starting" = emoji(2) + space(1) + "starting"(8) = 11 display width
		// width 15 - 11 = 4 spaces padding
		{"yellow emoji", "🟡 starting", 15, "🟡 starting    "},
		{"no emoji", "running", 15, "running        "},
		// "🟢🔴" = 2 + 2 = 4 display width, width 10 - 4 = 6 spaces
		{"multiple emojis", "🟢🔴", 10, "🟢🔴      "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadRightWithEmoji(tt.text, tt.width)
			if result != tt.expected {
				t.Errorf("PadRightWithEmoji(%q, %d) = %q, want %q", tt.text, tt.width, result, tt.expected)
			}
		})
	}
}
