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
