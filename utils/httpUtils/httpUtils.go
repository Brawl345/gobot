package httpUtils

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/Brawl345/gobot/logger"
)

var (
	log        = logger.New("httpUtils")
	HttpClient *http.Client
)

func init() {
	HttpClient = createHTTPClient()
}

func createHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: 10 * time.Second,
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

	resp, err := HttpClient.Get(url)

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

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := HttpClient.Do(req)

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
	log.Debug().
		Str("url", url).
		Interface("params", params).
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

	return HttpClient.Do(req)
}
