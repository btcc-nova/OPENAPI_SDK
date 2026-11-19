# BTCC Open API Go SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

这个 Go SDK 帮你连接 BTCC Open API：登录、查询账户和仓位、读取报价，以及开仓、平仓和修改止盈止损。SDK 负责签名和请求格式，你只需传入参数并处理结果。

先运行只读示例，再接入交易。本文不会要求你用下单来测试连接。

## 1. 开始前准备什么

- Go 1.22 或以上。
- 向服务方取得交易地址、用户名、密码、API Key、Secret Key 和公司 ID；需要报价时另取行情地址及行情产品名称。
- 请使用服务方提供的 HTTPS 地址，避免以明文发送登录凭证。

模块路径为 `github.com/btcc-nova/OPENAPI_SDK/go`。包名为 `btcc`，小数依赖为 `github.com/shopspring/decimal v1.4.0`。

## 2. 在自己的项目中安装

GitHub 仓库发布后，在已有 `go.mod` 的项目中执行：

```sh
go get github.com/btcc-nova/OPENAPI_SDK/go@latest
```

不从 GitHub 安装时，可在仓库的 `go/` 目录执行以下命令，创建使用本地源码的示例项目：

```sh
SDK_DIR="$(pwd)"
mkdir btcc-demo
cd btcc-demo
go mod init local/btcc-demo
go mod edit -require=github.com/btcc-nova/OPENAPI_SDK/go@v0.0.0
go mod edit -replace="github.com/btcc-nova/OPENAPI_SDK/go=$SDK_DIR"
```

将下一节的程序保存为示例项目中的 `main.go`。

## 3. 第一个程序：登录并查询余额

先在运行环境或密钥管理工具中设置以下变量。不要把凭证写进源码或提交到 Git，也不要把真实密码直接写进可能保留历史的终端命令。

| 变量 | 用途 |
|---|---|
| `BTCC_BASE_URL` | 必填，交易服务基础地址；SDK 自动追加 API 路径 |
| `BTCC_USER_NAME` | 必填，登录用户名 |
| `BTCC_PASSWORD` | 必填，服务方要求的密码值；SDK 原样发送，不额外哈希 |
| `BTCC_API_KEY` | 必填，API Key |
| `BTCC_SECRET_KEY` | 必填，用于签名的 Secret Key |
| `BTCC_COMPANY_ID` | 可选，默认 `1`；SDK 收到 `0` 也会使用 `1` |
| `BTCC_QUOTE_URL` | 仅查报价时需要，独立的行情基础地址 |

只有示例程序读取这些变量。SDK 不读取环境变量，也不会自动加载 `.env` 文件。地址不能包含用户名、密码、查询参数或片段；即使只查行情，创建客户端仍需有效的 `BaseURL`。

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

在示例项目中执行：

```sh
go mod tidy
go run .
```

成功时会显示账户 ID、余额和可用保证金；不会下单。余额不是固定值。`Session` 是登录后返回的会话凭证，只能用于创建它的 `Client`。不要自行构造，也不要跨客户端使用。

## 4. 查询产品、仓位和报价

保存为示例项目中的 `queries.go`。要查询产品、账户组和全部仓位，可把 `main.go` 中 `run()` 最后的 `return nil` 改为 `return inspect(ctx, client, login.Session)`。它仍然只查询，不交易。

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

- 产品名称使用 `GetSymbolList` 返回的 `Symbol.Name`。下单前检查数量最小值、步长、最大值及服务方的交易规则。
- `Symbol.Enable` 是数值状态，可能为 `3`；不要把所有非零值都当作可交易。它兼容布尔输入，但不支持 `bool(s.Enable)`。`AccountGroup.Enable` 则可用 `bool(group.Enable)` 转换。
- 仓位查询默认 `PageNo=1`、`PageSize=100`，SDK 不自动翻页。示例逐页读取；持仓在读取期间可能变化，这不是账户快照。可用 `PositionListFilter.Symbol` 或 `ID` 筛选，负数分页参数会被拒绝。
- 只查报价时，将 `run()` 最后的返回改为 `return printQuote(ctx, client, os.Getenv("BTCC_QUOTE_SYMBOL"))`，并设置 `BTCC_QUOTE_URL` 和 `BTCC_QUOTE_SYMBOL`。行情名称由服务方确认，不要直接假定它与交易产品名称相同。
- `GetQuotePrice` 本身不需要登录，也不发送 API Key、token 或签名；这里沿用登录示例只是为了复用客户端。SDK 会正确编码产品名称中的特殊字符。

