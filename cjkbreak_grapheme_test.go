package uniseg

import "testing"

// Grapheme cluster boundary tests (UAX #29, Grapheme Cluster Boundary
// Rules). Every case pins the exact boundary byte offsets, not just the
// cluster count, because mixed Chinese/English truncation code indexes its
// cuts by byte position.

func TestCJKGraphemeEmojiZWJ(t *testing.T) {
	// GB11: \p{Extended_Pictographic} Extend* ZWJ x
	// \p{Extended_Pictographic}. The sequences below are all one user
	// perceived character, i.e. the only boundary is the end of text.
	cases := []struct {
		name     string
		input    string
		breaks   []int
		clusters int
	}{
		{
			name:     "family man woman girl",
			input:    "👨\u200d👩\u200d👧", // U+1F468 ZWJ U+1F469 ZWJ U+1F467
			breaks:   []int{18},
			clusters: 1,
		},
		{
			name:     "woman teacher profession",
			input:    "👩\u200d🏫", // U+1F469 ZWJ U+1F3EB
			breaks:   []int{11},
			clusters: 1,
		},
		{
			name:     "thumbs up medium skin tone",
			input:    "👍🏽", // U+1F44D U+1F3FD (base + Extend skin tone modifier)
			breaks:   []int{8},
			clusters: 1,
		},
		{
			name:     "police officer medium skin tone woman with VS16",
			input:    "👮🏽\u200d♀️", // U+1F46E U+1F3FD ZWJ U+2640 U+FE0F
			breaks:   []int{17},
			clusters: 1,
		},
		{
			name:     "rainbow flag",
			input:    "🏳️\u200d🌈", // U+1F3F3 U+FE0F ZWJ U+1F308
			breaks:   []int{14},
			clusters: 1,
		},
		{
			name:     "man mechanic",
			input:    "👨\u200d🔧", // U+1F468 ZWJ U+1F527
			breaks:   []int{11},
			clusters: 1,
		},
		{
			name:     "two adjacent base emojis split",
			input:    "😀😀", // no ZWJ joining them: GB999 boundary between bases
			breaks:   []int{4, 8},
			clusters: 2,
		},
		{
			name: "dangling zwj after emoji does not absorb following letter",
			// GB9 makes "😀 ZWJ" one cluster (ZWJ extends backwards) but the
			// following "a" is not Extended_Pictographic, so GB11 does not
			// apply and GB999 inserts a boundary after the ZWJ.
			input:    "😀\u200da",
			breaks:   []int{7, 8},
			clusters: 2,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertInts(t, c.input+" -> grapheme breaks", graphemeBreaks(c.input), c.breaks)
			if n := GraphemeClusterCount(c.input); n != c.clusters {
				t.Errorf("%q -> GraphemeClusterCount = %d, want %d", c.input, n, c.clusters)
			}
		})
	}
}

func TestCJKGraphemeRegionalIndicators(t *testing.T) {
	// GB12: RI x RI (do not break between the first two RIs of a pair).
	// GB13: [^RI] RI x RI (an RI only pairs with an immediately preceding
	// unpaired RI). Odd RIs at the end of text therefore form a singleton
	// flag-letter cluster.
	cases := []struct {
		name   string
		input  string
		breaks []int
	}{
		{"flag pair", "🇩🇪", []int{8}},                                 // U+1F1E9 U+1F1EA -> one cluster
		{"two flags", "🇩🇪🇫🇷", []int{8, 16}},                           // pair, then pair
		{"three RIs", "\U0001F1E6\U0001F1E7\U0001F1E8", []int{8, 12}}, // pair, then singleton
		{"four RIs", "\U0001F1E6\U0001F1E7\U0001F1E8\U0001F1E9", []int{8, 16}},
		{"lone RI between letters", "x\U0001F1E6y", []int{1, 5, 6}}, // x | RI | y
		{"lone RI at end", "ab\U0001F1E6", []int{1, 2, 6}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertInts(t, c.input+" -> grapheme breaks", graphemeBreaks(c.input), c.breaks)
		})
	}
}

