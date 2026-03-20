# Backend 運用手順書

Ghostrunner バックエンドの運用に関する手順とトラブルシューティング。

## サーバー管理

### 起動

```bash
# フォアグラウンドで起動
make backend

# バックグラウンドで起動してログ表示
make start-backend

# バックエンド + フロントエンドを同時起動
make dev
```

### 停止

```bash
make stop-backend
```

### 再起動

```bash
# バックグラウンドで再起動
make restart-backend

# 再起動してログ表示
make restart-backend-logs
```

### ログ確認

```bash
make logs-backend
```

ログファイルは `/tmp/backend.log` に出力される。

### ヘルスチェック

```bash
make health
# または
curl http://localhost:8080/api/health
```

レスポンス: `{"status":"ok"}`

---

## ntfy.sh 通知

### 概要

ntfy.sh はオープンソースのプッシュ通知サービス。環境変数 `NTFY_TOPIC` を設定することで、コマンド完了時やエラー発生時にスマートフォンやブラウザへプッシュ通知を送信する。

### トピックの設定

#### 1. トピック名の決定

ntfy.sh のトピックは公開チャンネルであるため、推測されにくいユニークな名前を使用する。

```bash
# ランダムなトピック名を生成
echo "ghostrunner-$(openssl rand -hex 6)"
```

#### 2. 環境変数の設定

```bash
# backend/.env に追加
NTFY_TOPIC=ghostrunner-your-unique-id
```

#### 3. サーバーの再起動

```bash
make restart-backend-logs
```

#### 4. 有効化の確認

起動ログに以下が表示されることを確認する。

```
[NtfyService] Initialized with topic: https://ntfy.sh/ghostrunner-your-unique-id
```

### 通知の受信方法

#### スマートフォン

1. ntfy アプリをインストール
   - iOS: https://apps.apple.com/app/ntfy/id1625396347
   - Android: https://play.google.com/store/apps/details?id=io.heckel.ntfy
2. アプリ内で `NTFY_TOPIC` に設定したトピック名を購読

#### ブラウザ

`https://ntfy.sh/your-topic-name` にアクセスして購読する。

### 通知のタイミングと内容

| タイミング | タイトル | 優先度 | 説明 |
|-----------|---------|--------|------|
| コマンド正常完了 | Claude Code - Complete | default | 出力テキストの先頭100文字を通知本文に含む |
| コマンド実行エラー | Claude Code - Error | high | エラーメッセージを通知本文に含む |
| タイムアウト | Claude Code - Error | high | "Execution timeout" が通知される |
| パイプ生成失敗 | Claude Code - Error | high | エラー詳細が通知される |
| CLI起動失敗 | Claude Code - Error | high | エラー詳細が通知される |

### 通知の無効化

環境変数 `NTFY_TOPIC` を削除または空にしてサーバーを再起動する。

```bash
# backend/.env から NTFY_TOPIC の行を削除またはコメントアウト
# NTFY_TOPIC=ghostrunner-your-unique-id

make restart-backend
```

### 通知のテスト

ntfy.sh に直接 POST してトピックの疎通を確認できる。

```bash
curl -d "Test notification from Ghostrunner" \
  -H "Title: Test" \
  -H "Priority: default" \
  https://ntfy.sh/your-topic-name
```

---

## トラブルシューティング

### サーバーが起動しない

**症状**: `make backend` でエラーが発生する

**確認事項**:
1. Go 1.24 がインストールされているか: `go version`
2. ポート 8080 が他のプロセスで使用されていないか: `lsof -i :8080`
3. `.env` ファイルの構文が正しいか

**対処**:
```bash
# ポートを使用しているプロセスを停止
make stop-backend

# 再起動
make restart-backend-logs
```

### ntfy 通知が届かない

**症状**: コマンドを実行しても通知が届かない

**確認事項**:

1. 環境変数が設定されているか確認
   ```bash
   # サーバーのログを確認
   make logs-backend
   # 以下のログが出力されていれば有効
   # [NtfyService] Initialized with topic: https://ntfy.sh/your-topic
   ```

2. ntfy アプリで正しいトピックを購読しているか確認
   - アプリのトピック名と `NTFY_TOPIC` の値が一致していることを確認

3. 通知送信のログを確認
   ```bash
   # ログで以下を検索
   # [NtfyService] Sending notification: title=..., priority=...
   # [NtfyService] Notification sent successfully: title=...
   ```

4. ネットワーク接続を確認
   ```bash
   curl -s https://ntfy.sh/health
   ```

**対処**:
- トピック名に typo がないか確認
- ntfy アプリでトピックを再購読
- サーバーを再起動して環境変数を再読み込み

### ntfy 通知が送信失敗する

**症状**: ログに `[NtfyService] Failed to send notification` が表示される

**確認事項**:
1. ntfy.sh サーバーが稼働しているか: `curl -s https://ntfy.sh/health`
2. ネットワーク接続に問題がないか
3. HTTPタイムアウト（10秒）に達していないか

**対処**:
- ntfy.sh のステータスを確認: https://ntfy.sh
- 通知送信はfire-and-forget方式のため、送信失敗がコマンド実行には影響しない
- 一時的なネットワーク問題の場合は自然に回復する

### コマンドがタイムアウトする

**症状**: 60分後にタイムアウトエラーが返る

**確認事項**:
1. Claude CLI が正常に動作しているか: `claude --version`
2. 対象プロジェクトのパスが正しいか

**対処**:
- タイムアウト値はClaudeServiceの `timeout` フィールドで設定されている（60分）
- 長時間実行が必要な場合はタスクを分割して実行する

### ポートが既に使用されている

**症状**: `bind: address already in use` エラー

**対処**:
```bash
# バックエンドのプロセスを強制停止
make stop-backend

# 確認
lsof -i :8080

# それでも解決しない場合
lsof -ti:8080 | xargs kill -9
```
