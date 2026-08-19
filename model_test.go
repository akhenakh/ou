package main

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/akhenakh/maprender"
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
