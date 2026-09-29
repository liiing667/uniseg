package uniseg

import "testing"

// Sentence boundary tests (UAX #29, Sentence Boundary Rules) with exact byte
// offsets. The terminator and its trailing spaces belong to the sentence that
// ends (SB8/SB9), which shifts truncation cuts relative to word boundaries.

func TestCJKSentenceCRLFMatrix(t *testing.T) {
	// SB3: CR x LF (CRLF treated as one terminator). SB4: Sep | CR | LF ÷.
	// The terminator is included in the preceding sentence, so a single
	// sentence covers "a\r\n" and the boundary is after the whole sequence.
	cases := []struct {
		name   string
		input  string
		breaks []int
	}{
		{"CRLF", "a\r\nb", []int{3, 4}},
		{"lone CR", "a\rb", []int{2, 3}},
		{"lone LF", "a\nb", []int{2, 3}},
		{"bare CRLF", "\r\n", []int{2}},
		{"bare CR", "\rx", []int{1, 2}},
		{"bare LF", "\nx", []int{1, 2}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertInts(t, c.input+" -> sentence breaks", sentenceBreaks(c.input), c.breaks)
		})
	}
}

func TestCJKSentencePunctuation(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		breaks []int
	}{
		// Boundary after ". " (ATerm, Close*, Sp*): includes the space.
		{"dot space", "Hello. Hi!", []int{7, 10}},
		// "Mr." is an abbreviation (SB6-style ATerm followed by space); no
		// sentence boundary there.
		{"abbreviation", "Mr. Smith went. Yes?", []int{4, 16, 20}},
		// Han full stop U+3002 and ideographic exclamation U+FF01 both
		// terminate a sentence directly (STerm, no trailing space needed).
		{"han punctuation", "你好。再见！", []int{9, 18}},
		// Two trailing spaces stay with the first sentence via Sp*.
		{"double trailing spaces", "He said hi.  Then ran.", []int{13, 22}},
		// No terminator at all: single sentence.
		{"no terminator", "a) b", []int{4}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertInts(t, c.input+" -> sentence breaks", sentenceBreaks(c.input), c.breaks)
		})
	}
}
