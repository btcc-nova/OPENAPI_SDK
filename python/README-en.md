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

## Queries and quotes

`login()` returns a `Session` bound to the client. Pass it to `keep_alive`, `get_symbol_list`, `get_account_group`, `get_account` or `get_position_list`. Quotes do not require login: configure `Client(base_url, quote_url=...)` and call `get_quote_price(symbol)`.

## Trading and errors

Trading methods are `open_position`, `close_position` and `update_position`. Supply prices and quantities as strings or `Decimal`, not `float`. Responses are dictionaries or lists using the original field names; monetary values are `Decimal`. `APIError.code` contains the service error code; `HTTPError.status_code` contains the HTTP status. There are no automatic retries or heartbeats. After a trade timeout, check the position before sending another order.

For example, use `Decimal("0.01")` to represent a quantity. Before trading, check the product’s minimum volume and step size with `get_symbol_list(session)`.
