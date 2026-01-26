# 画面遷移フロー

Ghost Runner は SPA のため、ページ遷移ではなくコンポーネントの状態遷移で画面が変化する。

## 全体フロー

```mermaid
flowchart TD
    A[初期状態<br>コマンド入力のみ] -->|Execute Command| B[実行中<br>ローディング表示]
    B -->|イベント受信| C{イベント種別}
    C -->|tool_use, text, thinking| B
    C -->|question| D[質問待ち<br>QuestionSection表示]
    C -->|complete + 計画承認キーワード| E[計画承認待ち<br>PlanApproval表示]
    C -->|complete| F[完了<br>結果表示]
    C -->|error| G[エラー<br>エラー表示]
    B -->|Abort ボタン| H[中断<br>結果表示]
    D -->|回答（次の質問あり）| D
    D -->|回答（最後の質問）| B
    E -->|Approve/Reject| B
    F -->|新しいコマンド入力| A
    G -->|新しいコマンド入力| A
    H -->|新しいコマンド入力| A
```

## 状態遷移詳細

### 1. 初期状態 -> 実行中

| トリガー | 処理 |
|---------|------|
| Execute Command ボタンクリック | executeCommandStream API呼び出し、SSEストリーム開始 |

**渡すデータ**:
- `project`: プロジェクトパス
- `command`: 選択したコマンド（plan, research, etc.）
- `args`: 引数（選択ファイル + 入力テキスト）
- `images`: 画像データ配列（任意）
  - `name`: ファイル名
  - `data`: Base64エンコードされた画像データ
  - `mimeType`: MIME タイプ（image/jpeg, image/png, image/gif, image/webp）

### 2. 実行中 -> 質問待ち

| トリガー | 処理 |
|---------|------|
| question イベント受信 | questions 状態を更新、showQuestions を true に設定 |

**表示データ**:
- `question`: 質問文
- `header`: 質問のヘッダー
- `options`: 選択肢（label, description）
- `multiSelect`: 複数選択可否

### 3. 質問待ち -> 次の質問 / 実行中

複数の質問がある場合、最後の質問に回答するまでバックエンド通信は発生しない。

```mermaid
flowchart TD
    A[質問待ち<br>質問 N/M 表示] -->|回答| B{最後の質問?}
    B -->|いいえ| C[次の質問表示<br>質問 N+1/M]
    C -->|回答| B
    B -->|はい| D[実行中<br>バックエンドに送信]
```

| トリガー | 条件 | 処理 |
|---------|------|------|
| 選択肢クリック（単一選択時） | 最後の質問以外 | currentQuestionIndex をインクリメント |
| 選択肢クリック（単一選択時） | 最後の質問 | continueSessionStream API呼び出し |
| Submit ボタンクリック | 最後の質問以外 | currentQuestionIndex をインクリメント |
| Submit ボタンクリック | 最後の質問 | continueSessionStream API呼び出し |

**渡すデータ（最後の質問回答時のみ）**:
- `project`: プロジェクトパス
- `session_id`: セッションID
- `answer`: 回答テキスト

### 4. 実行中 -> 計画承認待ち

| トリガー | 処理 |
|---------|------|
| complete イベント + 承認キーワード | showPlanApproval を true に設定 |

**承認キーワード**:
- "承認をお待ち"
- "waiting for approval"
- "Ready for approval"

### 5. 計画承認待ち -> 実行中

| トリガー | 処理 |
|---------|------|
| Approve Plan ボタンクリック | "yes, proceed with the plan" で continueSessionStream 呼び出し |
| Reject ボタンクリック | "no, cancel the plan" で continueSessionStream 呼び出し |

### 6. 実行中 -> 完了/エラー

| トリガー | 処理 |
|---------|------|
| complete イベント（質問なし、承認不要） | resultOutput を設定、resultType を "success" に |
| error イベント | resultOutput を設定、resultType を "error" に |

### 7. 実行中 -> 中断

| トリガー | 処理 |
|---------|------|
| Abort ボタンクリック | AbortController.abort() で SSE 接続を切断 |

**表示条件**:
- ローディング中（`isLoading === true`）
- 質問待ちでない（`showQuestions === false`）
- 計画承認待ちでない（`showPlanApproval === false`）

**中断時の処理**:
- SSE ストリーム接続を切断
- イベントリストに "Execution aborted" を追加
- 結果表示に "Execution aborted by user" を表示（エラー扱い）
- ローディング状態を解除

## API通信フロー

