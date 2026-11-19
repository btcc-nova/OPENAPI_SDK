# BTCC Open API Go SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

這個 Go SDK 幫你連接 BTCC Open API：登入、查詢帳戶與倉位、讀取報價，以及開倉、平倉和修改停損停利。SDK 負責簽名與請求格式，你只需傳入參數並處理結果。

先執行唯讀範例，再串接交易。本文不會要求你用下單來測試連線。

## 1. 開始前準備什麼

- Go 1.22 或以上。
- 向服務方取得交易網址、使用者名稱、密碼、API Key、Secret Key 和公司 ID；需要報價時另取行情網址及行情商品名稱。
- 請使用服務方提供的 HTTPS 網址，避免以明文傳送登入憑證。

模組路徑為 [`github.com/btcc-nova/OPENAPI_SDK`](https://github.com/btcc-nova/OPENAPI_SDK)。套件名稱為 `btcc`，小數相依套件為 `github.com/shopspring/decimal v1.4.0`。

## 2. 在自己的專案中安裝

GitHub 倉庫發布後，在已有 `go.mod` 的專案中執行：

```sh
go get github.com/btcc-nova/OPENAPI_SDK@latest
```

不從 GitHub 安裝時，可在 SDK 根目錄執行以下指令，建立使用本機原始碼的範例專案：

```sh
SDK_DIR="$(pwd)"
mkdir btcc-demo
cd btcc-demo
go mod init local/btcc-demo
go mod edit -require=github.com/btcc-nova/OPENAPI_SDK@v0.0.0
go mod edit -replace="github.com/btcc-nova/OPENAPI_SDK=$SDK_DIR"
```

將下一節的程式儲存為範例專案中的 `main.go`。

## 3. 第一個程式：登入並查詢餘額

先在執行環境或金鑰管理工具中設定以下變數。不要把憑證寫進原始碼或提交到 Git，也不要把真實密碼直接寫進可能保留歷史紀錄的終端機指令。

| 變數 | 用途 |
|---|---|
| `BTCC_BASE_URL` | 必填，交易服務基礎網址；SDK 自動附加 API 路徑 |
| `BTCC_USER_NAME` | 必填，登入使用者名稱 |
| `BTCC_PASSWORD` | 必填，服務方要求的密碼值；SDK 原樣傳送，不額外雜湊 |
| `BTCC_API_KEY` | 必填，API Key |
| `BTCC_SECRET_KEY` | 必填，用於簽名的 Secret Key |
| `BTCC_COMPANY_ID` | 選填，預設 `1`；SDK 收到 `0` 也會使用 `1` |
| `BTCC_QUOTE_URL` | 僅查報價時需要，獨立的行情基礎網址 |

只有範例程式讀取這些變數。SDK 不讀取環境變數，也不會自動載入 `.env` 檔案。網址不能包含使用者名稱、密碼、查詢參數或片段；即使只查行情，建立客戶端仍需有效的 `BaseURL`。

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "strconv"
    "time"

    btcc "github.com/btcc-nova/OPENAPI_SDK"
)

func main() {
    if err := run(); err != nil {
        log.Fatal(err)
    }
}

