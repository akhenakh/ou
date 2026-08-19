package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/akhenakh/maprender"
	"github.com/peterstace/simplefeatures/carto"
	"github.com/peterstace/simplefeatures/geom"
)

// datum is a single key/value line shown in the data panel.
type datum struct {
	key   string
	value string
}

// describeOverlays builds the data panel rows for a set of overlays.
func describeOverlays(overlays []maprender.Overlay) []datum {
	var rows []datum
	for i, o := range overlays {
		if len(overlays) > 1 {
			rows = append(rows, datum{key: "Feature", value: strconv.Itoa(i + 1)})
		}
		rows = append(rows, describeGeometry(o.Geometry, o.Properties)...)
	}
	return rows
}

// describeGeometry computes the metadata for a single geometry: its type,
// bounding box, type-specific metrics (area, edges, vertex counts, ...) and any
// GeoJSON properties.
func describeGeometry(g geom.Geometry, props map[string]any) []datum {
	rows := []datum{{key: "Type", value: g.Type().String()}}

	if minXY, maxXY, ok := g.Envelope().MinMaxXYs(); ok {
		rows = append(rows, datum{
			key:   "Bounds",
			value: fmt.Sprintf("%.5f,%.5f → %.5f,%.5f", minXY.X, minXY.Y, maxXY.X, maxXY.Y),
		})
	}

	switch {
	case g.IsPoint():
		if xy, ok := g.MustAsPoint().XY(); ok {
			rows = append(rows, datum{key: "Longitude", value: fmt.Sprintf("%.6f", xy.X)})
			rows = append(rows, datum{key: "Latitude", value: fmt.Sprintf("%.6f", xy.Y)})
		}
	case g.IsMultiPoint():
		rows = append(rows, datum{key: "Points", value: strconv.Itoa(g.MustAsMultiPoint().NumPoints())})
	case g.IsLineString():
		ls := g.MustAsLineString()
		rows = append(rows, datum{key: "Vertices", value: strconv.Itoa(ls.Coordinates().Length())})
		rows = append(rows, datum{key: "Length", value: formatMeters(lengthM(ls.Coordinates()))})
	case g.IsMultiLineString():
		mls := g.MustAsMultiLineString()
		var total float64
		for i := 0; i < mls.NumLineStrings(); i++ {
			total += lengthM(mls.LineStringN(i).Coordinates())
		}
		rows = append(rows, datum{key: "Line strings", value: strconv.Itoa(mls.NumLineStrings())})
		rows = append(rows, datum{key: "Length", value: formatMeters(total)})
	case g.IsPolygon():
		p := g.MustAsPolygon()
		rows = append(rows, datum{key: "Edges", value: strconv.Itoa(ringEdges(p.ExteriorRing()))})
		rows = append(rows, datum{key: "Holes", value: strconv.Itoa(p.NumInteriorRings())})
		rows = append(rows, datum{key: "Area", value: formatArea(areaM2(g))})
	case g.IsMultiPolygon():
		mp := g.MustAsMultiPolygon()
		var rings, edges int
		for i := 0; i < mp.NumPolygons(); i++ {
			p := mp.PolygonN(i)
			rings += p.NumRings()
			edges += ringEdges(p.ExteriorRing())
		}
		rows = append(rows, datum{key: "Polygons", value: strconv.Itoa(mp.NumPolygons())})
		rows = append(rows, datum{key: "Rings", value: strconv.Itoa(rings)})
		rows = append(rows, datum{key: "Edges", value: strconv.Itoa(edges)})
		rows = append(rows, datum{key: "Area", value: formatArea(areaM2(g))})
	case g.IsGeometryCollection():
		rows = append(rows, datum{key: "Geometries", value: strconv.Itoa(len(g.MustAsGeometryCollection().Dump()))})
	}

	for _, k := range sortedKeys(props) {
		rows = append(rows, datum{key: k, value: formatProp(props[k])})
	}

	return rows
}

// ringEdges returns the number of distinct vertices of a closed ring, i.e. the
// number of its edges.
func ringEdges(ring geom.LineString) int {
	n := ring.Coordinates().Length()
	if n > 1 {
		n--
	}
	return n
}

// lengthM returns the length of a coordinate sequence in meters, computed as
// the sum of great-circle distances between consecutive vertices.
func lengthM(seq geom.Sequence) float64 {
	var total float64
	for i := 1; i < seq.Length(); i++ {
		total += haversineM(seq.GetXY(i-1), seq.GetXY(i))
	}
	return total
}

func haversineM(a, b geom.XY) float64 {
	const r = 6371008.8 // mean Earth radius in meters
	lat1, lon1 := a.Y*math.Pi/180, a.X*math.Pi/180
	lat2, lon2 := b.Y*math.Pi/180, b.X*math.Pi/180
	dLat := lat2 - lat1
	dLon := lon2 - lon1
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * r * math.Asin(math.Sqrt(h))
}

// areaM2 returns the area of a polygonal geometry in square meters using an
// equal-area cylindrical projection.
func areaM2(g geom.Geometry) float64 {
	proj := carto.NewLambertCylindricalEqualArea(carto.WGS84EllipsoidMeanRadiusM)
	return g.Area(geom.WithTransform(proj.Forward))
}

func formatArea(m2 float64) string {
	if m2 >= 1e6 {
		return fmt.Sprintf("%.3f km²", m2/1e6)
	}
	return fmt.Sprintf("%.0f m²", m2)
}

func formatMeters(m float64) string {
	if m >= 1000 {
		return fmt.Sprintf("%.2f km", m/1000)
	}
	return fmt.Sprintf("%.0f m", m)
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatProp(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	case nil:
		return ""
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}
