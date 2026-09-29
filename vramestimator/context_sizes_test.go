package vramestimator

import (
	"slices"
	"testing"
)

func TestGenerateContextSizesStopsAtRequestedTop(t *testing.T) {
	// --context 4k parses to 4096. The old list dropped 4096 and showed 8192.
	got := generateContextSizes(4096)
	want := []int{2048, 4096}
	if !slices.Equal(got, want) {
		t.Fatalf("generateContextSizes(4096) = %v, want %v", got, want)
	}

	got = generateContextSizes(8192)
	want = []int{2048, 8192}
	if !slices.Equal(got, want) {
		t.Fatalf("generateContextSizes(8192) = %v, want %v", got, want)
	}

	got = generateContextSizes(65536)
	want = []int{2048, 8192, 16384, 32768, 65536}
	if !slices.Equal(got, want) {
		t.Fatalf("generateContextSizes(65536) = %v, want %v", got, want)
	}

	got = generateContextSizes(20000)
	want = []int{2048, 8192, 16384, 20000}
	if !slices.Equal(got, want) {
		t.Fatalf("generateContextSizes(20000) = %v, want %v", got, want)
	}
}
