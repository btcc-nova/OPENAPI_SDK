# BTCC Open API Python SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

## インストール

Python 3.9 以降が必要です。実行時に外部パッケージは不要です。このディレクトリ（`python/`）で実行してください。

```sh
python3 -m pip install .
```

## ログインと残高照会

サービス提供者が指定した HTTPS 取引 URL、ユーザー名、パスワード、API Key、Secret Key を環境変数でサンプルに渡します。`BTCC_COMPANY_ID` は省略可能で、既定値は `1` です。認証情報をソースコードに記載しないでください。

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

このディレクトリで[照会専用サンプル](examples/login.py)を実行します。口座 ID と残高を表示するだけで、注文は行いません。

```sh
python3 examples/login.py
```

## 銘柄・口座グループ・ポジションの照会

次の関数を `main()` の前に定義し、ログイン後に `main()` 内で `inspect_account(client, session)` を呼び出します。セッションを作成したクライアントを使用してください。銘柄の `enable` がゼロ以外でも取引可能とは限りません。既定は 1 ページ目から、1 ページ 100 件です。`symbol` または `position_id` で絞り込めます。ページ間でポジションが変わる場合があります。

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

## ログインせずに価格を照会する

このコードは単独で実行できます。提供者が指定した取引 URL、価格配信 URL、価格照会用の銘柄名を使います。注文用の銘柄名と異なる場合があります。照会に認証情報は送信されません。

```python
import os
from btcc_sdk import Client

client = Client(os.environ["BTCC_BASE_URL"], quote_url=os.environ["BTCC_QUOTE_URL"])
quote = client.get_quote_price(os.environ["BTCC_QUOTE_SYMBOL"])
print(quote["Price"])
```

## 新規注文・損切りと利確の変更・決済

**以下の関数を呼び出すと取引リクエストが送信されます。** ログイン例には組み込まず、許可されたテスト環境で銘柄、最小数量、刻み幅、レバレッジを確認してください。価格と数量は `"0.01"` のような十進数文字列で渡し、`float` を経由しないでください。

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

新規注文は `position` 辞書を含む結果を返します。変更と決済はポジション辞書を直接返します。新規注文の結果から `id` を取得し、`get_position_list(session, position_id=...)` で銘柄と決済可能な数量を確認してください。方向は `1` が買い、`2` が売りです。決済時は提供者の規則に従って指定し、SDK は方向を反転しません。変更時は両方の価格を送信するため、`0` は「変更しない」という意味ではありません。

取引には `open_position`、`close_position`、`update_position` を使用します。価格や数量は `float` ではなく文字列か `Decimal` で指定します。応答はサーバーのフィールド名を保持する辞書またはリストで、金額は `Decimal` です。業務エラーは `APIError.code`、HTTP ステータスは `HTTPError.status_code` で確認できます。自動再試行や自動ハートビートはありません。取引がタイムアウトした場合は、再注文の前にポジションを確認してください。

## セッションの維持

長時間使う場合はアプリで定期的にハートビートを送信します。`threading.Event` を渡し、`stop.set()` で待機を終了できます。この関数はブロックし、20 秒後に初回送信します。失敗すると例外が伝播し、自動再ログインは行いません。

```python
def maintain_session(client, session, stop):
    while not stop.wait(20):
        client.keep_alive(session)
```

## エラーと取引のタイムアウト

次の例は残高照会のみです。`APIError.message` は機密情報を含む可能性のあるサーバーの原文なので、そのまま記録しないでください。タイムアウト、切断、取引応答の解析失敗時はポジションと約定記録を確認してください。結果不明のまま再送しないでください。`ref_id` による重複排除の保証はありません。

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

## API 一覧

| メソッド | HTTP | パス |
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
