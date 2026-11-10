// Package wire 实现 SDK 内部的签名与 HTTP 协议。
package wire

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxResponseBytes = 8 << 20

// HTTPError 表示非 2xx HTTP 状态，不包含可能泄露凭证的响应正文。
type HTTPError struct{ StatusCode int }

func (e *HTTPError) Error() string { return fmt.Sprintf("btcc: HTTP status %d", e.StatusCode) }

// Response 保存协议公共字段与原始 JSON。
type Response struct {
	Code    int
	Message string
	Body    []byte
}

// Client 不持有登录状态，可由多个会话并发使用。
type Client struct {
	BaseURL  string
	QuoteURL string
	HTTP     *http.Client
	Timeout  time.Duration
}

func New(base, quote string, client *http.Client, timeout time.Duration) (*Client, error) {
	if err := ValidateURL(base); err != nil {
		return nil, fmt.Errorf("btcc: BaseURL: %w", err)
	}
	if quote != "" {
		if err := ValidateURL(quote); err != nil {
			return nil, fmt.Errorf("btcc: QuoteURL: %w", err)
		}
	}
	if timeout < 0 {
		return nil, errors.New("btcc: Timeout must not be negative")
	}
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	h := http.Client{}
	if client != nil {
		h = *client
	}
	// 不跟随重定向，避免泄露 token、签名和登录表单，亦不重放交易。
	h.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{BaseURL: strings.TrimRight(base, "/"), QuoteURL: strings.TrimRight(quote, "/"), HTTP: &h, Timeout: timeout}, nil
}

func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("invalid URL")
	}
	if (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return errors.New("expected an HTTP(S) URL without credentials, query or fragment")
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return errors.New("invalid port")
		}
	}
	return nil
}

// Sign 遵循服务端 MD5 约定，不对签名原文进行 URL 编码。
func Sign(params url.Values, secret string) string {
	keys := make([]string, 0, len(params)+1)
	for key := range params {
		if key != "sign" && key != "secret_key" {
			keys = append(keys, key)
		}
	}
	keys = append(keys, "secret_key")
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		value := params.Get(key)
		if key == "secret_key" {
			value = secret
		}
		pairs = append(pairs, key+"="+value)
	}
	sum := md5.Sum([]byte(strings.Join(pairs, "&")))
	return hex.EncodeToString(sum[:])
}

func (c *Client) Do(ctx context.Context, method, path string, params url.Values, secret string, quote bool) (*Response, error) {
	if ctx == nil {
		return nil, errors.New("btcc: context is required")
	}
	base := c.BaseURL
	if quote {
		if c.QuoteURL == "" {
			return nil, errors.New("btcc: QuoteURL is required for quotes")
		}
		base = c.QuoteURL
	} else {
		// 每次复制参数，避免修改调用者持有的数据。
		signed := make(url.Values, len(params)+1)
		for key, value := range params {
			signed[key] = append([]string(nil), value...)
		}
		signed.Set("sign", Sign(params, secret))
		params = signed
	}
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	endpoint := base + path
	var body io.Reader
	if method == http.MethodGet {
		endpoint += "?" + params.Encode()
	} else {
		body = strings.NewReader(params.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, errors.New("btcc: cannot construct request")
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		// net/http 的 url.Error 包含完整查询参数，不能原样返回。
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return nil, fmt.Errorf("btcc: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{StatusCode: resp.StatusCode}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("btcc: read response: %w", err)
	}
	if len(data) > maxResponseBytes {
		return nil, errors.New("btcc: response exceeds 8 MiB")
	}
	var envelope struct {
		Code    *int   `json:"code"`
		Message string `json:"msg"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, errors.New("btcc: invalid JSON response")
	}
	if envelope.Code == nil {
		return nil, errors.New("btcc: response is missing code")
	}
	return &Response{Code: *envelope.Code, Message: envelope.Message, Body: data}, nil
}

// DecodeFlag 接受协议中出现的 bool 或 0/1，不接受其他值。
func DecodeFlag(data []byte) (bool, error) {
	switch string(bytes.TrimSpace(data)) {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, errors.New("btcc: expected boolean or 0/1")
	}
}

func Required(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("btcc: %s is required", name)
	}
	return nil
}

func Uint(v uint64) string { return strconv.FormatUint(v, 10) }
