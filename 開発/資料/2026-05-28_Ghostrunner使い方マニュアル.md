# Ghostrunner 使い方マニュアル

作成日: 2026-05-28

Ghostrunner を「開発フレームワーク」としても「統括ハブ」としても日常運用するための包括マニュアル。
セットアップ済みであることを前提とする(セットアップ手順は [../../概要.md](../../概要.md))。

---

## 目次

1. [Ghostrunner の二つの顔](#1-ghostrunner-の二つの顔)
2. [日常ワークフロー(開発)](#2-日常ワークフロー開発)
3. [統括ハブとしての運用](#3-統括ハブとしての運用)
4. [スマホ・ウェアラブルから操作する](#4-スマホウェアラブルから操作する)
5. [音声入出力(VOICEVOX)](#5-音声入出力voicevox)
6. [メモリリーク再発防止 三本柱](#6-メモリリーク再発防止-三本柱)
7. [コマンドリファレンス](#7-コマンドリファレンス)
8. [補助Tips](#8-補助tips)
9. [トラブルシュート](#9-トラブルシュート)
10. [関連資料](#10-関連資料)

---

## 1. Ghostrunner の二つの顔

Ghostrunner は役割が二つある。

### 1.1 開発フレームワーク

- `/init` で新規プロジェクトを対話生成
- `.claude/agents/`(30 体) と `.claude/skills/`(15 体) が組み込まれ、プロジェクト作成と同時にコピーされる
- 生成プロジェクトは Go + Gin / Next.js + React + TS + Tailwind 構成。PostgreSQL / Cloudflare R2 / Redis / Swift macOS のテンプレも切替可能

### 1.2 統括ハブ

複数プロジェクトを横断管理する司令塔としても動く。
ルール(`.claude/CLAUDE.md`)で **「把握＝読み取り(自動)」/ 「操作＝明示指示があってから」** が線引きされている。

| 機能 | 把握(自動・読み取り) | 操作(明示指示) |
|---|---|---|
| 開発 | `chief-director` が `開発/` 配下のカンバンを集約報告 | `/bulk-coding` で実装待ちタスクを一括 `gr-run` 起動 |
| 運用 | `chief-director` が `運用/状態/*.json` を読み、stale / blocked / 連続エラーを報告 | (運用 Phase2、未実装。Chrome 再起動・再点火はユーザー指示で別途) |

把握系の呼び出しトリガー語例: 「全プロジェクトの状況は？」「今日どう？」
操作系の呼び出しトリガー語例: 「一括 coding して」「実装待ちを一括で開始」

---

## 2. 日常ワークフロー(開発)

VS Code ターミナルから Claude Code を起動して、スキルを呼ぶのが基本。
スマホ・グラスからは even-terminal 経由で同じスキルを呼べる(後述)。

### 2.1 新規プロジェクト作成

```bash
/init
```

対話で「何を作りたいか」を答えると、テンプレ選定 → ディレクトリ作成 → 初期コミットまで自動。
ブラウザ操作派は http://localhost:3333/new から GUI で同じことができる。

### 2.2 機能追加サイクル(王道)

```
/discuss  →  /plan  →  /coding  →  /stage  →  /release
```

| ステップ | 役割 |
|---|---|
| `/discuss` | アイデアを深掘り、MVP案や複数案を提示。実装前の整理 |
| `/plan` | 仕様書を分析して実装計画書を作成(`開発/計画/...md`) |
| `/coding` | 計画書に基づき backend / frontend を実装。レビュー・テストまで自動で回す |
| `/stage` | feat ブランチを staging に squash merge して push |
| `/release` | staging を main にマージして本番デプロイ |

サブセットを単独で使うこともできる。

- `/go` … Go バックエンド側だけ実装サイクルを回す
- `/nextjs` … Next.js フロントエンド側だけ実装サイクルを回す
- `/plan` 単体 … 計画書だけ作って実装は人間がレビューしてから

### 2.3 デプロイ後の修正

| スキル | 用途 |
|---|---|
| `/fix` | 既存 plan の続投で直すか、plan を立て直すかを判定して修正 |
| `/hotfix` | 緊急修正(main 直接) |
| `/destroy` | 不要になった機能・コード・テーブルの削除サイクル |
| `/update` | 依存ライブラリのアップデート対応 |

### 2.4 補助スキル

| スキル | 用途 |
|---|---|
| `/research` | 外部情報を収集して調査レポートを作成 |
| `/devtools` | 進捗ビューアを起動(`make dev` と等価の Claude 側エントリ) |

---

## 3. 統括ハブとしての運用

### 3.1 把握(自動・読み取り) — chief-director

横断把握系のフレーズで `chief-director` エージェントが自動的に呼ばれる。

```
全プロジェクトの状況は？
今日どう？
```

- 各プロジェクトの `開発/` 配下のカンバン(計画書・レポート)を集約
- `運用/状態/*.json` がある場合は運用状態(progress / stale / blocked / 連続エラー)も並べて報告
- **読み取り専用**。状態変更は一切しない

### 3.2 一括操作(明示指示) — bulk-coding

実装待ちを一気に流したい時の明示動詞で `/bulk-coding` を呼ぶ。

```
一括 coding して
実装待ちを一括で開始
```

- スキルが対象プロジェクトを選定 → `gr-run` を背景起動
- 完了・確認事項は ntfy 通知で届く
- 確認事項が出たら chief-director がユーザーに噛み砕いて A案/B案 を提示し、回答を計画書に書き戻す(`**ステータス**: 回答済`)→ 必要なら再 dispatch

### 3.3 運用把握(読み取り)

- 運用を持つ判定は **フォルダベース** : プロジェクトに `運用/manifest.json` があれば運用あり、無ければ運用なし(`開発/` と対称、`実行中/` が任意なのと同じ前方互換)
- stale / blocked / 連続エラーは **検知・報告のみ**。解消(Chrome 再起動・再点火・調査)はユーザー指示で別途

### 3.4 統括対象プロジェクトの登録

`devtools/backend/patrol_projects.json` の `projects` 配列にエントリを追加する(gitignore 対象・ローカル専用)。
詳細手順: `devtools/backend/docs/BACKEND_RUNBOOK.md` の「プロジェクト登録」

---

## 4. スマホ・ウェアラブルから操作する

### 4.1 Tailscale 経由のブラウザアクセス

Mac で `make dev` → iPhone Safari で `http://<Mac の Tailscale IP>:3333` で devtools UI が開く。
詳細: [../../概要.md](../../概要.md) の「スマホからアクセスする」

### 4.2 even-terminal (テキスト/音声ブリッジ)

スマホ・グラスから Claude Code に話しかけるブリッジ。BRIDGE_TOKEN は `~/.zshrc` に固定保存(関連: [2026-05-27_even-terminal_BRIDGE_TOKEN設定.md](2026-05-27_even-terminal_BRIDGE_TOKEN設定.md))。

```bash
make start-even-terminal   # 通常起動(:3456)
make stop-even-terminal
make restart-even-terminal
```

### 4.3 Even G2 (スマートグラス)

#### 単体プロジェクト接続

```bash
make g2                    # G2 用に起動。QR を表示するので G2 アプリでスキャン
```

#### 全プロジェクト並列接続(マルチプロジェクト切替)

```bash
make g2-all                # patrol_projects.json の全プロジェクト分を並列起動(:3456 から index 順)
make g2-qr                 # 起動中の全 QR を再表示(登録忘れリカバリ用)
make g2-status             # 起動中インスタンス一覧
make stop-g2-all           # 全停止
```

- G2 アプリ側で複数接続先を保存・切替できるので、登録は **1 回だけでよい**
- 以後は G2 側でプロジェクト切替するだけ、Mac 側操作は不要

---

## 5. 音声入出力(VOICEVOX)

統括 GUI の音声出力は VOICEVOX (春日部つむぎ, speaker_id=8) を使う。
**VOICEVOX Engine は起動中 1〜17 GB のメモリを保持する** ため、必要なときだけ起動する on-demand 運用が原則。

```bash
make start-voicevox        # VOICEVOX.app を起動 → :50021 待機 → ready
make stop-voicevox         # 停止(最大 17 GB 解放)
make restart-voicevox
```

`make stop` は VOICEVOX も同時停止する(寝る前用)。

---

## 6. メモリリーク再発防止 三本柱

過去に 17 GB リークしたため、以下の三層で再発を防いでいる。

### 6.1 pprof による調査

```bash
make start-backend-debug   # ENABLE_PPROF=1 つきで起動
# 取得例:
go tool pprof http://localhost:8888/debug/pprof/heap
# ブラウザ:
open http://localhost:8888/debug/pprof/
```

### 6.2 VOICEVOX on-demand 起動

→ §5 参照。常駐させない。

### 6.3 launchd ガード付き auto-restart

毎日 03:00 にチェックして、**RSS > 1 GB かつ 巡回アイドル** の時だけ backend を restart する。

```bash
make install-restart-cron    # ~/Library/LaunchAgents/ に登録
make status-restart-cron     # 状態確認 + 最新ログ tail
make uninstall-restart-cron  # 解除
```

ログ: `/tmp/ghostrunner-backend-restart.log`
スクリプト: `scripts/launchd/maybe-restart-backend.sh`

---

## 7. コマンドリファレンス

### 7.1 Makefile (Ghostrunner ルート)

| カテゴリ | コマンド | 用途 |
|---|---|---|
| 起動(FG) | `make backend` / `make frontend` / `make dev` | フォアグラウンド、ログ直接 |
| 起動(BG) | `make start-backend` / `make start-frontend` | バックグラウンド + tail ログ |
| 停止 | `make stop-backend` / `make stop-frontend` / `make stop` | 個別 / 全停止 |
| 再起動 | `make restart-backend` / `make restart-frontend` / `make restart` | BG 再起動 |
| ログ | `make logs-backend` / `make logs-frontend` | tail + 色付け |
| ビルド | `make build` / `make gr-run` | server + gr-run + frontend ビルド |
| ヘルス | `make health` | backend/frontend 疎通 |
| even-terminal | `make start-even-terminal` / `make stop-even-terminal` | :3456 |
| Even G2 | `make g2` / `make g2-all` / `make g2-qr` / `make g2-status` / `make stop-g2-all` | グラス接続 |
| VOICEVOX | `make start-voicevox` / `make stop-voicevox` / `make restart-voicevox` | on-demand 17 GB |
| auto-restart | `make install-restart-cron` / `make status-restart-cron` / `make uninstall-restart-cron` | メモリリーク保険 |

### 7.2 Claude Code スキル

| スキル | 用途 |
|---|---|
| `/init` | 新規プロジェクト作成(対話) |
| `/discuss` | アイデア深掘り・複数案提示 |
| `/plan` | 実装計画書の作成 |
| `/coding` | 計画 → 実装 → レビュー → テスト |
| `/go` | Go バックエンドだけ /coding |
| `/nextjs` | Next.js フロントだけ /coding |
| `/stage` | feat → staging squash merge + push |
| `/release` | staging → main + 本番デプロイ |
| `/fix` | デプロイ後の修正(plan 続投 or 立て直し判定) |
| `/hotfix` | 緊急修正(main 直接) |
| `/destroy` | 機能・コード・テーブルの削除サイクル |
| `/update` | 依存ライブラリ更新 |
| `/bulk-coding` | 全プロジェクトの実装待ちを一括 gr-run |
| `/research` | 外部情報を収集して調査レポート |
| `/devtools` | 進捗ビューア起動 |

### 7.3 ポート一覧

| サービス | ポート |
|---|---|
| devtools frontend | 3333 |
| devtools backend | 8888 |
| even-terminal (default) | 3456 |
| even-terminal (g2-all) | 3456-3462 (index 順) |
| VOICEVOX Engine | 50021 |
| 生成プロジェクト frontend | 3xxx (ランダム) |
| 生成プロジェクト backend | 8xxx (ランダム) |
| PostgreSQL | 5xxx (ランダム) |
| MinIO | 9xxx (ランダム) |
| Redis | 6xxx (ランダム) |

---

## 8. 補助Tips

### 8.1 Claude Code スレッド名を日時にする (espanso `:now`)

espanso のスニペットを登録しておくと、CC スレッドを建てる時の初手メッセージに展開トリガーを入れるだけでスレ名を日時にできる。
スレ一覧が時系列ソートされるので、運用系の日次・時次タスクと相性がよい。

| トリガー | 展開後 | 用途 |
|---|---|---|
| `:date` | `2026-05-28` | 日次の運用スレッド名 |
| `:now` | `2026-05-28 14:35` | 同日複数スレを建てる時に分単位でソート |

CC は初手メッセージの冒頭からスレ名を自動生成するため、`:now` を最初に置くだけでスレ名が `2026-05-28 14:35 ...` になる。

---

## 9. トラブルシュート

| 症状 | 原因 / 対処 |
|---|---|
| even-terminal が 401 を返す | BRIDGE_TOKEN 不一致。`~/.zshrc` の `export BRIDGE_TOKEN=...` と frontend `.env.local` の `NEXT_PUBLIC_EVEN_TERMINAL_TOKEN` を一致させる |
| backend RSS が膨らんでいる | `make install-restart-cron` で 03:00 auto-restart 登録 + `make start-backend-debug` で pprof 確認 |
| VOICEVOX が応答しない | `make restart-voicevox`。:50021 が listen するまで 30 秒待機する |
| G2 アプリで QR が見えなくなった | `make g2-qr` で起動中インスタンスの全 QR を再表示 |
| 統括が「実装待ちタスク」を流してくれない | 把握フレーズ(「状況は？」)では発火しない。明示動詞(「一括 coding して」)で `/bulk-coding` を呼ぶ |
| `.claude/` 配下を main で直接編集してしまった | ブランチを切ってから作業し直す(関連: feedback_branch_for_claude_dir) |

---

## 10. 関連資料

- [../../概要.md](../../概要.md) - Ghostrunner README(セットアップ・Tailscale 詳細)
- [../../.claude/CLAUDE.md](../../.claude/CLAUDE.md) - プロジェクト規約・統括の設計思想
- [2026-04-05_マネージャー構想_全体図.md](2026-04-05_マネージャー構想_全体図.md) - 統括ハブの構想
- [2026-05-27_even-terminal_BRIDGE_TOKEN設定.md](2026-05-27_even-terminal_BRIDGE_TOKEN設定.md) - even-terminal 認証
- [2026-05-23_devtools仕組み解説.md](2026-05-23_devtools仕組み解説.md) - 進捗ビューアの内部構造
- [2026-03-27_Swift_SwiftUI_依存性逆転_DIパターン.md](2026-03-27_Swift_SwiftUI_依存性逆転_DIパターン.md) - Swift テンプレ DI 設計
- `devtools/backend/docs/BACKEND_RUNBOOK.md` - patrol プロジェクト登録手順
