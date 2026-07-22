package scene

import (
	"encoding/json"
	"fmt"
)

// Element kind discriminators (the JSON "kind" tag values).
const (
	KindBackground = "background"
	KindText       = "text"
	KindImage      = "image"
	KindStatWidget = "statWidget"
)

// Fit modes control how an image is mapped into its Rect.
const (
	FitCover   = "cover"   // fill the rect, cropping overflow (default)
	FitContain = "contain" // fit entirely inside, letterboxing
	FitStretch = "stretch" // distort to exactly fill
)

// Background is a full-canvas (or any-rect) image/GIF drawn beneath everything else.
type Background struct {
	base
	Path string `json:"path"`
	Fit  string `json:"fit"`
}

func (Background) Kind() string { return KindBackground }

// NewBackground makes a background covering the whole w×h canvas.
func NewBackground(path string, w, h int) *Background {
	return &Background{base: base{Rect: Rect{X: 0, Y: 0, W: w, H: h}}, Path: path, Fit: FitCover}
}

// TextElement is a run of text with a size and color. Mono selects the monospace face
// (the terminal/ASCII look); Color empty means "use the scene's Ink".
type TextElement struct {
	base
	Text     string  `json:"text"`
	FontSize float64 `json:"fontSize"`
	Color    string  `json:"color,omitempty"` // hex "#RRGGBB"; empty → scene Ink
	Mono     bool    `json:"mono,omitempty"`
}

func (TextElement) Kind() string { return KindText }

// NewText makes a text element at r with default styling.
func NewText(r Rect, text string) *TextElement {
	return &TextElement{base: base{Rect: r}, Text: text, FontSize: 48, Color: "#ffffff"}
}

// ImageElement is a free-floating image layer.
type ImageElement struct {
	base
	Path string `json:"path"`
	Fit  string `json:"fit"`
}

func (ImageElement) Kind() string { return KindImage }

// NewImage makes an image layer at r.
func NewImage(r Rect, path string) *ImageElement {
	return &ImageElement{base: base{Rect: r}, Path: path, Fit: FitContain}
}

// Stat metric identifiers a StatWidget can animate.
const (
	MetricCommits       = "commits"
	MetricFollowers     = "followers"
	MetricStars         = "stars"
	MetricContributions = "contributions"
)

// StatWidget renders an animated GitHub metric (a growing bar + ticking counter).
// Login identifies whose stats to fetch; if empty the scene's default login is used.
// Value is the resolved number the widget animates toward — populated by gifmaker after a
// GitHub fetch (or by the editor's mock/sample data), keeping the renderer pure.
// It renders as a monospace line: "label   value  [████░░░░]". Max is the meter's full-bar
// value (the bar shows Value/Max); BarCells is the bar width in characters.
type StatWidget struct {
	base
	Metric   string  `json:"metric"`
	Login    string  `json:"login"`
	Label    string  `json:"label"`
	Value    int     `json:"value"`
	Max      int     `json:"max,omitempty"`
	BarCells int     `json:"barCells,omitempty"`
	Color    string  `json:"color,omitempty"`
	FontSize float64 `json:"fontSize"`
}

func (StatWidget) Kind() string { return KindStatWidget }

// NewStatWidget makes a stat widget at r for the given metric and login.
func NewStatWidget(r Rect, metric, login string) *StatWidget {
	return &StatWidget{
		base:     base{Rect: r},
		Metric:   metric,
		Login:    login,
		Color:    "#39d353", // GitHub contribution green
		FontSize: 28,
	}
}

// unmarshalElement builds a concrete element from a type-tagged JSON object.
func unmarshalElement(raw json.RawMessage) (Element, error) {
	var probe struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	var el Element
	switch probe.Kind {
	case KindBackground:
		el = &Background{}
	case KindText:
		el = &TextElement{}
	case KindImage:
		el = &ImageElement{}
	case KindStatWidget:
		el = &StatWidget{}
	default:
		return nil, fmt.Errorf("scene: unknown element kind %q", probe.Kind)
	}
	if err := json.Unmarshal(raw, el); err != nil {
		return nil, err
	}
	return el, nil
}
