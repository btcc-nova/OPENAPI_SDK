# BTCC Open API SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

プロジェクトで使用する SDK を選んでください。どちらのログインサンプルも口座情報を照会するだけで、注文は行いません。

## Go SDK

Go 1.22 以降が必要です。 [利用ガイド](go/README-ja.md)

取引サービスの認証情報を設定してから、リポジトリのルートで実行してください。

```sh
cd go
go run ./examples/login
```

## Python SDK

Python 3.9 以上が必要です。リポジトリのルートからインストールしてください。実行時に外部パッケージは不要です。

```sh
python3 -m pip install ./python
```

`BTCC_BASE_URL`、`BTCC_USER_NAME`、`BTCC_PASSWORD`、`BTCC_API_KEY`、`BTCC_SECRET_KEY` を設定すると、照会専用サンプルを実行できます。

```sh
python3 python/examples/login.py
```

`login()` はクライアントに紐付いた `Session` を返します。これを `keep_alive`、`get_symbol_list`、`get_account_group`、`get_account`、`get_position_list` に渡します。価格照会にログインは不要です。`Client(base_url, quote_url=...)` を設定して `get_quote_price(symbol)` を呼び出します。

取引には `open_position`、`close_position`、`update_position` を使用します。価格や数量は `float` ではなく文字列か `Decimal` で指定します。応答はサーバーのフィールド名を保持する辞書またはリストで、金額は `Decimal` です。業務エラーは `APIError.code`、HTTP ステータスは `HTTPError.status_code` で確認できます。自動再試行や自動ハートビートはありません。取引がタイムアウトした場合は、再注文の前にポジションを確認してください。
