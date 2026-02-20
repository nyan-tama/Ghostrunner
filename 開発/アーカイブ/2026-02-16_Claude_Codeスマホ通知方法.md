# 調査レポート: Claude Code コマンド完了時のスマホ通知方法

## 概要

Claude Code の hooks 機能（`Stop` イベント、`Notification` イベント）を利用して、タスク完了時にシェルスクリプトを実行し、外部プッシュ通知サービス経由でスマートフォンに通知を送信できる。公式のクロスデバイス通知機能は未実装だが、hooks とサードパーティサービスの組み合わせで十分に実現可能。

## 背景

Claude Code で長時間のタスク（ビルド、リファクタリング、テスト実行等）を実行中にターミナルから離れた場合、処理完了を見逃すことがある。スマートフォンへのプッシュ通知により、離席中やモバイル環境でもタスク完了を即座に把握できる仕組みが求められている。

本プロジェクトの `.claude/settings.json` には既に `PreToolUse`、`PostToolUse`、`Stop` フックが設定されているが、通知送信の仕組みは未設定の状態。

---

## 調査結果

### 1. Claude Code の hooks 機能（公式）

Claude Code には 2025年6月に導入され、2026年に大幅に拡充されたフック機能がある。設定ファイルの JSON でイベント発火時に実行するコマンドを定義できる。

#### 通知に関連するフックイベント

| フックイベント | 発火タイミング | 通知用途 |
|---|---|---|
| **`Stop`** | Claude が応答を完了した時 | **タスク完了通知（最重要）** |
| **`Notification`** | Claude Code が通知を送信する時 | 入力待ち・権限要求の通知 |
| `TaskCompleted` | タスクが完了としてマークされた時 | エージェントチームのタスク完了通知 |
| `SubagentStop` | サブエージェントが完了した時 | 並列タスクの完了通知 |
| `SessionEnd` | セッションが終了した時 | セッション終了通知 |

#### `Notification` イベントの matcher 値

| matcher | 説明 |
|---|---|
| `permission_prompt` | 権限要求時 |
| `idle_prompt` | 入力待ち（60秒以上のアイドル後） |
| `auth_success` | 認証成功時 |
| `elicitation_dialog` | MCP ツールの入力要求時 |

#### 設定ファイルの配置場所

| ファイル | スコープ | Git 管理 |
|---|---|---|
| `~/.claude/settings.json` | ユーザーグローバル（全プロジェクト共通） | 対象外 |
| `.claude/settings.json` | プロジェクト固有（チーム共有可能） | 対象 |
| `.claude/settings.local.json` | ローカルプロジェクト設定 | 対象外 |

通知設定はプラットフォーム固有のため、**`~/.claude/settings.json`（ユーザーグローバル）** に配置するのが推奨。

#### hooks の JSON 設定構造

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "シェルコマンド"
          }
        ]
      }
    ],
    "Notification": [
      {
        "matcher": "permission_prompt|idle_prompt",
        "hooks": [
          {
            "type": "command",
            "command": "シェルコマンド"
          }
        ]
      }
    ]
  }
}
```

#### Stop イベントで受信する JSON 入力

```json
{
  "session_id": "abc123",
  "transcript_path": "/Users/.../.claude/projects/.../transcript.jsonl",
  "cwd": "/Users/user/Ghostrunner",
  "permission_mode": "default",
  "hook_event_name": "Stop",
  "stop_hook_active": false
}
```

`cwd` フィールドからプロジェクト名を抽出して通知メッセージに含めることができる。

#### async hooks（非同期フック）

```json
{
  "type": "command",
  "command": "curl -s ...",
  "async": true,
  "timeout": 30
}
```

`async: true` を指定すると、フックがバックグラウンドで実行され Claude の処理をブロックしない。通知送信のようにレスポンスを待つ必要がないケースに適している。ただし、async hooks は decision 制御（ブロック等）ができない点に注意。

---

### 2. Claude Code の組み込み通知機能

Claude Code 自体にはスマートフォンへのプッシュ通知機能は搭載されていない。

- **デスクトップ通知**: Claude Code は `Notification` イベントで権限要求や入力待ちを通知するが、これはターミナルベルまたは OS 通知に依存する
- **モバイル通知**: 未搭載。GitHub Issue #7590 で機能リクエストが出されたが、「not planned」としてクローズされた
- **公式見解**: hooks システムを使って自分で実装することが推奨されている

---

### 3. スマートフォン通知サービスの比較

#### 3-1. ntfy.sh（無料・オープンソース）

**概要**: HTTP PUT/POST で通知を送信するシンプルな pub-sub 通知サービス。

```bash
# 最もシンプルな送信方法
curl -d "Claude Code: タスク完了" ntfy.sh/your-secret-topic

