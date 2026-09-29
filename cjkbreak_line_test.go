package uniseg

import "testing"

// Line break tests (UAX #14) with exact byte offsets and mandatory/optional
// distinction. The offsets returned by FirstLineSegmentInString are the points
// after which wrapping can/must occur; the offset len(s) entry is the
// synthetic mandatory break required by LB3 and is excluded from these
// expectations where it adds no information (assertSegments compares both
// offsets and flags, including the LB3 end entry).

func TestCJKLineCRLFMatrix(t *testing.T) {
	// LB5: BK, CR, LF, NL force a break; CR x LF is kept together. The
	// terminator stays attached to the preceding segment, so the mandatory
	// break is reported after the whole CRLF (byte 3), not after the CR.
	cases := []struct {
		name  string
		input string
		want  []segmentEnd
	}{
		{"CRLF", "a\r\nb", []segmentEnd{{3, true}, {4, true}}},
		{"lone CR", "a\rb", []segmentEnd{{2, true}, {3, true}}},
		{"lone LF", "a\nb", []segmentEnd{{2, true}, {3, true}}},
		{"bare CRLF", "\r\n", []segmentEnd{{2, true}}},
		{"bare CR", "\rx", []segmentEnd{{1, true}, {2, true}}},
		{"bare LF", "\nx", []segmentEnd{{1, true}, {2, true}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertSegments(t, c.input+" -> line segments", lineSegments(c.input), c.want)
		})
	}
}

func TestCJKLineNewlineClasses(t *testing.T) {
	// U+000B (VT) and U+2028 (LINE SEPARATOR) are BK (LB4 mandatory).
	// U+0085 (NEL) is NL (LB5 mandatory).
	cases := []struct {
		name  string
		input string
		want  []segmentEnd
	}{
		{"vertical tab BK", "a\x0bb", []segmentEnd{{2, true}, {3, true}}},
		{"line separator BK", "a\u2028b", []segmentEnd{{4, true}, {5, true}}},
		{"NEL", "a\u0085b", []segmentEnd{{3, true}, {4, true}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertSegments(t, c.input+" -> line segments", lineSegments(c.input), c.want)
		})
	}
}

func TestCJKLineNoStartNoEnd(t *testing.T) {
	// Break decisions (LineDontBreak/LineCanBreak) emitted by StepString at
	// specific grapheme-cluster byte offsets.
	cases := []struct {
		name   string
		input  string
		offset int // byte offset of the grapheme cluster boundary
		want   int // LineDontBreak / LineCanBreak / LineMustBreak
		basis  string
	}{
		{"OP no line end", "(a)", 1, LineDontBreak, "LB14: OP x; opening punctuation cannot end a line"},
		{"HY no line start", "a-b", 1, LineDontBreak, "LB21: x HY; hyphen cannot start a line"},
		{"BA soft hyphen no line start", "a\u00adb", 1, LineDontBreak, "LB21: x BA; break-after char cannot start a line"},
		{"no break immediately before space", "a b", 1, LineDontBreak, "LB18 moves the break after the space"},
		{"break after the space", "a b", 2, LineCanBreak, "LB18: break after spaces"},
		{"break after RI pair", "\U0001F1E9\U0001F1EAx", 8, LineCanBreak, "LB30a: RIs pair; break allowed after the pair"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := lineBreakAtClusterEnd(t, c.input, c.offset); got != c.want {
				t.Errorf("%s\n%q offset %d: line break = %d, want %d", c.basis, c.input, c.offset, got, c.want)
			}
		})
	}

	// LB30a keeps the two RIs of a flag in the same grapheme cluster, so byte
	// 4 (inside the flag) is not even a cluster boundary and can never be a
	// legal cut.
	t.Run("no cut inside RI flag", func(t *testing.T) {
		input := "🇩🇪x"
		if got := graphemeBreaks(input); len(got) != 2 || got[0] != 8 || got[1] != 9 {
			t.Fatalf("UAX #14 LB30a / UAX #29 GB12: %q breaks=%v, want [8 9]", input, got)
		}
	})
}

