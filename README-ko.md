# BTCC Open API Go SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

이 Go SDK로 BTCC Open API에 연결하여 로그인, 계좌·포지션·시세 조회, 포지션 개설과 청산, 손절·익절 가격 변경을 할 수 있습니다. SDK가 서명과 요청 형식을 처리하므로 개발자는 입력값을 전달하고 결과를 처리하면 됩니다.

먼저 읽기 전용 예제를 실행한 뒤 거래 기능을 연결하세요. 연결 확인을 위해 주문할 필요는 없습니다.

## 1. 시작 전 준비

- Go 1.22 이상.
- 서비스 제공자가 발급한 거래 기본 URL, 사용자 이름, 비밀번호, API Key, Secret Key, 회사 ID. 시세를 조회하려면 별도의 시세 기본 URL과 시세 종목명도 필요합니다.
- 제공자가 확인한 HTTPS URL을 사용하세요. 운영 주소를 추측하지 마세요. HTTP는 테스트 또는 신뢰할 수 있는 내부 네트워크용이며 인증 정보를 암호화하지 않습니다. 예전 참고 자료의 `90816`은 유효한 포트가 아닙니다.

모듈 경로는 [`github.com/btcc-nova/OPENAPI_SDK`](https://github.com/btcc-nova/OPENAPI_SDK)입니다. 패키지 이름은 `btcc`이며 소수 처리에는 `github.com/shopspring/decimal v1.4.0`을 사용합니다.

## 2. 내 프로젝트에 설치

GitHub 저장소가 공개되면 `go.mod`가 있는 프로젝트에서 다음 명령을 실행하세요.

```sh
go get github.com/btcc-nova/OPENAPI_SDK@latest
```

공개 전이거나 로컬 소스를 테스트할 때는 대신 아래 단계를 따르세요.

SDK 루트에서 다음 명령을 실행해 별도의 `btcc-demo` 프로젝트를 만듭니다. 같은 디렉터리가 있으면 새 이름을 사용하고 기존 코드를 덮어쓰지 마세요. `replace`는 SDK를 다운로드하지 않고 로컬에서 읽도록 Go에 알려줍니다.

```sh
SDK_DIR="$(pwd)"
mkdir btcc-demo
cd btcc-demo
go mod init local/btcc-demo
go mod edit -require=github.com/btcc-nova/OPENAPI_SDK@v0.0.0
go mod edit -replace="github.com/btcc-nova/OPENAPI_SDK=$SDK_DIR"
```

다음 예제를 `btcc-demo` 안의 `main.go`로 저장한 뒤 `go mod tidy`를 실행하세요. 기존 Go 프로젝트에서는 `SDK_DIR`을 SDK의 절대 경로로 설정하고 두 개의 `go mod edit` 명령만 사용할 수 있습니다.

## 3. 첫 프로그램: 로그인하고 잔액 조회

실행 환경이나 비밀 관리 도구에서 다음 변수를 먼저 설정하세요. 인증 정보를 소스에 넣거나 Git에 커밋하지 마세요. 기록에 남을 수 있는 터미널 명령에 실제 비밀번호를 직접 입력하는 것도 피하세요.

| 변수 | 용도 |
|---|---|
| `BTCC_BASE_URL` | 필수 거래 기본 URL. SDK가 API 경로를 추가 |
| `BTCC_USER_NAME` | 필수 로그인 사용자 이름 |
| `BTCC_PASSWORD` | 제공자가 요구하는 형식의 필수 비밀번호 값. SDK가 추가 해시 없이 그대로 전송 |
| `BTCC_API_KEY` | 필수 API Key |
| `BTCC_SECRET_KEY` | 서명용 필수 Secret Key |
| `BTCC_COMPANY_ID` | 선택 사항. 기본값 `1`, SDK는 `0`도 `1`로 처리 |
| `BTCC_QUOTE_URL` | 시세 요청에만 필요한 별도 시세 기본 URL |

이 변수는 예제 프로그램만 읽습니다. SDK는 환경 변수를 읽거나 `.env` 파일을 자동으로 불러오지 않습니다. URL에 인증 정보, 쿼리 매개변수, 프래그먼트를 넣을 수 없습니다. 시세만 조회하더라도 클라이언트를 만들 때 유효한 `BaseURL`이 필요합니다.

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

예제 프로젝트에서 실행하세요.

```sh
go mod tidy
go run .
```

성공하면 계좌 ID, 잔액, 가용 증거금을 출력하며 주문하지 않습니다. 잔액은 고정된 예상값이 아닙니다. `Session`은 로그인으로 받은 인증 정보를 보관하며 이를 만든 `Client`에서만 사용할 수 있습니다. 직접 만들거나 다른 클라이언트에 재사용하지 마세요.

## 4. 종목·포지션·시세 조회

예제 프로젝트에 `queries.go`로 저장하세요. 종목, 계좌 그룹, 모든 포지션 페이지를 조회하려면 `main.go`의 `run()` 마지막 `return nil`을 `return inspect(ctx, client, login.Session)`으로 바꿉니다. 여전히 조회만 하며 거래하지 않습니다.

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

- 거래 종목명은 `GetSymbolList`가 반환한 `Symbol.Name`을 사용하세요. 주문 전 최소 수량, 수량 단위, 최대 수량과 제공자의 거래 규칙을 확인하세요.
- `Symbol.Enable`은 숫자 상태이며 `3`일 수도 있습니다. 0이 아닌 값이 모두 거래 가능을 뜻하지는 않습니다. 불리언 입력은 지원하지만 `bool(s.Enable)` 변환은 지원하지 않습니다. `AccountGroup.Enable`은 `bool(group.Enable)`로 변환할 수 있습니다.
- 포지션 조회 기본값은 `PageNo=1`, `PageSize=100`입니다. SDK는 다음 페이지를 자동으로 조회하지 않습니다. 예제는 페이지별로 읽지만 그동안 포지션이 바뀔 수 있어 계좌 스냅샷은 아닙니다. `PositionListFilter.Symbol` 또는 `ID`로 필터링할 수 있으며 음수 페이지 값은 거부됩니다.
- 시세 조회는 `run()`의 마지막 반환문을 `return printQuote(ctx, client, os.Getenv("BTCC_QUOTE_SYMBOL"))`으로 바꾸고 `BTCC_QUOTE_URL`, `BTCC_QUOTE_SYMBOL`을 설정하세요. 시세 종목명은 제공자에게 확인하고 거래 종목명과 같다고 가정하지 마세요.
- `GetQuotePrice` 자체는 로그인이 필요 없으며 API Key, token, 서명을 보내지 않습니다. 여기서는 클라이언트 설정을 재사용하기 위해 로그인 예제를 유지했습니다. 종목명의 특수 문자는 올바르게 인코딩됩니다.

## 5. 거래 예제: 개설·청산·손절과 익절 변경

경고: 아래 함수를 호출하면 실제 거래 요청이 전송됩니다. 파일을 저장하거나 컴파일하는 것만으로는 주문하지 않습니다. 허가된 테스트 환경에서 먼저 검증하고 종목, 방향, 가격, 수량, 레버리지를 확인한 뒤 사용하세요. 앞의 연결 확인 코드에 넣지 마세요.

`trades.go`로 저장하세요. 호출자가 확인한 값을 전달하도록 하여 실수로 주문할 수 있는 종목이나 가격을 고정하지 않았습니다. 가격과 수량은 `"0.001"` 같은 문자열로 전달하고 `decimal.NewFromString`으로 정확한 소수로 변환하세요. 먼저 `float64`로 변환하지 마세요.

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

- `openBuy`는 매수 포지션을 엽니다. 매도에는 `btcc.DirectionSell`을 사용하세요. 개설 가격, 수량, `Multiple`은 양수여야 하며 레버리지도 서버 규칙을 따라야 합니다. 성공하면 `Position`, `SLTPOrders`를 반환합니다.
- `closePosition`은 기존 포지션 ID와 `DirectionBuy`(1) 또는 `DirectionSell`(2)을 받습니다. SDK는 청산 방향을 추론하거나 반대로 바꾸지 않습니다. 제공자의 규칙을 먼저 확인하세요. 가격과 수량은 양수여야 합니다.
- `setStops`는 손절과 익절을 항상 함께 전송합니다. `"0"`은 필드 생략이 아닙니다. 0의 의미를 제공자에게 확인하고 한 필드만 바꾸려다 다른 필드를 실수로 0으로 두지 마세요.
- 청산에는 `PositionID`, 변경에는 `ID`를 사용합니다. 전송 필드는 각각 `positionid`, `id`입니다. 두 작업 모두 `PositionResult.Position`을 반환합니다.
- 개설에는 `RefID`, `RefAccountID`, `StopLoss`, `TakeProfit`도 설정할 수 있으며 0이면 전송하지 않습니다. 청산의 `RefID`도 0이면 전송하지 않습니다.

SDK는 재시도나 자동 재로그인을 하지 않으며 중복 주문 방지를 보장하지 않습니다. `RefID`는 서버가 요청을 한 번만 실행한다는 보장이 아닙니다. 시간 초과, 연결 끊김, 취소가 발생해도 거래가 이미 체결되었을 수 있습니다. 재시도 여부를 결정하기 전에 서버 상태를 확인하세요.

## 6. 오래 실행할 때 세션 유지

로그인만으로 하트비트가 시작되지는 않습니다. 짧은 조회는 보통 완료 후 종료합니다. 오래 실행하는 프로그램은 `KeepAlive`를 직접 주기적으로 호출하거나 아래처럼 계속 실행되는 함수를 사용하세요. `heartbeat.go`로 저장합니다.

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

`RunKeepAlive`는 취소되거나 첫 하트비트 실패가 발생할 때까지 실행됩니다. 다른 작업과 함께 실행하려면 goroutine에서 시작하고 channel로 반환 오류를 받으세요. 오류를 무시하지 마세요. 작업이 끝나면 context를 취소하고 하트비트 종료를 기다리세요.

간격 `0`은 20초이며 첫 하트비트는 한 간격 후 전송됩니다. 음수 간격은 거부됩니다. 실패해도 자동 로그인이나 로컬 Session 상태 변경은 없으므로 상태가 그대로라고 세션이 유효한 것은 아닙니다. `Client`와 `Session`은 동시에 공유할 수 있지만 요청 데이터나 사용자 지정 HTTP Transport를 동시에 수정하지 마세요. SDK는 백그라운드 작업을 자동으로 시작하지 않아 `Close`를 호출할 필요가 없습니다.

## 7. 인증 정보를 노출하지 않고 오류 처리

`errors.go`로 저장하고 `errorSummary(err)`로 오류 종류를 구분하세요. 서버 업무 오류는 `APIError`, 2xx가 아닌 HTTP 응답은 `HTTPError`입니다.

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

0이 아닌 모든 업무 코드는 오류로 반환됩니다. 청산 코드 `4`, 시세 코드 `-4`도 성공이나 0 가격으로 바꾸지 않습니다. 기본 오류 문자열에는 전체 URL, 응답 본문, 서버 원본 메시지가 포함되지 않습니다. `APIError.Message`는 원본 메시지를 보관하며 민감한 정보가 있을 수 있으므로 공개 로그에 그대로 출력하지 마세요.

SDK 자체는 로그를 남기지 않습니다. Session과 로그인 요청의 인증 필드는 JSON 직렬화에서 제외되지만 `fmt`나 디버거로 객체 전체를 출력해도 안전하다는 뜻은 아닙니다. 사용자 지정 Transport, 모니터링, 프록시 로그에서도 인증 정보를 숨겨야 합니다.

## 8. API와 요청 규칙

| 메서드 | HTTP | 경로 |
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

`KeepAliveResult.ServerTime`은 서버가 `server_time`을 반환할 때만 설정됩니다. STG 하트비트 응답에는 `accountid`만 있을 수 있으며, 이 경우 값은 현재 시각이 아닌 `0`입니다.

- 로그인과 시세 조회 외에는 같은 클라이언트의 Session이 필요합니다. 요청과 반환 타입은 [client.go](client.go), [types.go](types.go)를 참고하세요. `internal/wire/`는 내부 구현이며 외부 프로젝트에서 직접 가져올 수 없습니다.
- GET은 쿼리 매개변수를, POST는 JSON이 아닌 `application/x-www-form-urlencoded`를 사용합니다. 서명은 자동입니다. `secret_key`를 포함한 매개변수를 이름순으로 정렬하고 URL 인코딩 전의 `key=value`를 `&`로 연결한 뒤 소문자 MD5를 계산합니다. `sign`만 전송하고 `secret_key`는 보내지 않습니다. MD5는 기존 프로토콜 요구 사항이며 새 설계에 대한 권장이 아닙니다.
- 요청별 기본 제한 시간은 10초이며 `Options.Timeout`으로 바꿀 수 있습니다. context 또는 사용자 지정 `HTTPClient`의 더 짧은 제한도 적용됩니다. `NewClient`는 요청을 보내지 않습니다.
- 인증 정보 유출과 거래 재전송을 막기 위해 리디렉션을 따르지 않습니다. 사용자 지정 `HTTPClient`는 얕게 복사한 뒤 복사본의 리디렉션 정책을 덮어쓰며 원본은 변경하지 않습니다.
- 응답 상한은 8 MiB입니다. `code` 누락, 잘못된 JSON, 성공한 로그인에 필수 인증 정보가 없는 경우, 성공한 시세 응답에 `data`가 없는 경우는 오류입니다.

## 9. 검증과 관리자 도구

SDK 루트에서 아래 오프라인 검사를 실행하세요. 주문하지 않습니다.

```sh
go test -race -count=1 ./...
go vet ./...
go build ./...
```

로컬 테스트는 `httptest`로 서버를 모의하여 10개 API, 서명, 소수 정밀도, 입력 검증, 오류, 시간 초과, 리디렉션, 응답 제한, 동시 실행을 검사합니다. 프로젝트 규칙에 따라 `*_test.go`는 개발 기기에만 보관하고 `.gitignore`로 제외합니다. 새로 받은 소스에는 테스트가 없을 수 있으며, “테스트 파일 없음”은 이 검사를 통과했다는 뜻이 아닙니다.

다음 명령은 로컬 `uat_test.go`와 허가된 인증 정보를 가진 관리자 전용입니다. UAT에 연결하며 외부 개발자용 설치 절차가 아닙니다.

```sh
make help
make test-login
BTCC_UAT_CHECK=1 go test -run '^TestUATInspect$' -count=1 -v -timeout=90s
```

`make test-login`은 로그인만 하고 포지션·시세 조회, 주문, 하트비트는 실행하지 않습니다. 로컬 테스트가 없으면 실패합니다. `TestUATInspect`는 로그인 후 종목, 포지션, 시세를 조회하지만 거래하지 않습니다. 일반 `go test ./...`는 온라인 검사를 건너뜁니다.

UAT 테스트는 SDK 루트의 `uat.env`를 직접 읽습니다. 카피 트레이딩 봇 설정이나 환경 변수로 덮어쓴 인증 정보는 사용하지 않습니다. 필수 키는 `BTCC_UAT_BASE_URL`, `BTCC_UAT_QUOTE_URL`, `BTCC_UAT_USERNAME`, `BTCC_UAT_PASSWORD`, `BTCC_UAT_API_KEY`, `BTCC_UAT_SECRET_KEY`, `BTCC_UAT_COMPANY_ID`입니다. `KEY=VALUE` 형식이며 큰따옴표를 지원하지만 shell 실행이나 변수 확장은 하지 않습니다. 허용된 서비스 URL도 검사합니다.

`uat.env` 권한은 `600`으로 유지하세요. `*.env`는 제외되어 있으며 커밋, 출력, 공유하면 안 됩니다. `make test-login`의 요청 제한 시간은 15초이고 실패하면 0이 아닌 종료 코드를 반환하며 비밀번호, API Key, Secret, token을 출력하지 않습니다. Makefile은 PATH의 Go를 우선 사용하고 다음으로 `/opt/homebrew/bin/go`를 시도합니다. `make test-login GO="$(command -v go)"`로 명시할 수 있습니다.

운영 적용 전 실제 URL, 인증 정보 형식, 종목 상태, 수량·레버리지 제한, 청산 방향, 손절·익절 0 값의 의미를 제공자에게 확인하세요. 컴파일과 오프라인 테스트는 대상 환경 검증을 대신할 수 없습니다.