# タイトル・優先度指定
curl \
  -H "Title: Claude Code" \
  -H "Priority: default" \
  -H "Tags: white_check_mark" \
  -d "タスクが完了しました" \
  ntfy.sh/your-secret-topic
```

**Claude Code hooks 設定例**:
```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "curl -s -H 'Title: Claude Code' -d \"Task completed: $(basename \"$PWD\")\" ntfy.sh/YOUR_SECRET_TOPIC 2>/dev/null || true"
          }
        ]
      }
    ]
  }
}
```

**セットアップ手順**:
1. iPhone/Android に ntfy アプリをインストール（無料）
2. アプリ内で推測困難なトピック名を購読（例: `claude-work-a8x9k2m7`）
3. macOS で動作確認: `curl -d "テスト" ntfy.sh/claude-work-a8x9k2m7`
4. hooks 設定を追加し Claude Code を再起動

| 項目 | 内容 |
|---|---|
| 費用 | 無料（公開サーバー）/ セルフホスト可能 |
| iOS | App Store で無料 |
| Android | Google Play / F-Droid で無料 |
| セットアップ時間 | 約5分 |
| アカウント登録 | 不要（匿名利用可） |
| Apple Watch | 対応（iOS アプリ経由） |
| セキュリティ | トピック名がパスワード代わり。有料版でアクセストークン認証あり |

---

#### 3-2. Pushover（有料・高信頼性）

**概要**: 商用のプッシュ通知サービス。高い信頼性と豊富な機能。

```bash
curl -s \
  --form-string "token=$PUSHOVER_APP_TOKEN" \
  --form-string "user=$PUSHOVER_USER_KEY" \
  --form-string "title=Claude Code" \
  --form-string "message=タスク完了: $(basename $PWD)" \
  --form-string "sound=pushover" \
  https://api.pushover.net/1/messages.json
```

**セットアップ手順**:
1. pushover.net でアカウント作成、User Key を取得
2. アプリケーション登録で API Token を取得
3. iPhone に Pushover アプリをインストール（30日無料、以降 $5 ワンタイム）
4. 環境変数 `PUSHOVER_APP_TOKEN` と `PUSHOVER_USER_KEY` を設定

| 項目 | 内容 |
|---|---|
| 費用 | $5 ワンタイム購入（iOS/Android 各プラットフォーム） |
| セットアップ時間 | 約10分 |
| Apple Watch | 対応 |
| 優先度指定 | 対応（-2〜2、緊急通知で確認応答あり） |
| デバイス指定 | 対応（特定デバイスのみに通知可能） |
| 配信追跡 | 対応 |

---

#### 3-3. Pushcut（iPhone/Apple Watch 特化）

**概要**: iOS ショートカットとの統合に優れた通知サービス。

```bash
curl -s \
  -H "API-Key: $PUSHCUT_WEBHOOK_SECRET" \
  -H "Content-Type: application/json" \
  -d '{"title":"Claude Code","text":"タスクが完了しました"}' \
  https://api.pushcut.io/v1/notifications/terminal
```

**セットアップ手順**:
1. iOS App Store から Pushcut をインストール
2. アプリ内で「terminal」という名前の通知を作成
3. アカウントタブからシークレットをコピー
4. 環境変数に設定

| 項目 | 内容 |
|---|---|
| 費用 | 基本無料 / Pro 月額$2（動的タイトル・テキスト等） |
| Apple Watch | 対応（watchOS 6以降） |
| iOS ショートカット連携 | 強力（通知タップでショートカット実行可能） |
| Android | 非対応（iOS 専用） |

---

#### 3-4. Slack Webhook

**概要**: Slack のインカミング Webhook でチャンネルにメッセージを送信。

```bash
curl -X POST \
  -H 'Content-type: application/json' \
  -d '{"text":"Claude Code: タスク完了 - '"$(basename $PWD)"'"}' \
  "$SLACK_WEBHOOK_URL"
