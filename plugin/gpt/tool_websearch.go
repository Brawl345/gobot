package gpt

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Brawl345/gobot/utils/httpUtils"
)

const (
	BraveSearchURL    = "https://api.search.brave.com/res/v1/web/search"
	MaxSourceLinks    = 10
	BraveDefaultCount = 5
	BraveMaxCount     = 20
)

type (
	BraveWebResult struct {
		Title       string `json:"title"`
		URL         string `json:"url"`
		Description string `json:"description"`
		Age         string `json:"age"`
		PageAge     string `json:"page_age"`
	}

	BraveSearchResponse struct {
		Web struct {
			Results []BraveWebResult `json:"results"`
		} `json:"web"`
	}

	SearchResultsTool interface {
		Tool
		SearchResults() []BraveWebResult // this kinda sucks but we need the links at the end and the parse mode
	}

	WebsearchTool struct {
		apiKey  string
		chatID  int64
		mu      sync.Mutex
		results []BraveWebResult
	}
)

func NewWebsearchTool(apiKey string, chatID int64) *WebsearchTool {
	return &WebsearchTool{apiKey: apiKey, chatID: chatID}
}

func (t *WebsearchTool) Definition() FunctionTool {
	return FunctionTool{
		Type:        "function",
		Name:        "websearch",
		Description: "Sucht im Web. Gibt Titel, URLs, Snippets und Alter der Ergebnisse zurück. Nutze dieses Tool, wenn du aktuelle Informationen benötigst oder eine Frage beantworten musst, die wahrscheinlich durch eine Websuche beantwortet werden kann. Nutze webfetch um den vollständigen Inhalt einer Ergebnis-URL abzurufen.",
		Parameters: FunctionParameters{
			Type: "object",
			Properties: map[string]Property{
				"query": {
					Type:        "string",
					Description: "Suchanfrage",
				},
				"count": {
					Type:        "integer",
					Description: fmt.Sprintf("Anzahl der Ergebnisse (Standard: %d, Max: %d)", BraveDefaultCount, BraveMaxCount),
				},
				"country": {
					Type:        "string",
					Description: "Zweistelliger Ländercode für Ergebnisse (Standard: DE)",
				},
				"freshness": {
					Type:        "string",
					Description: "Nach Aktualität filtern: pd (letzter Tag), pw (letzte Woche), pm (letzter Monat), py (letztes Jahr)",
					Enum:        []string{"pd", "pw", "pm", "py"},
				},
			},
			Required:             []string{"query"},
			AdditionalProperties: false,
		},
		Strict: false,
	}
}

func (t *WebsearchTool) Execute(arguments string) (string, error) {
	var args struct {
		Query     string `json:"query"`
		Count     int    `json:"count"`
		Country   string `json:"country"`
		Freshness string `json:"freshness"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	log.Debug().
		Str("query", args.Query).
		Int64("chat_id", t.chatID).
		Msg("websearch tool call")
	results, output, err := braveSearch(args.Query, t.apiKey, args.Count, args.Country, args.Freshness)
	if err != nil {
		return "", err
	}
	t.mu.Lock()
	t.results = append(t.results, results...)
	t.mu.Unlock()
	return output, nil
}

func (t *WebsearchTool) Emoji() string {
	return "🔎"
}

func (t *WebsearchTool) SearchResults() []BraveWebResult {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]BraveWebResult(nil), t.results...)
}

const (
	braveMaxRetries  = 3
	braveMaxWaitSecs = 5
)

func braveSearch(query, braveKey string, count int, country, freshness string) ([]BraveWebResult, string, error) {
	if count <= 0 {
		count = BraveDefaultCount
	}
	if count > BraveMaxCount {
		count = BraveMaxCount
	}
	if country == "" {
		country = "DE"
	}

	params := url.Values{
		"q":       []string{query},
		"count":   []string{fmt.Sprintf("%d", count)},
		"country": []string{strings.ToUpper(country)},
	}
	if freshness != "" {
		params.Set("freshness", freshness)
	}

	reqURL := BraveSearchURL + "?" + params.Encode()

	var result BraveSearchResponse
	var lastErr error
	for attempt := range braveMaxRetries {
		var respHeaders http.Header
		err := httpUtils.MakeRequest(httpUtils.RequestOptions{
			Method:          httpUtils.MethodGet,
			URL:             reqURL,
			Headers:         map[string]string{"X-Subscription-Token": braveKey, "Accept": "application/json"},
			Response:        &result,
			ResponseHeaders: &respHeaders,
		})
		if err == nil {
			lastErr = nil
			break
		}

		httpErr, ok := errors.AsType[*httpUtils.HttpError](err)
		if !ok || httpErr.StatusCode != http.StatusTooManyRequests {
			return nil, "", fmt.Errorf("brave search failed: %w", err)
		}

		wait := time.Duration(1<<attempt) * time.Second // 1s, 2s, 4s
		if raw := respHeaders.Get("X-RateLimit-Reset"); raw != "" {
			if secs, parseErr := strconv.Atoi(raw); parseErr == nil && secs > 0 {
				if secs > braveMaxWaitSecs {
					return nil, "", fmt.Errorf("brave search rate limited, reset in %ds (too long to wait)", secs)
				}
				wait = time.Duration(secs) * time.Second
			}
		}
		log.Warn().
			Str("query", query).
			Int("attempt", attempt+1).
			Dur("wait", wait).
			Msg("brave search rate limited, retrying")
		time.Sleep(wait)
		lastErr = err
	}
	if lastErr != nil {
		return nil, "", fmt.Errorf("brave search failed after %d attempts: %w", braveMaxRetries, lastErr)
	}

	results := result.Web.Results
	if len(results) == 0 {
		return nil, "No results found.", nil
	}

	var sb strings.Builder
	for i, r := range results {
		age := r.Age
		if age == "" {
			age = r.PageAge
		}
		_, _ = fmt.Fprintf(&sb, "--- Result %d ---\nTitle: %s\nLink: %s\n", i+1, r.Title, r.URL)
		if age != "" {
			_, _ = fmt.Fprintf(&sb, "Age: %s\n", age)
		}
		_, _ = fmt.Fprintf(&sb, "Snippet: %s\n\n", sanitizeSnippet(r.Description))
	}
	return results, strings.TrimRight(sb.String(), "\n"), nil
}

func searchLinks(results []BraveWebResult) string {
	if len(results) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("(")
	for i, r := range results {
		host := r.URL
		if parsed, err := url.Parse(r.URL); err == nil {
			host = strings.TrimPrefix(parsed.Hostname(), "www.")
		}
		_, _ = fmt.Fprintf(&sb, `<a href="%s">%s</a>`, r.URL, host)
		if i < len(results)-1 {
			sb.WriteString(", ")
		}
		if i == MaxSourceLinks-1 && len(results) > MaxSourceLinks {
			sb.WriteString("...")
			break
		}
	}
	sb.WriteString(")")
	return sb.String()
}

func sanitizeSnippet(s string) string {
	s = strings.NewReplacer(
		"&amp;", "&", "&lt;", "<", "&gt;", ">",
		"&quot;", `"`, "&#39;", "'", "&nbsp;", " ",
	).Replace(s)
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
			result.WriteRune(' ')
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	out := result.String()
	for strings.Contains(out, "  ") {
		out = strings.ReplaceAll(out, "  ", " ")
	}
	return strings.TrimSpace(out)
}
