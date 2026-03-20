
# /devtools - 開発進捗ビューア

devtools（開発ドキュメントビューア）を起動する。

カレントディレクトリの `開発/` 配下のMDファイルをブラウザで閲覧できるようにする。

## 実行

```bash
cd ${CLAUDE_PLUGIN_ROOT}/devtools && PROJECT_DIR=$(pwd) npm run dev -- -p 3001
```

起動後、http://localhost:3001 でアクセス可能。

## 停止

```bash
lsof -ti:3001 | xargs kill -9 2>/dev/null || true
```
