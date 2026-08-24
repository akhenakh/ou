package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/akhenakh/maprender"
	"github.com/akhenakh/ouca"
	"github.com/akhenakh/tiletea"
)

// ANSI styles used by the data panel and help bar.
const (
	titleStyle = "\x1b[1m"                     // bold title
	keyStyle   = "\x1b[48;2;58;68;98m\x1b[97m" // slate background + bright fg
	helpStyle  = "\x1b[48;2;38;42;50m"         // dark slate status bar
	resetStyle = "\x1b[0m"
)

type app struct {
	mapModel *tiletea.Map
	data     []datum
	showData bool
	width    int
	height   int

	// Set by the map's click callback during Update; consumed right after to
	// move the marker dot and trigger a re-render.
	clicked    bool
	clickedLat float64
	clickedLng float64

	// selectMode disables mouse tracking so the terminal can select and copy
	// text (e.g. the lat/lng in the status line); toggled with "m".
	selectMode bool

	// Marker position on screen; hasMarker reports whether one is displayed.
	hasMarker bool
	markerLat float64
	markerLng float64

	// ouca reverse geocoder, created lazily on the first "c" press, sharing
	// maprender's tile cache so matched tiles are not re-downloaded.
	matcher  *ouca.Index
	matching bool
	match    *ouca.Address
}

// matchResultMsg carries the outcome of an async closest-road lookup.
type matchResultMsg struct {
	addr *ouca.Address
	err  error
}

func newApp(m *tiletea.Map, data []datum) *app {
	a := &app{mapModel: m, data: data}
	m.SetClickCallback(func(lat, lng float64) {
		a.clicked = true
		a.clickedLat = lat
		a.clickedLng = lng
	})
	return a
}

// setMarker records that a marker is displayed at the given coordinates.
func (a *app) setMarker(lat, lng float64) {
	a.hasMarker = true
	a.markerLat = lat
	a.markerLng = lng
}

func (a *app) Init() tea.Cmd {
	return a.mapModel.Init()
}

func (a *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		m, cmd := a.mapModel.Update(a.sizeMsg())
		a.mapModel = m.(*tiletea.Map)
		return a, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "d", "D":
			a.showData = !a.showData
			m, cmd := a.mapModel.Update(a.sizeMsg())
			a.mapModel = m.(*tiletea.Map)
			return a, cmd
		case "m", "M":
			a.selectMode = !a.selectMode
			return a, nil
		case "c", "C":
			return a, a.matchClosest()
		}
	}

	m, cmd := a.mapModel.Update(msg)
	a.mapModel = m.(*tiletea.Map)

	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		// The click callback ran inside the map's Update; if it recorded a new
		// click, move the marker dot to the clicked coordinates and re-render.
		if a.clicked {
			a.mapModel.SetMarker(&a.clickedLat, &a.clickedLng)
			a.setMarker(a.clickedLat, a.clickedLng)
			a.mapModel.SetStatusExtra(fmt.Sprintf("Clicked: %.6f, %.6f", a.clickedLat, a.clickedLng))
			a.clicked = false
			return a, tea.Batch(cmd, a.mapModel.Refresh())
		}
	case matchResultMsg:
		a.matching = false
		if msg.err != nil {
			a.mapModel.SetStatusExtra("Match failed: " + msg.err.Error())
			return a, nil
		}
		a.match = msg.addr
		a.mapModel.SetStatusExtra(fmt.Sprintf("Closest road: %s (%.0f m)",
			roadLabel(msg.addr), msg.addr.Distance))
		return a, nil
	}
	return a, cmd
}

// matchClosest starts an async ouca lookup for the road closest to the marker.
// It does nothing when no marker is on screen or a lookup is already running.
func (a *app) matchClosest() tea.Cmd {
	if !a.hasMarker || a.matching {
		return nil
	}
	if a.matcher == nil {
		dir, err := maprender.DefaultCacheDir()
		if err != nil {
			a.mapModel.SetStatusExtra("Match failed: " + err.Error())
			return nil
		}
		a.matcher = ouca.NewIndex(ouca.WithCacheDir(dir))
	}
	a.matching = true
	lat, lng := a.markerLat, a.markerLng
	return func() tea.Msg {
		addr, err := a.matcher.Reverse(context.Background(), lat, lng)
		return matchResultMsg{addr: addr, err: err}
	}
}

// roadLabel returns a human readable name for a matched address, preferring
// the street name over the road reference and class.
func roadLabel(addr *ouca.Address) string {
	switch {
	case addr.Street != "":
		return addr.Street
	case addr.Ref != "":
		return addr.Ref
	case addr.Class != "":
		return addr.Class
	default:
		return "unnamed road"
	}
}

// sizeMsg returns the window size the map should render at. One row is
// reserved for the map's header (handled by tiletea) and one for the help bar,
// so the map renders at height-2 rows. When the data panel is shown the map is
// additionally rendered at half width.
func (a *app) sizeMsg() tea.WindowSizeMsg {
	w := a.width
	if a.showData {
		w = a.width / 2
	}
	return tea.WindowSizeMsg{Width: w, Height: a.height - 1}
}

