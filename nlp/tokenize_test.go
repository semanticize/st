package nlp

import "testing"

func TestTokenize(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"Kahaani is a 2012 Indian mystery",
			[]string{"Kahaani", "is", "a", "<NUM>", "Indian", "mystery"}},
	} {
		got := Tokenize(c.in)
		if len(got) != len(c.want) {
			t.Errorf("len(tokenize(%q)) != len(%q)", c.in, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q != %q", got[i], c.want[i])
			}
		}
	}
}
