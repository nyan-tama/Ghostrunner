# commands から skills への全面移行 実装計画

## 概要

`.claude/commands/*.md` の13コマンドを `.claude/skills/*/SKILL.md` に移行する。
Claude Code の推奨形式に合わせ、フロントマター追加と関連ファイルの参照更新を行う。

## 懸念点と決定事項

| 懸念点 | 決定 |
|--------|------|
| `templates/` を skills 内に統合するか | プロジェクトルートに維持。テンプレートが大きすぎるため |
| `devtools.md` の `${CLAUDE_PLUGIN_ROOT}` | `.devtools` シンボリックリンク参照に修正 |
| `/init` の Step 7（.claude/資産コピー） | `commands/` → `skills/` にコピー先を変更 |
| CLAUDE.md の参照 | `.claude/commands/` → `.claude/skills/` に更新 |

## 移行対象一覧

全13コマンドを以下の形式で移行する:

| 元ファイル | 移行先 | disable-model-invocation |
|-----------|--------|--------------------------|
| `commands/init.md` | `skills/init/SKILL.md` | true |
| `commands/plan.md` | `skills/plan/SKILL.md` | true |
| `commands/discuss.md` | `skills/discuss/SKILL.md` | true |
| `commands/fullstack.md` | `skills/fullstack/SKILL.md` | true |
| `commands/go.md` | `skills/go/SKILL.md` | true |
| `commands/nextjs.md` | `skills/nextjs/SKILL.md` | true |
| `commands/fix.md` | `skills/fix/SKILL.md` | true |
| `commands/hotfix.md` | `skills/hotfix/SKILL.md` | true |
| `commands/stage.md` | `skills/stage/SKILL.md` | true |
| `commands/release.md` | `skills/release/SKILL.md` | true |
| `commands/devtools.md` | `skills/devtools/SKILL.md` | true |
| `commands/research.md` | `skills/research/SKILL.md` | true |
| `commands/destroy.md` | `skills/destroy/SKILL.md` | true |

全コマンドがタスク型（ユーザーが明示的に呼び出す）のため、全て `disable-model-invocation: true` とする。

## SKILL.md フロントマター形式

```yaml
---
name: <skill名>
description: <1行の説明。Claude が自動トリガー判定に使用>
disable-model-invocation: true
---
```

## 実装ステップ

### Step 1: skills ディレクトリ構造の作成

13個のスキルディレクトリを作成する:

```bash
mkdir -p .claude/skills/{init,plan,discuss,fullstack,go,nextjs,fix,hotfix,stage,release,devtools,research,destroy}
```

### Step 2: 各コマンドの SKILL.md 化

各 `commands/*.md` に対して:

1. フロントマター（name, description, disable-model-invocation）を先頭に追加
2. 内容はそのまま維持
3. `skills/<name>/SKILL.md` として保存

各スキルの description:

| スキル | description |
|--------|-------------|
| init | プロジェクトスターター。新規プロジェクトを対話的に生成する |
| plan | 仕様書を分析し実装計画を作成する |
| discuss | アイデアや課題を対話で深掘りし複数案を提示する |
| fullstack | バックエンドとフロントエンドの両方を実装する |
| go | Goバックエンドの実装サイクルを実行する |
| nextjs | Next.jsフロントエンドの実装サイクルを実行する |
| fix | デプロイ後の修正を判定し適切なフローで実行する |
| hotfix | 本番環境の緊急修正をstaging経由でリリースする |
| stage | featブランチをstagingにsquash mergeしデプロイする |
| release | stagingをmainにマージし本番リリースする |
| devtools | 開発進捗ビューアを起動する |
| research | 外部情報を収集し調査レポートを作成する |
| destroy | プロジェクトのリソースを検出し選択的に削除する |

### Step 3: devtools スキルの修正

`${CLAUDE_PLUGIN_ROOT}` を `.devtools` シンボリックリンク参照に変更:

変更前:
```bash
cd ${CLAUDE_PLUGIN_ROOT}/devtools && PROJECT_DIR=$(pwd) npm run dev -- -p 3001
```

変更後:
```bash
cd .devtools && PROJECT_DIR=$(pwd) npm run dev -- -p 3001
```

### Step 4: init スキルの修正

Step 7.1（.claude/資産コピー）のコマンドを更新:

変更前:
```bash
mkdir -p /Users/user/<プロジェクト名>/.claude/agents /Users/user/<プロジェクト名>/.claude/commands
cp /Users/user/Ghostrunner/.claude/agents/*.md /Users/user/<プロジェクト名>/.claude/agents/
cp /Users/user/Ghostrunner/.claude/commands/*.md /Users/user/<プロジェクト名>/.claude/commands/
```

変更後:
```bash
mkdir -p /Users/user/<プロジェクト名>/.claude/agents
cp /Users/user/Ghostrunner/.claude/agents/*.md /Users/user/<プロジェクト名>/.claude/agents/
cp -r /Users/user/Ghostrunner/.claude/skills/ /Users/user/<プロジェクト名>/.claude/skills/
```

### Step 5: CLAUDE.md の更新

以下の箇所を修正:

1. プロジェクト概要の構成リスト:
   - ``.claude/commands/` - 13コマンド`` → ``.claude/skills/` - 13スキル``

2. ファイル構造の図:
   - `commands/` → `skills/`
   - コメント更新: `コマンド定義（.md）` → `スキル定義（SKILL.md）`

### Step 6: 旧 commands ディレクトリの削除

```bash
rm -rf .claude/commands/
```

## 変更ファイル一覧

### 新規作成（13ファイル）

| ファイル |
|---------|
| `.claude/skills/init/SKILL.md` |
| `.claude/skills/plan/SKILL.md` |
| `.claude/skills/discuss/SKILL.md` |
| `.claude/skills/fullstack/SKILL.md` |
| `.claude/skills/go/SKILL.md` |
| `.claude/skills/nextjs/SKILL.md` |
| `.claude/skills/fix/SKILL.md` |
| `.claude/skills/hotfix/SKILL.md` |
| `.claude/skills/stage/SKILL.md` |
| `.claude/skills/release/SKILL.md` |
| `.claude/skills/devtools/SKILL.md` |
| `.claude/skills/research/SKILL.md` |
| `.claude/skills/destroy/SKILL.md` |

### 修正（1ファイル）

| ファイル | 変更内容 |
|---------|---------|
| `.claude/CLAUDE.md` | commands → skills への参照更新 |

### 削除（13ファイル）

| ファイル |
|---------|
| `.claude/commands/init.md` |
| `.claude/commands/plan.md` |
| `.claude/commands/discuss.md` |
| `.claude/commands/fullstack.md` |
| `.claude/commands/go.md` |
| `.claude/commands/nextjs.md` |
| `.claude/commands/fix.md` |
| `.claude/commands/hotfix.md` |
| `.claude/commands/stage.md` |
| `.claude/commands/release.md` |
| `.claude/commands/devtools.md` |
| `.claude/commands/research.md` |
| `.claude/commands/destroy.md` |

## 実装順序

1. Step 1: ディレクトリ作成
2. Step 2: 全13コマンドの SKILL.md 化（フロントマター追加 + コピー）
3. Step 3: devtools の `${CLAUDE_PLUGIN_ROOT}` 修正
4. Step 4: init の資産コピーコマンド修正
5. Step 5: CLAUDE.md 更新
6. Step 6: 旧 commands 削除
7. コミット
