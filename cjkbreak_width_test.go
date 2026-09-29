package uniseg

import "testing"

// Monospace width tests (RuneWidth/StringWidth level). There is no public
// per-rune width function in this Unicode-15 version of the package, so the
// per-rune cases are checked through runeWidth (same package) and the
// aggregate cases through StringWidth / the width returned by
// FirstGraphemeClusterInString. Every case asserts concrete cell counts for
// mixed Chinese/English truncation, where an off-by-one width corrupts the
// cut position.

func TestCJKWidthWideAndAmbiguous(t *testing.T) {
	// East_Asian_Width W and F take two cells; N, Na and (by default) A take
	// one. H (halfwidth) takes one.
	cases := []struct {
		name  string
		input string
		want  int
	}{
		{"han ideograph W", "你", 2},
		{"fullwidth exclamation F", "！", 2},
		{"fullwidth digit F", "１", 2},
		{"latin letter Na", "a", 1},
		{"halfwidth katakana H", "ｱ", 1},
		{"multiplication sign N", "×", 1},
		{"per ten thousand sign N", "‱", 1},      // U+2031 EAW=N
		{"ambiguous dingbat A", "🄀", 1},          // U+1F100 EAW=A, default width 1
		{"mixed han latin fullwidth", "你好a！", 7}, // 2+2+1+2
		{"emoji presentation wide", "🙂", 2},
		{"copyright text presentation", "©", 1},
		{"regional indicator singleton", "\U0001F1E6", 2},
		{"flag cluster two cells", "🇩🇪", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := StringWidth(c.input); got != c.want {
				t.Errorf("%q -> StringWidth = %d, want %d", c.input, got, c.want)
			}
		})
	}
}

func TestCJKWidthAmbiguousConfig(t *testing.T) {
	// EastAsianAmbiguousWidth switches Ambiguous (A) between 1 and 2 cells.
	original := EastAsianAmbiguousWidth
	defer func() { EastAsianAmbiguousWidth = original }()

	ambiguous := "🄀" // DIGIT ZERO FULL STOP, EAW=Ambiguous
	EastAsianAmbiguousWidth = 1
	if got := StringWidth(ambiguous); got != 1 {
		t.Errorf("A=1: %q -> %d, want 1", ambiguous, got)
	}
	EastAsianAmbiguousWidth = 2
	if got := StringWidth(ambiguous); got != 2 {
		t.Errorf("A=2: %q -> %d, want 2", ambiguous, got)
	}
}

func TestCJKWidthZeroWidthCodePoints(t *testing.T) {
	// Default control/format code points and combining marks occupy no cell.
	// U+200B ZERO WIDTH SPACE and U+2060 WORD JOINER both have
	// Grapheme_Cluster_Break=Control in this table; U+200D ZWJ and U+0301
	// combining acute have GCB=ZWJ/Extend. All map to width 0.
	cases := []struct {
		name string
		r    rune
		want int
	}{
		{"ZWSP U+200B Control", 0x200B, 0},
		{"WORD JOINER U+2060 Control", 0x2060, 0},
		{"ZWNJ U+200C Extend", 0x200C, 0},
		{"BOM/ZWNBSP U+FEFF Control", 0xFEFF, 0},
		{"SOFT HYPHEN U+00AD Control", 0x00AD, 0},
		{"ZWJ U+200D", 0x200D, 0},
		{"combining acute U+0301 Extend", 0x0301, 0},
		{"CR", 0x000D, 0},
		{"LF", 0x000A, 0},
		{"skin tone modifier U+1F3FD Extend", 0x1F3FD, 0},
		{"VS16 U+FE0F Extend", 0xFE0F, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := runeWidth(c.r, propertyGraphemes(c.r)); got != c.want {
				t.Errorf("U+%04X -> runeWidth = %d, want %d", c.r, got, c.want)
			}
		})
	}

	// Zero-width characters embedded between Latin letters add no cells.
	if got := StringWidth("a\u200bb"); got != 2 {
		t.Errorf("%q -> StringWidth = %d, want 2", "a\u200bb", got)
	}
	if got := StringWidth("a\ufeffb"); got != 2 {
		t.Errorf("%q -> StringWidth = %d, want 2", "a\ufeffb", got)
	}
}

func TestCJKWidthCombiningAndClusters(t *testing.T) {
	// A base plus combining marks is one cell wide even though it contains
	// several runes; the marks contribute 0.
	cases := []struct {
		name  string
		input string
		want  int
	}{
		{"letter plus combining", "e\u0301", 1},
		{"letter plus two marks", "a\u0308\u0301", 1},
		{"devanagari base plus spacing mark", "अः", 2},
		{"skin-toned emoji", "👍🏽", 2},
		{"zwj family", "👨\u200d👩\u200d👧", 2},
		{"woman teacher", "👩\u200d🏫", 2},
		{"rainbow flag", "🏳️\u200d🌈", 2},
		{"police woman sequence", "👮🏽\u200d♀️", 2},
		{"emoji plus VS15 text", "🙂︎", 1},
		{"emoji plus VS16 emoji", "🙂️", 2},
		{"copyright plus VS16", "©️", 2},
		{"copyright plus VS15", "©︎", 1},
		{"female sign default", "♀", 1},
		{"female sign VS16", "♀️", 2},
		{"two em dash U+2E3A", "⸺", 3},
		{"three em dash U+2E3B", "⸻", 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := StringWidth(c.input); got != c.want {
				t.Errorf("%q -> StringWidth = %d, want %d", c.input, got, c.want)
			}
		})
	}

	// The width is also exposed per grapheme cluster by
	// FirstGraphemeClusterInString and must agree with StringWidth when summed.
	const mixed = "aé🇩🇪"
	state := -1
	rest := mixed
	total := 0
	wantWidths := []int{1, 1, 2}
	var gotWidths []int
	for len(rest) > 0 {
		var w int
		_, rest, w, state = FirstGraphemeClusterInString(rest, state)
		gotWidths = append(gotWidths, w)
		total += w
	}
	assertInts(t, mixed+" cluster widths", gotWidths, wantWidths)
	if total != StringWidth(mixed) || total != 4 {
		t.Errorf("summed cluster width = %d, want StringWidth(%q)=4", total, mixed)
	}
}
