package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/Brawl345/gobot/logger"
	"gopkg.in/telebot.v3"
)

var (
	DefaultSendOptions = &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
		DisableNotification:   true,
		ParseMode:             telebot.ModeHTML,
	}
	log = logger.New("utils")
)

type (
	MultiPartParam struct {
		Name  string
		Value string
	}

	MultiPartFile struct {
		FieldName string
		FileName  string
		Content   io.Reader
	}

	VersionInfo struct {
		GoVersion  string
		GoOS       string
		GoArch     string
		Revision   string
		LastCommit time.Time
		DirtyBuild bool
	}
)

func GermanTimezone() *time.Location {
	timezone, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		log.Err(err).Msg("Failed to load timezone, using UTC")
		timezone, _ = time.LoadLocation("UTC")
	}
	return timezone
}

func AnyEntities(message *telebot.Message) telebot.Entities {
	entities := message.Entities
	if message.Entities == nil {
		entities = message.CaptionEntities
	}
	return entities
}

func ReadVersionInfo() (VersionInfo, error) {
	buildInfo, ok := debug.ReadBuildInfo()

	if !ok {
		return VersionInfo{}, errors.New("could not read build info")
	}

	versionInfo := VersionInfo{
		GoVersion: buildInfo.GoVersion,
	}

	for _, kv := range buildInfo.Settings {
		switch kv.Key {
		case "GOOS":
			versionInfo.GoOS = kv.Value
		case "GOARCH":
			versionInfo.GoArch = kv.Value
		case "vcs.revision":
			versionInfo.Revision = kv.Value
		case "vcs.time":
			versionInfo.LastCommit, _ = time.Parse(time.RFC3339, kv.Value)
		case "vcs.modified":
			versionInfo.DirtyBuild = kv.Value == "true"
		}
	}

	return versionInfo, nil
}

func IsAdmin(user *telebot.User) bool {
	adminId, _ := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	return adminId == user.ID
}

func GetRequest(url string, result any) error {
	log.Debug().
		Str("url", url).
		Send()

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(url)

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

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)

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

	client := &http.Client{}
	return client.Do(req)
}
