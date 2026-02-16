# 画面一覧

Ghost Runner は単一ページアプリケーション（SPA）として構成されており、メインページ内でコンポーネントの表示/非表示を切り替えることで、異なる状態を表現する。

## ページ一覧

| パス | コンポーネント | 役割 |
|-----|--------------|------|
| `/` | `app/page.tsx` | メインページ（コマンド入力と実行結果表示） |
| `/docs` | `app/docs/page.tsx` | 開発ドキュメントのルート表示（フォルダ一覧） |
| `/docs/[...path]` | `app/docs/[...path]/page.tsx` | 開発ドキュメントのサブフォルダ/ファイル表示 |
| `/gemini-live` | `app/gemini-live/page.tsx` | Gemini Live API を使用した音声 AI インターフェース |
| `/openai-realtime` | `app/openai-realtime/page.tsx` | OpenAI Realtime API を使用した音声 AI インターフェース |

## メインページの構成要素

### ヘッダーエリア

| 要素 | 役割 | 備考 |
|-----|------|------|
| タイトル | "Ghost Runner" の表示 | |
| Docs リンク | 開発ドキュメントページへの遷移 | projectPath を `?project=` クエリで引き渡し |
| Gemini Live リンク | Gemini Live ページへの遷移 | |
| OpenAI Realtime リンク | OpenAI Realtime ページへの遷移 | |
| Restart Servers ボタン | バックエンド・フロントエンドサーバーの再起動 | 開発環境のみ表示 |

### コマンド入力エリア

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| CommandForm | `components/CommandForm.tsx` | プロジェクトパス、コマンド、ファイル、引数の入力フォーム |

#### フォーム要素

- **Project Path**: 対象プロジェクトのドロップダウン選択（バックエンドからプロジェクト一覧を取得） + 履歴ドロップダウン
- **Command**: 実行コマンドの選択（plan, research, discuss, fullstack, go, nextjs）
- **File**: 開発フォルダ内のファイル選択（任意、複数選択可）
- **Arguments**: コマンドへの引数入力
- **PR workflow**: トグルスイッチ（ON で develop ブランチ経由 PR の指示を追加）
- **Voice notification**: トグルスイッチ（ON で音声通知を有効化）
- **Images**: 画像のアップロード（任意）

#### Project Path 選択

プロジェクトパスはドロップダウンから選択する。バックエンド `GET /api/projects` から取得したプロジェクト一覧と、履歴ドロップダウンの横並びレイアウトで構成される。

| 項目 | 内容 |
|-----|------|
| データソース | バックエンド `GET /api/projects` からプロジェクト一覧を取得（ページ読み込み時に1回） |
| 選択方法 | プロジェクト一覧ドロップダウンから選択 |
| 履歴機能 | 履歴ドロップダウン（幅固定）が横に配置され、過去に使用したパスを選択可能 |
| カスタムパス | プロジェクト一覧に含まれないパスも、履歴経由で選択可能。選択時はドロップダウン内に `(custom)` 付きで表示される |
| レイアウト | プロジェクト一覧ドロップダウン（flex-1）+ 履歴ドロップダウン（w-20）の横並び |

#### ファイル選択（複数対応）

複数のファイルをコマンドの引数として渡すことができる。

| 項目 | 内容 |
|-----|------|
| 選択方法 | ドロップダウンから選択、選択ごとにリストに追加 |
| 選択済み表示 | タグ形式で表示（ファイル名のみ） |
| 削除方法 | 各タグの x ボタンをクリック |
| 重複防止 | 同じファイルは追加されない |
| 選択済みマーク | ドロップダウン内で選択済みファイルに checkmark を表示、disabled |
| 実行後の動作 | 選択は保持される（手動で削除可能） |
| 自動リフレッシュ | ドロップダウンにフォーカスすると、ファイルリストをサイレント更新（ローディング表示なし） |

**引数生成**:
- 選択ファイルがある場合: `ファイルパス1 ファイルパス2 ... 引数テキスト`
- 選択ファイルがない場合: `引数テキスト`

### 画像アップロードエリア

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| ImageUploader | `components/ImageUploader.tsx` | 画像のアップロードとプレビュー表示 |

#### 仕様

| 項目 | 内容 |
|-----|------|
| 最大枚数 | 5枚 |
| 最大サイズ | 5MB/枚 |
| 対応形式 | JPEG, PNG, GIF, WebP |
| エンコード | Base64 |

#### レイアウト

