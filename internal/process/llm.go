package process

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

type Review struct {
	Send       bool    `json:"send"`
	Title      string  `json:"title"`
	Body       string  `json:"body"`
	Type       string  `json:"type"`
	Importance float64 `json:"importance"`
	Reason     string  `json:"reason"`
}

func ReviewMessage(ctx context.Context, st kernel.Settings, content string, mediaCount int) (Review, error) {
	payload := map[string]any{
		"model":       st.LLMModel,
		"temperature": st.LLMTemperature,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt(st.TranslateTo)},
			{"role": "user", "content": userPrompt(content, mediaCount, st.TranslateTo)},
		},
		"response_format": map[string]string{"type": "json_object"},
	}
	raw, _ := json.Marshal(payload)
	endpoint, err := chatEndpoint(st.LLMBaseURL)
	if err != nil {
		return Review{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return Review{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if st.LLMAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+st.LLMAPIKey)
	}
	client := &http.Client{Timeout: 90 * time.Second, CheckRedirect: refuseRedirect}
	resp, err := client.Do(req)
	if err != nil {
		return Review{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return Review{}, fmt.Errorf("llm HTTP %d", resp.StatusCode)
	}
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return Review{}, err
	}
	if len(envelope.Choices) == 0 {
		return Review{}, fmt.Errorf("llm empty choices")
	}
	text := strings.TrimSpace(envelope.Choices[0].Message.Content)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	var review Review
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &review); err != nil {
		return Review{}, fmt.Errorf("llm json: %w", err)
	}
	if review.Type == "" {
		review.Type = "Other"
	}
	if review.Importance == 0 {
		review.Importance = 5
	}
	return review, nil
}

func chatEndpoint(base string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil || u.User != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("llm base url 无效")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/chat/completions"
	u.RawQuery, u.Fragment = "", ""
	return u.String(), nil
}

func refuseRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func systemPrompt(translateTo string) string {
	block := "翻译关闭。title/body 保持原文，只删除文末广告、频道签名和引流，不要改写正经内容。"
	if needs(translateTo) {
		block = "翻译开启，目标语言 " + translateTo + "。先删文末推广，再把 title 与 body 译为目标语言。产品名、公司名、模型名、CVE、缩写等专有名词保留原文。"
	}
	return "你是科技资讯审核员。只返回 JSON：{\"send\":true,\"title\":\"\",\"body\":\"\",\"type\":\"AI\",\"importance\":7,\"reason\":\"\"}。\n" + block + "\n拒绝广告、拉群、与科技无关的水贴。拒绝时 send=false。"
}

func userPrompt(content string, mediaCount int, translateTo string) string {
	note := ""
	if mediaCount > 0 {
		note = fmt.Sprintf("\n（另有 %d 张图片，不要写入 body）", mediaCount)
	}
	return "审核并整理下面的消息。只返回 JSON。\n\n" + content + note
}

func needs(code string) bool {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "", "off", "none", "false", "0":
		return false
	default:
		return true
	}
}
