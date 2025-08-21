package dcrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
)

var (
	RcKey = []byte("cb99b5cbc24db398")
	RcIV  = []byte("9bc24cb995cb8db3")
)

const ApiUrl = "http://service.jdownloader.org/dlcrypt/service.php?srcType=dlc&destType=pylo&data=%s"

type (
	Base64String string

	DLC struct {
		XMLName xml.Name `xml:"dlc"`
		Header  struct {
			Generator struct {
				App     Base64String `xml:"app"`
				Version Base64String `xml:"version"`
				URL     Base64String `xml:"url"`
			} `xml:"generator"`
			Tribute       Base64String `xml:"tribute"`
			Dlcxmlversion Base64String `xml:"dlcxmlversion"`
		} `xml:"header"`
		Content struct {
			Package []struct {
				Category Base64String `xml:"category,attr"`
				Comment  Base64String `xml:"comment,attr"`
				Name     Base64String `xml:"name,attr"`
				File     []struct {
					URL      Base64String `xml:"url"`
					Filename Base64String `xml:"filename"`
					Size     Base64String `xml:"size"`
				} `xml:"file"`
			} `xml:"package"`
		} `xml:"content"`
	}
)

// Every entry is based64 encoded, so we will decode it on-the-fly while unmarshalling.
func (b *Base64String) decode(content string) error {
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return errors.New("failed to decode base64 string")
	}
	*b = Base64String(decoded)
	return nil
}

func (b *Base64String) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var content string
	if err := d.DecodeElement(&content, &start); err != nil {
		return err
	}
	return b.decode(content)
}

func (b *Base64String) UnmarshalXMLAttr(attr xml.Attr) error {
	return b.decode(attr.Value)
}

func (d *DLC) HasLinks() bool {
	for _, pkg := range d.Content.Package {
		for _, file := range pkg.File {
			if file.URL != "" {
				return true
			}
		}
	}
	return false
}

func (d *DLC) GeneratedBy() string {
	generator := d.Header.Generator

	if generator.App == "" {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("Generiert von ")

	if generator.URL != "" {
		sb.WriteString(fmt.Sprintf("<a href=\"%s\">", generator.URL))
	}

	sb.WriteString(utils.Escape(string(generator.App)))
	if generator.Version != "" && generator.App != generator.Version {
		sb.WriteString(fmt.Sprintf(" %s", utils.Escape(string(generator.Version))))
	}

	if generator.URL != "" {
		sb.WriteString("</a>")
	}

	return sb.String()
}

func (d *DLC) TotalSize() string {
	var totalSize int64

	for _, pkg := range d.Content.Package {
		for _, file := range pkg.File {
			size, err := strconv.ParseInt(string(file.Size), 10, 64)
			if err != nil {
				return "" // Just give up when one is missing
			}
			totalSize += size
		}
	}

	if totalSize == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<b>Größe:</b> %s", utils.HumanizeSize(totalSize)))

	return sb.String()
}

func DecryptDLC(data []byte) (DLC, error) {
	data = bytes.TrimSpace(data)

	// Add padding if necessary
	padding := len(data) % 4
	if padding != 0 {
		data = append(data, bytes.Repeat([]byte("="), 4-padding)...)
	}

	encryptedDlcKey := data[len(data)-88:]
	encryptedDlcData, err := base64.StdEncoding.DecodeString(string(data[:len(data)-88]))
	if err != nil {
		return DLC{}, err
	}

	var resp string
	err = httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodGet,
		URL:      fmt.Sprintf(ApiUrl, string(encryptedDlcKey)),
		Response: &resp,
	})

	if err != nil {
		return DLC{}, err
	}

	rc, err := extractRC(resp)
	if err != nil {
		return DLC{}, err
	}

	key, err := decryptData(rc, RcKey, RcIV)
	if err != nil {
		return DLC{}, err
	}

	xmlData, err := decryptData(encryptedDlcData, key, key) // Yes, key is also the IV
	if err != nil {
		return DLC{}, err
	}

	trimmed := bytes.TrimSpace(xmlData)
	trimmed = bytes.TrimRight(trimmed, "\x00\u0010")
	trimmed = bytes.TrimRight(trimmed, "\x00")
	trimmed = bytes.TrimRight(trimmed, "\u0010")
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(trimmed)))
	if err != nil {
		return DLC{}, err
	}

	var dlc DLC
	err = xml.Unmarshal(decoded, &dlc)
	if err != nil {
		return DLC{}, err
	}

	return dlc, nil
}

func extractRC(dlcContent string) ([]byte, error) {
	re := regexp.MustCompile(`<rc>(.+)</rc>`)
	matches := re.FindStringSubmatch(dlcContent)
	if len(matches) < 2 {
		return nil, errors.New("RC not found in API response")
	}

	rc, err := base64.StdEncoding.DecodeString(matches[1])
	if err != nil {
		return nil, err
	}

	return rc[:16], nil
}

func decryptData(data []byte, key []byte, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(data))
	mode.CryptBlocks(decrypted, data)

	return decrypted, nil
}
