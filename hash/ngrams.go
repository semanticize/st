package hash

import "hash/fnv"

// Returns hashes of all N-grams in tokens with minN ≤ N ≤ maxN.
//
// The hash of an N-gram is the hash of the tokens, joined by NUL characters.
// We assume these don't occur in text. The hash function used is FNV-32.
func NGrams(tokens []string, minN, maxN int) []uint32 {
	h := fnv.New32()
	out := make([]uint32, 0, len(tokens)*maxN)

	for i := 0; i < len(tokens); i++ {
		h.Reset()
		h.Write([]byte(tokens[i]))
		if minN == 1 {
			out = append(out, h.Sum32())
		}

		for n := 2; n <= min(maxN, len(tokens)-i); n++ {
			h.Write([]byte("\x00"))
			h.Write([]byte(tokens[i+n-1]))
			if n >= minN {
				out = append(out, h.Sum32())
			}
		}
	}
	return out
}

// Hash of n-gram at position Start through End (exclusive) in the input.
//
// The length of input n-gram is (End-Start).
type HashPos struct {
	Hash       uint32
	Start, End int
}

// Like NGrams, but returns position info and minN is hardcoded to one.
func NGramsPos(tokens []string, maxN int) []HashPos {
	h := fnv.New32()
	out := make([]HashPos, 0, len(tokens)*maxN)

	for i := 0; i < len(tokens); i++ {
		h.Reset()
		h.Write([]byte(tokens[i]))
		out = append(out, HashPos{h.Sum32(), i, i + 1})

		for n := 2; n <= min(maxN, len(tokens)-i); n++ {
			h.Write([]byte("\x00"))
			h.Write([]byte(tokens[i+n-1]))
			out = append(out, HashPos{h.Sum32(), i, i + n})
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
