package gpt

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/go-analyze/charts"
)

const (
	ChartWidth     = 1280
	ChartHeight    = 720
	ChartMaxLabels = 40
	ChartMaxSeries = 8
)

var chartHexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type (
	chartSeries struct {
		Name   string    `json:"name"`
		Values []float64 `json:"values"`
		Color  string    `json:"color"`
	}

	chartArgs struct {
		Type     string        `json:"type"`
		Title    string        `json:"title"`
		Subtitle string        `json:"subtitle"`
		Labels   []string      `json:"labels"`
		Series   []chartSeries `json:"series"`
		Colors   []string      `json:"colors"`
		Unit     string        `json:"unit"`
	}

	ChartTool struct {
		chatID int64
		mu     sync.Mutex
		media  map[string]fetchedImage
	}
)

func NewChartTool(chatID int64) *ChartTool {
	return &ChartTool{chatID: chatID, media: make(map[string]fetchedImage)}
}

func (t *ChartTool) RichMedia() map[string]fetchedImage {
	t.mu.Lock()
	defer t.mu.Unlock()
	return maps.Clone(t.media)
}

func (t *ChartTool) Definition() FunctionTool {
	str := func(desc string) Property { return Property{Type: "string", Description: desc} }
	strList := func(desc string) Property {
		return Property{Type: "array", Description: desc, Items: &Property{Type: "string"}}
	}
	return FunctionTool{
		Type:        "function",
		Name:        "chart",
		Description: "Rendert ein Diagramm als Bild (Balken, horizontale Balken, Linie, Torte) und liefert eine Chart-URL samt Vorschau. Unaufgefordert aufrufen, sobald die Antwort vergleichbare Zahlen enthält, z. B. Umfragen, Sonntagsfrage, Wahlergebnisse, Zeitreihen, Ranglisten oder Vergleiche. Die Daten müssen aus Tool-Ergebnissen oder dem Gespräch stammen, nie erfunden.",
		Parameters: FunctionParameters{
			Type: "object",
			Properties: map[string]Property{
				"type": {
					Type:        "string",
					Description: "pie: Werte sind Teile einer Gesamtsumme, z. B. Verkäufe nach Region, Marktanteile, Sitzverteilung, Budgetaufteilung (genau eine Serie, 2-8 Segmente, keine negativen Werte); bar: unabhängige Werte nebeneinander, z. B. Umfragewerte von Parteien oder Vergleich mehrerer Serien; horizontal_bar: bei langen Beschriftungen oder Ranglisten; line: Zeitreihen",
					Enum:        []string{"bar", "horizontal_bar", "line", "pie"},
				},
				"title":    str("Titel des Diagramms"),
				"subtitle": str("Untertitel, z. B. Quelle und Stand, oder leer"),
				"labels":   strList("Kategorien bzw. X-Werte, bei pie die Segmente"),
				"series": {
					Type:        "array",
					Description: fmt.Sprintf("Datenreihen (1-%d), jede mit genau so vielen Werten wie labels", ChartMaxSeries),
					Items: &Property{
						Type: "object",
						Properties: map[string]Property{
							"name": str("Name der Reihe für die Legende, bei nur einer Reihe leer"),
							"values": {
								Type:  "array",
								Items: &Property{Type: "number"},
							},
							"color": str("Farbe als #RRGGBB oder leer für automatisch"),
						},
						Required:             []string{"name", "values", "color"},
						AdditionalProperties: new(false),
					},
				},
				"colors": strList("Farben je Label als #RRGGBB (z. B. Parteifarben), nur bei einer Reihe in bar/horizontal_bar oder bei pie; sonst leeres Array"),
				"unit":   str("Einheit hinter den Werten, z. B. \"%\" oder \"€\", oder leer"),
			},
			Required:             []string{"type", "title", "subtitle", "labels", "series", "colors", "unit"},
			AdditionalProperties: false,
		},
		Strict: true,
	}
}

func (t *ChartTool) Execute(arguments string) (any, error) {
	var args chartArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	log.Debug().
		Str("type", args.Type).
		Str("title", args.Title).
		Int64("chat_id", t.chatID).
		Msg("chart tool call")

	if err := args.validate(); err != nil {
		return "", err
	}
	data, err := renderChart(args)
	if err != nil {
		return "", fmt.Errorf("rendering failed: %w", err)
	}

	t.mu.Lock()
	chartURL := fmt.Sprintf("chart://%d", len(t.media)+1)
	t.media[chartURL] = fetchedImage{mimeType: "image/png", data: data}
	t.mu.Unlock()

	return []any{
		InputText{Type: TypeInputText, Text: "Chart URL: " + chartURL},
		InputImage{
			Type:     TypeInputImage,
			ImageURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(data),
			Detail:   "low",
		},
	}, nil
}