横2分割（`grid grid-cols-2 gap-2`）のゾーン構成。

| ゾーン | 位置 | 操作 | 説明 |
|-------|------|------|------|
| 画像ドロップゾーン | 左 | クリックでファイル選択、またはドラッグ&ドロップ | 複数ファイルを一度に選択可能 |
| カメラ撮影ゾーン | 右 | クリックでカメラ起動 | `capture="environment"` により背面カメラを使用。モバイルではカメラアプリが起動し、デスクトップではファイル選択ダイアログにフォールバックする |

#### 機能

- ドラッグ&ドロップによるアップロード（左ゾーン）
- ファイル選択ダイアログによるアップロード（左ゾーン）
- カメラ撮影による画像追加（右ゾーン、モバイルで背面カメラ起動）
- アップロード済み画像のサムネイルプレビュー（5列グリッド）
- 個別画像の削除（ホバー時に削除ボタン表示）
- 重複ファイルの自動スキップ
- バリデーションエラーの表示

### 進捗表示エリア

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| ProgressContainer | `components/ProgressContainer.tsx` | 進捗表示全体のコンテナ |
| LoadingIndicator | `components/LoadingIndicator.tsx` | 処理中のスピナーとテキスト表示 |
| EventList | `components/EventList.tsx` | イベントリストの表示（自動スクロール） |
| EventItem | `components/EventItem.tsx` | 個別イベントの表示（展開可能） |
| QuestionSection | `components/QuestionSection.tsx` | AIからの質問への回答UI（逐次表示） |
| PlanApproval | `components/PlanApproval.tsx` | 計画の承認/拒否ボタン |

#### 表示状態

1. **初期状態**: ProgressContainer は非表示
2. **実行中**: ローディング表示 + イベントリスト + 中断ボタン（Abort）
3. **質問待ち**: QuestionSection を表示（中断ボタンは非表示）
4. **計画承認待ち**: PlanApproval を表示（中断ボタンは非表示）
5. **完了**: 結果出力を表示（成功は緑、エラーは赤）
6. **中断**: 結果出力に "Execution aborted by user" を表示（赤背景）

### 質問の逐次表示

複数の質問がある場合、一度にすべてを表示するのではなく、1問ずつ順番に表示する。

#### 表示要素

- **進捗表示**: 「質問 N/M」形式で現在の質問番号と総質問数を表示
- **質問内容**: ヘッダー、質問文、選択肢
- **回答入力**: 選択肢ボタン、またはカスタム回答のテキスト入力

#### 動作仕様

| 操作 | 条件 | 処理 |
|-----|------|------|
| 選択肢クリック（単一選択） | 最後の質問以外 | 次の質問を表示 |
| 選択肢クリック（単一選択） | 最後の質問 | バックエンドに回答を送信 |
| Submit ボタンクリック | 最後の質問以外 | 次の質問を表示 |
| Submit ボタンクリック | 最後の質問 | バックエンドに回答を送信 |

#### 状態管理

- `questions`: バックエンドから受信した質問の配列
- `currentQuestionIndex`: 現在表示中の質問インデックス（0始まり）
- 新しい質問セットを受信した際、インデックスは自動的にリセットされる

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

### 音声通知エリア

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| VoiceNotificationSection | `components/VoiceNotificationSection.tsx` | 音声通知のトグル、接続状態インジケーター、マイクボタン |

#### 構成要素

| 要素 | 役割 | 備考 |
|-----|------|------|
| トグルスイッチ | 音声通知機能の ON/OFF | 状態は localStorage に保存 |
| 接続状態ドット | OpenAI Realtime API への接続状態を色で表示 | ツールチップで詳細を表示 |
| マイクボタン | 対話モードの開始/停止 | 接続時のみ表示 |
| エラー表示 | 接続エラーの表示 | エラー発生時のみ表示 |

#### 接続状態インジケーター

| 状態 | ドット色 | ツールチップ |
|-----|---------|------------|
| disconnected | グレー | Disconnected |
| connecting | 黄色 | Connecting... |
| connected | 緑 | Connected |
| error | 赤 | エラーメッセージ |

#### 動作仕様

| 操作 | 条件 | 処理 |
|-----|------|------|
| トグル ON | - | OpenAI Realtime API に自動接続 |
| トグル OFF | - | 接続を切断、通知キューをクリア |
| マイクボタンクリック（録音停止中） | 接続状態 = connected | 音声入力を開始（対話モード） |
| マイクボタンクリック（録音中） | 接続状態 = connected | 音声入力を停止 |
| コマンド完了 | トグル ON、録音停止中 | 完了メッセージを音声で通知 |
| コマンドエラー | トグル ON、録音停止中 | エラーメッセージを音声で通知 |

