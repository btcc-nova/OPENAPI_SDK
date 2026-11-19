package btcc

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
)

// QuotePrice 表示 Open API 的产品报价。
type QuotePrice struct {
	Symbol   string          `json:"Symbol"`
	Price    decimal.Decimal `json:"Price"`
	LastTime int64           `json:"lastTime"`
}

// Account 表示 Open API 的资金账户。
type Account struct {
	ID             uint64          `json:"id"`
	UserID         uint64          `json:"userid"`
	AccountGroupID uint64          `json:"account_groupid"`
	Balance        decimal.Decimal `json:"balance"`
	Equity         decimal.Decimal `json:"equity"`
	Margin         decimal.Decimal `json:"margin"`
	FreeMargin     decimal.Decimal `json:"free_margin"`
	MarginLevel    decimal.Decimal `json:"margin_level"`
	MarginStopout  decimal.Decimal `json:"margin_stopout"`
	AccountNo      string          `json:"account_no"`
	Positions      int64           `json:"positions"`
	Bonus          decimal.Decimal `json:"bonus"`
	Status         int             `json:"status"`
}

// SymbolMarginLevel 表示 Open API 的保证金档位。
type SymbolMarginLevel struct {
	SymbolID                 uint64          `json:"symbol_id"`
	RangeMin                 decimal.Decimal `json:"range_min"`
	RangeMax                 decimal.Decimal `json:"range_max"`
	MarginInitial            decimal.Decimal `json:"margin_initial"`
	MarginMaintenance        decimal.Decimal `json:"margin_maintenance"`
	MarginStopout            decimal.Decimal `json:"margin_stopout"`
	MarginHedged             decimal.Decimal `json:"margin_hedged"`
	SpecialMarginInitial     decimal.Decimal `json:"special_margin_initial"`
	SpecialMarginMaintenance decimal.Decimal `json:"special_margin_maintenance"`
	SpecialMarginStopout     decimal.Decimal `json:"special_margin_stopout"`
	SpecialMarginHedged      decimal.Decimal `json:"special_margin_hedged"`
	HolidayMarginInitial     decimal.Decimal `json:"holiday_margin_initial"`
	HolidayMarginMaintenance decimal.Decimal `json:"holiday_margin_maintenance"`
	HolidayMarginStopout     decimal.Decimal `json:"holiday_margin_stopout"`
	HolidayMarginHedged      decimal.Decimal `json:"holiday_margin_hedged"`
	CreateTime               int64           `json:"create_time"`
	CreateUserID             uint64          `json:"create_user_id"`
	ModiTime                 int64           `json:"modi_time"`
	ModiUserID               uint64          `json:"modi_user_id"`
	Status                   int             `json:"status"`
}

// Commission 表示 Open API 的手续费配置。
type Commission struct {
	SymbolID         uint64          `json:"symbol_id"`
	CommissionRebate decimal.Decimal `json:"commission_rebate"`
	TriggerMode      int             `json:"trigger_mode"`
	Type             int             `json:"type"`
	Method           int             `json:"method"`
	RangeMin         decimal.Decimal `json:"range_min"`
	RangeMax         decimal.Decimal `json:"range_max"`
	Value            decimal.Decimal `json:"value"`
	CurrencyType     int             `json:"currency_type"`
	Currency         string          `json:"currency"`
	Comment          string          `json:"comment"`
	Status           int             `json:"status"`
}

// SymbolEnable 保留服务端的产品状态数值，不将非零状态等同于允许交易。
// 兼容旧协议的布尔值：true 为 1，false 为 0。
type SymbolEnable int

func (s *SymbolEnable) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	switch string(data) {
	case "true":
		*s = 1
	case "false":
		*s = 0
	case "null":
		return errors.New("btcc: symbol enable must be an integer or boolean")
	default:
		var value int
		if err := json.Unmarshal(data, &value); err != nil {
			return errors.New("btcc: symbol enable must be an integer or boolean")
		}
		*s = SymbolEnable(value)
	}
	return nil
}

// Symbol 表示 Open API 的产品配置。
type Symbol struct {
	ID                 uint64              `json:"id"`
	UUID               uint64              `json:"uuid"`
	Name               string              `json:"name"`
	Enable             SymbolEnable        `json:"enable"`
	Digits             int                 `json:"digits"`
	VolumesMin         decimal.Decimal     `json:"volumes_min"`
	VolumesStep        decimal.Decimal     `json:"volumes_step"`
	VolumesMax         decimal.Decimal     `json:"volumes_max"`
	BaseCurrency       string              `json:"base_currency"`
	ProfitCurrency     string              `json:"profit_currency"`
	MarginCurrency     string              `json:"margin_currency"`
	Tradable           int                 `json:"tradable"`
	StartTime          int64               `json:"start_time"`
	ExpiryTime         int64               `json:"expiry_time"`
	CreateTime         int64               `json:"create_time"`
	Status             int                 `json:"status"`
	ContractSize       decimal.Decimal     `json:"contract_size"`
	SymbolMarginLevels []SymbolMarginLevel `json:"symbol_margin_level"`
	Commissions        []Commission        `json:"commissions"`
	LongSwap           decimal.Decimal     `json:"long_swap"`
	ShortSwap          decimal.Decimal     `json:"short_swap"`
	StopLevel          decimal.Decimal     `json:"stop_level"`
	MaxStopLevel       decimal.Decimal     `json:"maxstop_level"`
	ThreeDaysSwap      decimal.Decimal     `json:"three_days_swap"`
	PipsRatio          decimal.Decimal     `json:"pips_ratio"`
	Spread             decimal.Decimal     `json:"spread"`
	SpreadBalance      decimal.Decimal     `json:"spread_balance"`
	RangeInit          decimal.Decimal     `json:"range_init"`
	CodeID             uint64              `json:"code_id"`
	PositionVolumesMax decimal.Decimal     `json:"position_volumes_max"`
	MarginInitial      decimal.Decimal     `json:"margin_initial"`
	CalcMarginType     int                 `json:"calc_margin_type"`
	DefaultFirst       int                 `json:"default_first"`
	VolumesList        string              `json:"volumes_list"`
	MaxPosValue        decimal.Decimal     `json:"max_pos_value"`
	GapValue           decimal.Decimal     `json:"gap_value"`
}