func (t *ChartTool) Emoji() string {
	return "📊"
}

func (t *ChartTool) MaxCalls() int {
	return 2
}

func (a chartArgs) validate() error {
	if len(a.Labels) == 0 || len(a.Labels) > ChartMaxLabels {
		return fmt.Errorf("labels must contain 1-%d entries", ChartMaxLabels)
	}
	if len(a.Series) == 0 || len(a.Series) > ChartMaxSeries {
		return fmt.Errorf("series must contain 1-%d entries", ChartMaxSeries)
	}
	for i, s := range a.Series {
		if len(s.Values) != len(a.Labels) {
			return fmt.Errorf("series %d has %d values, expected %d (one per label)", i+1, len(s.Values), len(a.Labels))
		}
		for _, v := range s.Values {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return errors.New("values must be finite numbers")
			}
			if a.Type == "pie" && v < 0 {
				return errors.New("pie charts cannot contain negative values")
			}
		}
		if s.Color != "" && !chartHexColor.MatchString(s.Color) {
			return fmt.Errorf("invalid color %q, expected #RRGGBB", s.Color)
		}
	}
	if len(a.Colors) > 0 {
		if len(a.Colors) != len(a.Labels) {
			return fmt.Errorf("colors has %d entries, expected %d (one per label)", len(a.Colors), len(a.Labels))
		}
		if len(a.Series) > 1 || a.Type == "line" {
			return errors.New("colors per label only work with a single series in bar, horizontal_bar or pie")
		}
		for _, c := range a.Colors {
			if !chartHexColor.MatchString(c) {
				return fmt.Errorf("invalid color %q, expected #RRGGBB", c)
			}
		}
	}
	if a.Type == "pie" && len(a.Series) != 1 {
		return errors.New("pie charts need exactly one series")
	}
	return nil
}

func renderChart(a chartArgs) ([]byte, error) {
	theme := charts.GetTheme(charts.ThemeLight)
	format := chartValueFormatter(a.Unit)

	title := charts.TitleOption{
		Text:             a.Title,
		Subtext:          a.Subtitle,
		Offset:           charts.OffsetCenter,
		FontStyle:        charts.FontStyle{FontSize: 22},
		SubtextFontStyle: charts.FontStyle{FontSize: 14},
	}
	names := make([]string, len(a.Series))
	values := make([][]float64, len(a.Series))
	colors := make([]charts.Color, len(a.Series))
	for i, s := range a.Series {
		names[i] = s.Name
		values[i] = s.Values
		colors[i] = theme.GetSeriesColor(i)
		if s.Color != "" {
			colors[i] = charts.ParseColor(s.Color)
		}
	}
	legend := charts.LegendOption{
		Show:        new(len(a.Series) > 1),
		SeriesNames: names,
		Offset:      charts.OffsetStr{Top: charts.PositionBottom},
		FontStyle:   charts.FontStyle{FontSize: 14},
	}

	p := charts.NewPainter(charts.PainterOptions{
		OutputFormat: charts.ChartOutputPNG,
		Width:        ChartWidth,
		Height:       ChartHeight,
		Theme:        theme,
	})
	padding := charts.NewBoxEqual(24)

	var err error
	switch a.Type {
	case "pie":
		perLabel := labelColors(a.Colors, len(a.Labels), theme)
		opt := charts.NewPieChartOptionWithData(a.Series[0].Values)
		opt.Theme = theme.WithSeriesColors(perLabel)
		opt.Padding = padding
		opt.Title = title
		opt.Legend = charts.LegendOption{Show: new(false)}
		for i := range opt.SeriesList {
			opt.SeriesList[i].Name = a.Labels[i]
			opt.SeriesList[i].Label.Show = new(true)
			opt.SeriesList[i].Label.FontStyle = charts.FontStyle{FontSize: 15}
			opt.SeriesList[i].Label.LabelFormatter = func(_ int, name string, val float64) (string, *charts.LabelStyle) {
				return name + ": " + format(val), nil
			}
		}
		err = p.PieChart(opt)
	case "line":
		opt := charts.NewLineChartOptionWithData(values)
		opt.Theme = theme.WithSeriesColors(colors)
		opt.Padding = padding
		opt.Title = title
		opt.Legend = legend
		opt.XAxis.Labels = a.Labels
		opt.XAxis.LabelFontStyle = charts.FontStyle{FontSize: 14}
		opt.YAxis = []charts.YAxisOption{{ValueFormatter: format, LabelFontStyle: charts.FontStyle{FontSize: 14}}}
		for i := range opt.SeriesList {
			opt.SeriesList[i].Name = names[i]
		}
		opt.LineStrokeWidth = 3
		err = p.LineChart(opt)
	default:
		opt := barChartOption(a, values, names, colors, theme, format)
		opt.Padding = padding
		opt.Title = title
		if len(a.Colors) == 0 {
			opt.Legend = legend
		}
		err = p.BarChart(opt)
	}
	if err != nil {
		return nil, err
	}
	return p.Bytes()
}

