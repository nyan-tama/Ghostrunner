# 画面一覧

Ghost Runner は単一ページアプリケーション（SPA）として構成されており、メインページ内でコンポーネントの表示/非表示を切り替えることで、異なる状態を表現する。

## ページ一覧

| パス | コンポーネント | 役割 |
|-----|--------------|------|
| `/` | `app/page.tsx` | メインページ（コマンド入力と実行結果表示） |
| `/docs` | `app/docs/page.tsx` | 開発ドキュメントのルート表示（フォルダ一覧） |
| `/docs/[...path]` | `app/docs/[...path]/page.tsx` | 開発ドキュメントのサブフォルダ/ファイル表示 |
| `/gemini-live` | `app/gemini-live/page.tsx` | Gemini Live API を使用した音声 AI インターフェース |
| `/new` | `app/new/page.tsx` | プロジェクト作成（フォーム入力、進捗表示、完了） |
| `/openai-realtime` | `app/openai-realtime/page.tsx` | OpenAI Realtime API を使用した音声 AI インターフェース |

## メインページの構成要素

### ヘッダーエリア

| 要素 | 役割 | 備考 |
|-----|------|------|
| タイトル | "Ghost Runner" の表示 | |
| New Project リンク | プロジェクト作成ページへの遷移 | |
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
| EventItem | `components/EventItem.tsx` | 個別イベントの表示（Markdown レンダリング） |
| OutputText | `components/OutputText.tsx` | テキスト出力の Markdown レンダリング（太字、コード、リスト、テーブル等） |
| QuestionSection | `components/QuestionSection.tsx` | AIからの質問への回答UI（逐次表示） |
| PlanApproval | `components/PlanApproval.tsx` | 計画の承認/拒否ボタン |

#### 表示状態

1. **初期状態**: ProgressContainer は非表示
2. **実行中**: ローディング表示 + イベントリスト + 中断ボタン（Abort）
3. **質問待ち**: QuestionSection を表示（中断ボタンは非表示）
4. **計画承認待ち**: PlanApproval を表示（中断ボタンは非表示）
5. **完了**: 結果出力を Markdown レンダリングして表示（成功は緑、エラーは赤）
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

## New Project ページ

プロジェクトの新規作成機能を提供するページ。フォーム入力、SSE による進捗表示、完了後のアクションの3フェーズで構成される。

### 構成要素

| 要素 | 役割 | 備考 |
|-----|------|------|
| タイトル | "New Project" の表示 | |
| Back リンク | メインページへの遷移 | Link コンポーネント |
| メインコンテンツ | フェーズに応じたコンポーネントの切り替え表示 | form / creating / complete |

### フェーズ別表示

| フェーズ | 表示コンポーネント | 説明 |
|---------|-----------------|------|
| form | ProjectForm | プロジェクト名、概要、Data Services の入力 |
| error | ProjectForm + エラーメッセージ | エラー内容を赤背景で表示、フォームは入力値を保持 |
| creating | CreateProgress | 進捗バーとステップチェックリストの表示 |
| complete | CreateComplete | 成功メッセージとアクションボタンの表示 |

### コンポーネント

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| ProjectForm | `components/create/ProjectForm.tsx` | プロジェクト名、概要、Data Services の入力フォーム |
| ServiceSelector | `components/create/ServiceSelector.tsx` | Data Services のチェックボックス選択 |
| CreateProgress | `components/create/CreateProgress.tsx` | 進捗バーとステップチェックリスト |
| CreateComplete | `components/create/CreateComplete.tsx` | 完了画面（VS Code で開く、もう1つ作る） |

### カスタムフック

| フック | ファイル | 役割 |
|-------|---------|------|
| useProjectCreate | `hooks/useProjectCreate.ts` | SSE 通信と状態遷移管理（form / creating / complete / error） |
| useProjectValidation | `hooks/useProjectValidation.ts` | 300ms デバウンス付きプロジェクト名バリデーション |

### API 関数

| 関数 | ファイル | 役割 |
|-----|---------|------|
| validateProjectName | `lib/createApi.ts` | `GET /api/projects/validate?name=` でプロジェクト名の重複・パスを検証 |
| createProjectStream | `lib/createApi.ts` | `POST /api/projects/create/stream` で SSE ストリームによるプロジェクト作成 |
| openInVSCode | `lib/createApi.ts` | `POST /api/projects/open` で VS Code を開く |

