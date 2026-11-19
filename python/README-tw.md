# BTCC Open API Python SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

## 安裝

需要 Python 3.9 以上，執行時不需第三方套件。在本目錄（`python/`）執行：

```sh
python3 -m pip install .
```

## 登入並查詢餘額

透過環境變數向範例提供服務方確認的 HTTPS 交易網址、使用者名稱、密碼、API Key 與 Secret Key。`BTCC_COMPANY_ID` 可省略，預設為 `1`。不要把憑證寫入程式碼。

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

在本目錄執行[唯讀範例](examples/login.py)，它只會顯示帳戶 ID 和餘額，不會下單：

```sh
python3 examples/login.py
```

## 查詢與報價

`login()` 回傳綁定目前客戶端的 `Session`。將它傳給 `keep_alive`、`get_symbol_list`、`get_account_group`、`get_account` 或 `get_position_list`。報價不需要登入：設定 `Client(base_url, quote_url=...)` 後呼叫 `get_quote_price(symbol)`。

## 交易與錯誤處理

交易方法是 `open_position`、`close_position`、`update_position`。價格與數量使用字串或 `Decimal`，不要傳 `float`。回應是保留服務端欄位名稱的字典或串列；金額以 `Decimal` 回傳。`APIError.code` 是業務錯誤碼，`HTTPError.status_code` 是 HTTP 狀態碼。SDK 不會自動重試或啟動心跳；交易逾時請先查倉位，不要直接重送。

例如，用 `Decimal("0.01")` 表示數量。下單前，先用 `get_symbol_list(session)` 核對商品的最小數量與增量。
