package hash

import (
    "math"
    "math/rand"
    "testing"
)

func TestCountMin(t *testing.T) {
    sketch := New(210, 1300)
    freq := make(map[uint32]uint32)

    rng := rand.New(rand.NewSource(42))
    for i := 0; i < 10000; i++ {
        h := rng.Uint32()
        sketch.Add(h, 1)
        freq[h] += 1
    }

    // XXX Should test if error is within margin with some probability.
    for k, v := range freq {
        if math.Abs(float64(sketch.Get(k) - v)) > 4 {
            t.Errorf("difference too big: got %d, want %d", sketch.Get(k), v)
        }
    }
}

func BenchmarkCountMinAdd(b *testing.B) {
    sketch := New(256, 256)

    rng := rand.New(rand.NewSource(42))
    for i := 0; i < b.N; i++ {
        for j := 0; j < 2000000; j++ {
            sketch.Add(rng.Uint32(), 1)
        }
    }
}
