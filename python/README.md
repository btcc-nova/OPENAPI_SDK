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

## 查询与报价

`login()` 返回绑定当前客户端的 `Session`。将它传给 `keep_alive`、`get_symbol_list`、`get_account_group`、`get_account` 或 `get_position_list`。报价不需要登录：设置 `Client(base_url, quote_url=...)` 后调用 `get_quote_price(symbol)`。

## 交易与错误处理

交易方法为 `open_position`、`close_position`、`update_position`。价格和数量使用字符串或 `Decimal`，不要传 `float`。响应使用服务端字段名的字典或列表；金额按 `Decimal` 返回。`APIError.code` 是业务错误码，`HTTPError.status_code` 是 HTTP 状态码。SDK 不自动重试或启动心跳；交易超时请先查仓位，不要直接重送。

例如，用 `Decimal("0.01")` 表示数量。下单前，先用 `get_symbol_list(session)` 核对实际产品的最小数量和步长。
