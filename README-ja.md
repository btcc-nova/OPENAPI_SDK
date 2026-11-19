# BTCC Open API Go SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

この Go SDK を使うと、BTCC Open API へのログイン、口座・ポジション・価格の照会、新規注文、決済、損切り・利確価格の変更ができます。署名とリクエスト形式は SDK が処理するため、開発者は入力値を渡して結果を処理します。

まず照会専用のサンプルを実行し、その後に取引処理を追加してください。接続確認のために注文する必要はありません。

## 1. 事前に用意するもの

- Go 1.22 以降。
- サービス提供者から取得した取引ベース URL、ユーザー名、パスワード、API Key、Secret Key、会社 ID。価格照会には別途、価格配信のベース URL と銘柄名が必要です。
- 認証情報を平文で送信しないよう、サービス提供者が指定した HTTPS URL を使用してください。

モジュールパスは [`github.com/btcc-nova/OPENAPI_SDK`](https://github.com/btcc-nova/OPENAPI_SDK) です。パッケージ名は `btcc`、小数処理には `github.com/shopspring/decimal v1.4.0` を使用します。

## 2. 自分のプロジェクトに導入する

GitHub リポジトリの公開後は、`go.mod` があるプロジェクトで次のコマンドを実行してください。

```sh
go get github.com/btcc-nova/OPENAPI_SDK@latest
```

GitHub からインストールせずローカルのソースを使う場合は、SDK のルートで以下を実行してサンプルプロジェクトを作成します。

```sh
SDK_DIR="$(pwd)"
mkdir btcc-demo
cd btcc-demo
go mod init local/btcc-demo
go mod edit -require=github.com/btcc-nova/OPENAPI_SDK@v0.0.0
go mod edit -replace="github.com/btcc-nova/OPENAPI_SDK=$SDK_DIR"
```

次のセクションのプログラムをサンプルプロジェクトの `main.go` に保存してください。

## 3. 最初のプログラム：ログインして残高を確認する

実行環境またはシークレット管理ツールで以下の変数を設定してください。認証情報をソースに書いたり、Git にコミットしたりしないでください。履歴に残る可能性がある端末コマンドへ実際のパスワードを直接入力することも避けてください。

| 変数 | 用途 |
|---|---|
| `BTCC_BASE_URL` | 必須。取引サービスのベース URL。API パスは SDK が追加 |
| `BTCC_USER_NAME` | 必須。ログインユーザー名 |
| `BTCC_PASSWORD` | 必須。提供者が指定する形式のパスワード値。SDK は追加のハッシュ処理をせず、そのまま送信 |
| `BTCC_API_KEY` | 必須。API Key |
| `BTCC_SECRET_KEY` | 必須。署名用の Secret Key |
| `BTCC_COMPANY_ID` | 任意。既定値は `1`。SDK は `0` も `1` として扱う |
| `BTCC_QUOTE_URL` | 価格照会でのみ必要な、独立した価格配信のベース URL |

これらの変数を読むのはサンプルプログラムだけです。SDK 自体は環境変数も `.env` ファイルも読み込みません。URL に認証情報、クエリパラメーター、フラグメントは指定できません。価格照会だけでも、クライアント作成には有効な `BaseURL` が必要です。

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

サンプルプロジェクトで実行します。

```sh
go mod tidy
go run .
```

成功すると口座 ID、残高、余剰証拠金を表示し、注文は行いません。残高は固定の期待値ではありません。`Session` はログインで返された認証情報を保持し、作成元の `Client` でのみ使えます。自分で作成したり、別のクライアントへ渡したりしないでください。

## 4. 銘柄・ポジション・価格を照会する

サンプルプロジェクトに `queries.go` として保存します。銘柄、口座グループ、全ページのポジションを取得するには、`main.go` の `run()` の最後の `return nil` を `return inspect(ctx, client, login.Session)` に変更します。これも照会のみで、取引は行いません。

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

- 取引銘柄には `GetSymbolList` が返す `Symbol.Name` を使います。注文前に最小数量、数量刻み、最大数量と提供者の取引ルールを確認してください。
- `Symbol.Enable` は数値の状態で、`3` の場合もあります。ゼロ以外がすべて取引可能とは限りません。真偽値の入力には対応しますが、`bool(s.Enable)` には対応しません。`AccountGroup.Enable` は `bool(group.Enable)` で変換できます。
- ポジション照会の既定値は `PageNo=1`、`PageSize=100` です。SDK は自動で次ページを読みません。サンプルはページごとに読みますが、途中でポジションが変わる可能性があり、口座のスナップショットではありません。`PositionListFilter.Symbol` または `ID` で絞り込めます。負のページ指定は拒否されます。
- 価格照会には `run()` の最後を `return printQuote(ctx, client, os.Getenv("BTCC_QUOTE_SYMBOL"))` に変更し、`BTCC_QUOTE_URL` と `BTCC_QUOTE_SYMBOL` を設定します。価格配信の銘柄名は提供者に確認し、取引銘柄名と同じだと決めつけないでください。
- `GetQuotePrice` 自体はログイン不要で、API Key、token、署名も送信しません。ここではクライアント設定を再利用するためログイン例をそのまま使っています。銘柄名の特殊文字は正しくエンコードされます。

## 5. 取引サンプル：新規注文・決済・損切りと利確の変更

警告：以下の関数を呼ぶと、実際の取引リクエストが送信されます。保存やコンパイルだけでは注文しません。許可されたテスト環境で検証し、銘柄、方向、価格、数量、レバレッジを確認してから使用してください。前述の接続確認には組み込まないでください。

`trades.go` として保存します。確認済みの値を呼び出し側から渡すため、誤発注につながる銘柄や価格を固定していません。価格と数量は `"0.001"` のような文字列で渡し、`decimal.NewFromString` で正確な小数に変換します。先に `float64` へ変換しないでください。

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

- `openBuy` は買いポジションを開きます。売りには `btcc.DirectionSell` を使います。新規注文の価格、数量、`Multiple` は正の値が必要です。レバレッジはサーバーの制限にも従います。成功時は `Position` と `SLTPOrders` が返ります。
- `closePosition` には既存のポジション ID と `DirectionBuy`（1）または `DirectionSell`（2）を渡します。SDK は決済方向の推測や反転をしません。提供者のルールを先に確認してください。価格と数量は正の値が必要です。
- `setStops` は損切りと利確の両方を毎回送信します。`"0"` は項目の省略ではありません。ゼロの意味を提供者に確認し、一方だけを変えるつもりで他方をゼロにしないでください。
- 決済は `PositionID`、変更は `ID` を使います。送信フィールドはそれぞれ `positionid` と `id` です。どちらも `PositionResult.Position` を返します。
- 新規注文では `RefID`、`RefAccountID`、`StopLoss`、`TakeProfit` も指定でき、ゼロなら送信されません。決済の `RefID` もゼロなら送信されません。

SDK は再試行、自動再ログイン、重複注文の防止を保証しません。`RefID` はサーバーが一度だけ処理する保証ではありません。タイムアウト、切断、キャンセル時にも取引が成立している可能性があります。再試行の判断前にサーバーの状態を確認してください。

## 6. 長時間の実行でセッションを維持する

ログインだけではハートビートは開始されません。短い照会は通常そのまま終了します。長時間動くアプリでは `KeepAlive` を自分で定期実行するか、以下の待機し続ける関数を使用します。`heartbeat.go` として保存してください。

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

`RunKeepAlive` はキャンセルまたは最初のハートビート失敗まで実行し続けます。他の処理と同時に動かす場合は goroutine で起動し、channel で戻り値のエラーを受け取ってください。エラーを無視しないでください。処理終了時には context をキャンセルし、ハートビートの終了を待ちます。

間隔 `0` は 20 秒です。最初の送信は 1 間隔後で、負の間隔は拒否されます。失敗しても再ログインやローカル Session 状態の変更は行われないため、状態が変わらないことは有効性の証明になりません。`Client` と `Session` は並行共有できますが、リクエストデータや独自の HTTP Transport を同時に変更しないでください。SDK はバックグラウンド処理を自動起動しないため、`Close` は不要です。

## 7. 認証情報を漏らさずにエラーを処理する

`errors.go` として保存し、`errorSummary(err)` でエラーの種類を判別します。サーバーの業務エラーは `APIError`、2xx 以外の HTTP 応答は `HTTPError` です。

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

ゼロ以外の業務コードはすべてエラーとして返り、決済の `4` や価格照会の `-4` も例外ではありません。成功やゼロ価格には置き換えません。既定のエラー文字列には完全な URL、応答本文、サーバーの生メッセージを含めません。`APIError.Message` は生メッセージを保持し、機密情報を含む可能性があるため、公開ログへ直接出力しないでください。

SDK 自体はログを記録しません。Session とログイン要求の認証情報は JSON シリアライズから除外されますが、`fmt` やデバッガーでオブジェクト全体を表示して安全という意味ではありません。独自 Transport、監視、プロキシのログでも認証情報を隠してください。

## 8. API とリクエストのルール

| メソッド | HTTP | パス |
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

サーバーが時刻を返さない場合、`KeepAliveResult.ServerTime` は `0` になります。現在時刻ではありません。

- ログインと価格照会を除き、同じ `Client` でログインして取得した `Session` が必要です。
- 各要求の既定タイムアウトは 10 秒で、`Options.Timeout` で変更できます。context や独自 `HTTPClient` に短い期限があれば、その期限も有効です。`NewClient` は要求を送りません。
- SDK はリダイレクトに追従しません。応答サイズの上限は 1 回につき 8 MiB です。
