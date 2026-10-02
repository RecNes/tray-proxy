package core

import "unicode"

// FlagEmoji maps an ISO 3166-1 alpha-2 code to a regional-indicator flag emoji.
func FlagEmoji(code string) string {
	if len(code) != 2 {
		return ""
	}
	a := unicode.ToUpper(rune(code[0]))
	b := unicode.ToUpper(rune(code[1]))
	if a < 'A' || a > 'Z' || b < 'A' || b > 'Z' {
		return ""
	}
	return string([]rune{0x1F1E6 + (a - 'A'), 0x1F1E6 + (b - 'A')})
}