```

**セットアップ手順**:
1. Slack で Incoming Webhooks アプリを追加
2. 通知先チャンネルを選択
3. Webhook URL を取得
4. 環境変数 `SLACK_WEBHOOK_URL` に設定

| 項目 | 内容 |
|---|---|
| 費用 | 無料（Slack アカウント必要） |
| セットアップ時間 | 約5分 |
| スマホ通知 | Slack アプリの通知設定に依存 |
| メッセージ装飾 | Block Kit で豊富なフォーマット可能 |
| 既にチーム利用中なら | 追加コスト/アプリなし |

---

#### 3-5. Discord Webhook

**概要**: Discord のチャンネル Webhook でメッセージを送信。

```bash
curl -H "Content-Type: application/json" \
  -d '{"content":"Claude Code: タスク完了 - '"$(basename $PWD)"'"}' \
  "$DISCORD_WEBHOOK_URL"
```

**セットアップ手順**:
1. Discord サーバーで通知用チャンネルを作成
2. チャンネル設定 > 連携サービス > ウェブフック > 新しいウェブフック
3. Webhook URL をコピー
4. 環境変数に設定

| 項目 | 内容 |
|---|---|
| 費用 | 無料（Discord アカウント必要） |
| セットアップ時間 | 約5分 |
| スマホ通知 | Discord アプリの通知設定に依存 |
| Embed 対応 | 可能（リッチなカード形式表示） |

---

#### 3-6. Pushbullet

**概要**: デバイス間でリンク・ファイル・通知を共有するサービス。

```bash
curl -u "$PUSHBULLET_TOKEN": \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{"type":"note","title":"Claude Code","body":"タスク完了"}' \
  https://api.pushbullet.com/v2/pushes
```

**セットアップ手順**:
1. pushbullet.com でアカウント作成
2. Settings > Access Tokens からトークンを取得
3. スマホアプリをインストール
4. 環境変数に設定

| 項目 | 内容 |
|---|---|
| 費用 | 基本無料 / Pro $4.99/月（SMS, ユニバーサルコピペ等） |
| iOS | 対応（機能制限あり） |
| Android | フル対応 |
| ブラウザ拡張 | Chrome, Firefox 対応 |

---

#### 3-7. LINE Messaging API（LINE Notify の代替）

**注意**: LINE Notify は 2025年3月31日にサービス終了。後継の Messaging API を使用する。

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $LINE_CHANNEL_ACCESS_TOKEN" \
  -d '{
    "to": "YOUR_USER_ID",
    "messages": [{"type": "text", "text": "Claude Code: タスク完了"}]
  }' \
  https://api.line.me/v2/bot/message/push
```

**セットアップ手順**:
1. LINE Developers Console でプロバイダーとチャネルを作成
2. Messaging API チャネルを設定
3. チャネルアクセストークン（長期）を発行
4. LINE 公式アカウントを友だち追加
5. ユーザー ID を取得

| 項目 | 内容 |
|---|---|
| 費用 | 無料（月200通まで。超過は有料） |
| セットアップ時間 | 約20-30分（LINE Developers 登録が必要） |
| 日本での普及率 | 最も高い |
| セットアップ難易度 | 高（チャネル設定、Bot 作成が必要） |

---

#### 3-8. IFTTT Webhook

**概要**: Webhook トリガーでスマホ通知を送信するノーコードサービス。

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"value1":"Claude Code","value2":"タスク完了"}' \
  https://maker.ifttt.com/trigger/claude_done/with/key/$IFTTT_KEY
