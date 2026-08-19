package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/akhenakh/tiletea"
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

func (a app) sizeMsg() tea.WindowSizeMsg {
	w := a.width
	if a.showData {
		w = a.width / 2
	}
	return tea.WindowSizeMsg{Width: w, Height: a.height}
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
		b.WriteString(a.panel(halfW, a.panelLines()))
	} else {
		b.WriteString(a.panel(halfW, nil))
	}

	if kitty != "" {
		b.WriteString("\x1b[2;1H")
		b.WriteString(kitty)
	}

	out := tea.NewView(b.String())
	out.AltScreen = true
	return out
}

// panelLines builds the list of lines shown in the data panel.
func (a app) panelLines() []string {
	lat, lng := a.mapModel.Center()
	lines := []string{
		"Data  (d to hide)",
		fmt.Sprintf("Center: %.4f, %.4f", lat, lng),
		fmt.Sprintf("Zoom: %d", a.mapModel.Zoom()),
	}
	for _, d := range a.data {
		lines = append(lines, fmt.Sprintf("%s: %s", d.key, d.value))
	}
	return lines
}

// panel renders the area to the right of the map. It always emits a
// fixed-height block (header + panel + kitty = height-1 rows) so the kitty
// sequence stays within the screen and the content height is stable across
// toggles: when lines is empty the right half is blanked out, which clears a
// previously shown panel.
func (a app) panel(halfW int, lines []string) string {
	rightW := a.width - halfW
	if rightW < 1 {
		rightW = 1
	}
	maxLines := a.height - 2
	if maxLines < 1 {
		maxLines = 1
	}

	var b strings.Builder
	for i := 0; i < maxLines; i++ {
		text := ""
		if i < len(lines) {
			text = truncate(lines[i], rightW)
		}

		pad := rightW - runeLen(text)
		if pad < 0 {
			pad = 0
		}

		b.WriteString(strings.Repeat(" ", halfW)) // transparent over the map
		b.WriteString(text)
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString("\n")
	}
	return b.String()
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
