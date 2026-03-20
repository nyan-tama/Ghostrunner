# 調査レポート: Claude Code Remote Control（スマホからの操作機能）

## 概要

2026年2月25日、AnthropicはClaude Codeの新機能「Remote Control」をリリースした。これはローカルマシンで動作するClaude Codeセッションを、スマートフォン（iOS/Android）やタブレット、別PCのブラウザからリモート操作できる機能である。Research Preview段階だが、全プラン（Pro/Max/Team/Enterprise）で利用可能。

## 背景

Claude Codeはターミナルベースのコーディングエージェントであり、従来はPCのターミナルの前にいなければ操作できなかった。長時間のタスク実行中にPCから離れたい（散歩、移動、休憩など）というニーズに応えるため、Remote Control機能が開発された。

## 調査結果

### 公式ドキュメント

#### 機能概要

Remote Controlは `claude.ai/code` またはClaudeモバイルアプリ（iOS/Android）を、マシン上で実行中のClaude Codeセッションに接続する機能。主な特徴は以下の通り。

- **ローカル実行を維持**: Claudeはローカルマシンで実行され続け、何もクラウドに移動しない
- **ローカル環境全体をリモート利用**: ファイルシステム、MCPサーバー、ツール、プロジェクト設定が全て利用可能
- **複数デバイス間の同期**: ターミナル、ブラウザ、スマートフォンから相互にメッセージ送信可能
- **自動再接続**: ラップトップがスリープやネットワーク切断になっても、復帰時に自動再接続

#### 要件

| 項目 | 内容 |
|------|------|
| 対応プラン | Pro, Max, Team, Enterprise |
| 必要バージョン | Claude Code v2.1.51 以降 |
| 認証 | claude.ai経由でサインイン済みであること（`/login`） |
| ワークスペース信頼 | プロジェクトディレクトリで少なくとも1回 `claude` を実行し信頼ダイアログを承認 |
| APIキー | サポートされない（claude.aiアカウントでのログインが必須） |

#### セットアップ手順

**方法1: 新規セッションとして開始**

```bash
claude remote-control
```

ターミナルにセッションURLが表示される。スペースバーを押すとQRコードが表示され、スマートフォンでスキャンして接続できる。

オプションフラグ:
- `--name "My Project"`: セッションにカスタム名を設定
- `--verbose`: 詳細ログを表示
- `--sandbox` / `--no-sandbox`: サンドボックスの有効/無効（デフォルトはオフ）

**方法2: 既存セッションから切り替え**

Claude Codeセッション内で以下を実行:

```
/remote-control
```

省略形: `/rc`

**方法3: 全セッションで自動有効化**

`/config` で「Enable Remote Control for all sessions」を `true` に設定。
または `~/.claude.json` に以下を追加:

```json
{
  "remoteControlAtStartup": true
}
```

#### 接続方法（スマートフォン側）

1. **QRコードをスキャン**: ターミナルに表示されるQRコードをスマホのカメラでスキャン
2. **セッションURLを開く**: ブラウザで `claude.ai/code` にアクセスしセッションを選択
3. **Claudeアプリから接続**: iOS/Androidアプリのセッションリストから選択（緑色のステータスドットが目印）

#### セキュリティモデル

- ローカルClaude Codeは**アウトバウンドHTTPSリクエストのみ**を発行
- マシン上のインバウンドポートは開かない
- Remote Control開始時にAnthropic APIに登録し、作業をポーリング
- 全トラフィックはTLS経由でAnthropic APIを通過
- 複数の短命な認証情報を使用（各認証情報は単一目的にスコープ、独立して有効期限切れ）

### サンプルコード / 実装パターン

#### 基本的な使い方フロー

```
1. PC のターミナルで作業開始
   $ cd /path/to/project
   $ claude

2. タスクを指示して実行開始
   > 新しいAPIエンドポイントを追加して

3. 外出する前にリモートコントロールを有効化
   > /rc

4. QR コードをスマートフォンでスキャン

5. スマートフォンから進捗確認・承認操作・追加指示が可能
```

#### スマホ通知の設定（Bark連携）

Remote Controlと組み合わせて、Claude Codeの承認要求をスマホにプッシュ通知する設定が可能。Hooksを使って`Bark`（iOS向け）などのプッシュ通知サービスと連携できる。

```json
{
  "hooks": {
    "Notification": [{
      "matcher": "permission_prompt|idle_prompt|elicitation_dialog",
      "hooks": [{"type": "command", "command": "~/.claude/hooks/bark-notify.sh"}]
    }]
  }
}
```

### Remote Control vs Web上のClaude Code

