# BTCC Open API SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

Choose the SDK for your project. Both login examples read account information without placing orders.

## Go SDK

Requires Go 1.22 or later. [Usage guide](go/README-en.md)

After setting your trading credentials, run this from the repository root:

```sh
cd go
go run ./examples/login
```

## Python SDK

Requires Python 3.9 or later. Install from the repository root; there are no third-party runtime dependencies.

```sh
python3 -m pip install ./python
```

Set `BTCC_BASE_URL`, `BTCC_USER_NAME`, `BTCC_PASSWORD`, `BTCC_API_KEY` and `BTCC_SECRET_KEY`, then run the read-only example:

```sh
python3 python/examples/login.py
```

[Usage guide](python/README-en.md)