```

**セットアップ手順**:
1. IFTTT アカウント作成
2. Webhooks サービスを接続
3. 「If Webhook, Then Notification」のアプレットを作成
4. Webhook キーを取得

| 項目 | 内容 |
|---|---|
| 費用 | 無料（アプレット2個まで）/ Pro $3.49/月 |
| セットアップ時間 | 約10分 |
| 拡張性 | 高（他の IoT サービスとの連携可能） |
| 遅延 | 数秒〜数十秒（リアルタイムではない場合あり） |

---

### 4. macOS ターミナルでの完了通知（一般的な方法）

#### 4-1. iTerm2 の「Alert on Next Mark」機能

iTerm2 のシェルインテグレーションがインストールされている場合、`Opt+Cmd+A` で「次のマーク時にアラート」を設定可能。長時間コマンド実行前に押しておけば、完了時にデスクトップ通知が届く。

#### 4-2. osascript（macOS ネイティブ通知）

```bash
# コマンド完了後に通知
osascript -e 'display notification "完了しました" with title "Terminal" sound name "Hero"'
```

#### 4-3. terminal-notifier（Homebrew）

```bash
brew install terminal-notifier
terminal-notifier -title 'Claude Code' -message 'タスク完了' -sound Hero \
  -sender com.anthropic.claudefordesktop
