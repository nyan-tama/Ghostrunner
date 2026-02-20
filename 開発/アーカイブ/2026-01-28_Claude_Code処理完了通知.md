# 調査レポート: Claude Code (CLI) 処理完了時の通知方法

## 概要

Claude Code にはフック (hooks) 機能が公式に搭載されており、`Stop` イベントと `Notification` イベントを利用して処理完了時にシェルコマンドを実行できる。macOS のネイティブ通知、サウンド再生、さらに ntfy.sh や Pushover を使った iPhone へのプッシュ通知も実現可能。

## 背景

Claude Code で長時間のタスクを実行中にターミナルから離れた際、処理完了を見逃さないための通知の仕組みが必要。現在のプロジェクト設定 (`.claude/settings.json`) には `PreToolUse`、`PostToolUse`、`Stop` フックが既に設定されているが、通知（デスクトップ通知やサウンド）は未設定。

## 調査結果

### 1. Claude Code Hooks 機能（公式）

Claude Code には 2025年6月に導入されたフック機能があり、設定ファイルの JSON で通知用コマンドを定義できる。

#### 利用可能なフックイベント一覧

| フックイベント | 発火タイミング |
|---|---|
| `SessionStart` | セッション開始・再開時 |
| `UserPromptSubmit` | ユーザーがプロンプトを送信した時 |
| `PreToolUse` | ツール実行前 |
| `PermissionRequest` | 権限確認ダイアログ表示時 |
| `PostToolUse` | ツール正常完了後 |
| `PostToolUseFailure` | ツール失敗後 |
| `SubagentStart` | サブエージェント起動時 |
| `SubagentStop` | サブエージェント完了時 |
| **`Stop`** | **Claude が応答を完了した時（通知に最適）** |
| `PreCompact` | コンテキスト圧縮前 |
| `SessionEnd` | セッション終了時 |
| **`Notification`** | **Claude Code が通知を送信する時（権限要求・入力待ち）** |

#### 通知に関連する主要イベント

- **`Stop`**: Claude がタスクを完了した時に発火。ユーザー割り込みによる停止では発火しない。
- **`Notification`**: Claude Code が通知を送信する時に発火。matcher でフィルタ可能。
  - `permission_prompt`: 権限要求時
  - `idle_prompt`: 入力待ち（60秒以上のアイドル後）
  - `auth_success`: 認証成功時
  - `elicitation_dialog`: MCP ツールの入力要求時

#### 設定ファイルの場所

| ファイル | 用途 | Git管理 |
|---|---|---|
| `~/.claude/settings.json` | ユーザーグローバル設定 | 対象外 |
| `.claude/settings.json` | プロジェクト設定（チーム共有） | 対象 |
| `.claude/settings.local.json` | ローカルプロジェクト設定 | 対象外 |

通知設定はプラットフォーム固有のため、**`~/.claude/settings.json`（ユーザーグローバル）** に配置するのが推奨。

### 2. 通知方法の選択肢

#### 方法A: osascript（macOS 標準、追加インストール不要）

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "osascript -e 'display notification \"Claude Code: タスク完了\" with title \"Claude Code\" sound name \"Hero\"'"
          }
        ]
      }
    ],
    "Notification": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "osascript -e 'display notification \"Claude Code: 入力待ち\" with title \"Claude Code\" sound name \"Glass\"'"
          }
        ]
      }
    ]
  }
}
```

利用可能なシステムサウンド（`/System/Library/Sounds/` 内）:
Basso, Blow, Bottle, Frog, Funk, Glass, Hero, Morse, Ping, Pop, Purr, Sosumi, Submarine, Tink

**利点**: 追加インストール不要、macOS 標準機能
**欠点**: Script Editor のアイコンが表示される（カスタマイズ不可）

#### 方法B: terminal-notifier（カスタムアイコン対応）

```bash
# インストール
brew install terminal-notifier
```

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "terminal-notifier -title 'Claude Code' -message 'タスク完了' -sound Hero -sender com.anthropic.claudefordesktop"
          }
        ]
      }
    ],
    "Notification": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "terminal-notifier -title 'Claude Code' -message '入力待ち' -sound Glass -sender com.anthropic.claudefordesktop"
          }
        ]
      }
    ]
  }
}
```

**利点**: Claude Desktop のアイコンで通知表示可能（`-sender` でバンドル識別子を指定）、カスタマイズ性が高い
**欠点**: Homebrew でのインストールが必要

