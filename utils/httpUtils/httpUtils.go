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

type MultiPartParam struct {
	Name  string
	Value string
}

type MultiPartFile struct {
	FieldName string
	FileName  string
	Content   io.Reader
}

func GetRequest(url string, result any) error {
	log.Debug().
		Str("url", url).
		Send()

	resp, err := DefaultHttpClient.Get(url)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return &HttpError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close response body")
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, result); err != nil {
		return err
	}

	log.Debug().
		Str("url", url).
		Interface("result", result).
		Send()
	return nil
}

type HttpOptions struct {
	Client *http.Client
}

func PostRequest(url string, headers map[string]string, input any, result any, options *HttpOptions) error {
	log.Debug().
		Str("url", url).
		Interface("input", input).
		Interface("headers", headers).
		Send()

	var reqBody io.Reader
	isJson := true
	switch v := input.(type) {
	case io.ReadCloser:
		isJson = false
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

	req, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return err
	}

	if isJson {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	httpClient := DefaultHttpClient

	if options != nil {
		if options.Client != nil {
			httpClient = options.Client
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return &HttpError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close response body")
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, result); err != nil {
		return err
	}

	log.Debug().
		Str("url", url).
		Interface("result", result).
		Send()
	return nil
}

func GetRequestWithHeader(url string, headers map[string]string, result any) error {
	log.Debug().
		Str("url", url).
		Interface("headers", headers).
		Send()

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := DefaultHttpClient.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return &HttpError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close response body")
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, result); err != nil {
		return err
	}

	log.Debug().
		Str("url", url).
		Interface("result", result).
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
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
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

	if resp.StatusCode != 200 {
		return nil, &HttpError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	return resp.Body, nil
}
