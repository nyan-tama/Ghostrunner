# {{PROJECT_NAME}} - Development Commands
PROJECT_ROOT := $(shell pwd)

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "{{PROJECT_NAME}} - 利用可能なコマンド:"
	@echo ""
	@echo "ビルド・実行:"
	@echo "  make build            - ビルド"
	@echo "  make run              - ビルドして実行"
	@echo "  make test             - テスト実行"
	@echo "  make clean            - ビルド成果物を削除"
	@echo ""

# ビルド
.PHONY: build
build:
	swift build

# 実行
.PHONY: run
run: build
	swift run App

# テスト
.PHONY: test
test:
	swift test

# クリーン
.PHONY: clean
clean:
	swift package clean
