// Package btcc 提供 BTCC Open API 的 Go 客户端。
// 交易地址必须由调用者提供；SDK 不自动重试、重登或启动后台任务。
package btcc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/btcc-nova/OPENAPI_SDK/go/internal/wire"
)

// Options 配置交易与行情地址以及 HTTP 超时。
// HTTPClient 会被浅复制，SDK 禁止重定向，不修改调用者的客户端。
// Timeout 为每次请求的上限，零值表示 10 秒。
type Options struct {
	BaseURL    string
	QuoteURL   string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// Client 可并发使用；创建后配置不可变。
type Client struct{ transport *wire.Client }

// NewClient 创建客户端，不发出网络请求。QuoteURL 可省略。
func NewClient(options Options) (*Client, error) {
	transport, err := wire.New(options.BaseURL, options.QuoteURL, options.HTTPClient, options.Timeout)
	if err != nil {
		return nil, err
	}
	return &Client{transport: transport}, nil
}

// APIError 保留服务端的业务错误码和消息。Error 不输出可能含敏感内容的消息。
// Message 来自远端，记录或展示前请按业务要求处理。
type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("btcc: API error code=%d", e.Code) }

// HTTPError 表示非 2xx 响应，可通过 errors.As 检查 StatusCode。
type HTTPError = wire.HTTPError

// Flag 兼容服务端配置字段的 true/false 和 1/0 两种表示。
type Flag bool

func (f *Flag) UnmarshalJSON(data []byte) error {
	value, err := wire.DecodeFlag(data)
	if err == nil {
		*f = Flag(value)
	}
	return err
}

// Session 保存一次登录的凭证，绑定创建它的 Client；不可直接构造。
// 内部字段不可变，允许并发请求；不会通过 JSON 序列化暴露凭证。
type Session struct {
	client    *Client
	token     string
	secret    string
	accountID uint64
	groupID   uint64
}

// LoginRequest 包含登录凭证。CompanyID 零值按文档使用 1。
// 请勿记录整个请求；凭证字段不会被 JSON 序列化。
type LoginRequest struct {
	UserName  string `json:"-"`
	Password  string `json:"-"`
	APIKey    string `json:"-"`
	SecretKey string `json:"-"`
	CompanyID uint32
}

// LoginResult 返回账户信息及后续请求使用的 Session。
type LoginResult struct {
	Session    *Session `json:"-"`
	Account    Account  `json:"account"`
	ServerTime int64    `json:"server_time"`
}

// KeepAliveResult 表示心跳结果。
type KeepAliveResult struct {
	AccountID  uint64 `json:"accountid"`
	ServerTime int64  `json:"server_time"`
}

// PositionListResult 表示一页仓位；SDK 不自动翻页。
type PositionListResult struct {
	Positions []Position `json:"positions"`
	Total     int        `json:"total"`
}

// OpenPositionResult 包含开仓后的仓位及止盈止损委托。
type OpenPositionResult struct {
	Position   Position    `json:"position"`
	SLTPOrders []SLTPOrder `json:"sltp_orders"`
}

// PositionResult 包含平仓或修改后的仓位。
type PositionResult struct {
	Position Position `json:"position"`
}

// DirectionBuy 和 DirectionSell 是服务端的买卖方向值。
const (
	DirectionBuy  uint32 = 1
	DirectionSell uint32 = 2
)

// Login 登录，但不自动启动心跳。
func (c *Client) Login(ctx context.Context, req LoginRequest) (*LoginResult, error) {
	for _, field := range []struct{ name, value string }{{"user_name", req.UserName}, {"password", req.Password}, {"api_key", req.APIKey}, {"secret_key", req.SecretKey}} {
		if err := wire.Required(field.name, field.value); err != nil {
			return nil, err
		}
	}
	if req.CompanyID == 0 {
		req.CompanyID = 1
	}
	params := url.Values{"user_name": {req.UserName}, "password": {req.Password}, "api_key": {req.APIKey}, "company_id": {wire.Uint(uint64(req.CompanyID))}}
	var response struct {
		Token      string  `json:"token"`
		Account    Account `json:"account"`
		ServerTime int64   `json:"server_time"`
	}
	if err := c.request(ctx, http.MethodPost, "/v1/user/login", params, req.SecretKey, false, &response); err != nil {
		return nil, err
	}
	if wire.Required("token", response.Token) != nil || response.Account.ID == 0 || response.Account.AccountGroupID == 0 {
		return nil, errors.New("btcc: incomplete login response")
	}
	session := &Session{client: c, token: response.Token, secret: req.SecretKey, accountID: response.Account.ID, groupID: response.Account.AccountGroupID}
	return &LoginResult{Session: session, Account: response.Account, ServerTime: response.ServerTime}, nil
}

