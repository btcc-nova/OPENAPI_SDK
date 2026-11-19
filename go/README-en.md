# BTCC Open API Go SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

This Go SDK connects your application to the BTCC Open API. Use it to sign in, read accounts, positions and quotes, open or close positions, and update stop-loss and take-profit prices. The SDK handles signing and request formatting; you supply the inputs and handle the results.

Start with the read-only example before adding trading. You do not need to place an order to check connectivity.

## 1. Before you start

- Go 1.22 or later.
- Obtain the trading base URL, username, password, API Key, Secret Key and company ID from the service provider. For quotes, also obtain the quote base URL and quote symbol name.
- Use the HTTPS URLs supplied by your provider to avoid sending credentials in plain text.

The module path is `github.com/btcc-nova/OPENAPI_SDK/go`. The package name is `btcc`; decimal values use `github.com/shopspring/decimal v1.4.0`.

## 2. Install in your project

Once the GitHub repository is published, run this in a project with a `go.mod` file:

```sh
go get github.com/btcc-nova/OPENAPI_SDK/go@latest
```

To use a local copy instead of installing from GitHub, run these commands from the repository’s `go/` directory to create a demo project:

```sh
SDK_DIR="$(pwd)"
mkdir btcc-demo
cd btcc-demo
go mod init local/btcc-demo
go mod edit -require=github.com/btcc-nova/OPENAPI_SDK/go@v0.0.0
go mod edit -replace="github.com/btcc-nova/OPENAPI_SDK/go=$SDK_DIR"
```

Save the program in the next section as `main.go` in your demo project.

## 3. First program: sign in and read your balance

Set these variables in your runtime environment or secret manager first. Do not embed credentials in source or commit them to Git. Avoid entering actual passwords in terminal commands that may be saved in shell history.

| Variable | Purpose |
|---|---|
| `BTCC_BASE_URL` | Required trading base URL; the SDK appends API paths |
| `BTCC_USER_NAME` | Required login username |
| `BTCC_PASSWORD` | Required password value in the provider's expected format; sent unchanged, with no additional hashing |
| `BTCC_API_KEY` | Required API Key |
| `BTCC_SECRET_KEY` | Required Secret Key for signing |
| `BTCC_COMPANY_ID` | Optional, defaults to `1`; the SDK also treats `0` as `1` |
| `BTCC_QUOTE_URL` | Separate quote base URL, needed only for quote requests |

Only the example reads these variables. The SDK does not read environment variables or load `.env` files. URLs must not contain credentials, query parameters or fragments. Creating a client requires a valid `BaseURL` even for quote-only use.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "strconv"
    "time"

    btcc "github.com/btcc-nova/OPENAPI_SDK/go"
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

Run in the example project:

```sh
go mod tidy
go run .
```

On success, it prints the account ID, balance and free margin without placing an order. The balance is not a fixed expected value. `Session` holds the credentials returned by login and only works with the `Client` that created it. Do not construct one yourself or reuse it with another client.

## 4. Read products, positions and quotes

Save this as `queries.go` in the example project. To read products, the account group and all position pages, replace the final `return nil` in `main.go`'s `run()` with `return inspect(ctx, client, login.Session)`. This still only reads data; it does not trade.

```go
package main

import (
    "context"
    "fmt"

    btcc "github.com/btcc-nova/OPENAPI_SDK/go"
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

- Use `Symbol.Name` returned by `GetSymbolList` as the trading symbol. Check minimum volume, volume step, maximum volume and the provider's trading rules before placing orders.
- `Symbol.Enable` is a numeric status and may be `3`. Nonzero does not necessarily mean trading is allowed. It accepts boolean input but does not support `bool(s.Enable)`. `AccountGroup.Enable` does support `bool(group.Enable)`.
- Position queries default to `PageNo=1`, `PageSize=100`. The SDK does not fetch subsequent pages automatically. The example does; positions may change while you read them, so this is not an account snapshot. Filter with `PositionListFilter.Symbol` or `ID`. Negative page values are rejected.
- For a quote, change the final return in `run()` to `return printQuote(ctx, client, os.Getenv("BTCC_QUOTE_SYMBOL"))` and set `BTCC_QUOTE_URL` and `BTCC_QUOTE_SYMBOL`. Ask the provider for the quote symbol; do not assume it matches the trading symbol.
- `GetQuotePrice` itself does not require login and sends no API Key, token or signature. This example retains login only to reuse the existing client setup. Special characters in symbol names are correctly encoded.

## 5. Trading examples: open, close and update stops

Warning: calling these functions sends real trading requests. Saving or compiling the file alone does not place orders. Validate in an authorized test environment first, then confirm the symbol, direction, price, volume and leverage. Do not add these functions to the connectivity check above.

Save as `trades.go`. The caller supplies checked inputs; the example does not hard-code a symbol or price that could cause an accidental trade. Pass prices and volumes as strings, such as `"0.001"`. `decimal.NewFromString` preserves decimal precision; do not convert through `float64` first.

```go
package main

