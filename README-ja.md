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

[利用ガイド](python/README-ja.md)
