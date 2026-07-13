package httpUtils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

var (
	log               = logger.New("httpUtils")
	DefaultHttpClient *http.Client
	// SSRFSafeClient guards against SSRF: its transport resolves and validates
	// every address at dial time (also covering redirects), so an attacker
	// cannot point it at an internal address even via DNS rebinding.
	SSRFSafeClient *http.Client

	// cgnatNet is the 100.64.0.0/10 carrier-grade NAT range, which is not
	// covered by net.IP.IsPrivate but must still be treated as internal.
	cgnatNet = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

	// botTokenInURL matches the "bot<id>:<token>" segment of Telegram API URLs
	// so it can be stripped before logging.
	botTokenInURL = regexp.MustCompile(`bot[0-9]+:[A-Za-z0-9_-]+`)

	// sensitiveHeaders are header names whose values must be redacted from logs.
	sensitiveHeaders = map[string]struct{}{
		"authorization":       {},
		"proxy-authorization": {},
		"api-key":             {},
		"x-api-key":           {},
		"x-goog-api-key":      {},
		"x-auth-token":        {},
	}
)

// redactURL masks a Telegram bot token embedded in a URL before logging.
func redactURL(u string) string {
	return botTokenInURL.ReplaceAllString(u, "bot[REDACTED]")
}

// redactHeaders returns a copy of h with sensitive header values masked.
func redactHeaders(h map[string]string) map[string]string {
	if h == nil {
		return nil
	}
	redacted := make(map[string]string, len(h))
	for k, v := range h {
		if _, ok := sensitiveHeaders[strings.ToLower(k)]; ok {
			redacted[k] = "[REDACTED]"
		} else {
			redacted[k] = v
		}
	}
	return redacted
}

type (
	HTTPMethod string

	RequestOptions struct {
		Method  HTTPMethod
		URL     string
		Headers map[string]string
		Body    any

		// Response can either be a pointer to a JSON struct or a pointer to a string
		Response any
		// ErrorResponse can either be a pointer to a JSON struct or a pointer to a string.
		// NOTE: An error will still be returned!
		ErrorResponse any
		// ResponseHeaders receives the response headers if non-nil. Populated on
		// both success and error responses (but not on transport-level errors).
		ResponseHeaders *http.Header

		Client *http.Client
	}

	MultiPartParam struct {
		Name  string
		Value string
	}

	MultiPartFile struct {
		FieldName string
		FileName  string
		Content   io.Reader
	}
)

const (
	MethodGet  HTTPMethod = http.MethodGet
	MethodPost HTTPMethod = http.MethodPost

	MaxResponseBodySize = 50 * 1024 * 1024
)

// readResponseBody reads r fully but stops at MaxResponseBodySize, returning an
// error if the body exceeds the cap.
func readResponseBody(r io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, MaxResponseBodySize+1))
	if err != nil {
		return nil, err
	}
	if len(body) > MaxResponseBodySize {
		return nil, fmt.Errorf("response body exceeds %d bytes", MaxResponseBodySize)
	}
	return body, nil
}

func init() {
	DefaultHttpClient = createHTTPClient()

	ssrfSafeTransport := http.DefaultTransport.(*http.Transport).Clone()
	ssrfSafeTransport.DialContext = ssrfSafeDialContext
	ssrfSafeTransport.TLSHandshakeTimeout = 7 * time.Second
	ssrfSafeTransport.ResponseHeaderTimeout = 15 * time.Second
	ssrfSafeTransport.MaxIdleConnsPerHost = 20
	ssrfSafeTransport.IdleConnTimeout = 5 * time.Minute

	SSRFSafeClient = &http.Client{
		Transport: ssrfSafeTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
}

// ssrfSafeDialContext resolves the target host and refuses to connect if any
// resolved address is internal, then dials a validated IP directly. Dialing
// the exact IP that was validated closes the DNS-rebinding TOCTOU window.
func ssrfSafeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("could not resolve host %q: %w", host, err)
	}

	for _, ip := range ips {
		if isBlockedIP(ip.IP) {
			return nil, fmt.Errorf("host %q resolves to a private/internal address (%s)", host, ip.IP)
		}
	}

	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	var firstErr error
	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
		if err != nil {
			firstErr = err
			continue
		}
		return conn, nil
	}
	return nil, firstErr
}

// isBlockedIP reports whether ip is loopback, private, link-local,
// unspecified (0.0.0.0/::) or in the carrier-grade NAT range.
func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if v4 := ip.To4(); v4 != nil && cgnatNet.Contains(v4) {
		return true
	}
	return false
}

func createHTTPClient() *http.Client {
	return NewHTTPClientWithTimeout(15 * time.Second)
}

