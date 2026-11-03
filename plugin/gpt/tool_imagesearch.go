package gpt

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	_ "golang.org/x/image/webp"
)

const (
	BraveImageSearchURL   = "https://api.search.brave.com/res/v1/images/search"
	ImageSearchDefaultCnt = 4
	ImageSearchMaxCount   = 8
	MaxPhotoBytes         = 10_000_000
	MaxPreviewBytes       = 2_000_000
	MaxPhotoDimensionSum  = 10_000
	MaxPhotoAspectRatio   = 20
)

var (
	imageClient = httpUtils.NewHTTPClientWithTimeout(10 * time.Second)

	richPhotoTypes = map[string]struct{}{
		"image/jpeg": {},
		"image/png":  {},
		"image/webp": {},
	}
)

type (
	BraveImageResult struct {
		Title     string `json:"title"`
		URL       string `json:"url"`
		Source    string `json:"source"`
		Thumbnail struct {
			Src string `json:"src"`
		} `json:"thumbnail"`
		Properties struct {
			URL string `json:"url"`
		} `json:"properties"`
	}

	BraveImageSearchResponse struct {
		Results []BraveImageResult `json:"results"`
	}

	fetchedImage struct {
		mimeType string
		data     []byte
	}

	imageCandidate struct {
		result  BraveImageResult
		fullURL string
		full    fetchedImage
		preview fetchedImage
	}

	RichMediaTool interface {
		Tool
		RichMedia() map[string]fetchedImage
	}

	ImagesearchTool struct {
		apiKey string
		chatID int64
		mu     sync.Mutex
		media  map[string]fetchedImage
	}
)

func NewImagesearchTool(apiKey string, chatID int64) *ImagesearchTool {
	return &ImagesearchTool{apiKey: apiKey, chatID: chatID, media: make(map[string]fetchedImage)}
}

func (t *ImagesearchTool) RichMedia() map[string]fetchedImage {
	t.mu.Lock()
	defer t.mu.Unlock()
	return maps.Clone(t.media)
}

func (t *ImagesearchTool) Definition() FunctionTool {
	return FunctionTool{
		Type:        "function",
		Name:        "imagesearch",
		Description: "Sucht Bilder im Web und liefert geprüfte Bild-URLs samt Vorschau, damit du siehst, was die Bilder zeigen. Nur sinnvoll, wenn Bilder die Antwort wirklich bereichern (z. B. 'wie sieht X aus?'). Bilder lassen sich nur in einer Rich-Nachricht anzeigen; verwende ausschließlich die hier gelieferten Bild-URLs. Prüfe die Vorschauen und lass unpassende, verstörende oder eklige Bilder weg.",
		Parameters: FunctionParameters{
			Type: "object",
			Properties: map[string]Property{
				"query": {
					Type:        "string",
					Description: "Suchanfrage",
				},
				"count": {
					Type:        "integer",
					Description: fmt.Sprintf("Anzahl der Bilder (Standard: %d, Max: %d)", ImageSearchDefaultCnt, ImageSearchMaxCount),
				},
			},
			Required:             []string{"query"},
			AdditionalProperties: false,
		},
		Strict: false,
	}
}