#### 方法C: afplay（サウンドのみ）

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "afplay /System/Library/Sounds/Hero.aiff &"
          }
        ]
      }
    ]
  }
}
```

**利点**: シンプル、確実にサウンドが鳴る
**欠点**: デスクトップ通知なし、サウンドのみ

#### 方法D: claude-code-notification（専用ツール）

```bash
# インストール
brew install wyattjoh/stable/claude-code-notification
```

Rust 製の専用ツール。クロスプラットフォーム対応、サウンドの並列再生、高速起動。

**利点**: Claude Code 専用に最適化、Homebrew で簡単インストール
**欠点**: 外部依存

### 3. macOS から iPhone に通知を送る方法

#### ntfy.sh（無料・オープンソース）

```bash
# 送信（curl のみで可能、インストール不要）
curl -d "Claude Code: タスク完了" ntfy.sh/your-secret-topic-name

# 優先度とタイトル付き
curl \
  -H "Title: Claude Code" \
  -H "Priority: default" \
  -d "タスクが完了しました" \
  ntfy.sh/your-secret-topic-name
```

Claude Code hooks での設定例:
```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "curl -s -H 'Title: Claude Code' -d 'タスク完了' ntfy.sh/your-secret-topic-name"
          }
        ]
      }
    ]
  }
}
```

**セットアップ手順**:
1. iPhone に ntfy アプリをインストール（App Store で無料）
2. アプリ内で推測困難なトピック名を登録して購読
3. 上記のフック設定を追加

**利点**: 完全無料、サーバー不要（公式サーバー利用時）、curl だけで送信可能、セルフホスト可能
**欠点**: トピック名がパスワード代わり（推測困難な名前を使うこと）、公開サーバーのためプライバシー面に注意

#### Pushover（有料・高機能）

```bash
# 送信
curl -s \
  --form-string "token=YOUR_APP_TOKEN" \
  --form-string "user=YOUR_USER_KEY" \
  --form-string "title=Claude Code" \
  --form-string "message=タスク完了" \
  https://api.pushover.net/1/messages.json
```

Claude Code hooks での設定例:
```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "curl -s --form-string 'token=YOUR_APP_TOKEN' --form-string 'user=YOUR_USER_KEY' --form-string 'title=Claude Code' --form-string 'message=タスク完了' https://api.pushover.net/1/messages.json"
          }
        ]
      }
    ]
  }
}
```

**セットアップ手順**:
1. pushover.net でアカウント作成、User Key を取得
2. アプリケーション登録で API Token を取得
3. iPhone に Pushover アプリをインストール（30日無料、以降ワンタイム購入 $5）
4. フック設定にトークンを追加

**利点**: 安定性が高い、Apple Watch 対応、優先度・デバイス指定など高機能、iOS Shortcuts 連携可能
**欠点**: 有料（ワンタイム $5）、APIトークン管理が必要

### 4. 現在のプロジェクト設定の状況

プロジェクトの `.claude/settings.json` には既に以下のフックが設定済み:

- **PreToolUse**: tmux外での開発サーバー実行ブロック、git push 前の確認、不要な .md ファイル作成ブロック
- **PostToolUse**: PR作成後の URL 表示、Prettier/gofmt 自動フォーマット、TypeScript/go vet チェック、console.log/fmt.Print 警告
- **Stop**: セッション終了前の console.log/fmt.Print 最終監査

`Notification` イベントのフックは**未設定**。
`Stop` イベントには監査フックのみで、**通知は未設定**。

## 比較表

| 項目 | osascript | terminal-notifier | afplay | ntfy.sh | Pushover |
|---|---|---|---|---|---|
| 通知先 | macOS | macOS | macOS (音のみ) | iPhone/macOS | iPhone/macOS |
| 追加インストール | 不要 | brew必要 | 不要 | アプリのみ | アプリ+登録 |
| 費用 | 無料 | 無料 | 無料 | 無料 | $5 (ワンタイム) |
| カスタムアイコン | 不可 | 可能 | N/A | 可能 | 可能 |
| サウンド | 可能 | 可能 | 可能 | iOSサウンド | iOSサウンド |
| 外出先で受信 | 不可 | 不可 | 不可 | **可能** | **可能** |
| セットアップ難易度 | 低 | 低 | 最低 | 低 | 中 |

## 既知の問題・注意点

- [Issue #16114](https://github.com/anthropics/claude-code/issues/16114): VSCode 拡張機能では Notification フックが動作しない（ターミナル CLI では動作する）。2026年1月3日報告。
- claude-code v1.0.95 以降、`~/.claude/settings.json` に無効な JSON があるとフックが全て無効化される。JSON の構文エラーに注意。
- フックは起動時にスナップショットが取られ、セッション中に外部から設定を変更しても即座には反映されない。`/hooks` メニューでのレビューが必要。
- 複数のフックが同じイベントにマッチする場合、全て並列実行される。

## 推奨設定

### 推奨案: macOS 通知 + ntfy.sh（iPhone）の組み合わせ

`~/.claude/settings.json`（ユーザーグローバル設定）に以下を追加:

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "osascript -e 'display notification \"タスクが完了しました\" with title \"Claude Code\" sound name \"Hero\"' && curl -s -H 'Title: Claude Code' -d 'タスク完了' ntfy.sh/YOUR_SECRET_TOPIC 2>/dev/null || true"
          }
        ],
        "description": "タスク完了時にmacOS通知とiPhoneプッシュ通知を送信"
      }
    ],
    "Notification": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "osascript -e 'display notification \"Claudeが入力を待っています\" with title \"Claude Code\" sound name \"Glass\"'"
          }
        ],
        "description": "入力待ち・権限要求時にmacOS通知を送信"
      }
    ]
  }
}
```

