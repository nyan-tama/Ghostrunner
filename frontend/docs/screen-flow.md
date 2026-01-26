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

## ファイル選択フロー（複数選択）

コマンド実行前に、複数のファイルを選択して引数に含めることができる。

```mermaid
flowchart TD
    A[未選択<br>ドロップダウン表示] -->|フォーカス| R[ファイルリスト<br>サイレント更新]
    R -->|ファイル選択| B[選択済み<br>タグリスト表示]
    A -->|ファイル選択| B
    B -->|別のファイル選択| B
    B -->|x ボタンクリック| C{残りファイル?}
    C -->|あり| B
    C -->|なし| A
    B -->|Execute Command| D[実行中<br>選択は保持]
    D -->|実行完了| B
```

### ファイル選択の状態遷移

| 現在の状態 | トリガー | 次の状態 | 処理 |
|-----------|---------|---------|------|
| 任意 | ドロップダウンにフォーカス | 同じ | refreshFiles でファイルリストをサイレント更新（ローディング表示なし） |
| 未選択 | ドロップダウンでファイル選択 | 選択済み | addSelectedFile で配列に追加 |
| 選択済み | ドロップダウンで別ファイル選択 | 選択済み | addSelectedFile で配列に追加（重複は無視） |
| 選択済み | タグの x ボタンクリック | 選択済み/未選択 | removeSelectedFile で配列から削除 |
| 選択済み | コマンド実行完了 | 選択済み | 選択状態は保持される |

### ファイル選択 UI 詳細

```mermaid
flowchart LR
    subgraph ドロップダウン
        A[optgroup: 開発/資料] --> A1[file1.md]
        A --> A2[checkmark file2.md<br>disabled]
        B[optgroup: 開発/検討中] --> B1[file3.md]
        B --> B2[checkmark file4.md<br>disabled]
    end

    subgraph 選択済みタグリスト
        C1[file2.md x]
        C2[file4.md x]
    end
```

## 状態遷移詳細

### 1. 初期状態 -> 実行中

| トリガー | 処理 |
|---------|------|
| Execute Command ボタンクリック | executeCommandStream API呼び出し、SSEストリーム開始 |

**渡すデータ**:
- `project`: プロジェクトパス
- `command`: 選択したコマンド（plan, research, etc.）
- `args`: 引数（選択ファイル群 + 入力テキスト）
  - 複数ファイル選択時: `file1.md file2.md file3.md 引数テキスト`
  - ファイル未選択時: `引数テキスト`
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

---

## Gemini Live: 音声会話フロー

`/gemini-live` は独立したページで、Gemini Live API を使用したリアルタイム音声会話機能を提供する。
メインページ（`/`）との画面遷移はなく、直接 URL でアクセスする。

### 状態遷移

```mermaid
flowchart TD
    A[未接続<br>Connect ボタン表示] -->|Connect ボタンクリック| B[接続中<br>Connecting...]
    B -->|setupComplete 受信| C[接続完了<br>マイクボタン有効]
    B -->|エラー発生| E[エラー<br>エラーメッセージ表示]
    C -->|Start Recording ボタンクリック| D[録音中<br>音声入力送信]
    D -->|Stop Recording ボタンクリック| C
    D -->|音声応答受信| D
    C -->|Disconnect ボタンクリック| A
    D -->|Disconnect ボタンクリック| A
    E -->|Connect ボタンクリック| B
```

### 状態遷移詳細

| 現在の状態 | トリガー | 次の状態 | 処理 |
|-----------|---------|---------|------|
| 未接続 | Connect ボタンクリック | 接続中 | エフェメラルトークン取得、WebSocket 接続開始 |
| 接続中 | setupComplete メッセージ受信 | 接続完了 | connectionStatus を "connected" に設定 |
| 接続中 | WebSocket エラー | エラー | エラーメッセージを表示 |
| 接続完了 | Start Recording ボタンクリック | 録音中 | マイク取得、AudioWorklet 開始、音声送信開始 |
| 録音中 | Stop Recording ボタンクリック | 接続完了 | 音声入力停止、リソース解放 |
| 録音中 | 音声応答受信 | 録音中 | 音声をキューに追加、順次再生 |
| 接続完了/録音中 | Disconnect ボタンクリック | 未接続 | WebSocket 切断、全リソース解放 |
| エラー | Connect ボタンクリック | 接続中 | エラーをクリアし、再接続を試行 |

### WebSocket 通信フロー

```mermaid
sequenceDiagram
    participant User
    participant Frontend
    participant Backend as Backend API
    participant Gemini as Gemini Live API

    User->>Frontend: Connect ボタンクリック
    Frontend->>Backend: POST /api/gemini/token
    Backend-->>Frontend: エフェメラルトークン
    Frontend->>Gemini: WebSocket 接続 + setup メッセージ
    Gemini-->>Frontend: setupComplete
    Frontend->>User: 接続完了表示

    User->>Frontend: Start Recording ボタンクリック
    Frontend->>Frontend: マイク取得、AudioWorklet 開始

    loop 録音中
        Frontend->>Frontend: 音声をダウンサンプリング + Base64 エンコード
        Frontend->>Gemini: realtimeInput（音声データ）
        Gemini-->>Frontend: serverContent（音声応答）
        Frontend->>Frontend: 音声をキューに追加、再生
    end

    User->>Frontend: Stop Recording ボタンクリック
    Frontend->>Frontend: 音声入力停止

    User->>Frontend: Disconnect ボタンクリック
    Frontend->>Gemini: WebSocket 切断
    Frontend->>User: 未接続表示
```

### 音声処理フロー

```mermaid
flowchart LR
    subgraph 音声入力
        A[マイク] --> B[AudioContext]
        B --> C[AudioWorklet]
        C --> D[ダウンサンプリング<br>48kHz -> 16kHz]
        D --> E[Int16 PCM 変換]
        E --> F[Base64 エンコード]
        F --> G[WebSocket 送信]
    end

    subgraph 音声出力
        H[WebSocket 受信] --> I[Base64 デコード]
        I --> J[PCM -> AudioBuffer]
        J --> K[再生キュー]
        K --> L[順次再生]
        L --> M[スピーカー]
    end
```

### エラーハンドリング

| エラー | 表示内容 | 対処 |
|-------|---------|------|
| トークン取得失敗 | "Failed to get ephemeral token" | Connect ボタンで再試行 |
| WebSocket 接続エラー | "Failed to connect to Gemini Live API" | Connect ボタンで再試行 |
| Gemini API エラー | "Gemini API error: [メッセージ]" | 内容を確認し、Connect ボタンで再試行 |
| マイク許可拒否 | "Microphone permission denied" | ブラウザ設定でマイク許可を付与 |
| 録音開始失敗 | "Failed to start recording: [詳細]" | Disconnect 後に再接続 |
