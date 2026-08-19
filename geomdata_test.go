package main

import (
	"testing"

	"github.com/akhenakh/maprender"
)

func TestDescribePolygon(t *testing.T) {
	ov, err := maprender.OverlayFromWKT("POLYGON((2.34 48.85, 2.36 48.85, 2.36 48.87, 2.34 48.87, 2.34 48.85))")
	if err != nil {
		t.Fatal(err)
	}
	rows := describeGeometry(ov.Geometry, map[string]any{"name": "Paris"})
	for _, r := range rows {
		t.Logf("%s = %s", r.key, r.value)
	}
}

func TestDescribeMultiPolygon(t *testing.T) {
	ov, err := maprender.OverlayFromGeoJSON([]byte(`{"type":"Feature","properties":{"foo":"bar","n":3},"geometry":{"type":"MultiPolygon","coordinates":[[[[0,0],[1,0],[1,1],[0,1],[0,0]]],[[[2,2],[3,2],[3,3],[2,3],[2,2]]]]}}`))
	if err != nil {
		t.Fatal(err)
	}
	rows := describeOverlays(ov)
	for _, r := range rows {
		t.Logf("%s = %s", r.key, r.value)
	}
}
