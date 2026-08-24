package main

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/akhenakh/maprender"
	"github.com/akhenakh/ouca"
	"github.com/akhenakh/tiletea"
)

var colsRE = regexp.MustCompile(`\x1b_G[^\x1b]*c=(\d+)`)

func kittyCols(content string) string {
	m := colsRE.FindStringSubmatch(content)
	if m == nil {
		return "none"
	}
	return m[1]
}

func TestClickMovesMarker(t *testing.T) {
	ov, err := maprender.OverlayFromWKT("POINT(2.35 48.85)")
	if err != nil {
		t.Fatal(err)
	}
	m := tiletea.New(0, 0, 0,
		tiletea.WithOverlays(ov),
		tiletea.WithTileURLTemplate("http://127.0.0.1:1/{z}/{x}/{y}.pbf"),
	)
	var model tea.Model = newApp(m, nil)

	model, cmd := model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if r := cmd(); r != nil {
		model, _ = model.Update(r)
	}

	model, cmd = model.Update(tea.MouseClickMsg{X: 10, Y: 5, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatal("click produced no command, want a re-render cmd")
	}
	if r := cmd(); r != nil {
		model, _ = model.Update(r)
	}

	if !strings.Contains(model.View().Content, "Clicked:") {
		t.Fatal("status line does not report the clicked coordinates")
	}
}

func TestSplitResize(t *testing.T) {
	ov, err := maprender.OverlayFromWKT("POINT(2.35 48.85)")
	if err != nil {
		t.Fatal(err)
	}
	m := tiletea.New(0, 0, 0,
		tiletea.WithOverlays(ov),
		tiletea.WithFitOverlays(),
		tiletea.WithTileURLTemplate("http://127.0.0.1:1/{z}/{x}/{y}.pbf"),
	)
	var model tea.Model = newApp(m, describeOverlays([]maprender.Overlay{ov}))
	var cmd tea.Cmd

	// step applies a message and runs the returned command to completion,
	// mimicking a synchronous Bubble Tea loop.
	step := func(msg tea.Msg) {
		model, cmd = model.Update(msg)
		if cmd == nil {
			return
		}
		if r := cmd(); r != nil {
			model, _ = model.Update(r)
		}
	}

	step(tea.WindowSizeMsg{Width: 100, Height: 40})
	if got := kittyCols(model.View().Content); got != "100" {
		t.Fatalf("full-width map cols = %s, want 100", got)
	}
	if strings.Contains(model.View().Content, "d to hide") {
		t.Fatal("data panel shown in full-width mode, want hidden")
	}

	step(tea.KeyPressMsg(tea.Key{Text: "d", Code: 'd'}))
	if got := kittyCols(model.View().Content); got != "50" {
		t.Fatalf("split map cols = %s, want 50", got)
	}
	if !strings.Contains(model.View().Content, "d to hide") {
		t.Fatal("data panel not shown in split mode")
	}

	step(tea.KeyPressMsg(tea.Key{Text: "d", Code: 'd'}))
	if got := kittyCols(model.View().Content); got != "100" {
		t.Fatalf("full-width map cols = %s, want 100", got)
	}
	if strings.Contains(model.View().Content, "d to hide") {
		t.Fatal("data panel shown after toggling back, want hidden")
	}
}

func newTestApp(t *testing.T) *app {
	t.Helper()
	m := tiletea.New(0, 0, 0,
		tiletea.WithTileURLTemplate("http://127.0.0.1:1/{z}/{x}/{y}.pbf"),
	)
	a := newApp(m, nil)
	// Run the initial resize + render so the map leaves its loading state and
	// the status line appears in the view.
	model, cmd := a.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if r := cmd(); r != nil {
		model, _ = model.Update(r)
	}
	return model.(*app)
}

func TestClosestRoadWithoutMarker(t *testing.T) {
	a := newTestApp(t)

	model, cmd := a.Update(tea.KeyPressMsg(tea.Key{Text: "c", Code: 'c'}))
	if cmd != nil {
		t.Fatal("c with no marker on screen produced a command, want none")
	}
	if strings.Contains(model.View().Content, "Match") {
		t.Fatal("status line reports matching without a marker")
	}
}

func TestClosestRoadWithMarker(t *testing.T) {
	a := newTestApp(t)
	a.setMarker(48.8566, 2.3522)

	_, cmd := a.Update(tea.KeyPressMsg(tea.Key{Text: "c", Code: 'c'}))
	if cmd == nil {
		t.Fatal("c with a marker produced no command, want an async match")
	}

	// While the lookup is running, further presses are ignored.
	if _, cmd := a.Update(tea.KeyPressMsg(tea.Key{Text: "c", Code: 'c'})); cmd != nil {
		t.Fatal("c while matching produced a second command, want none")
	}

	model, _ := a.Update(matchResultMsg{addr: &ouca.Address{
		Street:   "Rue de Rivoli",
		Distance: 12.5,
		Lat:      48.8566,
		Lng:      2.3522,
	}})

	if !strings.Contains(model.View().Content, "Closest road: Rue de Rivoli") {
		t.Fatal("status line does not report the matched street")
	}

	// The data panel lists the match details when shown.
	model, _ = model.Update(tea.KeyPressMsg(tea.Key{Text: "d", Code: 'd'}))
	v := model.View().Content
	if !strings.Contains(v, "Rue de Rivoli") || !strings.Contains(v, "Snap") {
		t.Fatal("data panel does not show the match details")
	}
}

func TestClosestRoadFailure(t *testing.T) {
	a := newTestApp(t)
	a.setMarker(48.8566, 2.3522)

	_, cmd := a.Update(tea.KeyPressMsg(tea.Key{Text: "c", Code: 'c'}))
	if r := cmd(); r != nil {
		a.Update(r)
	}
	model, _ := a.Update(matchResultMsg{err: errors.New("no road found")})

	if !strings.Contains(model.View().Content, "Match failed") {
		t.Fatal("status line does not report the match failure")
	}
}
