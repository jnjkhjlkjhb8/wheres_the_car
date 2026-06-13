package main

import "testing"

func TestMakeThatSame(t *testing.T) {
	// A4 fix verification: trailing "01"/"02" suffixes are stripped for THB routes.
	cases := []struct {
		input, want string
	}{
		{"THB123401", "THB1234"}, // trailing "01" stripped
		{"THB123402", "THB1234"}, // trailing "02" stripped
		{"TPE1234", "TPE1234"},   // non-THB prefix: unchanged
	}
	for _, tc := range cases {
		got := makethatsame(tc.input)
		if got != tc.want {
			t.Errorf("makethatsame(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
