# {{PROJECT_NAME}}

## 起動方法

```bash
make dev
```

これだけで、バックエンドとフロントエンドが起動します。

- フロントエンド: http://localhost:{{PORT_FRONTEND}}
- バックエンド API: http://localhost:{{PORT_BACKEND}}/api/health

## 停止方法

```bash
make stop
```

## 実装済みの機能

### ページ

| URL | 内容 |
|-----|------|
| http://localhost:{{PORT_FRONTEND}} | トップページ |

### API

| メソッド | URL | 内容 |
|---------|-----|------|
| GET | /api/health | ヘルスチェック |

### 技術スタック

- **バックエンド**: Go + Gin（REST API）
- **フロントエンド**: Next.js + React + TypeScript + Tailwind CSS

## 開発の進め方

VS Code のターミナルで Claude Code を使って開発できます:

1. `/discuss` - 追加機能のアイデアを相談
2. `/plan` - 実装計画を作成
3. `/fullstack` - 計画に基づいて自動実装

## その他のコマンド

```bash
make help     # 全コマンド一覧
make build    # ビルド
make health   # ヘルスチェック
```