func (t *ImagesearchTool) Execute(arguments string) (any, error) {
	var args struct {
		Query string `json:"query"`
		Count int    `json:"count"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Count <= 0 {
		args.Count = ImageSearchDefaultCnt
	}
	args.Count = min(args.Count, ImageSearchMaxCount)
	log.Debug().
		Str("query", args.Query).
		Int64("chat_id", t.chatID).
		Msg("imagesearch tool call")

	params := url.Values{
		"q":           []string{args.Query},
		"count":       []string{fmt.Sprintf("%d", args.Count*3)},
		"country":     []string{"DE"},
		"search_lang": []string{"de"},
		"safesearch":  []string{"strict"},
		"spellcheck":  []string{"false"},
	}
	var resp BraveImageSearchResponse
	if err := braveGet(BraveImageSearchURL+"?"+params.Encode(), t.apiKey, args.Query, &resp); err != nil {
		return "", err
	}

	candidates := fetchImages(resp.Results)
	if len(candidates) == 0 {
		return "No usable images found.", nil
	}
	candidates = candidates[:min(len(candidates), args.Count)]

	t.mu.Lock()
	for _, c := range candidates {
		t.media[c.fullURL] = c.full
	}
	t.mu.Unlock()

	parts := make([]any, 0, len(candidates)*2)
	for i, c := range candidates {
		source := c.result.Source
		if source == "" {
			if parsed, err := url.Parse(c.result.URL); err == nil {
				source = strings.TrimPrefix(parsed.Hostname(), "www.")
			}
		}
		parts = append(parts,
			InputText{
				Type: TypeInputText,
				Text: fmt.Sprintf("--- Image %d ---\nImage URL: %s\nTitle: %s\nSource: %s (%s)",
					i+1, c.fullURL, stripCitationChars(c.result.Title), source, c.result.URL),
			},
			InputImage{
				Type:     TypeInputImage,
				ImageURL: fmt.Sprintf("data:%s;base64,%s", c.preview.mimeType, base64.StdEncoding.EncodeToString(c.preview.data)),
				Detail:   "low",
			},
		)
	}
	return parts, nil
}

func (t *ImagesearchTool) Emoji() string {
	return "🖼️"
}

func (t *ImagesearchTool) MaxCalls() int {
	return 2
}

// Brave's thumbnail proxy passes upstream errors (e.g. 429) through and fails for Telegram's fetcher, so images are downloaded here and uploaded with the message.
func fetchImages(results []BraveImageResult) []imageCandidate {
	fetched := make([]*imageCandidate, len(results))
	var wg sync.WaitGroup
	for i, r := range results {
		wg.Go(func() {
			fetched[i] = fetchCandidate(r)
		})
	}
	wg.Wait()

	out := make([]imageCandidate, 0, len(results))
	seen := make(map[string]struct{}, len(results))
	for _, c := range fetched {
		if c == nil {
			continue
		}
		if _, dup := seen[c.fullURL]; dup {
			continue
		}
		seen[c.fullURL] = struct{}{}
		out = append(out, *c)
	}
	return out
}

func fetchCandidate(r BraveImageResult) *imageCandidate {
	var original, thumbnail fetchedImage
	var originalErr, thumbnailErr error
	var wg sync.WaitGroup
	wg.Go(func() {
		original, originalErr = fetchImage(r.Properties.URL, r.URL)
	})
	wg.Go(func() {
		thumbnail, thumbnailErr = fetchImage(r.Thumbnail.Src, "")
	})
	wg.Wait()

	c := &imageCandidate{result: r}
	switch {
	case originalErr == nil:
		c.fullURL, c.full = r.Properties.URL, original
	case thumbnailErr == nil:
		log.Debug().Err(originalErr).Str("url", r.Properties.URL).Msg("original image unavailable, using thumbnail")
		c.fullURL, c.full = r.Thumbnail.Src, thumbnail
	default:
		log.Debug().Err(thumbnailErr).Str("url", r.Thumbnail.Src).Msg("image unavailable")
		return nil
	}

	switch {
	case thumbnailErr == nil:
		c.preview = thumbnail
	case len(c.full.data) <= MaxPreviewBytes:
		c.preview = c.full
	default:
		return nil
	}
	return c
}

func fetchImage(rawURL, referer string) (fetchedImage, error) {
	if !strings.HasPrefix(rawURL, "https://") && !strings.HasPrefix(rawURL, "http://") {
		return fetchedImage{}, fmt.Errorf("invalid URL %q", rawURL)
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return fetchedImage{}, err
	}
	req.Header.Set("User-Agent", utils.UserAgent)
	req.Header.Set("Accept", "image/webp,image/png,image/jpeg;q=0.9,*/*;q=0.5")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := imageClient.Do(req)
	if err != nil {
		return fetchedImage{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fetchedImage{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxPhotoBytes+1))
	if err != nil {
		return fetchedImage{}, err
	}
	if len(data) == 0 || len(data) > MaxPhotoBytes {
		return fetchedImage{}, fmt.Errorf("invalid image size %d", len(data))
	}

	mimeType := http.DetectContentType(data)
	if _, ok := richPhotoTypes[mimeType]; !ok {
		return fetchedImage{}, fmt.Errorf("unsupported content type %q", mimeType)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fetchedImage{}, fmt.Errorf("invalid image: %w", err)
	}
	w, h := cfg.Width, cfg.Height
	if w <= 0 || h <= 0 || w+h > MaxPhotoDimensionSum || max(w, h) > MaxPhotoAspectRatio*min(w, h) {
		return fetchedImage{}, fmt.Errorf("unsupported dimensions %dx%d", w, h)
	}
	return fetchedImage{mimeType: mimeType, data: data}, nil
}
