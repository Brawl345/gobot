package utils

import (
	"bytes"
	"encoding/json"
	"gopkg.in/telebot.v3"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
)

const MaxFilesizeDownload = int(20e6)

var DefaultSendOptions = &telebot.SendOptions{
	AllowWithoutReply:     true,
	DisableWebPagePreview: true,
	DisableNotification:   true,
	ParseMode:             telebot.ModeHTML,
}

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
)

func IsAdmin(user *telebot.User) bool {
	adminId, _ := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	return adminId == user.ID
}

func GetRequest(url string, result any) error {
	resp, err := http.Get(url)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return &HttpError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, result); err != nil {
		return err
	}

	return nil
}

func MultiPartFormRequest(url string, params []MultiPartParam, files []MultiPartFile) (*http.Response, error) {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	defer writer.Close()

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

	writer.Close()

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	return client.Do(req)
}
