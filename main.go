// Command ou opens an interactive terminal map showing a location or the
// geometry of a file.
//
// "où" is French for "where".
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/akhenakh/maprender"
	"github.com/akhenakh/tiletea"
)

func main() {
	lat := flag.Float64("lat", 0, "latitude of the marker")
	lng := flag.Float64("lng", 0, "longitude of the marker")
	zoom := flag.Int("zoom", 14, "initial zoom level")
	file := flag.String("file", "", "geometry file to display (GeoJSON, WKT, WKB)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if os.Getenv("DEBUG") != "" {
		f, err := tea.LogToFile("debug.log", "ou")
		if err != nil {
			fmt.Fprintln(os.Stderr, "fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
		logger = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	opts := []tiletea.Option{tiletea.WithLogger(logger)}

	var model tea.Model
	if *file != "" {
		overlays, err := loadGeometryFile(*file)
		if err != nil {
			fmt.Fprintln(os.Stderr, "fatal:", err)
			os.Exit(1)
		}
		m := tiletea.New(0, 0, 0,
			append(opts,
				tiletea.WithOverlays(overlays...),
				tiletea.WithFitOverlays(),
			)...,
		)
		model = newApp(m, describeOverlays(overlays))
	} else {
		if !isSet("lat") || !isSet("lng") {
			usage()
			os.Exit(2)
		}
		if *lat < -90 || *lat > 90 {
			fmt.Fprintln(os.Stderr, "fatal: -lat must be between -90 and 90")
			os.Exit(2)
		}
		if *lng < -180 || *lng > 180 {
			fmt.Fprintln(os.Stderr, "fatal: -lng must be between -180 and 180")
			os.Exit(2)
		}
		model = tiletea.New(*lat, *lng, *zoom,
			append(opts, tiletea.WithMarker(*lat, *lng))...,
		)
	}

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: ou -lat <lat> -lng <lng> [-zoom <zoom>]")
	fmt.Fprintln(os.Stderr, "       ou -file <path>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Open an interactive terminal map at a location, or display the")
	fmt.Fprintln(os.Stderr, "geometry of a GeoJSON, WKT, or WKB file.")
}

// isSet reports whether the named flag was explicitly provided on the command
// line.
func isSet(name string) bool {
	set := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

// loadGeometryFile reads a file and parses its geometry into overlays, guessing
// the format (GeoJSON, WKT, or WKB) from its content.
func loadGeometryFile(path string) ([]maprender.Overlay, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return guessGeometry(data)
}

func guessGeometry(data []byte) ([]maprender.Overlay, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty geometry file")
	}

	if trimmed[0] == '{' || trimmed[0] == '[' {
		var probe struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(trimmed, &probe); err == nil && probe.Type != "" {
			return maprender.OverlayFromGeoJSON(trimmed)
		}
	}

	if utf8.Valid(trimmed) {
		if ov, err := maprender.OverlayFromWKT(string(trimmed)); err == nil {
			return []maprender.Overlay{ov}, nil
		}
	}

	if ov, err := maprender.OverlayFromWKB(trimmed); err == nil {
		return []maprender.Overlay{ov}, nil
	}

	return nil, fmt.Errorf("unrecognized geometry format (expected GeoJSON, WKT, or WKB)")
}
