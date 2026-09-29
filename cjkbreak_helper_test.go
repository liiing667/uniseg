package uniseg

import "testing"

// This file contains the shared helpers for the cjkbreak_*_test.go files.
//
// Those tests pin down the exact break positions (byte offsets) produced by
// the four segmentation layers used by Chinese/English mixed-text typesetting
// and truncation:
//
//   - grapheme cluster boundaries (UAX #29, GB rules)
//   - word boundaries (UAX #29, WB rules)
//   - sentence boundaries (UAX #29, SB rules)
//   - line break opportunities (UAX #14, LB rules)
//
// Assertions always list the concrete byte offsets rather than only the number
// of resulting segments, so that a one-byte shift (e.g. inside an emoji ZWJ
// sequence) is detected. The final offset is always len(s): per UAX #14 LB3
// the end of text is reported as a (synthetic) mandatory break; word and
// sentence rules similarly carry a boundary at the end of text.

// graphemeBreaks returns the byte offsets of every grapheme cluster boundary
// (excluding the boundary at offset 0, including len(s)).
func graphemeBreaks(s string) []int {
	var offsets []int
	state := -1
	original := len(s)
	for len(s) > 0 {
		var cluster string
		cluster, s, _, state = FirstGraphemeClusterInString(s, state)
		_ = cluster
		offsets = append(offsets, original-len(s))
	}
	return offsets
}

// wordBreaks returns the byte offsets of every word boundary, including the
// boundary at the end of text.
func wordBreaks(s string) []int {
	var offsets []int
	state := -1
	original := len(s)
	for len(s) > 0 {
		var _ string
		_, s, state = FirstWordInString(s, state)
		offsets = append(offsets, original-len(s))
	}
	return offsets
}

// sentenceBreaks returns the byte offsets of every sentence boundary,
// including the boundary at the end of text.
func sentenceBreaks(s string) []int {
	var offsets []int
	state := -1
	original := len(s)
	for len(s) > 0 {
		var _ string
		_, s, state = FirstSentenceInString(s, state)
		offsets = append(offsets, original-len(s))
	}
	return offsets
}

// segmentEnd is one result of [lineSegments]: a break byte offset tagged with
// whether the break is mandatory.
type segmentEnd struct {
	offset    int
	mustBreak bool
}

// lineSegments returns the byte offsets at which FirstLineSegmentInString
// produces a break (optional or mandatory), tagged with the mustBreak flag.
func lineSegments(s string) []segmentEnd {
	var results []segmentEnd
	state := -1
	original := len(s)
	for len(s) > 0 {
		var _ string
		var mustBreak bool
		_, s, mustBreak, state = FirstLineSegmentInString(s, state)
		results = append(results, segmentEnd{offset: original - len(s), mustBreak: mustBreak})
	}
	return results
}

// lineBreakAtClusterEnd returns the line break decision (LineDontBreak,
// LineCanBreak or LineMustBreak) reported by [StepString] immediately after the
// grapheme cluster ending at byte offset. offset must be a grapheme cluster
// boundary.
func lineBreakAtClusterEnd(t *testing.T, s string, offset int) int {
	t.Helper()
	state := -1
	end := 0
	for len(s) > 0 {
		var cluster string
		var boundaries int
		cluster, s, boundaries, state = StepString(s, state)
		end += len(cluster)
		if end == offset {
			return boundaries & MaskLine
		}
	}
	t.Fatalf("byte offset %d is not a grapheme cluster boundary in %q", offset, s)
	return -1
}

// assertInts fails with a readable message if got != want.
func assertInts(t *testing.T, label string, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got %v, want %v", label, got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s: got %v, want %v", label, got, want)
			return
		}
	}
}

// assertSegments compares the offsets and mustBreak flags produced by
// [lineSegments].
func assertSegments(t *testing.T, label string, got []segmentEnd, want []segmentEnd) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got %v, want %v", label, got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s: got %v, want %v", label, got, want)
			return
		}
	}
}