### ProjectForm 詳細

#### フォーム要素

| 要素 | 必須 | 説明 |
|-----|------|------|
| Project Name | はい | プロジェクト名の入力。入力時に 300ms デバウンスでバリデーション API を呼び出す |
| Description | いいえ | プロジェクトの概要入力（textarea） |
| Data Services | いいえ | PostgreSQL + GORM / Cloudflare R2 + MinIO / Redis のチェックボックス選択 |
| Summary | - | 入力内容の確認表示（名前、概要、サービス、作成先パス） |
| Create Project ボタン | - | バリデーション成功時のみ有効 |

#### バリデーション表示

| 状態 | 表示内容 |
|-----|---------|
| 未入力 | 表示なし |
| バリデーション中 | "Validating..." （グレー） |
| 有効 | "Will be created at: {path}" （緑） |
| 無効 | エラーメッセージ（赤） |

#### Data Services 選択肢

| ID | ラベル | 説明 |
|----|-------|------|
| database | PostgreSQL + GORM | Database with migration support |
| storage | Cloudflare R2 / MinIO | Object storage for files |
| cache | Redis | In-memory cache and session store |

### CreateProgress 詳細

#### プログレスバー

進捗率（0-100%）をバーとパーセンテージで表示する。

#### ステップチェックリスト

| ステップ ID | ラベル |
|------------|-------|
| template_copy | Copy template files |
| placeholder_replace | Replace placeholders |
| env_create | Create .env file |
| dependency_install | Install dependencies |
| claude_assets | Copy Claude assets |
| claude_md | Generate CLAUDE.md |
| devtools_link | Register with devtools |
| git_init | Initialize git repository |
| server_start | Start development server |
| health_check | Health check |

#### ステップ状態

| 状態 | アイコン | テキスト色 |
|-----|---------|----------|
| pending | 空の丸（グレー枠） | グレー |
| active | スピナー（青） | 青・太字 |
| done | チェックマーク丸（緑） | 緑 |
| error | バツ丸（赤） | 赤 |

### CreateComplete 詳細

#### 表示要素

| 要素 | 説明 |
|-----|------|
| 成功アイコン | 緑のチェックマーク |
| タイトル | "Project Created" |
| サブテキスト | "{project.name} is ready for development" |
| プロジェクトパス | 作成されたプロジェクトの絶対パス表示 |

#### アクションボタン

| ボタン | 処理 |
|-------|------|
| Open in VS Code | `POST /api/projects/open` でプロジェクトを VS Code で開く |
| Create Another | フォームに戻る（入力値はクリア） |

### 関連ファイル

| ファイル | 役割 |
|---------|------|
| `app/new/page.tsx` | ページエントリーポイント |
| `components/create/ProjectForm.tsx` | 入力フォーム |
| `components/create/ServiceSelector.tsx` | Data Services チェックボックス |
| `components/create/CreateProgress.tsx` | 進捗チェックリスト |
| `components/create/CreateComplete.tsx` | 完了画面 |
| `hooks/useProjectCreate.ts` | SSE 通信と状態遷移管理 |
| `hooks/useProjectValidation.ts` | デバウンス付きバリデーション |
| `lib/createApi.ts` | API 呼び出し関数（validate, create/stream, open） |
| `types/index.ts` | `DataService`, `CreateProjectRequest`, `CreateProgressEvent`, `CreateStep`, `CreatedProject`, `CreatePhase` 型 |

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

---

## New Project ページ

`/new` はプロジェクトを新規作成するページ。フォーム入力、進捗表示、完了表示の3フェーズを単一ページ内で切り替える。

### 構成要素

| 要素 | 役割 | 備考 |
|-----|------|------|
| タイトル | "New Project" の表示 | |
| Back リンク | メインページへの遷移 | |
| メインコンテンツ | フェーズに応じた表示切替 | form / creating / complete / error |

### フェーズ

ページ内の表示は `phase` 状態に応じて切り替わる。

| フェーズ | 表示内容 | 説明 |
|---------|---------|------|
| form | ProjectForm | プロジェクト名、説明、サービス選択のフォーム |
| error | エラーメッセージ + ProjectForm | エラー表示付きフォーム（入力値は保持） |
| creating | CreateProgress | ステップごとの進捗表示（プログレスバー付き） |
| complete | CreateComplete | 成功メッセージ、パス表示、アクションボタン |

