.DEFAULT_GOAL := help
GO ?= $(or $(shell command -v go 2>/dev/null),$(wildcard /opt/homebrew/bin/go),go)

.PHONY: $(wildcard *) help test-login

## help: 顯示可用指令
help:
	@sed -n 's/^## //p' Makefile

## test-login: 讀取 uat.env，只測試 UAT 登入（不下單）
test-login:
	@test -f go/uat_test.go && grep -q '^func TestUATLogin(' go/uat_test.go || { echo '缺少本機 TestUATLogin 測試，無法執行登入檢查。' >&2; exit 1; }
	@cd go && BTCC_UAT_CHECK=1 $(GO) test -run '^TestUATLogin$$' -count=1 -v -timeout=30s .
