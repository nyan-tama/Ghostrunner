# ntfy Mac PC 通知設定 実装計画

## 仕様書

`開発/検討中/2026-02-16_ntfy_Mac通知設定.md`

## 概要

ntfy.sh の Mac PC 側受信環境を `make` コマンドで自動セットアップする。
3つの Make ターゲット（`setup-ntfy`, `uninstall-ntfy`, `status-ntfy`）とそれを支えるシェルスクリプトを実装する。

## 改定履歴

| 日付 | 内容 |
|------|------|
| 2026-02-16 | 初版: ntfy CLI + LaunchAgent 方式で実装 |
| 2026-02-20 | **改定: LaunchAgent 方式を廃止、terminal-notifier 直接実行方式に変更** |

## 改定: LaunchAgent 方式の廃止 (2026-02-20)

### 廃止理由

1. **構成が回りくどい**: バックエンド → ntfy.sh → ntfy CLI (LaunchAgent) → osascript → Mac通知、という多段構成になっていた
2. **LaunchAgent が2重起動**: `sh.ntfy.subscriber` と `sh.ntfy.subscribe` が別々に動いており管理が煩雑
3. **osascript の通知が VSCode 拡張から届かない**: 既知の問題 (Issue #16114)
4. **ntfy LaunchAgent は iPhone 通知には不要**: iPhone には ntfy アプリが直接受信するため、Mac 側の subscriber は Mac 通知のためだけに存在していた

### 新方式: terminal-notifier 直接実行

Mac 通知には `terminal-notifier` を直接実行する。常駐プロセス不要。

| 場面 | Mac 通知 | iPhone 通知 |
|------|---------|------------|
| Claude Code 完了/入力待ち | hooks → `terminal-notifier` | hooks → `curl ntfy.sh` |
| バックエンド業務通知（ローカル） | ntfy.go → `terminal-notifier` | ntfy.go → ntfy.sh |
| バックエンド業務通知（Cloud Run） | なし | ntfy.go → ntfy.sh |

### 削除したもの

| 項目 | パス |
|------|------|
| LaunchAgent (subscriber) | `~/Library/LaunchAgents/sh.ntfy.subscriber.plist` |
| LaunchAgent (subscribe) | `~/Library/LaunchAgents/sh.ntfy.subscribe.plist` |
| ntfy 設定 | `~/Library/Application Support/ntfy/client.yml` |
| 通知スクリプト | `~/.local/bin/ntfy-notify.sh` |
| セットアップスクリプト | `scripts/setup-ntfy.sh` |
| アンインストールスクリプト | `scripts/uninstall-ntfy.sh` |
| Makefile ターゲット | `setup-ntfy`, `uninstall-ntfy`, `status-ntfy` |
| scripts/ ディレクトリ | 空になったため削除 |

### 変更したファイル

| ファイル | 変更内容 |
|----------|----------|
| `~/.claude/settings.json` | Stop/Notification hooks を osascript → terminal-notifier に変更 |
| `backend/internal/service/ntfy.go` | ローカル実行時に terminal-notifier でMac通知も出す機能を追加 |
| `Makefile` | ntfy 関連ターゲット (setup-ntfy, uninstall-ntfy, status-ntfy) を削除 |

### ntfy.go の変更詳細

- `exec.LookPath("terminal-notifier")` で起動時に存在チェック
- 見つかった場合: ntfy.sh 送信 + terminal-notifier でMac通知（並行実行）
- 見つからない場合（Cloud Run）: ntfy.sh 送信のみ（従来通り）
- エラー通知時はサウンドを `Basso` に変更

### 前提条件

- `terminal-notifier` がインストール済み (`brew install terminal-notifier`)
- macOS の通知設定で terminal-notifier の通知を「許可」にする

---

## 以下は初版の計画（参考: 廃止済み）

### 懸念点

#### 1. ntfy CLI のパスがハードコードされている

**仕様書の記述**: plist 内で `/opt/homebrew/bin/ntfy` を使用

**懸念**: Intel Mac では `/usr/local/bin/ntfy` になる。

**解決策**: `which ntfy` の結果を使って動的にパスを設定する。

#### 2. scripts/ ディレクトリが存在しない

**現状**: プロジェクトに `scripts/` ディレクトリがまだ存在しない。

**解決策**: `setup-ntfy.sh` と `uninstall-ntfy.sh` を新規作成時に `scripts/` ディレクトリも作成。`status-ntfy` は短いので Makefile 内にインラインで記述する。

#### 3. 既存の LaunchAgent / 設定ファイルがある場合の挙動

**懸念**: 再実行時に既存ファイルを上書きしてよいか。

**解決策**: 既存ファイルがある場合はバックアップを取らず上書きする（冪等性を保つ）。`launchctl unload` を先に実行して安全にリロードする（仕様書の手順通り）。

### 変更ファイル一覧

| ファイル | 操作 | 概要 |
|----------|------|------|
| `Makefile` | 変更 | `setup-ntfy`, `uninstall-ntfy`, `status-ntfy` ターゲットを追加、help にも追記 |
| `scripts/setup-ntfy.sh` | 新規 | セットアップスクリプト本体 |
| `scripts/uninstall-ntfy.sh` | 新規 | アンインストールスクリプト |

### 実装ステップ

#### ステップ 1: `scripts/setup-ntfy.sh` の作成

以下の処理を順番に実行するシェルスクリプト:

1. `which ntfy` で CLI のインストール確認（未インストールなら `brew install ntfy` を案内して終了）
2. `backend/.env` から `NTFY_TOPIC` を読み取り（未設定なら案内して終了）
3. `~/Library/Application Support/ntfy/` ディレクトリを作成
4. `client.yml` を生成（`NTFY_TOPIC` を埋め込み、terminal-notifier 優先 / osascript フォールバックの command を設定）
5. `~/Library/LaunchAgents/sh.ntfy.subscriber.plist` を生成（ntfy CLI のパスは `which ntfy` の結果を使用）
6. `launchctl unload` → `launchctl load` で登録・起動
7. 3秒待機後、テスト通知を curl で送信
8. 通知が届いたか確認するよう案内メッセージを表示
9. 通知が届かない場合のトラブルシューティング手順を表示

#### ステップ 2: `scripts/uninstall-ntfy.sh` の作成

以下の処理を実行:

1. `launchctl unload ~/Library/LaunchAgents/sh.ntfy.subscriber.plist` で停止
2. plist ファイルを削除
3. `~/Library/Application Support/ntfy/client.yml` を削除
4. 完了メッセージを表示（ntfy CLI 自体はアンインストールしない旨を案内）

#### ステップ 3: Makefile の変更

1. help ターゲットに ntfy 関連コマンドのセクションを追加
2. `setup-ntfy` ターゲット: `scripts/setup-ntfy.sh` を実行
3. `uninstall-ntfy` ターゲット: `scripts/uninstall-ntfy.sh` を実行
4. `status-ntfy` ターゲット: 以下をインラインで実行
   - `launchctl list | grep sh.ntfy.subscriber` で LaunchAgent 状態確認
   - `pgrep -f "ntfy subscribe"` で プロセス確認
   - 設定ファイルの存在確認
   - `/tmp/ntfy-subscriber.log` の末尾10行を表示

### テストプラン

#### 手動テスト（シェルスクリプトのため自動テストは不要）

1. **`make setup-ntfy` の正常系**: 実行後に LaunchAgent が登録され、テスト通知が Mac 通知センターに届くこと
2. **`make status-ntfy`**: 各項目（LaunchAgent、プロセス、設定ファイル、ログ）が正しく表示されること
3. **`make uninstall-ntfy`**: plist と設定ファイルが削除され、ntfy プロセスが停止すること
4. **再実行の冪等性**: `make setup-ntfy` を2回連続実行しても問題なく動作すること
5. **エラーケース**: ntfy CLI 未インストール時、NTFY_TOPIC 未設定時にエラーメッセージが表示されること
