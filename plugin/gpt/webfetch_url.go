package gpt

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	rxCommitSHA    = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	rxLineFragment = regexp.MustCompile(`^L(\d+)(?:C\d+)?(?:-L?(\d+)(?:C\d+)?)?$`)
)

type lineRange struct {
	start int
	end   int
}

func (r lineRange) empty() bool { return r.start == 0 }

func parseLineFragment(fragment string) lineRange {
	m := rxLineFragment.FindStringSubmatch(fragment)
	if m == nil {
		return lineRange{}
	}
	start, _ := strconv.Atoi(m[1])
	end := start
	if m[2] != "" {
		end, _ = strconv.Atoi(m[2])
	}
	if end < start {
		start, end = end, start
	}
	return lineRange{start: start, end: end}
}

// rewriteURL points a GitHub file viewer URL at the raw file that carries the
// content. The second return value is an alternative URL to try when the first
// one 404s, because a ref can be a branch, a tag or a commit and only the
// branch form is guessable.
func rewriteURL(u *url.URL) (*url.URL, *url.URL, lineRange) {
	lines := parseLineFragment(u.Fragment)

	switch strings.ToLower(u.Hostname()) {
	case "github.com", "www.github.com":
	case "gist.github.com":
		parts := splitPath(u.Path)
		if len(parts) == 2 {
			return rawURL("gist.githubusercontent.com", parts[0], parts[1], "raw"), nil, lines
		}
		return u, nil, lines
	default:
		return u, nil, lines
	}

	parts := splitPath(u.Path)
	if len(parts) < 5 {
		return u, nil, lines
	}
	owner, repo, kind, rest := parts[0], parts[1], parts[2], parts[3:]
	if kind != "blob" && kind != "raw" && kind != "blame" {
		return u, nil, lines
	}

	ref, path := rest[0], rest[1:]
	if ref == "refs" && len(rest) >= 3 {
		// already fully qualified: refs/heads/main/... or refs/tags/v1/...
		return rawURL("raw.githubusercontent.com", owner, repo, rest...), nil, lines
	}
	if len(path) == 0 {
		return u, nil, lines
	}

	plain := rawURL("raw.githubusercontent.com", owner, repo, append([]string{ref}, path...)...)
	if rxCommitSHA.MatchString(ref) {
		return plain, nil, lines
	}

	branch := rawURL("raw.githubusercontent.com", owner, repo, append([]string{"refs", "heads", ref}, path...)...)
	return branch, plain, lines
}

func rawURL(host, owner, repo string, segments ...string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/" + strings.Join(append([]string{owner, repo}, segments...), "/"),
	}
}

func splitPath(p string) []string {
	var out []string
	for _, part := range strings.Split(strings.Trim(p, "/"), "/") {
		if part != "" {
			if decoded, err := url.PathUnescape(part); err == nil {
				part = decoded
			}
			out = append(out, part)
		}
	}
	return out
}

func sliceLines(text string, r lineRange) string {
	lines := strings.Split(text, "\n")
	if r.start > len(lines) {
		return text
	}
	return strings.Join(lines[r.start-1:min(r.end, len(lines))], "\n")
}
