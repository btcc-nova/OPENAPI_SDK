# BTCC Open API SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

选择适合项目的 SDK。两个登录示例都只查询账户，不会下单。

## Go SDK

Go 1.22 或以上。 [使用指南](go/README.md)

设置交易服务提供的登录凭证后，从仓库根目录运行：

```sh
cd go
go run ./examples/login
```

## Python SDK

需要 Python 3.9 或以上。从仓库根目录安装，无运行时第三方依赖。

```sh
python3 -m pip install ./python
```

设置 `BTCC_BASE_URL`、`BTCC_USER_NAME`、`BTCC_PASSWORD`、`BTCC_API_KEY` 和 `BTCC_SECRET_KEY` 后，可运行只读示例：

```sh
python3 python/examples/login.py
```

[使用指南](python/README.md)