### コンポーネント

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| ProjectForm | `components/create/ProjectForm.tsx` | プロジェクト名、説明、サービス選択のフォーム入力 |
| ServiceSelector | `components/create/ServiceSelector.tsx` | データサービス（PostgreSQL, R2/MinIO, Redis）のチェックボックス選択 |
| CreateProgress | `components/create/CreateProgress.tsx` | 作成ステップごとの進捗表示（アイコン + プログレスバー） |
| CreateComplete | `components/create/CreateComplete.tsx` | 作成完了後の成功メッセージと操作ボタン |

### フォーム要素

| 要素 | 必須 | 説明 |
|-----|------|------|
| Project Name | はい | プロジェクト名（300ms デバウンスでリアルタイムバリデーション） |
| Description | いいえ | プロジェクトの概要 |
| Data Services | いいえ | PostgreSQL + GORM / Cloudflare R2・MinIO / Redis から複数選択 |
| Summary | - | 入力内容の確認表示（名前、説明、サービス、作成先パス） |

### プロジェクト名バリデーション

入力中にリアルタイムでバックエンド API を呼び出し、プロジェクト名の妥当性と作成先パスを検証する。

| 状態 | 表示内容 |
|-----|---------|
| バリデーション中 | "Validating..." （グレー） |
| 有効 | "Will be created at: {path}" （緑） |
| 無効 | エラーメッセージ （赤） |
| 未入力 | 何も表示しない |

### 作成ステップ

プロジェクト作成中に表示されるステップ一覧。各ステップは pending / active / done / error の状態を持つ。

| ステップID | ラベル |
|-----------|-------|
| template_copy | Copy template files |
| placeholder_replace | Replace placeholders |
| env_create | Create .env file |
| dependency_install | Install dependencies |
| claude_assets | Copy Claude assets |
| claude_md | Generate CLAUDE.md |
| devtools_link | Register with devtools |
| git_init | Initialize git repository |
| server_start | Start development server |
| health_check | Health check |

### ステップアイコン

| 状態 | アイコン | 色 |
|-----|---------|---|
| pending | 空の丸 | グレー |
| active | 回転するスピナー | 青 |
| done | チェックマーク丸 | 緑 |
| error | バツマーク丸 | 赤 |

### 完了後のアクション

| ボタン | 処理 |
|-------|------|
| Open in VS Code | バックエンド API 経由で VS Code を起動 |
| Create Another | フォームをリセットして新規作成 |

### カスタムフック

| フック | ファイル | 役割 |
|-------|---------|------|
| useProjectCreate | `hooks/useProjectCreate.ts` | SSE 通信によるプロジェクト作成の状態管理（phase, steps, progress） |
| useProjectValidation | `hooks/useProjectValidation.ts` | 300ms デバウンス付きプロジェクト名バリデーション |

### API

| 関数 | ファイル | エンドポイント | 説明 |
|-----|---------|-------------|------|
| validateProjectName | `lib/createApi.ts` | `GET /api/projects/validate?name=` | プロジェクト名の妥当性検証 |
| createProjectStream | `lib/createApi.ts` | `POST /api/projects/create/stream` | SSE でプロジェクト作成（進捗イベント受信） |
| openInVSCode | `lib/createApi.ts` | `POST /api/projects/open` | VS Code でプロジェクトを開く |

### 関連ファイル

| ファイル | 役割 |
|---------|------|
| `app/new/page.tsx` | ページエントリーポイント |
| `components/create/ProjectForm.tsx` | フォーム入力コンポーネント |
| `components/create/ServiceSelector.tsx` | サービス選択コンポーネント |
| `components/create/CreateProgress.tsx` | 進捗表示コンポーネント |
| `components/create/CreateComplete.tsx` | 完了画面コンポーネント |
| `hooks/useProjectCreate.ts` | プロジェクト作成状態管理フック |
| `hooks/useProjectValidation.ts` | プロジェクト名バリデーションフック |
| `lib/createApi.ts` | プロジェクト作成関連 API 関数 |
| `types/index.ts` | `CreatePhase`, `CreateStep`, `CreatedProject`, `DataService`, `CreateProjectRequest`, `CreateProgressEvent` 型定義 |