```mermaid
sequenceDiagram
    participant User
    participant Frontend
    participant Backend

    User->>Frontend: Execute Command
    Note over User,Frontend: 画像がある場合は Base64 データを含む
    Frontend->>Backend: POST /api/command/stream
    Note right of Frontend: {project, command, args, images?}
    Backend-->>Frontend: SSE init
    Backend-->>Frontend: SSE thinking
    Backend-->>Frontend: SSE tool_use (複数回)
    Backend-->>Frontend: SSE question
    Frontend->>User: 質問表示
    User->>Frontend: 回答入力
    Frontend->>Backend: POST /api/command/continue/stream
    Backend-->>Frontend: SSE thinking
    Backend-->>Frontend: SSE tool_use (複数回)
    Backend-->>Frontend: SSE complete
    Frontend->>User: 結果表示
```

### 中断時の通信フロー

```mermaid
sequenceDiagram
    participant User
    participant Frontend
    participant Backend

    User->>Frontend: Execute Command
    Frontend->>Backend: POST /api/command/stream
    Backend-->>Frontend: SSE init
    Backend-->>Frontend: SSE thinking
    Backend-->>Frontend: SSE tool_use
    User->>Frontend: Abort ボタンクリック
    Frontend->>Frontend: AbortController.abort()
    Frontend--xBackend: 接続切断
    Frontend->>User: 中断結果表示
```

## イベント処理フロー

```mermaid
flowchart LR
    subgraph SSE Stream
        E1[init] --> E2[thinking]
        E2 --> E3[tool_use]
        E3 --> E3
        E3 --> E4{分岐}
        E4 --> E5[question]
        E4 --> E6[complete]
        E4 --> E7[error]
    end

    subgraph UI Updates
        E1 -.-> U1[Session started イベント追加]
        E2 -.-> U2[Thinking... 表示]
        E3 -.-> U3[ツール使用イベント追加]
        E5 -.-> U4[質問UI表示]
        E6 -.-> U5[結果表示 + 計画承認判定]
        E7 -.-> U6[エラー表示]
    end
```

## ツール使用イベントの表示

各ツールは使用時に以下の情報を表示する。

| ツール | 表示内容 |
|-------|---------|
| Read | ファイルパス（短縮）+ offset/limit |
| Write | ファイルパス（短縮）+ 文字数 |
| Edit | ファイルパス（短縮）+ 置換前後の文字数 |
| Glob | パターン + 対象パス |
| Grep | パターン + 対象パス + glob |
| Bash | 説明（あれば）+ コマンド |
| Task | タスク種別 + プロンプト（短縮） |
| TodoWrite | アイテム数 |
| WebFetch | URL（短縮） |
| WebSearch | 検索クエリ（短縮） |
| ExitPlanMode | "Requesting plan approval" |
| EnterPlanMode | "Starting plan mode" |

## 開発者機能：サーバー再起動フロー

開発環境でのみ利用可能なサーバー再起動機能のフロー。

```mermaid
flowchart TD
    A[idle<br>Restart Servers ボタン表示] -->|ボタンクリック| B[restarting<br>API呼び出し]
    B -->|Fire-and-Forget| C[ヘルスチェックポーリング開始]
    C -->|ヘルスチェック成功| D[success<br>ページリロード]
    C -->|30秒経過| E[timeout<br>手動リロード促す]
```

### 再起動フロー詳細

```mermaid
sequenceDiagram
    participant User
    participant Frontend
    participant RestartAPI as Route Handler
    participant Makefile
    participant Backend

    User->>Frontend: Restart Servers ボタンクリック
    Frontend->>Frontend: setRestartStatus("restarting")

    par Fire-and-Forget
        Frontend->>RestartAPI: POST /api/restart/backend
        RestartAPI->>Makefile: make restart-backend
        and
        Frontend->>RestartAPI: POST /api/restart/frontend
        RestartAPI->>Makefile: make restart-frontend
    end

    loop ヘルスチェック（最大30回、1秒間隔）
        Frontend->>Backend: GET /api/health
        alt 成功
            Backend-->>Frontend: 200 OK
            Frontend->>Frontend: setRestartStatus("success")
            Frontend->>Frontend: window.location.reload()
        else 失敗
            Backend--xFrontend: エラー
            Frontend->>Frontend: 1秒待機してリトライ
        end
    end

    Note over Frontend: タイムアウト時
    Frontend->>Frontend: setRestartStatus("timeout")
```

### 状態遷移

| 現在の状態 | トリガー | 次の状態 | 処理 |
|-----------|---------|---------|------|
| idle | ボタンクリック | restarting | 両API呼び出し、ポーリング開始 |
| restarting | ヘルスチェック成功 | success | 500ms後にページリロード |
| restarting | 30秒タイムアウト | timeout | ボタン表示を "Timeout - Reload manually" に変更 |
