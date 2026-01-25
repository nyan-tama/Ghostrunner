# /research コマンド フロントエンド・バックエンド追加 実装計画

## 概要

`/research` コマンドは Claude Code CLI 側（`.claude/commands/research.md` と `.claude/agents/research.md`）では既に実装済みだが、Web UI（フロントエンド）とバックエンドのホワイトリストに追加されていないため、Web UI から実行できない状態。

## 現状分析

### 実装済み
- `.claude/commands/research.md` - コマンド定義
- `.claude/agents/research.md` - research エージェント定義
- `開発/資料/` フォルダ構造

### 未実装
1. **バックエンド**: `AllowedCommands` に `"research"` が含まれていない
2. **フロントエンド**: `web/index.html` の select 要素に `/research` オプションがない

## 必要な変更

### バックエンド変更

#### ファイル: `internal/service/types.go`

**変更内容**: `AllowedCommands` マップに `"research": true` を追加

```go
// 変更前（行 5-11）
var AllowedCommands = map[string]bool{
	"plan":      true,
	"fullstack": true,
	"go":        true,
	"nextjs":    true,
	"discuss":   true,
}

// 変更後
var AllowedCommands = map[string]bool{
	"plan":      true,
	"fullstack": true,
	"go":        true,
	"nextjs":    true,
	"discuss":   true,
	"research":  true,
}
```

**理由**:
- `CommandHandler` は `AllowedCommands[req.Command]` でホワイトリスト検証を行う
- 追加しないと「許可されていないコマンドです」エラーが返る

### フロントエンド変更

#### ファイル: `web/index.html`

**変更内容**: select 要素に `/research` オプションを追加

```html
<!-- 変更前（行 370-376） -->
<select id="command" name="command">
    <option value="plan">/plan - 実装計画作成</option>
    <option value="discuss">/discuss - アイデア深掘り</option>
    <option value="fullstack">/fullstack - フルスタック実装</option>
    <option value="go">/go - Go バックエンド実装</option>
    <option value="nextjs">/nextjs - Next.js フロントエンド実装</option>
</select>

<!-- 変更後 -->
<select id="command" name="command">
    <option value="plan">/plan - 実装計画作成</option>
    <option value="research">/research - 外部情報調査</option>
    <option value="discuss">/discuss - アイデア深掘り</option>
    <option value="fullstack">/fullstack - フルスタック実装</option>
    <option value="go">/go - Go バックエンド実装</option>
    <option value="nextjs">/nextjs - Next.js フロントエンド実装</option>
</select>
```

**配置位置の理由**:
- `/plan` の直後に配置（計画 → 調査 → 議論 → 実装 の流れ）
- `/research` は実装前の調査フェーズで使うため、実装系コマンドより前が適切

## 実装ステップ

| ステップ | 対象 | 内容 |
|---------|------|------|
| 1 | バックエンド | `internal/service/types.go` の `AllowedCommands` に `"research": true` を追加 |
| 2 | フロントエンド | `web/index.html` の select 要素に `/research` オプションを追加 |
| 3 | 検証 | Web UI から `/research` コマンドを実行して動作確認 |

## 影響範囲

- **変更ファイル数**: 2ファイル
- **変更行数**: 各ファイル1行追加
- **既存機能への影響**: なし（追加のみ）

## 技術的懸念点

**懸念点なし**

既存の汎用コマンドフレームワークが完成しており、ホワイトリストへの追加とUI追加のみで動作する設計になっている。

## セルフチェック

- [x] 用語統一: コマンド名 `research`、表示名「外部情報調査」
- [x] 整合性: バックエンドとフロントエンドで同じコマンド名 `research` を使用
- [x] 網羅性: 変更ファイルは2つのみで漏れなし
- [x] 順序: バックエンド → フロントエンド（依存関係なし、並行可能）
