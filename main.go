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
	"io"
	"log/slog"
	"os"
	"strconv"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/akhenakh/maprender"
	"github.com/akhenakh/tiletea"
)

var (
	lat = flag.Float64("lat", 0, "latitude of the marker")
	lng = flag.Float64("lng", 0, "longitude of the marker")
)

func main() {
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
	switch {
	case *file != "":
		overlays, err := loadGeometryFile(*file)
		if err != nil {
			fmt.Fprintln(os.Stderr, "fatal:", err)
			os.Exit(1)
		}
		model = overlayModel(opts, overlays)
	case stdinIsPiped():
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "fatal:", err)
			os.Exit(1)
		}
		overlays, err := guessGeometry(data)
		if err != nil {
			fmt.Fprintln(os.Stderr, "fatal:", err)
			os.Exit(1)
		}
		model = overlayModel(opts, overlays)
	default:
		mlat, mlng, ok := coords()
		if !ok {
			usage()
			os.Exit(2)
		}
		if mlat < -90 || mlat > 90 {
			fmt.Fprintln(os.Stderr, "fatal: -lat must be between -90 and 90")
			os.Exit(2)
		}
		if mlng < -180 || mlng > 180 {
			fmt.Fprintln(os.Stderr, "fatal: -lng must be between -180 and 180")
			os.Exit(2)
		}
		model = newApp(tiletea.New(mlat, mlng, *zoom,
			append(opts, tiletea.WithMarker(mlat, mlng))...,
		), nil)
	}

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: ou -lat <lat> -lng <lng> [-zoom <zoom>]")
	fmt.Fprintln(os.Stderr, "       ou <lat> <lng> [-zoom <zoom>]")
	fmt.Fprintln(os.Stderr, "       ou -file <path>")
	fmt.Fprintln(os.Stderr, "       ou < geometry.geojson")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Open an interactive terminal map at a location, or display the")
	fmt.Fprintln(os.Stderr, "geometry of a GeoJSON, WKT, or WKB file or of data piped on stdin.")
}

// coords returns the marker position from either the -lat/-lng flags or, when
// no flag is passed, from two positional arguments that both parse as floats,
// e.g. "ou 48.8566 2.3522".
func coords() (float64, float64, bool) {
	if flag.NFlag() == 0 {
		return parseCoordsArgs(flag.Args())
	}
	if !isSet("lat") || !isSet("lng") {
		return 0, 0, false
	}
	return *lat, *lng, true
}

// parseCoordsArgs parses two positional arguments as a latitude and longitude.
func parseCoordsArgs(args []string) (float64, float64, bool) {
	if len(args) != 2 {
		return 0, 0, false
	}
	alat, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return 0, 0, false
	}
	alng, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return 0, 0, false
	}
	return alat, alng, true
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

// stdinIsPiped reports whether stdin is not a terminal, i.e. data is being
// piped or redirected in.
func stdinIsPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice == 0
}

// overlayModel builds the interactive map model for a set of parsed overlays.
func overlayModel(opts []tiletea.Option, overlays []maprender.Overlay) tea.Model {
	m := tiletea.New(0, 0, 0,
		append(opts,
			tiletea.WithOverlays(overlays...),
			tiletea.WithFitOverlays(),
		)...,
	)
	return newApp(m, describeOverlays(overlays))
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
