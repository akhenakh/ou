package main

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
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
}

func newApp(m *tiletea.Map, data []datum) *app {
	return &app{mapModel: m, data: data}
}

func (a app) Init() tea.Cmd {
	return a.mapModel.Init()
}

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		m, cmd := a.mapModel.Update(a.sizeMsg())
		a.mapModel = m.(*tiletea.Map)
		return a, cmd

	case tea.KeyMsg:
		if msg.String() == "d" || msg.String() == "D" {
			a.showData = !a.showData
			m, cmd := a.mapModel.Update(a.sizeMsg())
			a.mapModel = m.(*tiletea.Map)
			return a, cmd
		}
	}

	m, cmd := a.mapModel.Update(msg)
	a.mapModel = m.(*tiletea.Map)
	return a, cmd
}

// sizeMsg returns the window size the map should render at. One row is
// reserved for the map's header (handled by tiletea) and one for the help bar,
// so the map renders at height-2 rows. When the data panel is shown the map is
// additionally rendered at half width.
func (a app) sizeMsg() tea.WindowSizeMsg {
	w := a.width
	if a.showData {
		w = a.width / 2
	}
	return tea.WindowSizeMsg{Width: w, Height: a.height - 1}
}

func (a app) View() tea.View {
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
	return out
}

// panelRows builds the list of rows shown in the data panel. An empty key marks
// the title row.
func (a app) panelRows() []datum {
	lat, lng := a.mapModel.Center()
	rows := []datum{
		{value: "Data  (d to hide)"},
		{key: "Center", value: fmt.Sprintf("%.4f, %.4f", lat, lng)},
		{key: "Zoom", value: strconv.Itoa(a.mapModel.Zoom())},
	}
	rows = append(rows, a.data...)
	return rows
}

// panel renders the area to the right of the map. It always emits a
// fixed-height block so the kitty sequence stays within the screen and the
// content height is stable across toggles: when rows is empty the right half is
// blanked out, which clears a previously shown panel. Field names are drawn
// with a highlighted background and values in the default style.
func (a app) panel(halfW int, rows []datum) string {
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
func (a app) helpBar() string {
	text := "arrows/hjkl: pan   +/-: zoom   d: data   q: quit"
	if a.showData {
		text = "arrows/hjkl: pan   +/-: zoom   d: hide data   q: quit"
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