func NewHTTPClientWithTimeout(responseHeaderTimeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	transport.TLSHandshakeTimeout = 7 * time.Second
	transport.ResponseHeaderTimeout = responseHeaderTimeout
	transport.MaxIdleConnsPerHost = 20
	transport.IdleConnTimeout = 5 * time.Minute

	return &http.Client{
		Transport: transport,
	}
}

func MakeRequest(opts RequestOptions) error {
	log.Debug().
		Str("method", string(opts.Method)).
		Str("url", redactURL(opts.URL)).
		Interface("body", opts.Body).
		Interface("headers", redactHeaders(opts.Headers)).
		Send()

	var reqBody io.Reader
	isJsonBody := true
	if opts.Body != nil {
		switch v := opts.Body.(type) {
		case io.ReadCloser:
			isJsonBody = false
			reqBody = v
			defer func(v io.ReadCloser) {
				err := v.Close()
				if err != nil {
					log.Err(err).Msg("Failed to close response body")
				}
			}(v)
		default:
			jsonData, err := json.Marshal(v)
			if err != nil {
				return err
			}
			reqBody = bytes.NewBuffer(jsonData)
		}
	}

	req, err := http.NewRequest(string(opts.Method), opts.URL, reqBody)
	if err != nil {
		return err
	}

	if opts.Body != nil && isJsonBody {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	client := opts.Client
	if client == nil {
		client = DefaultHttpClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close response body")
		}
	}(resp.Body)

	if opts.ResponseHeaders != nil {
		*opts.ResponseHeaders = resp.Header
	}

	if resp.StatusCode != http.StatusOK {
		if opts.ErrorResponse != nil {
			bodyBytes, err := readResponseBody(resp.Body)
			if err != nil {
				return &HttpError{
					StatusCode: resp.StatusCode,
				}
			}

			switch v := opts.ErrorResponse.(type) {
			case *string:
				*v = string(bodyBytes)
			default:
				err = json.Unmarshal(bodyBytes, opts.ErrorResponse)
				if err != nil {
					return &HttpError{
						StatusCode: resp.StatusCode,
					}
				}
			}

			log.Debug().
				Str("url", redactURL(opts.URL)).
				Interface("result", opts.ErrorResponse).
				Send()
		}

		return &HttpError{
			StatusCode: resp.StatusCode,
		}
	}

	if opts.Response != nil {
		bodyBytes, err := readResponseBody(resp.Body)
		if err != nil {
			return err
		}

		switch v := opts.Response.(type) {
		case *string:
			*v = string(bodyBytes)
		default:
			err = json.Unmarshal(bodyBytes, opts.Response)
			if err != nil {
				return err
			}
		}
	}

	log.Debug().
		Str("url", redactURL(opts.URL)).
		Interface("result", opts.Response).
		Send()

	return nil
}

func MultiPartFormRequest(url string, params []MultiPartParam, files []MultiPartFile) (*http.Response, error) {
	return MultiPartFormRequestWithHeaders(url, nil, params, files)
}

func MultiPartFormRequestWithHeaders(url string, headers map[string]string, params []MultiPartParam, files []MultiPartFile) (*http.Response, error) {
	log.Debug().
		Str("url", redactURL(url)).
		Interface("params", params).
		Interface("headers", redactHeaders(headers)).
		Send()

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	defer func(writer *multipart.Writer) {
		err := writer.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close multipart writer")
		}
	}(writer)

	for _, param := range params {
		err := writer.WriteField(param.Name, param.Value)
		if err != nil {
			return nil, err
		}
	}

	for _, file := range files {
		fw, err := writer.CreateFormFile(file.FieldName, file.FileName)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(fw, file.Content)
		if err != nil {
			return nil, err
		}
	}

	err := writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return DefaultHttpClient.Do(req)
}

func DownloadFile(b *gotgbot.Bot, fileID string) (io.ReadCloser, error) {
	file, err := b.GetFile(fileID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from Telegram: %w", err)
	}

	return DownloadFileFromGetFile(b, file)
}

func DownloadFileFromGetFile(b *gotgbot.Bot, file *gotgbot.File) (io.ReadCloser, error) {
	fileUrl := file.URL(b, nil)
	log.Debug().
		Str("url", redactURL(fileUrl)).
		Send()
	resp, err := DefaultHttpClient.Get(fileUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &HttpError{
			StatusCode: resp.StatusCode,
		}
	}

	return resp.Body, nil
}

// IsPrivateURL returns an error if the given rawURL resolves to a loopback,
// private, or link-local address, preventing SSRF attacks.
func IsPrivateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("URL has no host")
	}

	addrs, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("could not resolve host %q: %w", host, err)
	}

	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		if isBlockedIP(ip) {
			return fmt.Errorf("host %q resolves to a private/internal address (%s)", host, ip)
		}
	}

	return nil
}
