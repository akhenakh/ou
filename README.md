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

> **Note:** `ou` depends on
> [`github.com/akhenakh/tiletea`](https://github.com/akhenakh/tiletea), which is
> currently a private repository. Building requires access to it (e.g. a
> configured `GOPRIVATE=github.com/akhenakh`); it will be opened to the public
> later.

## Usage

Open the map with a marker at a specific location:

```sh
ou -lat 48.8566 -lng 2.3522
```

Open a file and display its geometry. The file type is guessed from its content
(GeoJSON, WKT, WKB, ...):

```sh
ou -file /path/to/geometry.geojson
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
