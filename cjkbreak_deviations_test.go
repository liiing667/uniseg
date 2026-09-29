package uniseg

import (
	"testing"
)

// This file documents places where the observed behavior of the package
// diverges from the Unicode specification (UAX #29 / UAX #14, Unicode 15.0.0)
// or from the package's own doc.go width contract. Each deviation is pinned
// with a test of the *current* behavior so a future change is intentional and
// visible, and a skipped test records the behavior the cited rule requires.
//
// These divergences also exist in the upstream github.com/rivo/uniseg code the
// package is generated from; the UCD conformance suites
// (GraphemeBreakTest/WordBreakTest/SentenceBreakTest/LineBreakTest shipped in
// *_break_test.go) do not cover the specific sequences below.

// ---------------------------------------------------------------------------
// Deviation 1: FirstLineSegmentInString breaks inside an emoji ZWJ grapheme
// cluster when the emoji base is followed by an emoji modifier (skin tone) and
// then ZWJ.
//
// Basis: UAX #14
//   - LB8a: "Do not break between ZWJ and any character."
//     (ZWJ x), https://www.unicode.org/reports/tr14/#LB8a
//   - LB9: "Do not break a character sequence...": an emoji modifier such as
//     U+1F3FD has line break class EM here and attaches to its emoji base EB,
//     after which the following ZWJ must glue the rest on via LB8a.
//
// Root cause (linerules.go, transitionLineBreakState, "LB30b"): after
// EB x EM the function returns prAny (the numeric constant 1) as the new line
// state. 1 is also the value of the line state lbBK, so when the next code
// point is ZWJ the "mandatory break state" test
//
//	state == lbBK
//
//	in the LB9/LB10 combining-mark branch is true, and the ZWJ is emitted with
//	LineMustBreak instead of setting the LB8a no-break bit. Clean
//	ExtPictographic ZWJ ExtPictographic chains (no emoji modifier, e.g.
//	"👨\u200d🔧") are not affected (covered in TestCJKLineBreakCharactersAndZWJ).
//
// Impact on the four break layers: the grapheme/word/sentence layers (UAX
// #29) all keep the sequence as one unit; only the rune-wise UAX #14 line
// breaker splits it. Callers using Step/StepString (which emits line break
// decisions only at grapheme cluster boundaries) cannot observe the internal
// split, see UAX #14 section 8.2 example 6.
// ---------------------------------------------------------------------------

func TestCJKDeviationEmojiModifierZWJLineBreak(t *testing.T) {
	const policeWoman = "👮🏽\u200d♀️" // U+1F46E U+1F3FD ZWJ U+2640 U+FE0F

	// CURRENT BEHAVIOR: the rune-wise line breaker reports a MANDATORY break
	// after byte 8 (the base+skin-tone modifier), inside the single grapheme
	// cluster which ends at byte 17.
	got := lineSegments(policeWoman)
	want := []segmentEnd{
		{offset: 8, mustBreak: true}, // diverges from LB8a/LB9
		{offset: 17, mustBreak: true},
	}
	assertSegments(t, policeWoman+" -> CURRENT line segments (LB8a deviation)", got, want)

	// The UAX #29 layers are unaffected and correct.
	assertInts(t, policeWoman+" -> grapheme breaks (GB11)", graphemeBreaks(policeWoman), []int{17})
	assertInts(t, policeWoman+" -> word breaks (WB3c/WB4)", wordBreaks(policeWoman), []int{17})
	assertInts(t, policeWoman+" -> sentence breaks", sentenceBreaks(policeWoman), []int{17})

	// StepString reports the boundary decision for the whole cluster only;
	// at byte 17 there is nothing inside to break at.
	if got := lineBreakAtClusterEnd(t, policeWoman, 17); got != LineMustBreak {
		t.Errorf("StepString end-of-text break = %d, want LineMustBreak(LB3)", got)
	}
}

// TestCJKDeviationEmojiModifierZWJLineBreakSpecCompliance records what UAX #14
// LB8a/LB9 require: one segment of 17 bytes, no internal break. It is skipped
// so `go test ./...` stays green against the current implementation.
func TestCJKDeviationEmojiModifierZWJLineBreakSpecCompliance(t *testing.T) {
	t.Skip("UAX #14 LB8a/LB9: fix the prAny(=lbBK) state returned at the LB30b EB x EM transition; " +
		"see TestCJKDeviationEmojiModifierZWJLineBreak")

	const policeWoman = "👮🏽\u200d♀️"
	want := []segmentEnd{{offset: 17, mustBreak: true}} // single segment, only LB3 end break
	assertSegments(t, policeWoman+" -> spec line segments", lineSegments(policeWoman), want)
}

