package core

import "testing"

func TestFlagEmojiTR(t *testing.T) {
	got := FlagEmoji("TR")
	want := string([]rune{0x1F1F9, 0x1F1F7}) // 🇹🇷
	if got != want {
		t.Fatalf("FlagEmoji(TR) = %q want %q", got, want)
	}
}

func TestFlagEmojiEmpty(t *testing.T) {
	if FlagEmoji("") != "" {
		t.Fatal("empty code should yield empty flag")
	}
}