| 項目 | Remote Control | Web上のClaude Code |
|------|---------------|-------------------|
| 実行場所 | ローカルマシン | Anthropicのクラウド |
| ローカルファイル | アクセス可能 | 不可（クラウド環境） |
| MCPサーバー | 利用可能 | 不可 |
| プロジェクト設定 | 利用可能 | リポジトリクローンが必要 |
| セットアップ | ターミナルで起動が必要 | 不要 |
| 並列実行 | 1セッション | 複数タスク並列可能 |
| 適したユースケース | ローカル作業の継続 | 新規タスク、未クローンのリポジトリ |

### SSH + Tailscale による代替アプローチ

Remote Control機能が登場する前から、以下のようなSSHベースの方法もコミュニティで使われている:

1. デスクトップPCでClaude Codeを実行
2. Tailscaleでプライベートネットワークを構築
3. スマホにTermux（Android）またはBlink Shell（iOS）をインストール
4. SSHでデスクトップに接続、tmuxでセッションを維持

Blink ShellのMOSH対応により、Wi-Fiからセルラーへの切り替え時もセッションが維持される利点がある。ただし、公式のRemote Control機能の登場により、この方法の必要性は大幅に減った。

## 既知の問題・注意点

- **1セッション1接続**: 各Claude Codeインスタンスは一度に1つのリモート接続のみサポート
- **ターミナルを閉じると終了**: Remote Controlはローカルプロセスとして実行されるため、ターミナルを閉じるかclaudeプロセスを停止するとセッション終了
- **ネットワークタイムアウト**: マシンが起動中でも約10分以上ネットワーク到達不能な場合、セッションはタイムアウトしてプロセスが終了する
- **APIキー非対応**: claude.aiアカウントでのログインが必須。APIキーでの利用は不可

## コミュニティ事例

- Zennやクラスメソッド（DevelopersIO）で日本語の解説記事が複数公開されている
- Bark（iOSプッシュ通知アプリ）とHooksを組み合わせて、承認要求をスマホに通知する運用事例がある
- 「犬の散歩をしながらコーディング作業を監視できる」というユースケースが紹介されている
- TermuxでAndroid上に直接Claude Codeをインストールして実行する方法も報告されている（Remote Controlとは別アプローチ）

## 結論・推奨

Claude Code Remote Controlは、ターミナルベースのコーディングエージェントをモバイルに拡張する実用的な機能である。セットアップは非常に簡単（`/rc` コマンド1つ + QRコードスキャン）で、セキュリティモデルも堅実（インバウンドポート不要、TLS暗号化通信のみ）。

特に以下のシナリオで有用:
- 長時間のリファクタリングやビルドタスク実行中にPCから離れたい場合
- 承認が必要な操作をスマホから承認したい場合
- 移動中に進捗を確認し、追加の指示を出したい場合

全プランで利用可能（v2.1.51以降）。APIキーではなくclaude.aiアカウントでのログインが必要な点に注意。

## ソース一覧

- [任意のデバイスからローカルセッションを続行する Remote Control - Claude Code公式ドキュメント](https://code.claude.com/docs/ja/remote-control) - 公式ドキュメント（日本語）
- [Claude Code の Remote Control でスマホからローカルマシンの作業を継続可能に | DevelopersIO](https://dev.classmethod.jp/articles/claude-coderemotecontrol-enables-you-to-work-on-your-local-machine-from-your-smartphone/) - 解説記事
- [Claude Code のリモートコントロールとスマホ通知の始め方 - Zenn](https://zenn.dev/schroneko/articles/claude-code-remote-control-and-mobile-notification) - Hooks+Bark通知設定
- [Claude Code Remote Controlが登場 - Zenn (Ubie)](https://zenn.dev/ubie_dev/articles/claude-code-remote-control-intro) - 解説記事
- [Anthropic just released a mobile version of Claude Code called Remote Control | VentureBeat](https://venturebeat.com/orchestration/anthropic-just-released-a-mobile-version-of-claude-code-called-remote) - ニュース記事
- [Anthropic's Remote Control Brings Claude Code to Mobile Devices - WinBuzzer](https://winbuzzer.com/2026/02/28/anthropic-remote-control-claude-code-mobile-access-xcxwbn/) - ニュース記事
- [Claude Code on Your Phone - Builder.io](https://www.builder.io/blog/claude-code-mobile-phone) - 解説記事
- [Claude Code Remote Control Keeps Your Agent Local and Puts it in Your Pocket - DevOps.com](https://devops.com/claude-code-remote-control-keeps-your-agent-local-and-puts-it-in-your-pocket/) - 解説記事
- [「Claude Code」をスマホから操れる 遠隔操作機能「Remote Control」公開 - ITmedia](https://www.itmedia.co.jp/aiplus/articles/2602/25/news089.html) - ニュース記事

## 関連資料

- このレポートを参照: /discuss, /plan で活用