func (a *app) View() tea.View {
	v := a.mapModel.View()

	// The map view is "<header>\n" + kitty sequence. Split them so we can keep
	// the header and position the image below it, with the data panel to the
	// right. The kitty sequence must stay the last thing in the content for the
	// renderer to emit it verbatim, and it must land within the screen bounds.
	header := v.Content
	kitty := ""
	if i := strings.Index(v.Content, "\x1b_G"); i >= 0 {
		header = v.Content[:i]
		kitty = v.Content[i:]
	}

	halfW := a.width / 2
	if halfW < 1 {
		halfW = 1
	}

	var b strings.Builder
	b.WriteString(header)

	if a.showData {
		b.WriteString(a.panel(halfW, a.panelRows()))
	} else {
		b.WriteString(a.panel(halfW, nil))
	}

	b.WriteString(a.helpBar())

	if kitty != "" {
		b.WriteString("\x1b[2;1H")
		b.WriteString(kitty)
	}

	out := tea.NewView(b.String())
	out.AltScreen = true
	// The map handles clicks; enable mouse tracking like tiletea's own View
	// does. In select mode it stays off so text can be selected and copied.
	if !a.selectMode {
		out.MouseMode = tea.MouseModeCellMotion
	}
	return out
}

// panelRows builds the list of rows shown in the data panel. An empty key marks
// the title row.
func (a *app) panelRows() []datum {
	lat, lng := a.mapModel.Center()
	rows := []datum{
		{value: "Data  (d to hide)"},
		{key: "Center", value: fmt.Sprintf("%.4f, %.4f", lat, lng)},
		{key: "Zoom", value: strconv.Itoa(a.mapModel.Zoom())},
	}
	rows = append(rows, a.data...)
	if a.match != nil {
		rows = append(rows,
			datum{key: "Road", value: roadLabel(a.match)},
			datum{key: "Snap", value: fmt.Sprintf("%.6f, %.6f", a.match.Lat, a.match.Lng)},
			datum{key: "Dist", value: fmt.Sprintf("%.0f m", a.match.Distance)},
		)
		if a.match.Bearing != 0 {
			rows = append(rows, datum{key: "Bearing", value: fmt.Sprintf("%.0f°", a.match.Bearing)})
		}
	}
	return rows
}

// panel renders the area to the right of the map. It always emits a
// fixed-height block so the kitty sequence stays within the screen and the
// content height is stable across toggles: when rows is empty the right half is
// blanked out, which clears a previously shown panel. Field names are drawn
// with a highlighted background and values in the default style.
func (a *app) panel(halfW int, rows []datum) string {
	rightW := a.width - halfW
	if rightW < 1 {
		rightW = 1
	}
	maxLines := a.height - 2
	if maxLines < 1 {
		maxLines = 1
	}

	keyW := 0
	for _, r := range rows {
		if n := runeLen(r.key); n > keyW {
			keyW = n
		}
	}
	valueW := rightW - keyW - 2 // "key: value"
	if valueW < 0 {
		valueW = 0
	}

	var b strings.Builder
	for i := 0; i < maxLines; i++ {
		b.WriteString(strings.Repeat(" ", halfW)) // transparent over the map
		if i < len(rows) {
			r := rows[i]
			if r.key == "" {
				t := truncate(r.value, rightW)
				b.WriteString(titleStyle + t + resetStyle)
				b.WriteString(strings.Repeat(" ", rightW-runeLen(t)))
			} else {
				key := r.key
				if runeLen(key) > keyW {
					key = truncate(key, keyW)
				}
				value := truncate(r.value, valueW)
				b.WriteString(keyStyle + key + strings.Repeat(" ", keyW-runeLen(key)) + resetStyle)
				b.WriteString(": " + value)
				b.WriteString(strings.Repeat(" ", valueW-runeLen(value)))
			}
		} else {
			b.WriteString(strings.Repeat(" ", rightW))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// helpBar renders the help line at the bottom of the app. It has no trailing
// newline so the kitty sequence stays within the last screen row.
func (a *app) helpBar() string {
	mouse := "m: select"
	if a.selectMode {
		mouse = "m: clicks"
	}
	closest := ""
	if a.hasMarker {
		closest = "   c: road"
	}
	text := "arrows/hjkl: pan   +/-: zoom   d: data" + closest + "   " + mouse + "   q: quit"
	if a.showData {
		text = "arrows/hjkl: pan   +/-: zoom   d: hide data" + closest + "   " + mouse + "   q: quit"
	}
	t := truncate(text, a.width-1)
	pad := a.width - 1 - runeLen(t)
	if pad < 0 {
		pad = 0
	}
	return helpStyle + t + strings.Repeat(" ", pad) + resetStyle
}

func truncate(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 0 {
		return ""
	}
	return string(r[:w])
}

func runeLen(s string) int {
	return len([]rune(s))
}
