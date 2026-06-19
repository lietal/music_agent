package tme

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://u.y.qq.com/cgi-bin/musicu.fcg"
const streamDomain = "https://isure.stream.qqmusic.qq.com/"
const photoDomain = "https://y.gtimg.cn/music/photo_new/"
const photoSizeSegment = "T002R300x300M000"

type Client struct {
	baseURL    string
	httpClient *http.Client
	comm       CommParams
}

func NewClient() *Client {
	return &Client{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		comm: CommParams{
			Ct: 11,
		},
	}
}

func (c *Client) SetCredential(musicid, musickey string) {
	c.comm.QQ = musicid
	c.comm.Authst = musickey
}

func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

func (c *Client) Call(ctx context.Context, reqs map[string]MusicuSubRequest) (*MusicuResponse, error) {
	full := map[string]any{
		"comm": c.comm,
	}
	for key, req := range reqs {
		full[key] = req
	}

	body, err := json.Marshal(full)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tme api returned %d: %s", resp.StatusCode, string(truncateBytes(respBody, 500)))
	}

	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	result := &MusicuResponse{
		Code: int64(getFloat(raw, "code")),
		Req:  make(map[string]MusicuSubResponse),
	}

	for key := range reqs {
		if sub, ok := raw[key]; ok {
			if subMap, ok := sub.(map[string]any); ok {
				code := int64(getFloat(subMap, "code"))
				data, _ := subMap["data"].(map[string]any)
				if data == nil {
					data = make(map[string]any)
				}
				result.Req[key] = MusicuSubResponse{Code: code, Data: data}
			}
		}
	}

	if result.Code != 0 {
		return result, fmt.Errorf("tme api error: code=%d", result.Code)
	}

	return result, nil
}

func (c *Client) CallRaw(ctx context.Context, reqs map[string]MusicuSubRequest) (map[string]any, error) {
	full := map[string]any{
		"comm": c.comm,
	}
	for key, req := range reqs {
		full[key] = req
	}

	body, err := json.Marshal(full)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return raw, nil
}

func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case json.Number:
			f, _ := val.Float64()
			return f
		}
	}
	return 0
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case string:
			return val
		default:
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

func getStringField(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s := getString(m, key); s != "" {
			return s
		}
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	return int(getFloat(m, key))
}

func getStringSlice(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch arr := v.(type) {
	case []any:
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	case []string:
		return arr
	}
	return nil
}

func getMapSlice(m map[string]any, key string) []map[string]any {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	result := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			result = append(result, m)
		}
	}
	return result
}

func getNestedMapSlice(m map[string]any, keys ...string) []map[string]any {
	current := m
	for i, key := range keys {
		if i == len(keys)-1 {
			return getMapSlice(current, key)
		}
		v, ok := current[key]
		if !ok {
			return nil
		}
		next, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	return nil
}

func truncateBytes(b []byte, maxLen int) []byte {
	if len(b) > maxLen {
		return b[:maxLen]
	}
	return b
}

func buildArtworkURL(pmid string) string {
	if pmid == "" {
		return ""
	}
	if strings.HasPrefix(pmid, "http") {
		return pmid
	}
	return fmt.Sprintf("%s%s/%s.jpg", photoDomain, photoSizeSegment, pmid)
}
