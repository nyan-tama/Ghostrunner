# Tailscaleで外部から開発環境にアクセス

## 概要

外出先やスマホから開発中のアプリ（Next.js + Go API）を確認できるようにする。

## 要件

| 項目 | 内容 |
|------|------|
| 用途 | 開発中のアプリを外部から確認（UI確認、動作テスト） |
| アクセス元 | 自分のスマホ/タブレット |
| セキュリティ | 自分だけがアクセスできればOK |

## 選定した方式

**案A: Tailscale直接アクセス**

```
自分のスマホ (Tailscaleアプリ)
        ↓ 100.x.x.x:3000 / :8080
開発マシン (Tailscale)
   ├── Next.js (0.0.0.0:3000)
   └── Go API (0.0.0.0:8080)
```

### 選定理由

- 設定がシンプル
- 追加ツール不要
- 複数ポート同時アクセス可能
- Tailscale認証で十分なセキュリティ

### 他の案との比較

| 観点 | 案A: 直接アクセス | 案B: Serve | 案C: Funnel |
|------|------------------|------------|-------------|
| セットアップ | 最も簡単 | やや手間 | 簡単 |
| HTTPS | なし | あり | あり |
| アクセス元の制限 | Tailscale必須 | Tailscale必須 | 制限なし |
| 複数ポート | 同時アクセス可 | 個別設定 | 個別設定 |
| 推奨シナリオ | 自分のスマホで確認 | HTTPS必須の機能テスト | 外部へのデモ |

## 実装内容

### 1. Tailscaleインストール（開発マシン）

```bash
# macOS
brew install --cask tailscale

# または公式サイトからダウンロード
# https://tailscale.com/download/mac
```

### 2. Tailscaleアプリインストール（スマホ）

- iOS: App Store で「Tailscale」
- Android: Google Play で「Tailscale」

同じアカウントでログイン

### 3. 開発サーバーの設定変更

#### Next.js（0.0.0.0でリッスン）

```json
// package.json
{
  "scripts": {
    "dev": "next dev -H 0.0.0.0"
  }
}
```

#### Go API（すでに0.0.0.0の場合は変更不要）

```go
r.Run("0.0.0.0:8080")
```

### 4. Makefileにコマンド追加（任意）

```makefile
# 外部アクセス用の開発サーバー起動
dev-external:
	@echo "Starting servers for external access..."
	@echo "Access from Tailscale IP: $$(tailscale ip -4):3000 / :8080"
	$(MAKE) dev
```

### 5. アクセス方法

```bash
# TailscaleのIPを確認
tailscale ip -4
# 例: 100.100.100.1

# スマホのブラウザから
# http://100.100.100.1:3000  (フロントエンド)
# http://100.100.100.1:8080  (API)
```

## 将来の拡張

必要に応じて以下を追加:

- **案B（Tailscale Serve）**: HTTPS が必要な機能テスト時
- **案C（Tailscale Funnel）**: クライアントへのデモ時

## ステータス

- [x] 検討完了
- [ ] 実装待ち
