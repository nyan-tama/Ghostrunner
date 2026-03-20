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

## その他のコマンド

```bash
make help     # 全コマンド一覧
make build    # ビルド
make health   # ヘルスチェック
```

## 開発の進め方

VS Code のターミナルで Claude Code を使って開発できます:

- `/discuss` - 追加機能のアイデアを相談
- `/plan` - 実装計画を作成
- `/fullstack` - 計画に基づいて自動実装