## カスタムフック

| フック | ファイル | 役割 |
|-------|---------|------|
| useSSEStream | `hooks/useSSEStream.ts` | SSEストリームの処理（バッファリング対応） |
| useSessionManagement | `hooks/useSessionManagement.ts` | セッションID、累計コスト、プロジェクトパスの管理 |
| useFileSelector | `hooks/useFileSelector.ts` | 開発フォルダ内のファイル取得と複数選択管理、ドロップダウンフォーカス時のサイレントリフレッシュ |
| useVoiceNotification | `hooks/useVoiceNotification.ts` | 音声通知機能の状態管理、OpenAI Realtime API との連携 |

## データフロー

```
CommandForm
    |
    +---> useFileSelector (複数ファイル選択)
    |         |
    |         +---> addSelectedFile (ファイル追加)
    |         +---> removeSelectedFile (ファイル削除)
    |         +---> refreshFiles (ドロップダウンフォーカス時にサイレント更新)
    |         v
    |     selectedFiles[] (選択されたファイルパスの配列)
    |
    +---> ImageUploader (画像アップロード)
    |         |
    |         v
    |     images[] (Base64エンコード済み ImageData[])
    |
    v (executeCommandStream)
args 生成: selectedFiles.join(" ") + " " + args
SSEストリーム (AbortController で中断可能)
    |
    v (useSSEStream)
handleStreamEvent
    |
    +---> setEvents (EventList)
    +---> setQuestionsWithReset (QuestionSection、インデックスもリセット)
    +---> setShowPlanApproval (PlanApproval)
    +---> setResultOutput (結果表示)

質問回答:
QuestionSection
    |
    v (onAnswer)
handleAnswerWithNext
    |
    +---> 最後の質問以外: setCurrentQuestionIndex をインクリメント
    +---> 最後の質問: handleAnswer でバックエンドに送信

中断操作:
Abort ボタン
    |
    v (handleAbort)
AbortController.abort()
    |
    +---> SSE接続を切断
    +---> "Execution aborted" イベント追加
    +---> 結果表示に "Execution aborted by user"

通知経路:
コマンド完了/エラー
    |
    +---> フロントエンド: useVoiceNotification (音声通知、トグル ON 時)
    +---> バックエンド: ntfy.sh (プッシュ通知、NTFY_TOPIC 設定時)
```

## 永続化データ

| キー | 保存場所 | 内容 |
|-----|---------|------|
| `ghostrunner_project` | localStorage | プロジェクトパス |
| `ghostrunner_project_history` | localStorage | プロジェクトパス履歴 |
| `ghostrunner_git_workflow` | localStorage | PR workflow トグル状態 |
| `ghostrunner_voice_notification` | localStorage | 音声通知トグル状態 |

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

---

## Gemini Live ページ

Gemini Live API を使用したリアルタイム音声会話機能を提供する独立ページ。

### 構成要素

| 要素 | 役割 | 備考 |
|-----|------|------|
| タイトル | "Gemini Live" の表示 | |
| 接続状態インジケーター | WebSocket 接続状態を色とテキストで表示 | |
| エラー表示 | エラーメッセージの表示（エラー発生時のみ） | |
| 接続ボタン | Gemini Live API への接続/切断 | |
| マイクボタン | 音声入力の開始/停止 | 接続時のみ有効 |
| 使い方説明 | 操作手順の説明 | |
| デバッグ情報 | 接続状態、録音状態、エラー情報の表示 | 開発環境のみ表示 |

### コンポーネント

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| GeminiLiveClient | `components/GeminiLiveClient.tsx` | 音声 AI インターフェースの UI |

### カスタムフック

| フック | ファイル | 役割 |
|-------|---------|------|
| useGeminiLive | `hooks/useGeminiLive.ts` | WebSocket 接続と音声処理の管理 |

### 接続状態

| 状態 | インジケーター色 | 表示テキスト |
|-----|-----------------|-------------|
| disconnected | グレー | Disconnected |
| connecting | 黄色 | Connecting... |
| connected | 緑 | Connected |
| error | 赤 | Error |

### ボタン状態

#### 接続ボタン