// A single series cannot color bars individually, so per-label colors become one stacked series per label.
func barChartOption(a chartArgs, values [][]float64, names []string, colors []charts.Color, theme charts.ColorPalette, format charts.ValueFormatter) charts.BarChartOption {
	horizontal := a.Type == "horizontal_bar"
	labels := a.Labels
	if horizontal {
		labels = slices.Clone(labels)
		slices.Reverse(labels)
		values = slices.Clone(values)
		for i := range values {
			values[i] = slices.Clone(values[i])
			slices.Reverse(values[i])
		}
	}

	nonNegative := minValue(values) >= 0
	seriesColors := colors
	zeroAnchor := false
	if len(a.Colors) > 0 {
		seriesColors = labelColors(a.Colors, len(a.Labels), theme)
		if horizontal {
			slices.Reverse(seriesColors)
		}
		split := make([][]float64, len(labels))
		for i, v := range values[0] {
			split[i] = slices.Repeat([]float64{charts.GetNullValue()}, len(labels))
			split[i][i] = v
		}
		values = split
		// Stacked charts lower the axis minimum by 1 before applying Min, so Min=0 is ignored when the smallest value is below 1; a zero-height series keeps the minimum at 0.
		if nonNegative {
			anchor := slices.Repeat([]float64{charts.GetNullValue()}, len(labels))
			anchor[0] = 0
			values = append(values, anchor)
			seriesColors = append(seriesColors, charts.ColorTransparent)
			zeroAnchor = true
		}
	}

	opt := charts.NewBarChartOptionWithData(values)
	opt.Theme = theme.WithSeriesColors(seriesColors)
	opt.StackSeries = new(len(a.Colors) > 0)
	opt.Horizontal = horizontal
	opt.BarSize = 0.7
	for i := range opt.SeriesList {
		if len(a.Colors) == 0 {
			opt.SeriesList[i].Name = names[i]
		}
		opt.SeriesList[i].Label = charts.SeriesLabel{
			Show:           new(!zeroAnchor || i < len(opt.SeriesList)-1),
			FontStyle:      charts.FontStyle{FontSize: 15, FontColor: theme.GetLabelTextColor()},
			ValueFormatter: format,
		}
	}
	opt.CategoryAxis.Labels = labels
	opt.CategoryAxis.LabelFontStyle = charts.FontStyle{FontSize: 14}
	valueAxis := charts.ValueAxisOption{LabelFontStyle: charts.FontStyle{FontSize: 14}}
	if nonNegative {
		valueAxis.Min = new(0.0)
	}
	opt.ValueAxis = []charts.ValueAxisOption{valueAxis}
	opt.ValueFormatter = format
	return opt
}

func minValue(values [][]float64) float64 {
	m := math.Inf(1)
	for _, s := range values {
		for _, v := range s {
			if v != charts.GetNullValue() {
				m = min(m, v)
			}
		}
	}
	return m
}

func labelColors(hex []string, n int, theme charts.ColorPalette) []charts.Color {
	out := make([]charts.Color, n)
	for i := range out {
		if i < len(hex) {
			out[i] = charts.ParseColor(hex[i])
		} else {
			out[i] = theme.GetSeriesColor(i)
		}
	}
	return out
}

func chartValueFormatter(unit string) charts.ValueFormatter {
	suffix := ""
	if unit = strings.TrimSpace(unit); unit != "" {
		suffix = " " + unit
	}
	return func(v float64) string {
		s := strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
		return strings.Replace(s, ".", ",", 1) + suffix
	}
}
