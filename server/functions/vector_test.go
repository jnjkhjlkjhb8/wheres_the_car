package main

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestZip(t *testing.T) {
	vec := []float32{1.0, -2.5, 0}
	got := zip(vec)
	if len(got) != len(vec)*4 {
		t.Fatalf("zip() length = %d, want %d", len(got), len(vec)*4)
	}

	first := math.Float32frombits(binary.LittleEndian.Uint32(got[0:4]))
	second := math.Float32frombits(binary.LittleEndian.Uint32(got[4:8]))
	third := math.Float32frombits(binary.LittleEndian.Uint32(got[8:12]))
	if first != vec[0] || second != vec[1] || third != vec[2] {
		t.Fatalf("zip() decoded = %v, want %v", []float32{first, second, third}, vec)
	}
}

func TestZipEmpty(t *testing.T) {
	got := zip(nil)
	if len(got) != 0 {
		t.Fatalf("zip() length = %d, want 0", len(got))
	}
}