| 接続状態 | 表示テキスト | 操作 |
|---------|------------|------|
| disconnected | "Connect" | クリックで接続開始 |
| connecting | "Connecting..." | 無効化 |
| connected | "Disconnect" | クリックで切断 |
| error | "Connect" | クリックで再接続 |

#### マイクボタン

| 接続状態 | 録音状態 | 表示テキスト | 操作 |
|---------|---------|------------|------|
| 未接続 | - | "Start Recording" | 無効化 |
| 接続中 | 停止 | "Start Recording" | クリックで録音開始 |
| 接続中 | 録音中 | "Stop Recording" | クリックで録音停止 |

### 技術仕様

#### 音声入力

| 項目 | 値 |
|-----|-----|
| サンプルレート | 16kHz（Gemini 要求仕様） |
| チャンネル数 | モノラル |
| フォーマット | 16-bit PCM、Base64 エンコード |
| 処理方式 | AudioWorklet によるリアルタイム処理 |

#### 音声出力

| 項目 | 値 |
|-----|-----|
| サンプルレート | 24kHz（Gemini 出力仕様） |
| フォーマット | 16-bit PCM |
| 処理方式 | キュー方式による順次再生 |

### 関連ファイル

| ファイル | 役割 |
|---------|------|
| `app/gemini-live/page.tsx` | ページエントリーポイント（SSR 無効化） |
| `components/GeminiLiveClient.tsx` | UI コンポーネント |
| `hooks/useGeminiLive.ts` | WebSocket 接続・音声処理フック |
| `types/gemini.ts` | Gemini Live API 関連の型定義 |
| `lib/api.ts` | エフェメラルトークン取得 API |
| `lib/audioProcessor.ts` | 音声処理ユーティリティ |
| `public/audio-worklet-processor.js` | AudioWorklet プロセッサ |

---

## OpenAI Realtime ページ

OpenAI Realtime API を使用したリアルタイム音声会話機能を提供する独立ページ。Gemini Live ページと同等の機能を持つ。

### 構成要素

| 要素 | 役割 | 備考 |
|-----|------|------|
| タイトル | "OpenAI Realtime" の表示 | |
| 接続状態インジケーター | WebSocket 接続状態を色とテキストで表示 | |
| エラー表示 | エラーメッセージの表示（エラー発生時のみ） | |
| 接続ボタン | OpenAI Realtime API への接続/切断 | |
| マイクボタン | 音声入力の開始/停止 | 接続時のみ有効 |
| 使い方説明 | 操作手順の説明 | |
| デバッグ情報 | 接続状態、録音状態、エラー情報の表示 | 開発環境のみ表示 |

### コンポーネント

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| OpenAIRealtimeClient | `components/OpenAIRealtimeClient.tsx` | 音声 AI インターフェースの UI |

### カスタムフック

| フック | ファイル | 役割 |
|-------|---------|------|
| useOpenAIRealtime | `hooks/useOpenAIRealtime.ts` | WebSocket 接続と音声処理の管理 |

### 接続状態

| 状態 | インジケーター色 | 表示テキスト |
|-----|-----------------|-------------|
| disconnected | グレー | Disconnected |
| connecting | 黄色 | Connecting... |
| connected | 緑 | Connected |
| error | 赤 | Error |

### ボタン状態

#### 接続ボタン

| 接続状態 | 表示テキスト | 操作 |
|---------|------------|------|
| disconnected | "Connect" | クリックで接続開始 |
| connecting | "Connecting..." | 無効化 |
| connected | "Disconnect" | クリックで切断 |
| error | "Connect" | クリックで再接続 |

#### マイクボタン

| 接続状態 | 録音状態 | 表示テキスト | 操作 |
|---------|---------|------------|------|
| 未接続 | - | "Start Recording" | 無効化 |
| 接続中 | 停止 | "Start Recording" | クリックで録音開始 |
| 接続中 | 録音中 | "Stop Recording" | クリックで録音停止 |

### 技術仕様

#### 音声入力

| 項目 | 値 |
|-----|-----|
| サンプルレート | 24kHz（OpenAI 要求仕様） |
| チャンネル数 | モノラル |
| フォーマット | 16-bit PCM、Base64 エンコード |
| 処理方式 | ScriptProcessorNode によるリアルタイム処理 |

#### 音声出力

| 項目 | 値 |
|-----|-----|
| サンプルレート | 24kHz（OpenAI 出力仕様） |
| フォーマット | 16-bit PCM |
| 処理方式 | キュー方式による順次再生 |

