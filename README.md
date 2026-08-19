# où

`où` (French for "where") is a command-line tool that opens an interactive map in
your terminal and displays a location or geometry.

It is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), using
the [tiletea](https://github.com/akhenakh/tiletea) map component, itself based on
[maprender](https://github.com/akhenakh/maprender).

## Install

```sh
go install github.com/akhenakh/ou@latest
```

## Usage

Open the map with a marker at a specific location:

```sh
ou -lat 48.8566 -lng 2.3522
```

Display a file's geometry. The format is guessed from its content (GeoJSON,
WKT, WKB):

```sh
ou -file /path/to/geometry.geojson
```

Or pipe geometry in on stdin; it's guessed the same way as `-file`:

```sh
cat /path/to/geometry.geojson | ou
```

## Controls

| Keys | Action |
| --- | --- |
| Arrow keys / `h` `j` `k` `l` | Pan |
| `+` / `=` | Zoom in |
| `-` | Zoom out |
| `d` | Toggle the data panel (geometry mode only) |
| `q` / `ctrl+c` | Quit |

When viewing a geometry, press `d` to split the view in half vertically and show
a data panel listing the geometry's metadata — its type, bounding box, computed
area (polygons), number of edges and holes, number of polygons (multipolygons),
length (lines), and any GeoJSON properties — alongside the map.

## License

MIT
