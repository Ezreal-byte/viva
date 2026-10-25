package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type usage struct{ Prompt, Completion int }

var client = &http.Client{Timeout: 6 * time.Minute}

// chat sends one chat completions request and returns the reply, the finish reason and
// the tokens used. It retries rate limits and server errors twice. The key only appears
// in the Authorization header, so no error text can contain it.
func chat(m Model, key string, messages []message) (reply, finish string, u usage, err error) {
	request := map[string]any{"model": m.id(), "messages": messages, "max_tokens": 16000}
	if m.MaxTokens != 0 {
		request["max_tokens"] = m.MaxTokens
	}
	if m.Temperature >= 0 {
		request["temperature"] = m.Temperature
	}
	if m.Extra != "" {
		var extra map[string]json.RawMessage
		if err = json.Unmarshal([]byte(m.Extra), &extra); err != nil {
			return "", "", u, fmt.Errorf("model %s: extra is not a JSON object: %v", m.Name, err)
		}
		for k, v := range extra {
			request[k] = v
		}
	}
	body, err := json.Marshal(request)
	if err != nil {
		return "", "", u, err
	}
	url := strings.TrimRight(m.BaseURL, "/") + "/chat/completions"
	var data []byte
	for attempt := 0; ; attempt++ {
		var status int
		data, status, err = post(url, key, body)
		if err == nil && status == 200 {
			break
		}
		if err == nil {
			err = fmt.Errorf("HTTP %d: %.300s", status, data)
		}
		if attempt == 2 || (status != 0 && status != 429 && status < 500) {
			return "", "", u, err
		}
		time.Sleep(time.Duration(attempt+1) * 3 * time.Second)
	}
	var response struct {
		Choices []struct {
			FinishReason string  `json:"finish_reason"`
			Message      message `json:"message"`
		} `json:"choices"`
		Usage struct {
			Prompt     int `json:"prompt_tokens"`
			Completion int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err = json.Unmarshal(data, &response); err != nil || len(response.Choices) == 0 {
		return "", "", u, fmt.Errorf("unreadable reply: %.200s", data)
	}
	c := response.Choices[0]
	return c.Message.Content, c.FinishReason, usage{response.Usage.Prompt, response.Usage.Completion}, nil
}

func post(url, key string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	return data, resp.StatusCode, err
}

var goBlock = regexp.MustCompile("(?s)```go\n(.*?)```")

// extractGo returns the longest ```go block; a reply cut off by the token limit yields
// the unterminated block, which then fails to compile and is repaired like any other.
func extractGo(reply string) (string, bool) {
	best := ""
	for _, m := range goBlock.FindAllStringSubmatch(reply, -1) {
		if len(m[1]) > len(best) {
			best = m[1]
		}
	}
	if best != "" {
		return best, true
	}
	if i := strings.Index(reply, "```go\n"); i >= 0 {
		return reply[i+6:], true
	}
	return "", false
}