func run() error {
    companyID := uint64(1)
    if value := os.Getenv("BTCC_COMPANY_ID"); value != "" {
        parsed, err := strconv.ParseUint(value, 10, 32)
        if err != nil {
            return fmt.Errorf("invalid BTCC_COMPANY_ID")
        }
        companyID = parsed
    }
    client, err := btcc.NewClient(btcc.Options{
        BaseURL: os.Getenv("BTCC_BASE_URL"),
        QuoteURL: os.Getenv("BTCC_QUOTE_URL"),
        Timeout: 10 * time.Second,
    })
    if err != nil {
        return err
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    login, err := client.Login(ctx, btcc.LoginRequest{
        UserName: os.Getenv("BTCC_USER_NAME"),
        Password: os.Getenv("BTCC_PASSWORD"),
        APIKey: os.Getenv("BTCC_API_KEY"),
        SecretKey: os.Getenv("BTCC_SECRET_KEY"),
        CompanyID: uint32(companyID),
    })
    if err != nil {
        return err
    }
    account, err := client.GetAccount(ctx, login.Session)
    if err != nil {
        return err
    }
    fmt.Printf("account=%d balance=%s free_margin=%s\n",
        account.ID, account.Balance.String(), account.FreeMargin.String())
    return nil
}
```

在範例專案中執行：

```sh
go mod tidy
go run .
```

成功時會顯示帳戶 ID、餘額和可用保證金；不會下單。餘額不是固定值。`Session` 是登入後回傳的會話憑證，只能用於建立它的 `Client`。不要自行建構，也不要跨客戶端使用。

## 4. 查詢商品、倉位與報價

儲存為範例專案中的 `queries.go`。要查詢商品、帳戶群組與全部倉位，可把 `main.go` 中 `run()` 最後的 `return nil` 改為 `return inspect(ctx, client, login.Session)`。它仍然只查詢，不交易。

```go
package main

import (
    "context"
    "fmt"

    btcc "github.com/btcc-nova/OPENAPI_SDK"
)

func inspect(ctx context.Context, client *btcc.Client, session *btcc.Session) error {
    symbols, err := client.GetSymbolList(ctx, session)
    if err != nil {
        return err
    }
    for _, s := range symbols {
        fmt.Printf("symbol=%s enable=%d min=%s step=%s max=%s\n",
            s.Name, s.Enable, s.VolumesMin, s.VolumesStep, s.VolumesMax)
    }
    group, err := client.GetAccountGroup(ctx, session)
    if err != nil {
        return err
    }
    fmt.Printf("group=%d currency=%s enable=%t\n", group.ID, group.Currency, bool(group.Enable))

    const pageSize = 100
    seen := 0
    for page := 1; ; page++ {
        result, err := client.GetPositionList(ctx, session, btcc.PositionListFilter{
            PageNo: page, PageSize: pageSize,
        })
        if err != nil {
            return err
        }
        for _, p := range result.Positions {
            fmt.Printf("position=%d symbol=%s volume=%s\n", p.ID, p.Symbol, p.Volume)
        }
        seen += len(result.Positions)
        if len(result.Positions) == 0 || seen >= result.Total {
            return nil
        }
    }
}

func printQuote(ctx context.Context, client *btcc.Client, quoteSymbol string) error {
    quote, err := client.GetQuotePrice(ctx, quoteSymbol)
    if err != nil {
        return err
    }
    fmt.Printf("symbol=%s price=%s time=%d\n", quote.Symbol, quote.Price, quote.LastTime)
    return nil
}
```

- 商品名稱使用 `GetSymbolList` 回傳的 `Symbol.Name`。下單前檢查數量最小值、增量、最大值及服務方的交易規則。
- `Symbol.Enable` 是數值狀態，可能為 `3`；不要把所有非零值都當作可交易。它相容布林輸入，但不支援 `bool(s.Enable)`。`AccountGroup.Enable` 則可用 `bool(group.Enable)` 轉換。
- 倉位查詢預設 `PageNo=1`、`PageSize=100`，SDK 不自動翻頁。範例逐頁讀取；持倉在讀取期間可能改變，這不是帳戶快照。可用 `PositionListFilter.Symbol` 或 `ID` 篩選，負數分頁參數會被拒絕。
- 只查報價時，將 `run()` 最後的回傳改為 `return printQuote(ctx, client, os.Getenv("BTCC_QUOTE_SYMBOL"))`，並設定 `BTCC_QUOTE_URL` 和 `BTCC_QUOTE_SYMBOL`。行情名稱由服務方確認，不要直接假定它與交易商品名稱相同。
- `GetQuotePrice` 本身不需要登入，也不傳送 API Key、token 或簽名；這裡沿用登入範例只是為了重用客戶端。SDK 會正確編碼商品名稱中的特殊字元。

## 5. 交易範例：開倉、平倉、修改停損停利

警告：以下函式被呼叫時會傳送真實交易請求。僅儲存或編譯檔案不會下單。先在獲准的測試環境驗證，確認商品、方向、價格、數量和槓桿後再使用；不要把它們接到前面的連線檢查。

儲存為 `trades.go`。呼叫者傳入已檢查過的參數，範例不寫死一個可能誤下單的商品或價格。價格和數量用字串傳入，例如 `"0.001"`，由 `decimal.NewFromString` 轉成精確小數，不要先經過 `float64`。

```go
package main

import (
    "context"

    btcc "github.com/btcc-nova/OPENAPI_SDK"
    "github.com/shopspring/decimal"
)

func openBuy(ctx context.Context, client *btcc.Client, session *btcc.Session,
    symbol, volumeText, priceText string, multiple uint16,
) (*btcc.OpenPositionResult, error) {
    volume, err := decimal.NewFromString(volumeText)
    if err != nil {
        return nil, err
    }
    price, err := decimal.NewFromString(priceText)
    if err != nil {
        return nil, err
    }
    return client.OpenPosition(ctx, session, btcc.OpenPositionRequest{
        Direction: btcc.DirectionBuy,
        Symbol: symbol,
        RequestVolume: volume,
        RequestPrice: price,
        Multiple: multiple,
    })
}

func closePosition(ctx context.Context, client *btcc.Client, session *btcc.Session,
    positionID uint64, symbol string, direction uint32, volumeText, priceText string,
) (*btcc.PositionResult, error) {
    volume, err := decimal.NewFromString(volumeText)
    if err != nil {
        return nil, err
    }
    price, err := decimal.NewFromString(priceText)
    if err != nil {
        return nil, err
    }
    return client.ClosePosition(ctx, session, btcc.ClosePositionRequest{
        PositionID: positionID,
        Direction: direction,
        Symbol: symbol,
        RequestVolume: volume,
        RequestPrice: price,
    })
}

func setStops(ctx context.Context, client *btcc.Client, session *btcc.Session,
    positionID uint64, symbol, stopLossText, takeProfitText string,
) (*btcc.PositionResult, error) {
    stopLoss, err := decimal.NewFromString(stopLossText)
    if err != nil {
        return nil, err
    }
    takeProfit, err := decimal.NewFromString(takeProfitText)
    if err != nil {
        return nil, err
    }
    return client.UpdatePosition(ctx, session, btcc.UpdatePositionRequest{
        ID: positionID,
        Symbol: symbol,
        StopLoss: stopLoss,
        TakeProfit: takeProfit,
    })
}
```

- `openBuy` 開買倉；賣倉使用 `btcc.DirectionSell`。開倉價格、數量及 `Multiple` 必須大於零，槓桿範圍仍需符合伺服器規則。成功結果含 `Position` 和 `SLTPOrders`。
- `closePosition` 使用已查到的倉位 ID，方向為 `DirectionBuy`（1）或 `DirectionSell`（2）。SDK 不推斷或反轉平倉方向，必須先向服務方確認規則。價格和數量必須大於零。
- `setStops` 每次都傳送停損與停利。傳入 `"0"` 不是省略欄位，零值的業務意義需要服務方確認；不要只想改一個欄位，卻把另一個留為零。
- 平倉參數是 `PositionID`，修改參數是 `ID`；線上欄位分別為 `positionid` 和 `id`。兩個操作均回傳 `PositionResult.Position`。
- 開倉還可設定 `RefID`、`RefAccountID`、`StopLoss`、`TakeProfit`；這些欄位零值時不傳送。平倉可設定 `RefID`，零值時不傳送。

SDK 不重試、不自動重新登入，也不提供防重複下單保證。`RefID` 不代表伺服器保證請求只執行一次。逾時、斷線或取消請求後，交易仍可能已經成交；先核對伺服器狀態，再決定是否重試。

## 6. 長時間執行：維持會話

登入不會自動啟動心跳。短查詢通常執行後結束；長時間執行的程式需自己安排 `KeepAlive`，或使用以下會持續等待的函式。儲存為 `heartbeat.go`：

```go
package main

import (
    "context"
    "time"

    btcc "github.com/btcc-nova/OPENAPI_SDK"
)

func keepSessionAlive(ctx context.Context, client *btcc.Client, session *btcc.Session) error {
    return client.RunKeepAlive(ctx, session, 20*time.Second)
}
```

`RunKeepAlive` 會一直執行到取消或第一次心跳失敗。若同時執行其他業務，由呼叫者用 goroutine 啟動它，透過 channel 接收回傳的錯誤；不要忽略錯誤。結束業務時取消 context，並等待心跳結束。

間隔傳 `0` 時為 20 秒，首次心跳在一個間隔後傳送；負數間隔會被拒絕。失敗時不會自動重新登入，也不會改變 Session 的本機狀態，不能據此認定會話仍有效。`Client` 和 `Session` 可並行共用，但不要同時修改請求資料或自訂 HTTP Transport。SDK 沒有內建背景工作，不需要呼叫 `Close`。

## 7. 處理錯誤，不洩露憑證

儲存為 `errors.go`，用 `errorSummary(err)` 區分錯誤類型。伺服器業務錯誤是 `APIError`，非 2xx HTTP 狀態是 `HTTPError`。

```go
package main

import (
    "context"
    "errors"
    "fmt"

    btcc "github.com/btcc-nova/OPENAPI_SDK"
)

func errorSummary(err error) string {
    var apiErr *btcc.APIError
    var httpErr *btcc.HTTPError
    switch {
    case err == nil:
        return "ok"
    case errors.As(err, &apiErr):
        return fmt.Sprintf("API code=%d", apiErr.Code)
    case errors.As(err, &httpErr):
        return fmt.Sprintf("HTTP status=%d", httpErr.StatusCode)
    case errors.Is(err, context.DeadlineExceeded):
        return "timeout; check server state before another trade"
    case errors.Is(err, context.Canceled):
        return "canceled; an in-flight trade may still have completed"
    default:
        return "validation, network or response error"
    }
}
```

所有非零業務錯誤碼都會回傳，包括平倉的 `4` 和行情的 `-4`；SDK 不會把它們偽裝為成功或零價。預設錯誤文字不含完整網址、回應本文或伺服器原始訊息。`APIError.Message` 保留原始訊息，可能包含敏感內容，不要直接輸出到公開日誌。

SDK 不記錄日誌，Session 與登入請求的憑證欄位也不會參與 JSON 序列化，但這不代表可以安全地用 `fmt` 或除錯工具輸出整個物件。自訂 Transport、監控及代理日誌同樣需要隱藏憑證。

## 8. API 與請求規則

| 方法 | HTTP | 路徑 |
|---|---|---|
| `Login` | POST | `/v1/user/login` |
| `KeepAlive` | GET | `/v1/user/keepalive` |
| `GetSymbolList` | GET | `/v1/config/symbollist` |
| `GetAccountGroup` | GET | `/v1/config/accountgroup` |
| `GetAccount` | GET | `/v1/account/account` |
| `GetPositionList` | GET | `/v1/account/positionlist` |
| `OpenPosition` | POST | `/v1/account/openposition` |
| `ClosePosition` | POST | `/v1/account/closeposition` |
| `UpdatePosition` | POST | `/v1/account/updateposition` |
| `GetQuotePrice` | GET | `/quot/reqOnePrdPrice` |

伺服器未回傳時間時，`KeepAliveResult.ServerTime` 為 `0`，不代表目前時間。

- 除登入與查報價外，所有方法都需要由同一個 `Client` 登入取得的 `Session`。
- 預設每次請求逾時為 10 秒，可用 `Options.Timeout` 調整；context 或自訂 `HTTPClient` 的較短期限仍會生效。`NewClient` 不發送請求。
- SDK 不跟隨重新導向，單次回應上限為 8 MiB。