// AccountGroup 表示 Open API 的账户组配置。
type AccountGroup struct {
	CompanyID       uint64          `json:"companyid"`
	Currency        string          `json:"currency"`
	Enable          Flag            `json:"enable"`
	ID              uint64          `json:"id"`
	MarginCallLevel decimal.Decimal `json:"margin_call_level"`
	Name            string          `json:"name"`
	Platform        string          `json:"platform"`
	Status          int             `json:"status"`
	StopOutLevel    decimal.Decimal `json:"stop_out_level"`
	Digits          int             `json:"digits"`
}

// Position 表示 Open API 的仓位。
type Position struct {
	AccountID      uint64          `json:"accountid"`
	ClosePrice     decimal.Decimal `json:"close_price"`
	CloseTime      int64           `json:"close_time"`
	Commission     decimal.Decimal `json:"commission"`
	Direction      int             `json:"direction"`
	ExecCommission decimal.Decimal `json:"exec_commission"`
	ExecPrice      decimal.Decimal `json:"exec_price"`
	ExecProfit     decimal.Decimal `json:"exec_profit"`
	ExecSwap       decimal.Decimal `json:"exec_swap"`
	ExecTime       int64           `json:"exec_time"`
	ExecVolume     decimal.Decimal `json:"exec_volume"`
	GroupID        uint64          `json:"groupid"`
	ID             uint64          `json:"id"`
	InitVolume     decimal.Decimal `json:"init_volume"`
	Message        string          `json:"msg"`
	OpenPrice      decimal.Decimal `json:"open_price"`
	OpenTime       int64           `json:"open_time"`
	OrderRefID     uint64          `json:"order_refid"`
	Profit         decimal.Decimal `json:"profit"`
	Rate           decimal.Decimal `json:"rate"`
	RefAccountID   uint64          `json:"ref_accid"`
	Status         int             `json:"status"`
	StopLoss       decimal.Decimal `json:"stop_loss"`
	Swap           decimal.Decimal `json:"swap"`
	Symbol         string          `json:"symbol"`
	SymbolID       uint64          `json:"symbolid"`
	TakeProfit     decimal.Decimal `json:"take_profit"`
	Volume         decimal.Decimal `json:"volume"`
	DealID         uint64          `json:"deal_id"`
	OpenMargin     decimal.Decimal `json:"open_margin"`
	Multiple       uint16          `json:"multiple"`
	PosType        int             `json:"pos_type"`
}

// SLTPOrder 表示 Open API 的止盈止损委托。
type SLTPOrder struct {
	ID            uint64          `json:"id"`
	GroupID       uint64          `json:"groupid"`
	AccountID     uint64          `json:"accountid"`
	Direction     int             `json:"direction"`
	Type          int             `json:"type"`
	Status        int             `json:"status"`
	RequestVolume decimal.Decimal `json:"request_volume"`
	RequestPrice  decimal.Decimal `json:"request_price"`
	Symbol        string          `json:"symbol"`
	RequestTime   int64           `json:"request_time"`
	ExecuteVolume decimal.Decimal `json:"execute_volume"`
	ExecutePrice  decimal.Decimal `json:"execute_price"`
	ExecuteTime   int64           `json:"execute_time"`
	InitVolume    decimal.Decimal `json:"init_volume"`
	StopLoss      decimal.Decimal `json:"stop_loss"`
	TakeProfit    decimal.Decimal `json:"take_profit"`
	ExpireType    int             `json:"expire_type"`
	ExpireTime    int64           `json:"expire_time"`
	Multiple      uint16          `json:"multiple"`
	PositionID    uint64          `json:"pos_id"`
	RefID         uint64          `json:"ref_id"`
}

// PositionListFilter 表示 Open API 的仓位查询条件（页码默认 1，每页默认 100）。
type PositionListFilter struct {
	Symbol   string
	ID       uint64
	PageNo   int
	PageSize int
}

// OpenPositionRequest 表示 Open API 的开仓参数；价格与数量必须大于零。
type OpenPositionRequest struct {
	Direction     uint32
	Symbol        string
	RequestVolume decimal.Decimal
	RequestPrice  decimal.Decimal
	RefID         uint64
	RefAccountID  uint64
	StopLoss      decimal.Decimal
	TakeProfit    decimal.Decimal
	Multiple      uint16
}

// ClosePositionRequest 表示 Open API 的平仓参数；方向直接按调用者提供的值发送。
type ClosePositionRequest struct {
	PositionID    uint64
	Direction     uint32
	Symbol        string
	RequestVolume decimal.Decimal
	RequestPrice  decimal.Decimal
	RefID         uint64
}

// UpdatePositionRequest 表示 Open API 的止盈止损修改参数；零值也会发送。
type UpdatePositionRequest struct {
	ID         uint64
	StopLoss   decimal.Decimal
	TakeProfit decimal.Decimal
	Symbol     string
}
