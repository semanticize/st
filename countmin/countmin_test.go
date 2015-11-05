package countmin

import (
	"math"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/semanticize/st/nlp"
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
	freq := make(map[string]uint32)

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 10000; i++ {
		key := strconv.Itoa(rng.Int())
		sketch.Add([]byte(key))
		freq[key] += 1
	}

	// XXX Should test if error is within margin with some probability.
	for k, v := range freq {
		if got := sketch.Get([]byte(k)); math.Abs(float64(got-v)) > 4 {
			t.Errorf("difference too big: got %d, want %d", got, v)
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
	for _, x := range []int{216, 121, 7, 1, 834, 8015, 15, 1266, 162, 16} {
		cm.Add(itob(x))
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

func TestCounts(t *testing.T) {
	nrows := 14
	a, _ := New(nrows, 51)
	a.Add([]byte("2613621"))
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

func TestNGram(t *testing.T) {
	cm, _ := New(16, 1024)
	tokens := strings.Split("foo bar baz quux bla barney fred", " ")
	ngrams := nlp.NGrams(tokens, 1, 5)
	for _, ng := range ngrams {
		cm.AddNGram(ng)
	}
	counts := make([]uint32, len(ngrams))
	for i, ng := range ngrams {
		counts[i] = cm.GetNGram(ng)
		if counts[i] != 1 {
			t.Errorf("expected count=1, got %d", counts[i])
		}
	}
	for i, ng := range ngrams {
		// This test is here because it's too easy to reuse the hash.Hash
		// instances, changing the hash values for subsequent runs.
		if cm.GetNGram(ng) != counts[i] {
			t.Errorf("count estimate for %v not deterministic", ng)
		}
	}
}

func TestCountMinSum(t *testing.T) {
	a, _ := New(25, 126)
	b, _ := New(25, 126)
	sum, _ := New(25, 126)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 2000; i++ {
		key := itob(rng.Int())

		a.Add(key)

		b.Add(key)
		b.Add(key)

		sum.Add(key)
		sum.Add(key)
		sum.Add(key)
	}
	a.Sum(b)

	for i := 0; i < 126; i++ {
		key := itob(i)
		if a.Get(key) != sum.Get(key) {
			t.Errorf("expected %d, got %d", sum.Get(key), a.Get(key))
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
	b.StopTimer()

	sketch, _ := New(16, 256)

	rng := rand.New(rand.NewSource(42))
	var keys [313][]byte
	for i := range keys {
		keys[i] = itob(rng.Int())
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 20000; j++ {
			sketch.Add(keys[j%len(keys)])
		}
	}
}

func itob(i int) []byte { return []byte(strconv.Itoa(i)) }

func randstr(rng *rand.Rand) string {
	return strconv.Itoa(rng.Int())
}
