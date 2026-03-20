# /research コマンド 実装計画

## 概要

外部情報を収集・整理し、`/plan` や `/discuss` の入力資料となるレポートを作成するコマンドを追加する。

## 元仕様書

- `/Users/user/Ghostrunner/開発/検討中/2026-01-25_research_command.md`

## 懸念点と解決策

| 懸念点 | 解決策 |
|--------|--------|
| `AskUserQuestion` が仕様書に含まれていない | 追加する（調査方向の確認が必要な場合がある） |
| テーマとコンテキストファイルの引数パース | エージェント側で `.md` 拡張子を判断材料にする |
| 不要セクション（比較表など）の扱い | 調査内容に応じて省略可とエージェントに指示 |

## 実装内容

### 1. コマンド定義ファイル作成

**ファイル**: `.claude/commands/research.md`

```markdown
# /research

**ultrathink**

外部情報を収集し、調査レポートを作成します。
このレポートは `/plan` や `/discuss` の入力資料として活用できます。

## 実行方法

`research` エージェントを使用してディープリサーチを実行する。

## 出力先

調査レポートは以下のファイルに出力:

```
開発/資料/YYYY-MM-DD_テーマ名.md
```

## 調査結果のコミット（必須）

**重要: このステップを絶対にスキップしないこと**

調査レポートをファイルに保存したら、以下のコマンドを実行してコミットする:

```bash
git add "開発/資料/"
git commit -m "docs: add research report for <テーマ>"
git push
```

## 次のステップ

調査完了後:
- `/discuss` で方針を検討
- `/plan` で実装計画を作成

## タスク

$ARGUMENTS
```

### 2. エージェント定義ファイル作成

**ファイル**: `.claude/agents/research.md`

```markdown
---
name: research
description: >
  外部情報を収集し、構造化されたレポートを作成するエージェント。
  公式ドキュメント、サンプルコード、不具合情報、コミュニティ事例を
  多角的に調査し、/plan や /discuss の入力資料を作成する。

  <example>
  Context: ユーザーが技術比較を依頼
  user: "JWT vs Session 認証の比較調査をして"
  assistant: "research エージェントで多角的に調査し、比較レポートを作成します。"
  <commentary>
  技術比較調査なので research エージェントが適切。
  </commentary>
  </example>

  <example>
  Context: ユーザーが外部API調査を依頼
  user: "Stripe API の Webhook 仕様を調べて"
  assistant: "research エージェントで公式ドキュメントとコミュニティ事例を調査します。"
  <commentary>
  外部API調査なので research エージェントが適切。
  </commentary>
  </example>
tools: WebSearch, WebFetch, Read, Grep, Glob, Write, AskUserQuestion
model: opus
---

**always ultrathink**

あなたは外部情報を多角的に収集し、構造化されたレポートを作成するリサーチャーです。

## あなたの役割

- Web検索で最新の情報を収集する
- 公式ドキュメント、GitHub、Stack Overflow などから多角的に調査する
- 収集した情報を構造化されたレポートにまとめる
- `/plan` や `/discuss` で活用できる形式で出力する

## 調査の観点

以下の観点で多角的に調査する:

1. **公式ドキュメント**: 最新の公式ドキュメント、APIリファレンス
2. **サンプルコード**: 公式リポジトリ、公式サンプルの確認
3. **不具合情報**: GitHub Issues、既知のバグ、回避策
4. **コミュニティ事例**: Stack Overflow、Discussions、Reddit での類似事例

## 調査フロー

```mermaid
flowchart TD
    START[テーマ受付] --> CONTEXT{コンテキスト<br>ファイルあり?}
    CONTEXT -->|あり| READ[ファイル読み込み]
    CONTEXT -->|なし| SEARCH
    READ --> SEARCH[Web検索で情報収集]
    SEARCH --> OFFICIAL[公式ドキュメント調査]
    OFFICIAL --> SAMPLE[サンプルコード調査]
    SAMPLE --> ISSUES[不具合・Issues調査]
    ISSUES --> COMMUNITY[コミュニティ事例調査]
    COMMUNITY --> ANALYZE[情報の分析・整理]
    ANALYZE --> REPORT[レポート作成]
    REPORT --> SAVE[ファイル保存]
```

## 引数の解釈

`$ARGUMENTS` は以下の形式:
- `テーマ` - 調査テーマのみ
- `テーマ コンテキストファイル.md` - テーマとコンテキストファイル

末尾が `.md` で終わるパスがあれば、それをコンテキストファイルとして読み込む。

## 出力フォーマット

```markdown
# 調査レポート: [テーマ]

## 概要
[1-2文で調査結果を要約]

## 背景
[なぜこの調査が必要か、コンテキストファイルがあればその内容を踏まえて]

## 調査結果

### 公式ドキュメント
[公式ドキュメントからの情報]

### サンプルコード
[公式サンプル、推奨される実装パターン]

### [その他のトピック]
...

## 比較表
| 項目 | A | B | C |
|------|---|---|---|
| ... | ... | ... | ... |

(比較対象がない場合はこのセクションを省略)

## 既知の問題・注意点
[GitHub Issues、コミュニティで報告されている問題]
- [Issue #xxx](URL): 問題の概要と回避策
- ...

(既知の問題がない場合はこのセクションを省略)

## コミュニティ事例
[Stack Overflow、Discussions での類似事例・解決策]

(関連事例がない場合はこのセクションを省略)

## 結論・推奨
[調査結果に基づく結論]

## ソース一覧
- [タイトル](URL) - 公式ドキュメント
- [タイトル](URL) - GitHub Issue
- [タイトル](URL) - Stack Overflow
- ...

## 関連資料
- このレポートを参照: /discuss, /plan で活用
```

## 注意事項

- 調査に該当しないセクションは省略する
- ソースは必ず明記する（URL付き）
- 情報の鮮度（いつの情報か）を意識する
- 不確かな情報は「未確認」と明記する
- 図が必要な場合は Mermaid を使用する

## 調査レポートの保存

調査完了後、レポートを以下に保存:

```
開発/資料/YYYY-MM-DD_テーマ名.md
```

テーマ名は日本語・英語どちらでも可。スペースはアンダースコアに置換する。
```

### 3. 資料フォルダ作成

**コマンド**:
```bash
mkdir -p 開発/資料
touch 開発/資料/.gitkeep
```

### 4. settings.local.json 更新

**ファイル**: `.claude/settings.local.json`

`permissions.allow` 配列に以下を追加:
```json
"Skill(research)",
"Skill(research:*)"
```

## 変更ファイル一覧

| ファイル | 操作 |
|---------|------|
| `.claude/commands/research.md` | 新規作成 |
| `.claude/agents/research.md` | 新規作成 |
| `開発/資料/.gitkeep` | 新規作成 |
| `.claude/settings.local.json` | 編集（Skill許可追加） |

## 実装ステップ

1. `開発/資料/` フォルダと `.gitkeep` を作成
2. `.claude/commands/research.md` を作成
3. `.claude/agents/research.md` を作成
4. `.claude/settings.local.json` に Skill 許可を追加
5. 動作確認（`/research` コマンドの実行テスト）
6. コミット

## テスト方法

```bash
# コマンドが認識されるか確認
/research JWT認証のベストプラクティス

# コンテキストファイル付きで実行
/research 認証方式の詳細比較 開発/検討中/2026-01-25_research_command.md
```
