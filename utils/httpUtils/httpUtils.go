package httpUtils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

var (
	log               = logger.New("httpUtils")
	DefaultHttpClient *http.Client
)

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
)

func init() {
	DefaultHttpClient = createHTTPClient()
}

func createHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	transport.TLSHandshakeTimeout = 7 * time.Second
	transport.ResponseHeaderTimeout = 15 * time.Second
	transport.MaxIdleConnsPerHost = 20
	transport.IdleConnTimeout = 5 * time.Minute

	client := &http.Client{
		Transport: transport,
	}

	return client
}

func MakeRequest(opts RequestOptions) error {
	log.Debug().
		Str("method", string(opts.Method)).
		Str("url", opts.URL).
		Interface("body", opts.Body).
		Interface("headers", opts.Headers).
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

	if resp.StatusCode != http.StatusOK {
		if opts.ErrorResponse != nil {
			bodyBytes, err := io.ReadAll(resp.Body)
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
				Str("url", opts.URL).
				Interface("result", opts.ErrorResponse).
				Send()
		}

		return &HttpError{
			StatusCode: resp.StatusCode,
		}
	}

	if opts.Response != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
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
		Str("url", opts.URL).
		Interface("result", opts.Response).
		Send()

	return nil
}

func MultiPartFormRequest(url string, params []MultiPartParam, files []MultiPartFile) (*http.Response, error) {
	return MultiPartFormRequestWithHeaders(url, nil, params, files)
}

func MultiPartFormRequestWithHeaders(url string, headers map[string]string, params []MultiPartParam, files []MultiPartFile) (*http.Response, error) {
	log.Debug().
		Str("url", url).
		Interface("params", params).
		Interface("headers", headers).
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
		Str("url", fileUrl).
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
