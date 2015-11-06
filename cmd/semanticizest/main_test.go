package main

import (
	"bufio"
	"strings"
	"testing"
)

func TestSplitPara(t *testing.T) {
	r := strings.NewReader("first paragraph\n\nsecond\n")
	scanner := bufio.NewScanner(r)
	scanner.Split(splitPara)

	for i, expected := range []string{"first paragraph", "second"} {
		if scanner.Scan() == false {
			t.Fatalf("expected two paragraphs, got %d", i)
		}
		if text := scanner.Text(); text != expected {
			t.Errorf("expected %q, got %q", expected, text)
		}
	}
	if scanner.Scan() {
		t.Errorf("expected two paragraphs, got a third: %q\n", scanner.Text())
	}
}