**この構成の理由**:
- `Stop` イベントでタスク完了時にデスクトップ通知 + iPhone 通知（離席時対応）
- `Notification` イベントで入力待ち時はデスクトップ通知のみ（離席時は iPhone 通知も追加可能）
- ntfy.sh は無料でサーバー不要、curl のみで送信可能
- `|| true` で ntfy.sh への送信失敗時もフック全体が失敗しないようにする
- 既存のプロジェクト `.claude/settings.json` の `Stop` フック（console.log/fmt.Print 監査）とは別に、ユーザーグローバル設定に配置することで両方が並列実行される

### ntfy.sh セットアップ手順

1. iPhone の App Store で "ntfy" をインストール
2. アプリを開き、推測困難なトピック名で購読（例: `claude-notify-a8x9k2m7`）
3. macOS で動作確認: `curl -d "テスト通知" ntfy.sh/claude-notify-a8x9k2m7`
4. iPhone で通知を受信できたら、上記の設定を `~/.claude/settings.json` に追加
5. Claude Code を再起動

## ソース一覧

- [Hooks reference - Claude Code Docs](https://code.claude.com/docs/en/hooks) - 公式リファレンス
- [Get started with Claude Code hooks](https://code.claude.com/docs/en/hooks-guide) - 公式ガイド
- [Claude Code Notifications That Don't Suck - Boris Buliga](https://www.d12frosted.io/posts/2026-01-05-claude-code-notifications) - terminal-notifier活用例（2026年1月）
- [Get Notified When Claude Code Finishes With Hooks - alexop.dev](https://alexop.dev/posts/claude-code-notification-hooks/) - 通知フック設定例
- [Claude Code hooks for simple macOS notifications - Stanislav Khromov](https://khromov.se/claude-code-hooks-for-simple-macos-notifications/) - シンプルな通知設定
- [Claude Code Hooks: Automating macOS Notifications - Masato Naka](https://nakamasato.medium.com/claude-code-hooks-automating-macos-notifications-for-task-completion-42d200e751cc) - macOS通知の自動化
- [Automate Your AI Workflows with Claude Code Hooks - GitButler Blog](https://blog.gitbutler.com/automate-your-ai-workflows-with-claude-code-hooks/) - フック活用全般
- [claude-code-notification - GitHub (wyattjoh)](https://github.com/wyattjoh/claude-code-notification) - Rust製通知ツール
- [ntfy.sh](https://ntfy.sh/) - 無料プッシュ通知サービス
- [Pushover API](https://pushover.net/api) - Pushover API リファレンス
- [Using terminal-notifier in Claude Code - Andrea Grandi](https://www.andreagrandi.it/posts/using-terminal-notifier-claude-code-custom-notifications/) - terminal-notifier設定
- [Notification hooks not working in VSCode - Issue #16114](https://github.com/anthropics/claude-code/issues/16114) - 既知の不具合

## 関連資料

- このレポートを参照: /discuss, /plan で活用
- プロジェクトの既存フック設定: `/Users/user/Ghostrunner/.claude/settings.json`
- ユーザーグローバル設定の場所: `~/.claude/settings.json`
