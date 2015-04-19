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
	} {
		input, want := c[0], c[1:]
		got := Tokenize(input)
		if len(got) != len(want) {
			t.Errorf("len(tokenize(%q)) != len(%q)", input, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%q != %q", got[i], want[i])
			}
		}
	}
}