```

`-sender com.anthropic.claudefordesktop` で Claude Desktop のアイコンを通知に表示可能。

#### 4-4. say コマンド（音声読み上げ）

```bash
say "タスクが完了しました"
```

---

### 5. 現在のプロジェクト設定の状況

プロジェクトの `.claude/settings.json` には既に以下のフックが設定済み:

- **PreToolUse**: tmux 外での開発サーバー実行ブロック、git push 前の確認、不要な .md ファイル作成ブロック
- **PostToolUse**: PR 作成後の URL 表示、Prettier/gofmt 自動フォーマット、TypeScript/go vet チェック、console.log/fmt.Print 警告
- **Stop**: セッション終了前の console.log/fmt.Print 最終監査

`Notification` イベントのフックは**未設定**。
`Stop` イベントには監査フックのみで、**通知は未設定**。

`.claude/settings.local.json` には `terminal-notifier` の権限許可が設定済み（`"Bash(terminal-notifier:*)"` が allow リストにある）。

---

## 比較表

### スマホ通知サービス総合比較

| 項目 | ntfy.sh | Pushover | Pushcut | Slack | Discord | Pushbullet | LINE Messaging | IFTTT |
|---|---|---|---|---|---|---|---|---|
| **費用** | 無料 | $5 (一回) | 無料/Pro $2/月 | 無料 | 無料 | 無料/Pro $5/月 | 無料 (200通/月) | 無料 (2個) /Pro $3.5/月 |
| **iOS 対応** | 対応 | 対応 | 対応 | 対応 | 対応 | 対応(制限) | 対応 | 対応 |
| **Android 対応** | 対応 | 対応 | 非対応 | 対応 | 対応 | 対応 | 対応 | 対応 |
| **Apple Watch** | 対応 | 対応 | 対応 | 対応 | 非対応 | 非対応 | 非対応 | 非対応 |
| **アカウント登録** | 不要 | 必要 | 必要 | 必要 | 必要 | 必要 | 必要 | 必要 |
| **セットアップ時間** | 5分 | 10分 | 10分 | 5分 | 5分 | 10分 | 20-30分 | 10分 |
| **セットアップ難易度** | 最低 | 低 | 低 | 低 | 低 | 低 | 高 | 中 |
| **curl のみで送信** | 可能 | 可能 | 可能 | 可能 | 可能 | 可能 | 可能 | 可能 |
| **セルフホスト** | 可能 | 不可 | 不可 | 不可 | 不可 | 不可 | 不可 | 不可 |
| **追加アプリ不要** | 要 | 要 | 要 | 既存Slack利用可 | 既存Discord利用可 | 要 | 既存LINE利用可 | 要 |
| **通知の即時性** | 即時 | 即時 | 即時 | 即時 | 即時 | 即時 | 即時 | 数秒遅延あり |
| **日本語対応** | 対応 | 対応 | 対応 | 対応 | 対応 | 対応 | 完全対応 | 対応 |

### ユースケース別推奨

| ユースケース | 推奨サービス | 理由 |
|---|---|---|
| 最も手軽に始めたい | **ntfy.sh** | アカウント不要、5分でセットアップ完了 |
| 高信頼性が必要 | **Pushover** | 商用サービス、配信追跡・優先度制御あり |
| iPhone + Apple Watch | **Pushover** or **Pushcut** | ネイティブ Apple Watch 対応 |
| 既に Slack を使っている | **Slack Webhook** | 追加アプリ・費用なし |
| 既に Discord を使っている | **Discord Webhook** | 追加アプリ・費用なし |
| プライバシー重視 | **ntfy.sh (セルフホスト)** | 完全に自分のサーバーで運用 |
| 日本語環境・LINE ユーザー | **LINE Messaging API** | 普段使いのアプリで受信 |
| IoT/ホームオートメーション連携 | **IFTTT** | 他サービスとの連携が容易 |

---

## 既知の問題・注意点

- [Issue #7590](https://github.com/anthropics/claude-code/issues/7590): クロスデバイス通知の機能リクエストが出されたが「not planned」でクローズ。hooks での自前実装が公式推奨
- [Issue #16114](https://github.com/anthropics/claude-code/issues/16114): VSCode 拡張機能では Notification フックが動作しない（ターミナル CLI では正常動作）。2026年1月3日報告
- フックは起動時にスナップショットが取られ、セッション中に設定を変更しても即座には反映されない。変更後は `/hooks` メニューでの確認が必要
- `~/.claude/settings.json` に無効な JSON があるとフックが全て無効化される。JSON の構文エラーに注意
- ntfy.sh 公開サーバー利用時はトピック名がパスワード代わりになるため、推測困難な名前を使用すること
- LINE Notify は 2025年3月31日にサービス終了済み。LINE Messaging API への移行が必要

---

## コミュニティ事例

### Boris Buliga 氏の terminal-notifier 活用例（2026年1月）

macOS で terminal-notifier + yabai を組み合わせ、どのワークスペースのどのプロジェクトが完了したかを通知に含める高度な実装。`-sender com.anthropic.claudefordesktop` で Claude のアイコン表示に対応。

### motlin.com の Pushover + woof スクリプト（2026年）

`woof` コマンドとして、(1) ターミナル出力、(2) macOS 音声合成、(3) Pushover プッシュ通知の3つを同時に実行するスクリプトを実装。環境変数 `NOTIFICATIONS=true` で切り替え可能。

### Justin Searls 氏の Pushcut + Apple Watch 連携

Pushcut Pro を使い、ターミナルがアクティブでない場合やディスプレイスリープ時のみ通知を送信するスマート通知を実現。Apple Watch での受信にも対応。

### GitHub Issue #7590 での ntfy.sh 活用例

コミュニティメンバーが ntfy.sh を使った実装例を共有:

```json
{
  "hooks": {
    "Notification": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "curl -d \"Claude is waiting for input\" ntfy.sh/your-claude-alerts"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "curl -d \"Claude task finished!\" ntfy.sh/your-claude-alerts"
          }
        ]
      }
    ]
  }
}
```

---

## 結論・推奨

### 推奨構成: macOS デスクトップ通知 + ntfy.sh（スマホ通知）

以下の理由から、**osascript（macOS 通知） + ntfy.sh（iPhone 通知）** の組み合わせを推奨する。

1. **ntfy.sh が最もセットアップが簡単**: アカウント登録不要、curl のみで送信可能、5分で開始可能
2. **費用ゼロ**: 公開サーバー利用なら完全無料
3. **macOS 通知との併用**: PC 前にいる時はデスクトップ通知、離席時はスマホ通知の二段構え
4. **プロジェクト既存設定との共存**: ユーザーグローバル設定に配置することで、既存の Stop フック（console.log/fmt.Print 監査）と並列実行される

### 推奨設定例

`~/.claude/settings.json` に追加:

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "PROJECT=$(basename \"$PWD\") && osascript -e \"display notification \\\"$PROJECT: タスク完了\\\" with title \\\"Claude Code\\\" sound name \\\"Hero\\\"\" && curl -s -H 'Title: Claude Code' -d \"$PROJECT: タスク完了\" ntfy.sh/YOUR_SECRET_TOPIC 2>/dev/null || true",
            "async": true
          }
        ],
        "description": "タスク完了時にmacOS通知とiPhoneプッシュ通知を送信"
      }
    ],
    "Notification": [
      {
        "matcher": "permission_prompt|idle_prompt",
        "hooks": [
          {
            "type": "command",
            "command": "osascript -e 'display notification \"入力を待っています\" with title \"Claude Code\" sound name \"Glass\"' && curl -s -H 'Title: Claude Code' -H 'Priority: high' -d '入力待ち' ntfy.sh/YOUR_SECRET_TOPIC 2>/dev/null || true",
            "async": true
          }
        ],
        "description": "入力待ち・権限要求時にmacOS通知とiPhoneプッシュ通知を送信"
      }
    ]
  }
}
```

