package gpt

import (
	"bytes"
	"encoding/json"
	"image/png"
	"strings"
	"testing"
)

func TestChartToolExecute(t *testing.T) {
	cases := map[string]chartArgs{
		"bar colors": {Type: "bar", Title: "Sonntagsfrage", Unit: "%",
			Labels: []string{"CDU/CSU", "AfD", "SPD"},
			Series: []chartSeries{{Values: []float64{25.5, 26, 15}}},
			Colors: []string{"#151518", "#009EE0", "#E3000F"}},
		"horizontal": {Type: "horizontal_bar", Title: "Einwohner",
			Labels: []string{"Berlin", "Hamburg"},
			Series: []chartSeries{{Values: []float64{3.88, 1.91}}},
			Colors: []string{"#E3000F", "#1AA037"}},
		"grouped": {Type: "bar", Title: "Umfragen",
			Labels: []string{"A", "B"},
			Series: []chartSeries{{Name: "X", Values: []float64{1, -2}}, {Name: "Y", Values: []float64{3, 4}, Color: "#888888"}}},
		"line": {Type: "line", Title: "Inflation",
			Labels: []string{"2024", "2025"},
			Series: []chartSeries{{Name: "DE", Values: []float64{2.2, 2.1}}}},
		"pie": {Type: "pie", Title: "Mix",
			Labels: []string{"Wind", "Solar"},
			Series: []chartSeries{{Values: []float64{60, 40}}}},
	}
	tool := NewChartTool(1)
	for name, args := range cases {
		raw, _ := json.Marshal(args)
		out, err := tool.Execute(string(raw))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		parts, ok := out.([]any)
		if !ok || len(parts) != 2 {
			t.Fatalf("%s: unexpected output %T", name, out)
		}
		text := parts[0].(InputText).Text
		chartURL := strings.TrimPrefix(text, "Chart URL: ")
		img, ok := tool.RichMedia()[chartURL]
		if !ok {
			t.Fatalf("%s: %q not in rich media", name, chartURL)
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(img.data))
		if err != nil || cfg.Width != ChartWidth || cfg.Height != ChartHeight {
			t.Fatalf("%s: invalid PNG %+v: %v", name, cfg, err)
		}
	}
	if len(tool.RichMedia()) != len(cases) {
		t.Errorf("RichMedia() has %d entries, want %d", len(tool.RichMedia()), len(cases))
	}
}

func TestChartArgsValidate(t *testing.T) {
	base := func() chartArgs {
		return chartArgs{Type: "bar", Labels: []string{"A", "B"}, Series: []chartSeries{{Values: []float64{1, 2}}}}
	}
	cases := map[string]func(*chartArgs){
		"value count":      func(a *chartArgs) { a.Series[0].Values = []float64{1} },
		"no labels":        func(a *chartArgs) { a.Labels = nil },
		"bad series color": func(a *chartArgs) { a.Series[0].Color = "red" },
		"color count":      func(a *chartArgs) { a.Colors = []string{"#000000"} },
		"colors on line": func(a *chartArgs) {
			a.Type = "line"
			a.Colors = []string{"#000000", "#ffffff"}
		},
		"colors multi series": func(a *chartArgs) {
			a.Colors = []string{"#000000", "#ffffff"}
			a.Series = append(a.Series, chartSeries{Values: []float64{3, 4}})
		},
		"pie negative": func(a *chartArgs) {
			a.Type = "pie"
			a.Series[0].Values = []float64{-1, 2}
		},
		"pie multi series": func(a *chartArgs) {
			a.Type = "pie"
			a.Series = append(a.Series, chartSeries{Values: []float64{3, 4}})
		},
	}
	if err := base().validate(); err != nil {
		t.Fatalf("valid args rejected: %v", err)
	}
	for name, mutate := range cases {
		a := base()
		mutate(&a)
		if a.validate() == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestChartValueFormatter(t *testing.T) {
	cases := map[string]struct {
		unit string
		val  float64
		want string
	}{
		"percent": {"%", 25.5, "25,5 %"},
		"round":   {"", 1.23456, "1,23"},
		"integer": {"€", 3, "3 €"},
		"padded":  {" Mio. ", 7.95, "7,95 Mio."},
	}
	for name, c := range cases {
		if got := chartValueFormatter(c.unit)(c.val); got != c.want {
			t.Errorf("%s: got %q, want %q", name, got, c.want)
		}
	}
}

func TestEmbedRichMediaChart(t *testing.T) {
	media := map[string]fetchedImage{"chart://1": {mimeType: "image/png", data: []byte("x")}}
	got, files := embedRichMedia(`<figure><img src="chart://1"/><figcaption>Umfrage</figcaption></figure>`, media)
	if want := `<figure><img src="tg://photo?id=img1"/><figcaption>Umfrage</figcaption></figure>`; got != want || len(files) != 1 {
		t.Errorf("embedRichMedia() = %q (%d files), want %q", got, len(files), want)
	}
}