## 5. 交易示例：开仓、平仓、修改止盈止损

警告：以下函数被调用时会发送真实交易请求。仅保存或编译文件不会下单。先在获准的测试环境验证，确认产品、方向、价格、数量和杠杆后再使用；不要把它们接到前面的连通性检查。

保存为 `trades.go`。调用者传入已经检查过的参数，示例不硬编码一个可能误下单的产品或价格。价格和数量用字符串传入，例如 `"0.001"`，由 `decimal.NewFromString` 转成精确小数，不要先经过 `float64`。

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

- `openBuy` 开买仓；卖仓使用 `btcc.DirectionSell`。开仓价格、数量及 `Multiple` 必须大于零，杠杆范围仍需符合服务端规则。成功结果含 `Position` 和 `SLTPOrders`。
- `closePosition` 使用已查到的仓位 ID，方向为 `DirectionBuy`（1）或 `DirectionSell`（2）。SDK 不推断或反转平仓方向，必须先向服务方确认规则。价格和数量必须大于零。
- `setStops` 每次都发送止损与止盈。传入 `"0"` 不是省略字段，零值的业务含义需要服务方确认；不要只想改一个字段，却把另一个留为零。
- 平仓参数是 `PositionID`，修改参数是 `ID`；线上字段分别为 `positionid` 和 `id`。两个操作均返回 `PositionResult.Position`。
- 开仓还可设置 `RefID`、`RefAccountID`、`StopLoss`、`TakeProfit`；这些字段零值时不发送。平仓可设置 `RefID`，零值时不发送。

SDK 不重试、不自动重登，也不提供防重复下单保证。`RefID` 不代表服务端保证请求只执行一次。超时、断线或取消请求后，交易仍可能已经成交；先核对服务端状态，再决定是否重试。

## 6. 长时间运行：维持会话

登录不会自动启动心跳。短查询通常执行后退出；长时间运行的程序需自己安排 `KeepAlive`，或使用以下阻塞函数。保存为 `heartbeat.go`：

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

`RunKeepAlive` 会一直运行到取消或第一次心跳失败。若同时执行其他业务，由调用者用 goroutine 启动它，通过 channel 接收返回的错误；不要忽略错误。结束业务时取消 context，并等待心跳退出。

间隔传 `0` 时为 20 秒，首次心跳在一个间隔后发送；负数间隔会被拒绝。失败时不会自动重登，也不会改变 Session 的本地状态，不能据此认定会话仍有效。`Client` 和 `Session` 可并发共享，但不要并发修改请求数据或自定义 HTTP Transport。SDK 没有内建后台任务，不需要调用 `Close`。

## 7. 处理错误，不泄露凭证

保存为 `errors.go`，用 `errorSummary(err)` 区分错误类型。服务端业务错误是 `APIError`，非 2xx HTTP 状态是 `HTTPError`。

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

所有非零业务错误码都会返回，包括平仓的 `4` 和行情的 `-4`；SDK 不会把它们伪装为成功或零价。默认错误文本不含完整 URL、响应正文或服务端原始消息。`APIError.Message` 保留原始消息，可能包含敏感内容，不要直接输出到公开日志。

SDK 不记录日志，Session 与登录请求的凭证字段也不会参与 JSON 序列化，但这不代表可以安全地用 `fmt` 或调试器输出整个对象。自定义 Transport、监控及代理日志同样需要隐藏凭证。

## 8. API 与请求规则

| 方法 | HTTP | 路径 |
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

服务器未返回时间时，`KeepAliveResult.ServerTime` 为 `0`，不代表当前时间。

- 除登录和查询报价外，所有方法都需要由同一个 `Client` 登录获取的 `Session`。
- 默认每次请求超时 10 秒，可用 `Options.Timeout` 调整；context 或自定义 `HTTPClient` 的更短期限仍会生效。`NewClient` 不发请求。
- SDK 不跟随重定向，单次响应上限为 8 MiB。