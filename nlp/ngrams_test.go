package nlp

import (
	"strings"
	"testing"
)

var tokens = []string{
	"and", "or", "not", "xor", "lsh", "rsh", "shift", "foo", "bar", "baz",
}
var bigrams = [...][2]string{
	{"and", "or"}, {"or", "not"}, {"not", "xor"}, {"xor", "lsh"},
	{"lsh", "rsh"}, {"rsh", "shift"}, {"shift", "foo"}, {"foo", "bar"},
	{"bar", "baz"},
}
var trigrams = [...][3]string{
	{"and", "or", "not"}, {"or", "not", "xor"}, {"not", "xor", "lsh"},
	{"xor", "lsh", "rsh"}, {"lsh", "rsh", "shift"}, {"rsh", "shift", "foo"},
	{"shift", "foo", "bar"}, {"foo", "bar", "baz"},
}

func TestNGrams(t *testing.T) {
	for i, g := range NGrams(tokens, 1, 1) {
		if len(g) != 1 || g[0] != tokens[i] {
			t.Errorf("expected %s, got %v", tokens[i], g)
		}
	}

	for i, b := range NGrams(tokens, 2, 2) {
		expected := bigrams[i]
		if len(b) != 2 || b[0] != expected[0] || b[1] != expected[1] {
			t.Errorf("expected %v, got %v", expected, b)
		}
	}

	triIdx := 0
	for i, ng := range NGrams(tokens, 2, 3) {
		switch len(ng) {
		case 2:
			exp := bigrams[i-triIdx]
			if ng[0] != exp[0] || ng[1] != exp[1] {
				t.Errorf("expected %v, got %v", exp, ng)
			}
		case 3:
			exp := trigrams[triIdx]
			if ng[0] != exp[0] || ng[1] != exp[1] || ng[2] != exp[2] {
				t.Logf("%q %q", ng[0], exp[0])
				t.Errorf("expected %v, got %v", exp, ng)
			}
			triIdx++
		default:
			t.Errorf("expected bigrams and trigrams, not %v", ng)
		}
	}
}

// From https://en.wikipedia.org/wiki/Rabin%E2%80%93Karp_algorithm
var benchdata = strings.Split(
	`In computer science, the Rabin–Karp algorithm or Karp–Rabin algorithm is
     a string searching algorithm created by Richard M. Karp and Michael O.
     Rabin (1987) that uses hashing to find any one of a set of pattern
     strings in a text. For text of length n and p patterns of combined length
     m, its average and best case running time is O(n+m) in space O(p), but
     its worst-case time is O(nm). In contrast, the Aho–Corasick string
     matching algorithm has asymptotic worst-time complexity O(n+m) in space
     O(m).

     A practical application of the algorithm is detecting plagiarism. Given
     source material, the algorithm can rapidly search through a paper for
     instances of sentences from the source material, ignoring details such as
     case and punctuation. Because of the abundance of the sought strings,
     single-string searching algorithms are impractical.

     Rather than pursuing more sophisticated skipping,
     the Rabin–Karp algorithm seeks to speed up the testing of equality of the
     pattern to the substrings in the text by using a hash function. A hash
     function is a function which converts every string into a numeric value,
     called its hash value; for example, we might have hash("hello")=5. The
     algorithm exploits the fact that if two strings are equal, their hash
     values are also equal. Thus, it would seem all we have to do is compute
     the hash value of the substring we're searching for, and then look for a
     substring with the same hash value.

     However, there are two problems with this. First, because there are so
     many different strings, to keep the hash values small we have to assign
     some strings the same number. This means that if the hash values match,
     the strings might not match; we have to verify that they do, which can
     take a long time for long substrings. Luckily, a good hash function
     promises us that on most reasonable inputs, this won't happen too often,
     which keeps the average search time within an acceptable range.`,
	" ")

func BenchmarkNGrams(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for minN := 1; minN < 5; minN++ {
			for maxN := minN; maxN < 7; maxN++ {
				NGrams(benchdata, minN, maxN)
			}
		}
	}
}