import (
    "context"

    btcc "github.com/btcc-nova/OPENAPI_SDK/go"
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

- `openBuy` opens a buy position; use `btcc.DirectionSell` for a sell. Opening price, volume and `Multiple` must be positive. Leverage must also meet server rules. Success returns `Position` and `SLTPOrders`.
- `closePosition` takes an existing position ID and either `DirectionBuy` (1) or `DirectionSell` (2). The SDK neither infers nor reverses the closing direction; confirm the provider's rule first. Price and volume must be positive.
- `setStops` always sends both stop-loss and take-profit. Passing `"0"` does not omit a field; confirm what zero means with the provider. Do not leave one field at zero unintentionally when changing the other.
- Closing uses `PositionID`; updating uses `ID`. Their wire fields are `positionid` and `id`, respectively. Both operations return `PositionResult.Position`.
- Opening also accepts `RefID`, `RefAccountID`, `StopLoss` and `TakeProfit`, omitted when zero. Closing accepts `RefID`, also omitted when zero.

The SDK does not retry, sign in again automatically or guarantee duplicate-order prevention. `RefID` is not a guarantee that the server executes a request only once. A timeout, disconnection or cancellation may happen after a trade completes. Check server state before deciding whether to retry.

## 6. Keep a long-running session alive

Login does not start a heartbeat automatically. Short queries usually finish and exit. For long-running applications, schedule `KeepAlive` yourself or use this blocking function. Save as `heartbeat.go`:

```go
package main

import (
    "context"
    "time"

    btcc "github.com/btcc-nova/OPENAPI_SDK/go"
)

func keepSessionAlive(ctx context.Context, client *btcc.Client, session *btcc.Session) error {
    return client.RunKeepAlive(ctx, session, 20*time.Second)
}
```

`RunKeepAlive` runs until cancellation or the first heartbeat failure. To run other work at the same time, start it in a goroutine and receive its returned error through a channel. Do not ignore that error. When work ends, cancel the context and wait for the heartbeat to exit.

An interval of `0` means 20 seconds; the first heartbeat is sent after one interval. Negative intervals are rejected. Failure does not trigger login or change local Session state, so an unchanged Session does not prove it is valid. `Client` and `Session` can be shared concurrently; do not concurrently modify request data or a custom HTTP Transport. The SDK starts no background tasks, so there is no `Close` call.

## 7. Handle errors without exposing credentials

Save as `errors.go` and use `errorSummary(err)` to distinguish error types. Server business errors are `APIError`; non-2xx HTTP responses are `HTTPError`.

```go
package main

import (
    "context"
    "errors"
    "fmt"

    btcc "github.com/btcc-nova/OPENAPI_SDK/go"
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

Every nonzero business code is returned, including closing code `4` and quote code `-4`. They are never disguised as success or a zero price. Default error text excludes full URLs, response bodies and raw server messages. `APIError.Message` retains the raw message and may contain sensitive data; do not send it directly to public logs.

The SDK does not log. Credential fields in Session and login requests are excluded from JSON serialization, but this does not make printing the whole object with `fmt` or a debugger safe. Custom Transport, monitoring and proxy logs must also hide credentials.

## 8. API and request rules

| Method | HTTP | Path |
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

If the server omits the time, `KeepAliveResult.ServerTime` is `0`, not the current time.

- Every method except login and quote lookup requires a `Session` from the same `Client`.
- Each request defaults to a 10-second timeout, configurable with `Options.Timeout`. Shorter context or custom `HTTPClient` deadlines still apply. `NewClient` sends no request.
- The SDK does not follow redirects and limits each response to 8 MiB.