// ---------------------------------------------------------------------------
// Deviation 2: keycap-style and ASCII emoji bases with VS16 have width 1 even
// though the package doc.go states:
//
//	"Grapheme clusters ending with Variation Selector-16 (U+FE0F) have a
//	width of 2."
//
// Root cause: propertyGraphemes fast-tracks every ASCII character 0x20..0x7E
// to prAny, so U+0023 NUMBER SIGN and U+0030..U+0039 digits never receive
// their real Grapheme_Cluster_Break=Extended_Pictographic property. The width
// special case "firstProp == prExtendedPictographic -> VS16 forces width 2"
// therefore never fires. The clusters are segmented correctly (GB9/GB11),
// only their width disagrees with the documented contract.
// ---------------------------------------------------------------------------

func TestCJKDeviationASCIEmojiVS16Width(t *testing.T) {
	// CURRENT BEHAVIOR.
	cases := []struct {
		input string
		want  int
	}{
		{"#️", 1},  // U+0023 U+FE0F, "#\uFE0F keycap base"
		{"1️", 1},  // U+0031 U+FE0F
		{"1️⃣", 1}, // U+0031 U+FE0F U+20E3 combining enclosing keycap
		{"#️⃣", 1}, // U+0023 U+FE0F U+20E3
	}
	for _, c := range cases {
		if got := StringWidth(c.input); got != c.want {
			t.Errorf("%q -> StringWidth = %d, want current %d", c.input, got, c.want)
		}
		// Segmentation is still one grapheme cluster (GB9 Extend attaches
		// VS16 and U+20E3), the divergence is width only.
		assertInts(t, c.input+" -> grapheme breaks (GB9)", graphemeBreaks(c.input), []int{len(c.input)})
	}
}

// TestCJKDeviationASCIEmojiVS16WidthDocCompliance records the width doc.go
// promises for clusters ending in U+FE0F (2 cells).
func TestCJKDeviationASCIEmojiVS16WidthDocCompliance(t *testing.T) {
	t.Skip("doc.go width contract: ASCII keycap bases are fast-tracked to prAny and lose " +
		"Extended_Pictographic, so VS16 does not force width 2; " +
		"see TestCJKDeviationASCIEmojiVS16Width")
	for _, s := range []string{"#️", "1️"} {
		if got := StringWidth(s); got != 2 {
			t.Errorf("%q -> StringWidth = %d, want documented 2", s, got)
		}
	}
}

// ---------------------------------------------------------------------------
// Deviation 3: trailing conjoining Jamo in a cluster that starts with a
// precomposed LV/LVT syllable or with a V add their per-rune width, although
// doc.go states:
//
//	"For Hangul grapheme clusters composed of conjoining Jamo and for
//	Regional Indicators (flags), all code points except the first one have a
//	width of 0."
//
// The width accumulation in FirstGraphemeCluster/Step only suppresses
// subsequent widths for clusters starting with prL or prRegionalIndicator;
// GB7 (LV/V x V/T) and GB8 (LVT/T x T) clusters therefore over-count.
// ---------------------------------------------------------------------------

func TestCJKDeviationHangulTrailingJamoWidth(t *testing.T) {
	cases := []struct {
		name  string
		input string
		rule  string
		want  int // current width
	}{
		{"LV plus V", "가ᅡ", "GB7", 3},  // U+AC00(LV) U+1161(V): 2 + 1
		{"V plus T", "ᅡᆨ", "GB7", 2},   // U+1161(V) U+11A8(T): 1 + 1
		{"LVT plus T", "각ᆨ", "GB8", 3}, // U+AC01(LVT) U+11A8(T): 2 + 1
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := StringWidth(c.input); got != c.want {
				t.Errorf("%q (%s) -> StringWidth = %d, want current %d", c.input, c.rule, got, c.want)
			}
			// The cluster boundaries themselves are correct per GB7/GB8.
			assertInts(t, c.input+" -> grapheme breaks ("+c.rule+")", graphemeBreaks(c.input), []int{len(c.input)})
		})
	}

	// Contrast: an L-started GB6 cluster and a plain precomposed syllable do
	// follow the documented width (trailing Jamo contribute 0). These are the
	// non-deviating cases guarding the boundary of the bug.
	if got := StringWidth("가"); got != 2 { // U+1100(L) U+1161(V), GB6
		t.Errorf("L x V %q -> StringWidth = %d, want documented 2", "가", got)
	}
	if got := StringWidth("가"); got != 2 {
		t.Errorf("precomposed %q -> StringWidth = %d, want 2", "가", got)
	}
}

// TestCJKDeviationHangulTrailingJamoWidthDocCompliance records the width
// doc.go promises for GB7/GB8 Jamo clusters (2 cells).
func TestCJKDeviationHangulTrailingJamoWidthDocCompliance(t *testing.T) {
	t.Skip("doc.go width contract: only prL/prRegionalIndicator clusters suppress trailing " +
		"Jamo widths; GB7/GB8 clusters over-count; see TestCJKDeviationHangulTrailingJamoWidth")
	for _, s := range []string{"가ᅡ", "ᅡᆨ", "각ᆨ"} {
		if got := StringWidth(s); got != 2 {
			t.Errorf("%q -> StringWidth = %d, want documented 2", s, got)
		}
	}
}
