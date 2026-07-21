package scene

import (
	"encoding/json"
	"testing"
)

func sampleScene() *Scene {
	s := NewScene(800, 400)
	s.Add(NewBackground("bg.png", 800, 400))
	s.Add(NewText(Rect{X: 20, Y: 30, W: 300, H: 60}, "hello"))
	s.Add(NewStatWidget(Rect{X: 400, Y: 200, W: 200, H: 80}, MetricCommits, "sorfeb"))
	return s
}

func TestJSONRoundTrip(t *testing.T) {
	orig := sampleScene()

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Scene
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.W != orig.W || got.H != orig.H || got.FPS != orig.FPS || got.DurationMs != orig.DurationMs {
		t.Errorf("scene meta mismatch: got %+v", got)
	}
	if len(got.Layers) != len(orig.Layers) {
		t.Fatalf("layer count: got %d want %d", len(got.Layers), len(orig.Layers))
	}

	// Concrete types + z-order must survive the round trip.
	wantKinds := []string{KindBackground, KindText, KindStatWidget}
	for i, el := range got.Layers {
		if el.Kind() != wantKinds[i] {
			t.Errorf("layer %d kind: got %q want %q", i, el.Kind(), wantKinds[i])
		}
		if el.Bounds() != orig.Layers[i].Bounds() {
			t.Errorf("layer %d bounds: got %+v want %+v", i, el.Bounds(), orig.Layers[i].Bounds())
		}
	}

	txt, ok := got.Layers[1].(*TextElement)
	if !ok {
		t.Fatalf("layer 1 not *TextElement: %T", got.Layers[1])
	}
	if txt.Text != "hello" {
		t.Errorf("text content: got %q want %q", txt.Text, "hello")
	}

	sw, ok := got.Layers[2].(*StatWidget)
	if !ok {
		t.Fatalf("layer 2 not *StatWidget: %T", got.Layers[2])
	}
	if sw.Metric != MetricCommits || sw.Login != "sorfeb" {
		t.Errorf("stat widget fields: got metric=%q login=%q", sw.Metric, sw.Login)
	}
}

func TestUnmarshalUnknownKind(t *testing.T) {
	bad := `{"w":10,"h":10,"fps":1,"durationMs":1000,"layers":[{"kind":"nope","rect":{"x":0,"y":0,"w":1,"h":1}}]}`
	var s Scene
	if err := json.Unmarshal([]byte(bad), &s); err == nil {
		t.Fatal("expected error for unknown kind, got nil")
	}
}

func TestTopmostAt(t *testing.T) {
	s := NewScene(100, 100)
	s.Add(NewText(Rect{X: 0, Y: 0, W: 100, H: 100}, "bottom")) // index 0
	s.Add(NewText(Rect{X: 10, Y: 10, W: 20, H: 20}, "top"))    // index 1, overlaps

	// Point inside both → topmost (last) wins.
	i, el := s.TopmostAt(15, 15)
	if i != 1 {
		t.Errorf("overlap: got index %d want 1", i)
	}
	if tx := el.(*TextElement); tx.Text != "top" {
		t.Errorf("overlap: got %q want top", tx.Text)
	}

	// Point only inside the bottom element.
	if i, _ := s.TopmostAt(90, 90); i != 0 {
		t.Errorf("bottom only: got index %d want 0", i)
	}

	// Point outside everything.
	if i, el := s.TopmostAt(200, 200); i != -1 || el != nil {
		t.Errorf("miss: got (%d,%v) want (-1,nil)", i, el)
	}
}

func TestRaiseLower(t *testing.T) {
	s := NewScene(10, 10)
	a := NewText(Rect{}, "a")
	b := NewText(Rect{}, "b")
	c := NewText(Rect{}, "c")
	s.Add(a)
	s.Add(b)
	s.Add(c) // order: a,b,c

	if ni := s.Raise(0); ni != 1 { // a up → b,a,c
		t.Errorf("raise returned %d want 1", ni)
	}
	if s.Layers[0] != Element(b) || s.Layers[1] != Element(a) {
		t.Errorf("after raise order wrong: %q,%q", s.Layers[0].(*TextElement).Text, s.Layers[1].(*TextElement).Text)
	}

	if ni := s.Raise(2); ni != 2 { // top can't go higher
		t.Errorf("raise top returned %d want 2", ni)
	}
	if ni := s.Lower(0); ni != 0 { // bottom can't go lower
		t.Errorf("lower bottom returned %d want 0", ni)
	}
}

func TestRemove(t *testing.T) {
	s := sampleScene()
	n := len(s.Layers)
	s.Remove(1)
	if len(s.Layers) != n-1 {
		t.Fatalf("remove: got %d layers want %d", len(s.Layers), n-1)
	}
	if s.Layers[1].Kind() != KindStatWidget {
		t.Errorf("after remove, layer 1 kind: got %q", s.Layers[1].Kind())
	}
	s.Remove(99) // out of range → no-op, no panic
}

func TestFrameCount(t *testing.T) {
	s := NewScene(10, 10)
	s.FPS, s.DurationMs = 15, 2000
	if got := s.FrameCount(); got != 30 {
		t.Errorf("frame count: got %d want 30", got)
	}
	s.FPS, s.DurationMs = 0, 0
	if got := s.FrameCount(); got != 1 {
		t.Errorf("degenerate frame count: got %d want 1", got)
	}
}

func TestRectContains(t *testing.T) {
	r := Rect{X: 5, Y: 5, W: 10, H: 10} // covers [5,15) × [5,15)
	cases := []struct {
		x, y int
		want bool
	}{
		{5, 5, true}, {14, 14, true}, {15, 15, false}, {4, 10, false}, {10, 10, true},
	}
	for _, c := range cases {
		if got := r.Contains(c.x, c.y); got != c.want {
			t.Errorf("Contains(%d,%d): got %v want %v", c.x, c.y, got, c.want)
		}
	}
}
