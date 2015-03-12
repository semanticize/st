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

        for n := 2; n <= min(maxN, len(tokens) - i); n++ {
            h.Write([]byte("\x00"))
            h.Write([]byte(tokens[i+n-1]))
            if n >= minN {
                out = append(out, h.Sum32())
            }
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
