package main

import "testing"

func TestParseCoordsArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		lat  float64
		lng  float64
		ok   bool
	}{
		{"two floats", []string{"48.8566", "2.3522"}, 48.8566, 2.3522, true},
		{"negative lng", []string{"48.8", "-2.2"}, 48.8, -2.2, true},
		{"no args", nil, 0, 0, false},
		{"one arg", []string{"48.8"}, 0, 0, false},
		{"three args", []string{"48.8", "-2.2", "extra"}, 0, 0, false},
		{"non-float lat", []string{"paris", "2.3522"}, 0, 0, false},
		{"non-float lng", []string{"48.8566", "east"}, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lat, lng, ok := parseCoordsArgs(tt.args)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && (lat != tt.lat || lng != tt.lng) {
				t.Fatalf("coords = (%v, %v), want (%v, %v)", lat, lng, tt.lat, tt.lng)
			}
		})
	}
}
