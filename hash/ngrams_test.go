package hash

import (
	"hash/fnv"
	"strings"
	"testing"
)

func hashNGram(ngram []string) uint32 {
	h := fnv.New32()
	h.Write([]byte(ngram[0]))
	for _, w := range ngram[1:] {
		h.Write([]byte("\x00"))
		h.Write([]byte(w))
	}
	return h.Sum32()
}

// Generate n-grams.
func ngrams(tokens []string, minN, maxN int) [][]string {
	out := make([][]string, 0)
	for i := range tokens {
		for n := minN; n <= min(maxN, len(tokens)-i); n++ {
			out = append(out, tokens[i:i+n])
		}
	}
	return out
}

// Test that our n-gram hasher matches the naïve implementation.
func TestNGrams(t *testing.T) {
	tokens := strings.Split("and or not xor lsh rsh shift foo bar baz", " ")
	for minN := 1; minN < 4; minN++ {
		for maxN := minN; maxN < 6; maxN++ {

			hashes := NGrams(tokens, minN, maxN)
			grams := ngrams(tokens, minN, maxN)
			if len(hashes) != len(grams) {
				t.Errorf("length mismatch, %d != %d (%d, %d)",
					len(hashes), len(grams), minN, maxN)
			} else {
				for i, gram := range grams {
					if hashes[i] != hashNGram(gram) {
						t.Errorf("expected %d, got %d (%d, %d)",
							hashes[i], hashNGram(gram), minN, maxN)
					}
				}
			}
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
	for minN := 1; minN < 5; minN++ {
		for maxN := minN; maxN < 70; maxN++ {
			NGrams(benchdata, minN, maxN)
			// Uncomment to benchmark the naïve way of doing this (~2.5× slower).
			/*
			   for _, gram := range ngrams(benchdata, maxN) {
			       hashNGram(gram)
			   }
			*/
		}
	}
}
