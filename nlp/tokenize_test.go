package nlp

import "testing"

func TestTokenize(t *testing.T) {
	for _, c := range [][]string{
		{"C1000 is een Nederlandse supermarktorganisatie,",
			"C1000", "is", "een", "Nederlandse", "supermarktorganisatie"},
		{"1981 (MCMLXXXI) was a common year starting on Thursday of the" +
			" Gregorian calendar (dominical letter D), the 1981st year",
			"1981", "MCMLXXXI", "was", "a", "common", "year", "starting", "on",
			"Thursday", "of", "the", "Gregorian", "calendar", "dominical",
			"letter", "D", "the", "1981st", "year"},
		{"In 2012, Fortune ranked IBM the No. 2 largest U.S. firm in terms of" +
			" number of employees (435,000 worldwide)",
			"In", "2012", "Fortune", "ranked", "IBM", "the", "No", "<NUM>",
			"largest", "U.S", "firm", "in", "terms", "of", "number", "of",
			"employees", "<NUM>", "worldwide"},
		{"The €1,000,000 test case", "The", "€", "<NUM>", "test", "case"},
		{"That's about US$1,080,000", "That's", "about", "US$", "<NUM>"},
	} {
		input, want := c[0], c[1:]
		tokens1 := Tokenize(input)
		tokens2, pos := TokenizePos(input)

		if len(tokens1) != len(want) || len(tokens2) != len(want) {
			t.Errorf("length mismatch: wanted %d, got %d and %d",
				len(want), len(tokens1), len(tokens2))
		}
		if len(pos) != len(tokens2) {
			t.Errorf("number of positions %d doesn't match number of tokens %d",
				len(pos), len(tokens2))
		}
		for i, tok := range want {
			if tokens1[i] != tok {
				t.Errorf("Tokenize error: %q != %q", tokens1[i], tok)
			}
			if tokens2[i] != tok {
				t.Errorf("Tokenize error: %q != %q", tokens2[i], tok)
			}
			atpos := input[pos[i][0]:pos[i][1]]
			if tok != "<NUM>" && tok != atpos {
				t.Errorf("text at %d:%d is %q, not %q",
					pos[i][0], pos[i][1], atpos, tok)
			}
		}
	}
}
