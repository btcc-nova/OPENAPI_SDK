# BTCC Open API Python SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

## 설치

Python 3.9 이상이 필요하며 실행 시 외부 패키지는 필요하지 않습니다. 이 디렉터리(`python/`)에서 실행하세요.

```sh
python3 -m pip install .
```

## 로그인 및 잔액 조회

서비스 제공자가 지정한 HTTPS 거래 주소와 사용자 이름, 비밀번호, API Key, Secret Key를 환경 변수로 예제에 전달하세요. `BTCC_COMPANY_ID`는 선택 사항이며 기본값은 `1`입니다. 소스 코드에 인증 정보를 넣지 마세요.

```python
import os
from btcc_sdk import Client


def main():
    client = Client(os.environ["BTCC_BASE_URL"])
    session = client.login(user_name=os.environ["BTCC_USER_NAME"], password=os.environ["BTCC_PASSWORD"],
                           api_key=os.environ["BTCC_API_KEY"], secret_key=os.environ["BTCC_SECRET_KEY"],
                           company_id=int(os.getenv("BTCC_COMPANY_ID", "1")))
    account = client.get_account(session)
    print(f"account={account['id']} balance={account['balance']}")

main()
```

이 디렉터리에서 [조회 전용 예제](examples/login.py)를 실행하세요. 주문하지 않고 계좌 ID와 잔액만 출력합니다.

```sh
python3 examples/login.py
```

## 종목, 계좌 그룹 및 포지션 조회

다음 함수를 `main()` 앞에 정의하고 로그인 후 `main()` 안에서 `inspect_account(client, session)`을 호출하세요. 세션을 만든 클라이언트를 사용하세요. 종목의 `enable`이 0이 아니어도 거래 가능하다는 뜻은 아닙니다. 기본값은 1페이지부터 페이지당 100건입니다. `symbol` 또는 `position_id`로 필터링할 수 있으며 페이지를 읽는 동안 포지션이 바뀔 수 있습니다.

```python
def inspect_account(client, session):
    group = client.get_account_group(session)
    print(group["id"])
    for product in client.get_symbol_list(session):
        print(product["name"], product["enable"], product["volumes_min"], product["volumes_step"])

    page_no, seen = 1, 0
    while True:
        page = client.get_position_list(session, page_no=page_no, page_size=100)
        for position in page["positions"]:
            print(position["id"], position["symbol"], position["volume"])
        seen += len(page["positions"])
        if not page["positions"] or seen >= page["total"]:
            break
        page_no += 1
```

## 로그인 없이 시세 조회

이 코드는 단독으로 실행할 수 있습니다. 제공자가 확인한 거래 URL, 시세 URL 및 시세 종목명을 사용하세요. 시세 종목명은 주문에 쓰는 종목명과 다를 수 있습니다. 시세 요청에는 인증 정보가 전송되지 않습니다.

```python
import os
from btcc_sdk import Client

client = Client(os.environ["BTCC_BASE_URL"], quote_url=os.environ["BTCC_QUOTE_URL"])
quote = client.get_quote_price(os.environ["BTCC_QUOTE_SYMBOL"])
print(quote["Price"])
```

## 포지션 개설, 손절·익절 변경 및 청산

**아래 함수를 호출하면 거래 요청이 전송됩니다.** 로그인 예제에 연결하지 말고 먼저 승인된 테스트 환경에서 종목, 최소 주문 수량, 수량 단위 및 레버리지를 확인하세요. 가격과 수량은 `"0.01"` 같은 십진수 문자열로 전달하고 `float`를 거치지 마세요.

```python
from decimal import Decimal


def open_buy(client, session, *, symbol, volume_text, price_text, leverage):
    result = client.open_position(
        session, symbol=symbol, direction=1, request_volume=Decimal(volume_text),
        request_price=Decimal(price_text), multiple=leverage,
    )
    return result["position"]


def set_stops(client, session, *, position_id, symbol, stop_text, target_text):
    return client.update_position(
        session, position_id=position_id, symbol=symbol,
        stop_loss=Decimal(stop_text), take_profit=Decimal(target_text),
    )


def close_position(client, session, *, position_id, symbol, direction, volume_text, price_text):
    return client.close_position(
        session, position_id=position_id, symbol=symbol, direction=direction,
        request_volume=Decimal(volume_text), request_price=Decimal(price_text),
    )
```

개설 결과에는 `position` 딕셔너리가 들어 있고 변경과 청산은 포지션 딕셔너리를 직접 반환합니다. 개설 결과의 `id`를 사용해 `get_position_list(session, position_id=...)`로 종목과 청산 가능 수량을 확인하세요. 방향은 `1`이 매수, `2`가 매도입니다. 청산 방향은 제공자의 규칙에 따라 전달하며 SDK는 이를 반전하지 않습니다. 변경 시 두 가격을 모두 전송하므로 `0`은 ‘기존 값 유지’가 아닙니다.

거래에는 `open_position`, `close_position`, `update_position`을 사용합니다. 가격과 수량은 `float` 대신 문자열이나 `Decimal`로 전달하세요. 응답은 서버 필드 이름을 유지한 딕셔너리 또는 리스트이며 금액은 `Decimal`입니다. 업무 오류는 `APIError.code`, HTTP 상태는 `HTTPError.status_code`로 확인합니다. 재시도나 하트비트는 자동 실행되지 않습니다. 거래 시간이 초과되면 다시 주문하기 전에 포지션을 확인하세요.

## 세션 유지

장시간 사용할 때는 앱에서 주기적으로 하트비트를 보내세요. `threading.Event`를 전달하고 `stop.set()`으로 대기를 끝낼 수 있습니다. 아래 함수는 호출을 차단하며 20초 후 첫 요청을 보냅니다. 실패 시 예외를 전달하고 다시 로그인하지 않습니다.

```python
def maintain_session(client, session, stop):
    while not stop.wait(20):
        client.keep_alive(session)
```

## 오류와 거래 시간 초과

이 예제는 잔액만 조회합니다. `APIError.message`는 민감한 정보를 포함할 수 있는 서버 원문이므로 그대로 로그에 남기지 마세요. 시간 초과, 연결 중단 또는 거래 응답 해석 실패 시 포지션과 체결 기록을 확인하세요. 결과가 불분명하면 재전송하지 마세요. `ref_id`는 중복 주문 방지를 보장하지 않습니다.

```python
from btcc_sdk import APIError, HTTPError, ProtocolError, TransportError


def report_balance(client, session):
    try:
        balance = client.get_account(session)["balance"]
    except APIError as exc:
        print("API:", exc.code)
    except HTTPError as exc:
        print("HTTP:", exc.status_code)
    except TransportError:
        print("Connection failed or timed out")
    except ProtocolError:
        print("Unexpected response format")
    except ValueError:
        print("Invalid input or session")
    else:
        print(balance)
```

## API 목록

| 메서드 | HTTP | 경로 |
|---|---|---|
| `login` | POST | `/v1/user/login` |
| `keep_alive` | GET | `/v1/user/keepalive` |
| `get_symbol_list` | GET | `/v1/config/symbollist` |
| `get_account_group` | GET | `/v1/config/accountgroup` |
| `get_account` | GET | `/v1/account/account` |
| `get_position_list` | GET | `/v1/account/positionlist` |
| `open_position` | POST | `/v1/account/openposition` |
| `close_position` | POST | `/v1/account/closeposition` |
| `update_position` | POST | `/v1/account/updateposition` |
| `get_quote_price` | GET | `/quot/reqOnePrdPrice` |