func TestCJKLineClosingPunctuation(t *testing.T) {
	// Closing punctuation classes:
	//   ')' U+0029 is CP, '}' U+007D is CL,
	//   U+FF09 ）, U+300B 》, U+300D 」 are CL in this table.
	// LB17 keeps B2 characters together; the East-Asian CL closers and CP
	// are allowed before a line end by LB31 unless a more specific rule
	// applies. The important invariant for typesetting is that a *closing*
	// mark must not be wrapped to the next line after its preceding text when
	// joined by LB13-adjacent context, while opening marks never trail.
	cases := []struct {
		name  string
		input string
		want  []segmentEnd
	}{
		// "(a)" is unbreakable: LB14 pins "(" and LB30 makes AL x CP a
		// no-break, so the only segment is the synthetic LB3 end break.
		{"latin parens glued", "(a)", []segmentEnd{{3, true}}},
		// Closing CP after a letter: the pair "a)" itself is unbreakable;
		// the break opportunity is after the following space (LB18), so the
		// first segment ends at byte 3 as optional and the space goes with
		// the next segment.
		{"close then space", "a) b", []segmentEnd{{3, false}, {4, true}}},
		// Fullwidth opening + ideograph + fullwidth closing stays together
		// (LB14/LB30 pair behavior), then a break before the next ideograph.
		{"fullwidth bracket pair", "（你）好", []segmentEnd{{9, false}, {12, true}}},
		// CJK corner brackets form an unbreakable quoted unit.
		{"corner bracket unit", "a「b」c", []segmentEnd{{1, false}, {8, false}, {9, true}}},
		// A closing CJK bracket may legitimately end a line after Han text
		// (break allowed before the opening bracket chain, not after the
		// closer): "你」" is one unbreakable tail.
		{"han then closer", "你」", []segmentEnd{{6, true}}},
		{"opener then han", "「你", []segmentEnd{{6, true}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertSegments(t, c.input+" -> line segments", lineSegments(c.input), c.want)
		})
	}
}

func TestCJKLineBreakCharactersAndZWJ(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []segmentEnd
	}{
		// LB8a: ZWJ joins emoji across line breaking for a plain
		// ExtPict ZWJ ExtPict chain; the whole sequence is one segment.
		{"emoji zwj emoji", "👨\u200d🔧", []segmentEnd{{11, true}}},
		{"family zwj", "👨\u200d👩\u200d👧", []segmentEnd{{18, true}}},
		// Rainbow flag: ZWJ inside the flag is a no-break under LB8a/LB9.
		{"rainbow flag", "🏳️\u200d🌈", []segmentEnd{{14, true}}},
		// After the joined emoji, a break before the following letter is
		// allowed.
		{"zwj emoji then letter", "👨\u200d🔧x", []segmentEnd{{11, false}, {12, true}}},
		// LB20: CB (U+FFFC object replacement) allows a break on both sides.
		{"object replacement", "a￼b", []segmentEnd{{1, false}, {4, false}, {5, true}}},
		// U+FEFF WORD JOINER (WJ) forces no break on both sides (LB11).
		{"word joiner", "a\ufeffb", []segmentEnd{{5, true}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertSegments(t, c.input+" -> line segments", lineSegments(c.input), c.want)
		})
	}
}

func TestCJKLineHanAndEmojiOpportunities(t *testing.T) {
	// Han ideographs are ID; consecutive ID characters provide a break
	// opportunity between each pair (LB31 via the ID-related rules), which is
	// why CJK text can wrap anywhere. An emoji behaves like a wide
	// breakable unit.
	cases := []struct {
		name  string
		input string
		want  []segmentEnd
	}{
		{"han han latin", "你好a", []segmentEnd{{3, false}, {6, false}, {7, true}}},
		{"han emoji", "中😀", []segmentEnd{{3, false}, {7, true}}},
		{"mixed phrase", "你好 世界", []segmentEnd{{3, false}, {7, false}, {10, false}, {13, true}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertSegments(t, c.input+" -> line segments", lineSegments(c.input), c.want)
		})
	}
}
