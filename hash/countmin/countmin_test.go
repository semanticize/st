package countmin

import (
	"math"
	"math/rand"
	"testing"
)

func expectError(t *testing.T, sketch *Sketch, err error) {
	if err == nil {
		t.Error("expected an error, got nil")
	} else if sketch != nil {
		t.Errorf("non-nil *Sketch despite error %q", err)
	}
}

func TestCountMin(t *testing.T) {
	sketch, _ := New(210, 1300)
	sketch1, _ := New(210, 1300)
	freq := make(map[uint32]uint32)

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 10000; i++ {
		h := rng.Uint32()
		sketch.Add(h, 1)
		sketch1.Add1(h)
		freq[h] += 1
	}

	// XXX Should test if error is within margin with some probability.
	for k, v := range freq {
		if math.Abs(float64(sketch.Get(k)-v)) > 4 {
			t.Errorf("difference too big: got %d, want %d", sketch.Get(k), v)
		}
		if sketch.Get(k) != sketch1.Get(k) {
			t.Errorf("different counts for Add and Add1: %d, %d",
				sketch.Get(k), sketch1.Get(k))
		}
	}

	var err error
	check := func() {
		expectError(t, sketch, err)
	}

	sketch, err = New(0, 1)
	check()

	sketch, err = New(1, -1)
	check()

	sketch, err = New(MaxRows+1, 10)
	check()
}

func TestNewFromCounts(t *testing.T) {
	var rows [][]uint32
	checkerr := func() {
		cm, err := NewFromCounts(rows)
		expectError(t, cm, err)
	}

	checkerr()

	rows = make([][]uint32, 3)
	checkerr()

	rows[0] = make([]uint32, 4)
	rows[1] = make([]uint32, 4)
	checkerr()

	rows[2] = make([]uint32, 3)
	checkerr()

	rows[2] = make([]uint32, 4)
	cm, err := NewFromCounts(rows)
	if err != nil {
		t.Fatalf("unexpected error: %q", err)
	} else if cm == nil {
		t.Fatal("got nil *Sketch but no error")
	}
}

func TestNewFromProb(t *testing.T) {
	ε, δ := 0.001, .00001
	cm, _ := NewFromProb(ε, δ)
	if nrows := len(cm.Counts()); nrows != 12 {
		t.Errorf("expected %d rows, got %d", 12, nrows)
	}
	if ncols := len(cm.Counts()[0]); ncols != 2719 {
		t.Errorf("expected %d rows, got %d", 2719, ncols)
	}
}

func TestCopy(t *testing.T) {
	cm, _ := New(5, 8)
	for _, x := range []uint32{216, 121, 7, 1, 834, 8015, 15, 1266, 162, 16} {
		cm.Add1(x)
	}
	clone := cm.Copy()
	for i := 0; i < cm.NRows(); i++ {
		for j := 0; j < cm.NCols(); j++ {
			if old, cpy := cm.rows[i][j], clone.rows[i][j]; cpy != old {
				t.Errorf("Copy() not equal to original at %d, %d: %d != %d",
					i, j, cpy, old)
			}
		}
	}
}

// Test inserting more than 2**32 occurrences of an event.
func TestWraparound(t *testing.T) {
	a, _ := New(10, 4)
	check := func(expected uint32) {
		if got := a.Get(1); got != expected {
			t.Errorf("expected %d, got %d", expected, got)
		}
	}

	a.AddCU(1, 3e9)
	check(3e9)
	a.Add(1, 3e9)
	check(math.MaxUint32)
	a.Add1(1)
	check(math.MaxUint32)
	a.AddCU(1, 3)
	check(math.MaxUint32)

	b, _ := New(10, 4)
	b.Add(1, 4e9)
	a.Sum(b)
	check(math.MaxUint32)
}

func TestCounts(t *testing.T) {
	nrows := 14
	a, _ := New(nrows, 51)
	a.Add1(2613621)
	rows := a.Counts()

	var total int
	for _, row := range rows {
		for _, c := range row {
			total += int(c)
		}
	}
	if total != nrows {
		t.Errorf("expected %d, got %d", nrows, total)
	}
}

func TestCountMinSum(t *testing.T) {
	a, _ := New(25, 126)
	b, _ := New(25, 126)
	sum, _ := New(25, 126)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 2000; i++ {
		h := rng.Uint32()
		a.Add1(h)
		b.Add(h, h%100)
		sum.Add(h, h%100+1)
	}
	a.Sum(b)

	for i := uint32(0); i < 126; i++ {
		if a.Get(i) != sum.Get(i) {
			t.Errorf("expected %d, got %d", sum.Get(i), a.Get(i))
		}
	}

	b, _ = New(25, 127)
	err := a.Sum(b)
	if err == nil {
		t.Error("expected an error, got nil")
	}
	b, _ = New(26, 126)
	err = a.Sum(b)
	if err == nil {
		t.Error("expected an error, got nil")
	}
}

func BenchmarkCountMinAdd(b *testing.B) {
	sketch, _ := New(256, 256)

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < b.N; i++ {
		for j := 0; j < 20000; j++ {
			sketch.Add(rng.Uint32(), uint32(rng.Int31n(1000)))
		}
	}
}

func BenchmarkCountMinAdd1(b *testing.B) {
	sketch, _ := New(256, 256)

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < b.N; i++ {
		for j := 0; j < 2000000; j++ {
			sketch.Add1(rng.Uint32())
		}
	}
}
