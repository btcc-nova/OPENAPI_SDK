# BTCC Open API SDK

<p align="center">
  <a href="README-en.md"><img src="https://img.shields.io/badge/English-Click-yellow" alt="English"></a>
  <a href="README-tw.md"><img src="https://img.shields.io/badge/繁體中文-點擊查看-orange" alt="繁體中文"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/简体中文-点击查看-orange" alt="简体中文"></a>
  <a href="README-ja.md"><img src="https://img.shields.io/badge/日本語-クリック-blue" alt="日本語"></a>
  <a href="README-ko.md"><img src="https://img.shields.io/badge/한국어-클릭-yellow" alt="한국어"></a>
</p>

프로젝트에 맞는 SDK를 선택하세요. 두 로그인 예제는 계좌 정보만 조회하며 주문하지 않습니다.

## Go SDK

Go 1.22 이상이 필요합니다. [사용 안내](go/README-ko.md)

거래 서비스 인증 정보를 설정한 뒤 저장소 루트에서 실행하세요.

```sh
cd go
go run ./examples/login
```

## Python SDK

Python 3.9 이상이 필요합니다. 저장소 루트에서 설치하세요. 실행 시 외부 패키지가 필요하지 않습니다.

```sh
python3 -m pip install ./python
```

`BTCC_BASE_URL`, `BTCC_USER_NAME`, `BTCC_PASSWORD`, `BTCC_API_KEY`, `BTCC_SECRET_KEY`를 설정한 뒤 조회 전용 예제를 실행하세요.

```sh
python3 python/examples/login.py
```

[사용 안내](python/README-ko.md)
