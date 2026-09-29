package uniseg

import "testing"

// Word boundary tests (UAX #29, Word Boundary Rules) with exact byte
// offsets. These are the boundaries used by double-click selection and
// whole-word matching in mixed Chinese/English text.

func TestCJKWordCRLFMatrix(t *testing.T) {
	// WB3: CR x LF. WB3a: (Newline | CR | LF) ÷. WB3b: ÷
	// (Newline | CR | LF). Unlike sentence breaking (SB3/SB4), the terminator
	// itself is its own word; only CR and LF are merged.
	cases := []struct {
		name   string
		input  string
		breaks []int
	}{
		{"CRLF", "a\r\nb", []int{1, 3, 4}},  // a | CRLF | b
		{"lone CR", "a\rb", []int{1, 2, 3}}, // a | CR | b
		{"lone LF", "a\nb", []int{1, 2, 3}}, // a | LF | b
		{"bare CRLF", "\r\n", []int{2}},
		{"VT is WB Newline", "a\x0bb", []int{1, 2, 3}}, // U+000B: WB3a/WB3b
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertInts(t, c.input+" -> word breaks", wordBreaks(c.input), c.breaks)
		})
	}
}

func TestCJKWordPunctuationAndLatin(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		breaks []int
	}{
		// Punctuation is its own word.
		{"comma glued to letters", "a,b", []int{1, 2, 3}},
		{"comma space letters", "a, b", []int{1, 2, 3, 4}},
		// WB15c/WB15d keep the apostrophe inside a contraction.
		{"contraction", "can't", []int{5}},
		// Numeric stays one word around the decimal point (WB6/WB7/WB12).
		{"decimal number", "3.14", []int{4}},
		// ExtendNumLet links letters across the underscore (WB13a/WB13b).
		{"underscore join", "super_cafe", []int{10}},
		// WSegSpace runs merge (WB3d); two spaces are one word unit.
		{"double space", "hello  world", []int{5, 7, 12}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertInts(t, c.input+" -> word breaks", wordBreaks(c.input), c.breaks)
		})
	}
}

func TestCJKWordHanAndEmoji(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		breaks []int
	}{
		// Each Han ideograph is its own word (no word-joiner rule for two
		// OLetter-like Han characters): break after every 3 bytes.
		{"han han latin", "你好a", []int{3, 6, 7}},
		// A flag is one word; the following Latin letter starts another.
		{"flag then letter", "🇩🇪x", []int{8, 9}},
		// An emoji ZWJ sequence is one word (ZWJ is ignored by WB4, then the
		// emoji chain holds via WB3c/WB4).
		{"zwj family", "👨\u200d👩\u200d👧", []int{18}},
		// A skin-toned emoji followed by a letter splits.
		{"toned emoji then letter", "👍🏽a", []int{8, 9}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertInts(t, c.input+" -> word breaks", wordBreaks(c.input), c.breaks)
		})
	}
}
