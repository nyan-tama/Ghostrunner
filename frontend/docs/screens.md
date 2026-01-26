# 画面一覧

Ghost Runner は単一ページアプリケーション（SPA）として構成されており、メインページ内でコンポーネントの表示/非表示を切り替えることで、異なる状態を表現する。

## ページ一覧

| パス | コンポーネント | 役割 |
|-----|--------------|------|
| `/` | `app/page.tsx` | メインページ（コマンド入力と実行結果表示） |

## メインページの構成要素

### ヘッダーエリア

| 要素 | 役割 | 備考 |
|-----|------|------|
| タイトル | "Ghost Runner" の表示 | |
| Restart Servers ボタン | バックエンド・フロントエンドサーバーの再起動 | 開発環境のみ表示 |

### コマンド入力エリア

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| CommandForm | `components/CommandForm.tsx` | プロジェクトパス、コマンド、ファイル、引数の入力フォーム |

#### フォーム要素

- **Project Path**: 対象プロジェクトの絶対パス入力
- **Command**: 実行コマンドの選択（plan, research, discuss, fullstack, go, nextjs）
- **File**: 開発フォルダ内のファイル選択（任意）
- **Arguments**: コマンドへの引数入力

### 進捗表示エリア

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| ProgressContainer | `components/ProgressContainer.tsx` | 進捗表示全体のコンテナ |
| LoadingIndicator | `components/LoadingIndicator.tsx` | 処理中のスピナーとテキスト表示 |
| EventList | `components/EventList.tsx` | イベントリストの表示（自動スクロール） |
| EventItem | `components/EventItem.tsx` | 個別イベントの表示（展開可能） |
| QuestionSection | `components/QuestionSection.tsx` | AIからの質問への回答UI |
| PlanApproval | `components/PlanApproval.tsx` | 計画の承認/拒否ボタン |

#### 表示状態

1. **初期状態**: ProgressContainer は非表示
2. **実行中**: ローディング表示 + イベントリスト + 中断ボタン（Abort）
3. **質問待ち**: QuestionSection を表示（中断ボタンは非表示）
4. **計画承認待ち**: PlanApproval を表示（中断ボタンは非表示）
5. **完了**: 結果出力を表示（成功は緑、エラーは赤）
6. **中断**: 結果出力に "Execution aborted by user" を表示（赤背景）

## イベント種別

EventItem で表示されるイベントの種別と色。

| 種別 | 色 | 説明 |
|-----|---|------|
| tool | 青 | ツール使用（Read, Write, Edit, Glob, Grep, Bash等） |
| task | 紫 | サブタスク実行 |
| text | 緑 | テキスト出力 |
| info | シアン | 情報メッセージ（セッション開始、中断等） |
| error | 赤 | エラー |
| question | 黄 | 質問 |

## カスタムフック

| フック | ファイル | 役割 |
|-------|---------|------|
| useSSEStream | `hooks/useSSEStream.ts` | SSEストリームの処理（バッファリング対応） |
| useSessionManagement | `hooks/useSessionManagement.ts` | セッションID、累計コスト、プロジェクトパスの管理 |
| useFileSelector | `hooks/useFileSelector.ts` | 開発フォルダ内のファイル取得と選択 |

## データフロー

```
CommandForm
    |
    v (executeCommandStream)
SSEストリーム (AbortController で中断可能)
    |
    v (useSSEStream)
handleStreamEvent
    |
    +---> setEvents (EventList)
    +---> setQuestions (QuestionSection)
    +---> setShowPlanApproval (PlanApproval)
    +---> setResultOutput (結果表示)

中断操作:
Abort ボタン
    |
    v (handleAbort)
AbortController.abort()
    |
    +---> SSE接続を切断
    +---> "Execution aborted" イベント追加
    +---> 結果表示に "Execution aborted by user"
```

## 永続化データ

| キー | 保存場所 | 内容 |
|-----|---------|------|
| `ghostrunner_project` | localStorage | プロジェクトパス |

## 開発者機能

### サーバー再起動機能（開発環境のみ）

開発環境（`NODE_ENV === "development"`）でのみ表示されるサーバー再起動ボタン。

#### 仕組み

1. ボタンクリックで `/api/restart/backend` と `/api/restart/frontend` を Fire-and-Forget で呼び出し
2. Route Handler がプロジェクトルートの Makefile を実行（`make restart-backend`, `make restart-frontend`）
3. バックエンドのヘルスチェックエンドポイント（`/api/health`）をポーリング
4. ヘルスチェック成功後、ページを自動リロード

#### ボタン状態

| 状態 | 表示テキスト | 操作 |
|-----|------------|------|
| idle | "Restart Servers" | クリック可能 |
| restarting | "Restarting..." | 無効化 |
| success | （リロード） | ページ自動リロード |
| timeout | "Timeout - Reload manually" | 手動リロードが必要 |

#### 関連ファイル

| ファイル | 役割 |
|---------|------|
| `app/api/restart/backend/route.ts` | バックエンド再起動 Route Handler |
| `app/api/restart/frontend/route.ts` | フロントエンド再起動 Route Handler |
| `lib/constants.ts` | `BACKEND_HEALTH_URL` の定義 |
| `types/index.ts` | `RestartStatus` 型の定義 |
| プロジェクトルート `/Makefile` | 再起動コマンドの定義 |
