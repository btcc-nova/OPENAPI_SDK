# BTCC Open API Python SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

## Installation

Requires Python 3.9 or later, with no third-party runtime dependencies. From this directory (`python/`), run:

```sh
python3 -m pip install .
```

## Sign in and check your balance

Supply the provider-confirmed HTTPS trading URL, username, password, API Key and Secret Key to the example through environment variables. `BTCC_COMPANY_ID` is optional and defaults to `1`. Keep credentials out of source code.

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

Run the [read-only example](examples/login.py) from this directory. It prints the account ID and balance without placing an order:

```sh
python3 examples/login.py
```

## Products, account group and positions

Define this function before `main()`, then call `inspect_account(client, session)` inside `main()` after login. Use the client that created the session. A nonzero product `enable` status does not necessarily mean trading is allowed. Positions start at page 1 with 100 rows by default. Filter with `symbol` or `position_id`; positions can change between pages.

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

## Get a quote without signing in

Run this snippet independently. Use the trading URL, quote URL and quote symbol confirmed by your provider. The quote symbol can differ from the trading symbol. The quote request sends no login credentials.

```python
import os
from btcc_sdk import Client

client = Client(os.environ["BTCC_BASE_URL"], quote_url=os.environ["BTCC_QUOTE_URL"])
quote = client.get_quote_price(os.environ["BTCC_QUOTE_SYMBOL"])
print(quote["Price"])
```

## Open a position, update stops and close

**Calling these functions sends trading requests.** Keep them separate from the login example. Verify the product, minimum volume, volume step and leverage in an authorized test environment. Pass prices and volumes as decimal strings such as `"0.01"`, never through `float`.

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

Open returns a result containing a `position` dictionary; update and close return that dictionary directly. Take the `id` from the open result and confirm the product and remaining volume with `get_position_list(session, position_id=...)`. Direction `1` means buy, `2` means sell. Supply the close direction required by your provider; the SDK does not reverse it. Updating stops sends both prices: `0` does not mean “leave unchanged”.

Trading methods are `open_position`, `close_position` and `update_position`. Supply prices and quantities as strings or `Decimal`, not `float`. Responses are dictionaries or lists using the original field names; monetary values are `Decimal`. `APIError.code` contains the service error code; `HTTPError.status_code` contains the HTTP status. There are no automatic retries or heartbeats. After a trade timeout, check the position before sending another order.

## Keep a session alive

Schedule heartbeats in your application for long-running sessions. Pass a `threading.Event` and call `stop.set()` to stop waiting. This function blocks, sends its first heartbeat after 20 seconds and propagates failures without signing in again.

```python
def maintain_session(client, session, stop):
    while not stop.wait(20):
        client.keep_alive(session)
```

## Errors and trade timeouts

This example only reads the balance. Do not log `APIError.message` blindly: it is raw server text and may contain sensitive information. After a timeout, disconnect or unreadable trade response, check positions and execution records. Do not resend if the outcome is unknown; `ref_id` does not guarantee deduplication.

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

## API reference

| Method | HTTP | Path |
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