#### WebSocket 認証

ブラウザの WebSocket API はヘッダーを直接設定できないため、サブプロトコルにエフェメラルキーを含める方式を使用。

### 関連ファイル

| ファイル | 役割 |
|---------|------|
| `app/openai-realtime/page.tsx` | ページエントリーポイント（SSR 無効化） |
| `components/OpenAIRealtimeClient.tsx` | UI コンポーネント |
| `hooks/useOpenAIRealtime.ts` | WebSocket 接続・音声処理フック |
| `types/openai.ts` | OpenAI Realtime API 関連の型定義 |
| `lib/api.ts` | エフェメラルトークン取得 API |
| `lib/audioProcessor.ts` | 音声処理ユーティリティ |

---

## Docs ページ

プロジェクトの `開発/` フォルダ内のドキュメントをブラウザで閲覧する機能を提供する。クエリパラメータ `?project=` で任意のプロジェクトパスを指定でき、異なるプロジェクトのドキュメントを表示できる。

### クエリパラメータ

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `project` | string | いいえ | 対象プロジェクトの絶対パス。指定時はそのパス配下の `開発/` フォルダを表示する。未指定時は Ghostrunner プロジェクトの `開発/` フォルダにフォールバックする |

### 構成要素

| 要素 | 役割 | 備考 |
|-----|------|------|
| タイトル | "開発ドキュメント" の表示 | `/docs` のみ |
| Home リンク | メインページへの遷移 | |
| Breadcrumb | パンくずナビゲーション | サブフォルダ/ファイル閲覧時のみ表示、`project` パラメータを引き回す |
| FolderList | フォルダ・ファイル一覧 | ディレクトリ表示時に表示、`project` パラメータを引き回す |
| MarkdownViewer | Markdown ファイルの内容表示 | ファイル表示時に表示 |

### コンポーネント

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| FolderList | `components/docs/FolderList.tsx` | フォルダ・ファイルの一覧表示、リンクに `?project=` を付与 |
| Breadcrumb | `components/docs/Breadcrumb.tsx` | パンくずナビゲーション、リンクに `?project=` を付与 |
| MarkdownViewer | `components/docs/MarkdownViewer.tsx` | Markdown ファイルのレンダリング（Tailwind Typography による prose スタイリング、highlight.js による構文ハイライト） |
| MermaidRenderer | `components/docs/MermaidRenderer.tsx` | Mermaid 図のレンダリング |

### パス解決とセキュリティ

| 項目 | 内容 |
|-----|------|
| ドキュメントルート | `{projectPath}/開発/` （`projectPath` 未指定時は `{cwd}/../開発/`） |
| パストラバーサル防止 | 解決後のパスがドキュメントルート配下であることを検証し、範囲外アクセスを拒否 |
| 隠しファイル除外 | `.` で始まるファイル・フォルダは一覧に表示しない |
| ソート順 | ディレクトリを先に表示、ファイルは名前降順 |

### 表示モード

| パスの種別 | 表示内容 |
|-----------|---------|
| ディレクトリ | FolderList でフォルダ・ファイルの一覧を表示 |
| ファイル（`.md`） | MarkdownViewer で Markdown の内容をレンダリング表示（見出し、テーブル、リスト、コードブロック（構文ハイライト付き）、引用、水平線、Mermaid 図に対応） |
| 存在しないパス | 404 Not Found |

### 関連ファイル

| ファイル | 役割 |
|---------|------|
| `app/docs/page.tsx` | Docs ルートページ（`searchParams` から `project` を取得） |
| `app/docs/[...path]/page.tsx` | サブパスページ（`params` と `searchParams` から `project` を取得） |
| `components/docs/FolderList.tsx` | フォルダ・ファイル一覧コンポーネント |
| `components/docs/Breadcrumb.tsx` | パンくずナビゲーションコンポーネント |
| `components/docs/MarkdownViewer.tsx` | Markdown レンダリングコンポーネント |
| `components/docs/MermaidRenderer.tsx` | Mermaid 図レンダリングコンポーネント |
| `lib/docs/fileSystem.ts` | ファイルシステム操作（パス解決、ディレクトリ取得、ファイル読み取り、パストラバーサル防止） |
| `app/globals.css` | Tailwind Typography プラグイン（`@plugin "@tailwindcss/typography"`）と highlight.js テーマ（`@import "highlight.js/styles/github.css"`）の読み込み |
