# commands から skills への全面移行

作成日: 2026-03-20
ステータス: 方針決定済み
決定: 案A（全面移行）

## 背景

Ghostrunnerは `.claude/commands/` に13個のコマンドを配置しているが、Claude Codeは現在 `.claude/skills/` を推奨している。公式ドキュメントでは「Custom commands have been merged into skills」と明記されており、commands は旧形式として位置づけられている。

## skills にしかない機能

- 複数ファイル（テンプレート、スクリプト同梱）
- 自動トリガー（descriptionベースでClaudeが自動活用）
- `allowed-tools` でツール制限
- `context: fork` で隔離実行
- `disable-model-invocation` で自動実行禁止
- 動的コンテキスト注入

## 検討した案

### 案A: 全面移行（採用）

全13コマンドを `.claude/skills/` に移行。

```
.claude/skills/
├── init/
│   ├── SKILL.md
│   └── templates/
├── plan/
│   ├── SKILL.md
│   └── template.md
├── discuss/
│   └── SKILL.md
...
```

メリット:
- 最新の推奨に沿う
- supporting files活用でテンプレート同梱可能
- 自動トリガー対応
- `templates/` ディレクトリをskills内に統合し構造をシンプルに

デメリット:
- 既存ユーザーへの影響（ただしcommands自体は動き続ける）

### 案B: 段階的移行（不採用）

メリットの大きいものだけ先に移行。commandsとskillsが混在して分かりにくくなるため不採用。

### 案C: 現状維持（不採用）

新機能が使えず、旧形式のままになるため不採用。

## 移行対象（13コマンド）

1. init
2. plan
3. discuss
4. fullstack
5. go
6. nextjs
7. fix
8. hotfix
9. stage
10. release
11. devtools
12. research
13. destroy

## 移行方針

- `commands/foo.md` を `skills/foo/SKILL.md` に移動
- フロントマター（name, description等）を追加
- supporting filesが有効なもの（init, plan等）はテンプレートを同梱
- `templates/` ディレクトリの内容をskills内に統合するか検討
- 旧 `.claude/commands/` は移行完了後に削除

## 次のステップ

`/plan` で実装計画を作成