func TestCJKGraphemeCombiningAndVariationSelectors(t *testing.T) {
	// GB9: x Extend, x ZWJ. GB9a: x SpacingMark. GB9b: Prepend x.
	// Variation selectors U+FE00..U+FE0F have Grapheme_Cluster_Break=Extend.
	cases := []struct {
		name   string
		input  string
		breaks []int
	}{
		// "e" + combining acute accent U+0301: one cluster.
		{"letter plus combining mark", "e\u0301", []int{3}},
		// Two stacked combining marks both attach to the preceding base.
		{"letter plus two combining marks", "a\u0308\u0301", []int{5}},
		// Devanagari letter U+0905 + visarga U+0903 (SpacingMark).
		{"spacing mark", "अः", []int{6}},
		// Arabic number sign U+0600 has GCB=Prepend; the following letter is
		// absorbed (GB9b).
		{"prepend", "\u0600a", []int{3}},
		// VS16 (U+FE0F) on an emoji presentation base stays in the cluster.
		{"emoji base with VS16", "🙂️", []int{7}},
		// VS15 (U+FE0E) forces text presentation but is still part of the
		// cluster.
		{"emoji base with VS15", "🙂︎", []int{7}},
		// VS16 following a non-emoji letter still attaches via GB9.
		{"letter with VS16", "A️", []int{4}},
		// A combining mark never starts its own cluster when a base precedes
		// it; after a boundary it becomes the base of its own cluster (GB999
		// does not extend across the text start).
		{"combining mark after boundary", "a\u0301b", []int{3, 4}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertInts(t, c.input+" -> grapheme breaks", graphemeBreaks(c.input), c.breaks)
		})
	}
}

func TestCJKGraphemeHangulLVT(t *testing.T) {
	// GB6: L x (L | V | LV | LVT)
	// GB7: (LV | V) x (V | T)
	// GB8: (LVT | T) x T
	// U+1100 = L, U+1161 = V, U+11A8 = T, U+AC00 = LV (가), U+AC01 = LVT
	// (각), U+3131 = compatibility jamo ㄱ (GCB=Other).
	cases := []struct {
		name   string
		input  string
		breaks []int
	}{
		{"L plus V", "\u1100\u1161", []int{6}},    // GB6 L x V
		{"LV plus T (각)", "각", []int{3}},          // precomposed LVT singleton
		{"LVT plus conjoining T", "각ᆨ", []int{6}}, // GB8 LVT x T
		{"V plus T", "ᅡᆨ", []int{6}},              // GB7 V x T
		{"L plus L", "\u1100\u1100", []int{6}},    // GB6 L x L
		// T x L has no rule: GB999 inserts a boundary.
		{"T then L breaks", "ᆨᄀ", []int{3, 6}},
		// L x T has no rule: boundary after the L.
		{"L then T breaks", "\u1100\u11A8", []int{3, 6}},
		// LV (가) followed by a standalone V: GB7 applies (LV x V).
		{"LV plus V", "가ᅡ", []int{6}},
		// LV followed by a compatibility jamo ㄱ (GCB=Other): none of GB6-8
		// applies, so GB999 forces a boundary at byte 3.
		{"LV plus compatibility jamo breaks", "가ㄱ", []int{3, 6}},
		// Fully precomposed syllable remains a single cluster.
		{"precomposed syllable 간", "간", []int{3}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertInts(t, c.input+" -> grapheme breaks", graphemeBreaks(c.input), c.breaks)
		})
	}
}

func TestCJKGraphemeCRLF(t *testing.T) {
	// GB3: CR x LF (CRLF is one cluster). GB4: (Control | CR | LF) ÷.
	// GB5: ÷ (CR | LF | Control).
	cases := []struct {
		name   string
		input  string
		breaks []int
	}{
		{"CRLF one cluster", "a\r\nb", []int{1, 3, 4}},
		{"lone CR", "a\rb", []int{1, 2, 3}},
		{"lone LF", "a\nb", []int{1, 2, 3}},
		{"bare CRLF", "\r\n", []int{2}},
		{"CR CR split", "\r\r", []int{1, 2}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertInts(t, c.input+" -> grapheme breaks", graphemeBreaks(c.input), c.breaks)
		})
	}
}