### セットアップ手順

1. iPhone の App Store で「ntfy」をインストール（無料）
2. アプリを開き、推測困難なトピック名で購読（例: `claude-work-a8x9k2m7`）
3. macOS で動作確認: `curl -d "テスト" ntfy.sh/claude-work-a8x9k2m7`
4. iPhone で通知を受信できたら、上記の設定を `~/.claude/settings.json` に追加
5. `YOUR_SECRET_TOPIC` を自分のトピック名に置換
6. Claude Code を再起動（またはセッション内で `/hooks` からレビュー）

### 代替案

- **Slack/Discord を既に使っている場合**: Webhook に置き換えるだけで追加アプリ不要
- **Apple Watch で確実に受信したい場合**: Pushover ($5) または Pushcut (Pro $2/月) を検討
- **LINE で受信したい場合**: LINE Messaging API でセットアップ（やや手間がかかる）

---

## ソース一覧

- [Hooks reference - Claude Code Docs](https://code.claude.com/docs/en/hooks) - 公式リファレンス（最新）
- [Automate workflows with hooks - Claude Code Docs](https://code.claude.com/docs/en/hooks-guide) - 公式ガイド
- [Claude Code: Getting Phone Notifications When Tasks Complete](https://motlin.com/blog/claude-code-phone-notifications) - Pushover 活用例
- [Claude Code Notifications: Get Alerts When Tasks Finish](https://alexop.dev/posts/claude-code-notification-hooks/) - 通知フック設定例
- [Notify your iPhone or Watch when Claude Code finishes](https://justin.searls.co/posts/notify-your-iphone-or-watch-when-claude-code-finishes/) - Pushcut + Apple Watch
- [Claude Code Notifications That Don't Suck](https://www.d12frosted.io/posts/2026-01-05-claude-code-notifications) - terminal-notifier 活用例
- [Feature Request: Cross-Device Notifications - Issue #7590](https://github.com/anthropics/claude-code/issues/7590) - 公式リポジトリの機能リクエスト
- [ntfy.sh](https://ntfy.sh/) - 無料プッシュ通知サービス公式
- [ntfy documentation](https://docs.ntfy.sh/) - ntfy 公式ドキュメント
- [Pushover API](https://pushover.net/api) - Pushover API リファレンス
- [Pushcut Web API](https://www.pushcut.io/webapi) - Pushcut API リファレンス
- [Sending messages using incoming webhooks - Slack](https://docs.slack.dev/messaging/sending-messages-using-incoming-webhooks/) - Slack Webhook ドキュメント
- [Using Discord as a Notification System with Curl](https://wchesley.dev/posts/discord_curl_notifications/) - Discord Webhook 活用例
- [Pushbullet API](https://docs.pushbullet.com/) - Pushbullet API ドキュメント
- [Send messages - LINE Developers](https://developers.line.biz/en/docs/messaging-api/sending-messages/) - LINE Messaging API
- [LINE Notify サービス終了のお知らせ](https://internet.watch.impress.co.jp/docs/yajiuma/1629950.html) - LINE Notify 廃止情報
- [Webhooks Integrations - IFTTT](https://ifttt.com/maker_webhooks) - IFTTT Webhook

## 関連資料

- このレポートを参照: /discuss, /plan で活用
- 前回の調査レポート: `/Users/user/Ghostrunner/開発/資料/2026-01-28_Claude_Code処理完了通知.md`
- プロジェクトの既存フック設定: `/Users/user/Ghostrunner/.claude/settings.json`
- ユーザーグローバル設定の場所: `~/.claude/settings.json`
