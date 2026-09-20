package imgbed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
	"github.com/kiteyuan/nexa-pulse-bot/internal/ports"
)

// Client uploads to an existing Telegraph-Image / Cloudflare ImgBed instance.
type Client struct {
	BaseURL  string
	AuthCode string
	HTTP     *http.Client
}

var _ ports.ImageUploader = (*Client)(nil)

func (c *Client) Upload(ctx context.Context, filename string, data []byte) (string, error) {
	if c == nil || strings.TrimSpace(c.BaseURL) == "" {
		return "", errors.New("未配置图床")
	}
	base, err := kernel.AbsoluteHTTPURL(c.BaseURL)
	if err != nil {
		return "", err
	}
	if filename == "" {
		filename = "image.jpg"
	}
	filename = path.Base(filename)
	q := url.Values{}
	q.Set("uploadChannel", "telegram")
	q.Set("returnFormat", "full")
	if c.AuthCode != "" {
		q.Set("authCode", c.AuthCode)
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(data); err != nil {
		return "", err
	}
	if err := form.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/upload?"+q.Encode(), &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	resp, err := c.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return "", errors.New("图床上传失败")
	}
	link, err := parseLink(base, raw)
	if err != nil {
		return "", err
	}
	if err := sameHost(base, link); err != nil {
		return "", err
	}
	return link, nil
}

func sameHost(base, link string) error {
	baseURL, err := url.Parse(base)
	if err != nil || baseURL.Host == "" {
		return errors.New("图床地址无效")
	}
	u, err := url.Parse(link)
	if err != nil || u.Host == "" {
		return errors.New("图床返回地址无效")
	}
	if !strings.EqualFold(u.Host, baseURL.Host) {
		return errors.New("图床返回地址无效")
	}
	return nil
}

func (c *Client) client() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func parseLink(base string, raw []byte) (string, error) {
	var items []linkItem
	if err := json.Unmarshal(raw, &items); err == nil && len(items) > 0 {
		return items[0].abs(base)
	}
	var wrapped struct {
		Data []linkItem `json:"data"`
		Src  string     `json:"src"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return "", errors.New("图床返回无法解析")
	}
	if len(wrapped.Data) > 0 {
		return wrapped.Data[0].abs(base)
	}
	if wrapped.Src != "" {
		return linkItem{Src: wrapped.Src}.abs(base)
	}
	return "", errors.New("图床未返回链接")
}

type linkItem struct {
	Src       string `json:"src"`
	PublicURL string `json:"publicUrl"`
}

func (it linkItem) abs(base string) (string, error) {
	if strings.HasPrefix(it.PublicURL, "http://") || strings.HasPrefix(it.PublicURL, "https://") {
		return it.PublicURL, nil
	}
	src := it.Src
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return src, nil
	}
	if src == "" {
		return "", errors.New("图床未返回链接")
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimPrefix(src, "/"), nil
}