// KeepAlive 发送一次心跳。
func (c *Client) KeepAlive(ctx context.Context, s *Session) (*KeepAliveResult, error) {
	params, err := c.sessionParams(s, false)
	if err != nil {
		return nil, err
	}
	var result KeepAliveResult
	if err := c.request(ctx, http.MethodGet, "/v1/user/keepalive", params, s.secret, false, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RunKeepAlive 阻塞运行心跳，间隔零值表示 20 秒。
// 调用者负责启动 goroutine、取消 ctx 并等待返回；首次请求在一个间隔后发出。
// 取消或首次心跳失败即返回；不会自动重登、重试或使 Session 失效。
func (c *Client) RunKeepAlive(ctx context.Context, s *Session, interval time.Duration) error {
	if ctx == nil {
		return errors.New("btcc: context is required")
	}
	if _, err := c.sessionParams(s, false); err != nil {
		return err
	}
	if interval < 0 {
		return errors.New("btcc: keepalive interval must not be negative")
	}
	if interval == 0 {
		interval = 20 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := c.KeepAlive(ctx, s); err != nil {
				return err
			}
		}
	}
}

// GetSymbolList 获取当前账户组的产品配置。
func (c *Client) GetSymbolList(ctx context.Context, s *Session) ([]Symbol, error) {
	params, err := c.sessionParams(s, true)
	if err != nil {
		return nil, err
	}
	var result struct {
		Symbols []Symbol `json:"symbols"`
	}
	if err := c.request(ctx, http.MethodGet, "/v1/config/symbollist", params, s.secret, false, &result); err != nil {
		return nil, err
	}
	return result.Symbols, nil
}

// GetAccountGroup 获取当前账户组信息。
func (c *Client) GetAccountGroup(ctx context.Context, s *Session) (*AccountGroup, error) {
	params, err := c.sessionParams(s, true)
	if err != nil {
		return nil, err
	}
	var result struct {
		AccountGroup AccountGroup `json:"account_group"`
	}
	if err := c.request(ctx, http.MethodGet, "/v1/config/accountgroup", params, s.secret, false, &result); err != nil {
		return nil, err
	}
	return &result.AccountGroup, nil
}

// GetAccount 获取资金账户信息。
func (c *Client) GetAccount(ctx context.Context, s *Session) (*Account, error) {
	params, err := c.sessionParams(s, false)
	if err != nil {
		return nil, err
	}
	var result struct {
		Account Account `json:"account"`
	}
	if err := c.request(ctx, http.MethodGet, "/v1/account/account", params, s.secret, false, &result); err != nil {
		return nil, err
	}
	return &result.Account, nil
}

// GetPositionList 获取一页仓位，负数分页参数会被拒绝。
func (c *Client) GetPositionList(ctx context.Context, s *Session, filter PositionListFilter) (*PositionListResult, error) {
	params, err := c.sessionParams(s, false)
	if err != nil {
		return nil, err
	}
	if filter.PageNo < 0 || filter.PageSize < 0 {
		return nil, errors.New("btcc: pagination must not be negative")
	}
	if filter.PageNo == 0 {
		filter.PageNo = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 100
	}
	params.Set("pageNo", strconv.Itoa(filter.PageNo))
	params.Set("pageSize", strconv.Itoa(filter.PageSize))
	if filter.Symbol != "" {
		params.Set("symbol", filter.Symbol)
	}
	if filter.ID != 0 {
		params.Set("id", wire.Uint(filter.ID))
	}
	var result PositionListResult
	if err := c.request(ctx, http.MethodGet, "/v1/account/positionlist", params, s.secret, false, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// OpenPosition 开仓；不自动重试，超时不代表订单未成交。
func (c *Client) OpenPosition(ctx context.Context, s *Session, req OpenPositionRequest) (*OpenPositionResult, error) {
	params, err := c.sessionParams(s, false)
	if err != nil {
		return nil, err
	}
	if err := wire.Required("symbol", req.Symbol); err != nil {
		return nil, err
	}
	if req.Direction != DirectionBuy && req.Direction != DirectionSell {
		return nil, errors.New("btcc: direction must be 1 or 2")
	}
	if req.RequestVolume.Sign() <= 0 || req.RequestPrice.Sign() <= 0 || req.Multiple == 0 {
		return nil, errors.New("btcc: volume, price and multiple must be positive")
	}
	params.Set("direction", wire.Uint(uint64(req.Direction)))
	params.Set("symbol", req.Symbol)
	params.Set("request_volume", req.RequestVolume.String())
	params.Set("request_price", req.RequestPrice.String())
	params.Set("multiple", wire.Uint(uint64(req.Multiple)))
	if req.RefID != 0 {
		params.Set("refid", wire.Uint(req.RefID))
	}
	if req.RefAccountID != 0 {
		params.Set("ref_accid", wire.Uint(req.RefAccountID))
	}
	if !req.StopLoss.IsZero() {
		params.Set("stop_loss", req.StopLoss.String())
	}
	if !req.TakeProfit.IsZero() {
		params.Set("take_profit", req.TakeProfit.String())
	}
	var result OpenPositionResult
	if err := c.request(ctx, http.MethodPost, "/v1/account/openposition", params, s.secret, false, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ClosePosition 平仓，方向按请求原样发送；所有非零业务错误码均返回 APIError。
func (c *Client) ClosePosition(ctx context.Context, s *Session, req ClosePositionRequest) (*PositionResult, error) {
	params, err := c.sessionParams(s, false)
	if err != nil {
		return nil, err
	}
	if err := wire.Required("symbol", req.Symbol); err != nil {
		return nil, err
	}
	if req.Direction != DirectionBuy && req.Direction != DirectionSell {
		return nil, errors.New("btcc: direction must be 1 or 2")
	}
	if req.PositionID == 0 || req.RequestVolume.Sign() <= 0 || req.RequestPrice.Sign() <= 0 {
		return nil, errors.New("btcc: positionid, volume and price must be positive")
	}
	params.Set("positionid", wire.Uint(req.PositionID))
	params.Set("direction", wire.Uint(uint64(req.Direction)))
	params.Set("symbol", req.Symbol)
	params.Set("request_volume", req.RequestVolume.String())
	params.Set("request_price", req.RequestPrice.String())
	if req.RefID != 0 {
		params.Set("refid", wire.Uint(req.RefID))
	}
	var result PositionResult
	if err := c.request(ctx, http.MethodPost, "/v1/account/closeposition", params, s.secret, false, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdatePosition 修改止盈止损；两个字段都会发送，包括零值。
func (c *Client) UpdatePosition(ctx context.Context, s *Session, req UpdatePositionRequest) (*PositionResult, error) {
	params, err := c.sessionParams(s, false)
	if err != nil {
		return nil, err
	}
	if err := wire.Required("symbol", req.Symbol); err != nil {
		return nil, err
	}
	if req.ID == 0 {
		return nil, errors.New("btcc: position id is required")
	}
	params.Set("id", wire.Uint(req.ID))
	params.Set("symbol", req.Symbol)
	params.Set("stop_loss", req.StopLoss.String())
	params.Set("take_profit", req.TakeProfit.String())
	var result PositionResult
	if err := c.request(ctx, http.MethodPost, "/v1/account/updateposition", params, s.secret, false, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetQuotePrice 从独立行情地址查询价格，不发送交易凭证。
// 包含 -4 在内的非零业务错误码均返回 APIError，不伪造零价格。
func (c *Client) GetQuotePrice(ctx context.Context, symbol string) (*QuotePrice, error) {
	if err := wire.Required("Symbol", symbol); err != nil {
		return nil, err
	}
	var result struct {
		Data *QuotePrice `json:"data"`
	}
	if err := c.request(ctx, http.MethodGet, "/quot/reqOnePrdPrice", url.Values{"Symbol": {symbol}}, "", true, &result); err != nil {
		return nil, err
	}
	if result.Data == nil {
		return nil, errors.New("btcc: quote response is missing data")
	}
	return result.Data, nil
}

func (c *Client) sessionParams(s *Session, group bool) (url.Values, error) {
	if c == nil || c.transport == nil {
		return nil, errors.New("btcc: use NewClient to create a client")
	}
	if s == nil || s.client != c || s.token == "" || s.secret == "" {
		return nil, errors.New("btcc: session must belong to this client")
	}
	params := url.Values{"token": {s.token}}
	if group {
		params.Set("account_group_id", wire.Uint(s.groupID))
	} else {
		params.Set("accountid", wire.Uint(s.accountID))
	}
	return params, nil
}

func (c *Client) request(ctx context.Context, method, path string, params url.Values, secret string, quote bool, out any) error {
	if c == nil || c.transport == nil {
		return errors.New("btcc: use NewClient to create a client")
	}
	result, err := c.transport.Do(ctx, method, path, params, secret, quote)
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return &APIError{Code: result.Code, Message: result.Message}
	}
	if err := json.Unmarshal(result.Body, out); err != nil {
		return errors.New("btcc: invalid response fields")
	}
	return nil
}
