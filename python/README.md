# BTCC Open API Python SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

## 安装

需要 Python 3.9 或以上，无运行时第三方依赖。在本目录（`python/`）执行：

```sh
python3 -m pip install .
```

## 登录并查询余额

通过环境变量向示例提供服务方确认的 HTTPS 交易地址、用户名、密码、API Key 和 Secret Key。`BTCC_COMPANY_ID` 可选，默认是 `1`。不要将凭证写入源码。

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

在本目录运行[只读示例](examples/login.py)，它只显示账户 ID 和余额，不会下单：

```sh
python3 examples/login.py
```

## 产品、账户组与仓位

将函数放在 `main()` 前，登录后在 `main()` 中调用 `inspect_account(client, session)`。使用创建会话的同一个客户端。产品的 `enable` 是状态码，非零不一定可交易。仓位默认从第 1 页起，每页 100 条，可用 `symbol` 或 `position_id` 筛选；翻页期间持仓可能变化。

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

## 无需登录的报价查询

这段代码可独立运行。请使用服务方确认的交易地址、行情地址和行情产品名称。行情名称可能与交易名称不同。报价不会发送登录凭证。

```python
import os
from btcc_sdk import Client

client = Client(os.environ["BTCC_BASE_URL"], quote_url=os.environ["BTCC_QUOTE_URL"])
quote = client.get_quote_price(os.environ["BTCC_QUOTE_SYMBOL"])
print(quote["Price"])
```

## 开仓、修改止盈止损与平仓

**调用以下函数会发送交易请求。** 不要接入登录示例。先在获准的测试环境核对产品、最小数量、步长和杠杆。价格与数量以字符串传入，例如 `"0.01"`，不要经过 `float`。

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

开仓响应中的 `position` 是仓位字典；修改和平仓直接返回仓位字典。从开仓结果取出 `id`，用 `get_position_list(session, position_id=...)` 核对产品及可平数量。`1` 为买、`2` 为卖；按服务方规则传入平仓方向，SDK 不会反转。修改止盈止损时两个价格都会发送，`0` 不表示保留原值。

交易方法为 `open_position`、`close_position`、`update_position`。价格和数量使用字符串或 `Decimal`，不要传 `float`。响应使用服务端字段名的字典或列表；金额按 `Decimal` 返回。`APIError.code` 是业务错误码，`HTTPError.status_code` 是 HTTP 状态码。SDK 不自动重试或启动心跳；交易超时请先查仓位，不要直接重送。

## 维持登录状态

长时间运行时由应用定期发送心跳。传入 `threading.Event`，用 `stop.set()` 停止等待。本例会阻塞，20 秒后才发送第一次心跳；失败时异常向上抛出，不自动重新登录。

```python
def maintain_session(client, session, stop):
    while not stop.wait(20):
        client.keep_alive(session)
```

## 错误与交易超时

下例只查余额。`APIError.message` 是服务端原始文本，可能含敏感信息，不要直接记录。交易超时、断线或响应无法解析时，先查仓位及成交记录；结果不明不要重发。`ref_id` 不保证订单去重。

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

## API 列表

| 方法 | HTTP | 路径 |
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
