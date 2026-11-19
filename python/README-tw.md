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

## 商品、帳戶群組與倉位

將函式放在 `main()` 前，登入後在 `main()` 內呼叫 `inspect_account(client, session)`。使用建立會話的同一個客戶端。商品的 `enable` 是狀態碼，非零不一定可交易。倉位預設從第 1 頁起，每頁 100 筆，可用 `symbol` 或 `position_id` 篩選；翻頁期間持倉可能改變。

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

## 不登入查報價

這段可獨立執行。請使用服務方確認的交易網址、行情網址與行情商品名稱。行情名稱可能與交易名稱不同。報價不會傳送登入憑證。

```python
import os
from btcc_sdk import Client

client = Client(os.environ["BTCC_BASE_URL"], quote_url=os.environ["BTCC_QUOTE_URL"])
quote = client.get_quote_price(os.environ["BTCC_QUOTE_SYMBOL"])
print(quote["Price"])
```

## 開倉、修改停損停利與平倉

**呼叫以下函式會送出交易請求。** 不要把它們接到登入範例。先在獲准的測試環境核對商品、最小數量、增量與槓桿。請以字串傳入價格與數量，例如 `"0.01"`，不要經過 `float`。

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

開倉回應中的 `position` 是倉位字典；修改與平倉直接回傳倉位字典。從開倉結果取出 `id`，用 `get_position_list(session, position_id=...)` 核對商品和可平數量。`1` 為買、`2` 為賣；平倉方向依服務方規則傳入，SDK 不會反轉。修改停損停利時兩個價格都會送出，`0` 不等於保留原值。

交易方法是 `open_position`、`close_position`、`update_position`。價格與數量使用字串或 `Decimal`，不要傳 `float`。回應是保留服務端欄位名稱的字典或串列；金額以 `Decimal` 回傳。`APIError.code` 是業務錯誤碼，`HTTPError.status_code` 是 HTTP 狀態碼。SDK 不會自動重試或啟動心跳；交易逾時請先查倉位，不要直接重送。

## 維持登入狀態

長時間使用時，由應用程式定期發送心跳。傳入 `threading.Event`，用 `stop.set()` 停止等待。下例會阻塞，20 秒後才送第一次心跳；失敗時例外會往外拋，不會自動重新登入。

```python
def maintain_session(client, session, stop):
    while not stop.wait(20):
        client.keep_alive(session)
```

## 錯誤與交易逾時

下例只查餘額。`APIError.message` 是服務端原始文字，可能含敏感資訊，不要直接寫入日誌。交易逾時、斷線或回應無法解析時，先查倉位及成交紀錄；結果不明不要重送。`ref_id` 不保證訂單去重。

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

## API 一覽

| 方法 | HTTP | 路徑 |
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
