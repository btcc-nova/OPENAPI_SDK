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

## 照会と価格取得

`login()` はクライアントに紐付いた `Session` を返します。これを `keep_alive`、`get_symbol_list`、`get_account_group`、`get_account`、`get_position_list` に渡します。価格照会にログインは不要です。`Client(base_url, quote_url=...)` を設定して `get_quote_price(symbol)` を呼び出します。

## 取引とエラー処理

取引には `open_position`、`close_position`、`update_position` を使用します。価格や数量は `float` ではなく文字列か `Decimal` で指定します。応答はサーバーのフィールド名を保持する辞書またはリストで、金額は `Decimal` です。業務エラーは `APIError.code`、HTTP ステータスは `HTTPError.status_code` で確認できます。自動再試行や自動ハートビートはありません。取引がタイムアウトした場合は、再注文の前にポジションを確認してください。

数量の表現には、例えば `Decimal("0.01")` を使います。注文前に `get_symbol_list(session)` で銘柄の最小数量と刻み幅を確認してください。
