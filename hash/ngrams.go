package hash

import "hash/fnv"

// Returns hashes of all n-grams in tokens with 1 ≤ n ≤ maxn.
//
// The hash function used is FNV-32.
func NGrams(tokens []string, maxn int) []uint32 {
    h := fnv.New32()
    out := make([]uint32, 0, len(tokens)*maxn)

    for i := 0; i < len(tokens); i++ {
        h.Reset()
        h.Write([]byte(tokens[i]))
        out = append(out, h.Sum32())

        for n := 1; n < min(maxn, len(tokens) - i); n++ {
            h.Write([]byte("\x00"))
            h.Write([]byte(tokens[i+n]))
            out = append(out, h.Sum32())
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
