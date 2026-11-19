# BTCC Open API SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

選擇適合專案的 SDK。兩個登入範例都只查詢帳戶，不會下單。

## Go SDK

Go 1.22 或以上。 [使用指南](go/README-tw.md)

設定交易服務提供的登入憑證後，從倉庫根目錄執行：

```sh
cd go
go run ./examples/login
```

## Python SDK

需要 Python 3.9 以上。從倉庫根目錄安裝；執行時不需第三方套件。

```sh
python3 -m pip install ./python
```

設定 `BTCC_BASE_URL`、`BTCC_USER_NAME`、`BTCC_PASSWORD`、`BTCC_API_KEY` 與 `BTCC_SECRET_KEY` 後，可執行唯讀範例：

```sh
python3 python/examples/login.py
```

`login()` 回傳綁定目前客戶端的 `Session`。將它傳給 `keep_alive`、`get_symbol_list`、`get_account_group`、`get_account` 或 `get_position_list`。報價不需要登入：設定 `Client(base_url, quote_url=...)` 後呼叫 `get_quote_price(symbol)`。

交易方法是 `open_position`、`close_position`、`update_position`。價格與數量使用字串或 `Decimal`，不要傳 `float`。回應是保留服務端欄位名稱的字典或串列；金額以 `Decimal` 回傳。`APIError.code` 是業務錯誤碼，`HTTPError.status_code` 是 HTTP 狀態碼。SDK 不會自動重試或啟動心跳；交易逾時請先查倉位，不要直接重送。
