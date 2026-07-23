package gifmaker

import (
	"bytes"
	"image/gif"
	"testing"

	"github.com/sorfeb/profilegif/internal/scene"
)

func TestDefaultSceneLayout(t *testing.T) {
	sc := DefaultScene(Stats{Login: "sorfeb", TotalCommits: 4200, Followers: 88, Stars: 17})
	if len(sc.Layers) != 4 { // title + 3 stat widgets
		t.Fatalf("layer count: got %d want 4", len(sc.Layers))
	}
	// The commits widget must carry the fetched value through to the scene.
	sw, ok := sc.Layers[1].(*scene.StatWidget)
	if !ok {
		t.Fatalf("layer 1 not *StatWidget: %T", sc.Layers[1])
	}
	if sw.Value != 4200 {
		t.Errorf("commits value: got %d want 4200", sw.Value)
	}
}

func TestDefaultSceneThemed(t *testing.T) {
	dark := DefaultSceneTheme(Stats{Login: "x", TotalCommits: 100}, ThemeDark)
	light := DefaultSceneTheme(Stats{Login: "x", TotalCommits: 100}, ThemeLight)

	if !dark.Transparent || !light.Transparent {
		t.Error("themed default scenes should be transparent")
	}
	if dark.Ink == "" || light.Ink == "" {
		t.Error("themed scenes should set an ink color")
	}
	if dark.Ink == light.Ink {
		t.Errorf("dark and light ink should differ (both %q)", dark.Ink)
	}
	// Title should inherit the scene ink (no hardcoded color).
	if tx, ok := light.Layers[0].(*scene.TextElement); !ok || tx.Color != "" {
		t.Errorf("title should inherit ink (empty color), got %+v", light.Layers[0])
	}
}

func TestRenderProducesGIF(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, Stats{Login: "sorfeb", TotalCommits: 100, Followers: 5}); err != nil {
		t.Fatalf("render: %v", err)
	}
	g, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(g.Image) < 1 {
		t.Error("expected at least one frame")
	}
}
