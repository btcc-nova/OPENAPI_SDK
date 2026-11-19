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

## 조회 및 시세

`login()`은 해당 클라이언트에 연결된 `Session`을 반환합니다. 이를 `keep_alive`, `get_symbol_list`, `get_account_group`, `get_account`, `get_position_list`에 전달합니다. 시세 조회에는 로그인이 필요 없습니다. `Client(base_url, quote_url=...)`을 설정하고 `get_quote_price(symbol)`을 호출하세요.

## 거래 및 오류 처리

거래에는 `open_position`, `close_position`, `update_position`을 사용합니다. 가격과 수량은 `float` 대신 문자열이나 `Decimal`로 전달하세요. 응답은 서버 필드 이름을 유지한 딕셔너리 또는 리스트이며 금액은 `Decimal`입니다. 업무 오류는 `APIError.code`, HTTP 상태는 `HTTPError.status_code`로 확인합니다. 재시도나 하트비트는 자동 실행되지 않습니다. 거래 시간이 초과되면 다시 주문하기 전에 포지션을 확인하세요.

예를 들어 수량은 `Decimal("0.01")`로 표현합니다. 주문 전 `get_symbol_list(session)`으로 종목의 최소 수량과 주문 단위를 확인하세요.
