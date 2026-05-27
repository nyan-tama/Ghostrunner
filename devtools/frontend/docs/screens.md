# 画面一覧

Ghost Runner は単一ページアプリケーション（SPA）として構成されており、メインページ内でコンポーネントの表示/非表示を切り替えることで、異なる状態を表現する。

## ページ一覧

| パス | コンポーネント | 役割 |
|-----|--------------|------|
| `/` | `app/page.tsx` | メインページ（コマンド入力と実行結果表示） |
| `/dashboard` | `app/dashboard/page.tsx` | 統括ダッシュボード（全プロジェクト横断把握・チャット・音声読み上げ） |
| `/docs` | `app/docs/page.tsx` | 開発ドキュメントのルート表示（フォルダ一覧） |
| `/docs/[...path]` | `app/docs/[...path]/page.tsx` | 開発ドキュメントのサブフォルダ/ファイル表示 |
| `/gemini-live` | `app/gemini-live/page.tsx` | Gemini Live API を使用した音声 AI インターフェース |
| `/new` | `app/new/page.tsx` | プロジェクト作成（フォーム入力、進捗表示、完了） |
| `/openai-realtime` | `app/openai-realtime/page.tsx` | OpenAI Realtime API を使用した音声 AI インターフェース |
| `/patrol` | （リダイレクトのみ） | `/dashboard` への 308 Permanent Redirect。旧ブックマーク救済用。実体ページ・hooks・API は当面温存だが、トップからの導線は廃止済み |

## メインページの構成要素

### ヘッダーエリア

| 要素 | 役割 | 備考 |
|-----|------|------|
| タイトル | "Ghost Runner" の表示 | |
| 統括リンク | 統括ダッシュボード（`/dashboard`）への遷移。blue 系ボタン（`bg-blue-100 text-blue-700`）。`title="統括ダッシュボード"` |
| New Project リンク | プロジェクト作成ページへの遷移 | |
| Docs リンク | 開発ドキュメントページへの遷移 | projectPath を `?project=` クエリで引き渡し |
| Gemini Live リンク | Gemini Live ページへの遷移 | |
| OpenAI Realtime リンク | OpenAI Realtime ページへの遷移 | |
| Restart Servers ボタン | バックエンド・フロントエンドサーバーの再起動 | 開発環境のみ表示 |

### 削除モード

画面左下の「削除モード」ボタンでコマンド入力エリアとプロジェクト削除リストを切り替える。

| 状態 | 表示内容 | 備考 |
|-----|---------|------|
| 通常モード | CommandForm（コマンド入力エリア） | デフォルト状態 |
| 削除モード | ProjectDeleteList（プロジェクト削除リスト） | CommandForm は非表示 |

#### 削除モードボタン

| 要素 | 役割 | 備考 |
|-----|------|------|
| 削除モードボタン | 通常モードと削除モードの切り替え | 画面左下に固定配置。削除モード ON 時は赤背景 |

#### プロジェクト削除リスト

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| ProjectDeleteList | `components/ProjectDeleteList.tsx` | 登録済みプロジェクトの一覧表示と個別削除 |

#### 表示内容

- プロジェクト名とパスの一覧
- 各プロジェクトの横に「削除」ボタン
- 削除中のプロジェクトはボタンが「削除中...」に変化し disabled

#### 削除処理

| 操作 | 処理 |
|-----|------|
| 削除ボタンクリック | `window.confirm` で確認ダイアログを表示 |
| 確認ダイアログ OK | `POST /api/projects/destroy` でプロジェクト削除、プロジェクト一覧を再取得 |
| 確認ダイアログ キャンセル | 何もしない |
| 削除成功（選択中のプロジェクト） | プロジェクト選択をリセット |
| 削除失敗 | `alert` でエラーメッセージを表示 |

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
| usePatrol | `hooks/usePatrol.ts` | 巡回ダッシュボードの状態管理（プロジェクト登録・巡回制御・回答送信） |
| usePatrolSSE | `hooks/usePatrolSSE.ts` | 巡回ダッシュボード用 SSE 接続管理（EventSource による常時接続、自動再接続） |

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

---

## 巡回ダッシュボードページ（旧・リダイレクト対象）

`/patrol` は統括ダッシュボード（`/dashboard`）に役割を引き継いだ旧ページ。トップ `/` からの導線は撤去済みで、`/patrol` への直接アクセスは `next.config.ts` の `redirects()` により `/dashboard` へ 308 Permanent Redirect される。

実体のページ・コンポーネント（`app/patrol/`、`components/patrol/`）・hooks（`hooks/usePatrol.ts`、`hooks/usePatrolSSE.ts`）・API ハンドラ（`/api/patrol/*`）は段階的廃止のため当面温存されているが、新規導線は持たない。以下は旧仕様の参考記述。

### 構成要素

| 要素 | 役割 | 備考 |
|-----|------|------|
| タイトル | "巡回ダッシュボード" の表示 | |
| Back リンク | メインページへの遷移 | |
| エラー表示 | エラーメッセージの表示 | エラー発生時のみ |
| PatrolHeader | 巡回の開始/停止、ポーリングON/OFF、SSE接続状態の表示 | |
| ProjectRegister | 巡回対象プロジェクトの登録UI | |
| ProjectCard 一覧 | 各プロジェクトのステータスカード | グリッドレイアウト（1列/2列） |

### コンポーネント

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| PatrolHeader | `components/patrol/PatrolHeader.tsx` | 巡回開始/停止ボタン、ポーリングチェックボックス、SSE 接続状態インジケーター |
| ProjectCard | `components/patrol/ProjectCard.tsx` | プロジェクトのステータス表示（名前、パス、ステータスバッジ、コミット情報、タスク数、エラー、回答フォーム） |
| AnswerForm | `components/patrol/AnswerForm.tsx` | 承認待ちプロジェクトへの回答フォーム（選択肢ボタン、自由テキスト入力） |
| ProjectRegister | `components/patrol/ProjectRegister.tsx` | 巡回対象プロジェクトの追加/解除UI（バックエンドからプロジェクト一覧を取得して選択） |

### カスタムフック

| フック | ファイル | 役割 |
|-------|---------|------|
| usePatrol | `hooks/usePatrol.ts` | プロジェクト一覧・状態の管理、巡回操作（開始/停止/回答/ポーリング切替）の実行 |
| usePatrolSSE | `hooks/usePatrolSSE.ts` | EventSource による SSE 常時接続（自動再接続、指数バックオフ、最大10回リトライ） |

### API 関数

| 関数 | ファイル | エンドポイント | 説明 |
|-----|---------|-------------|------|
| fetchPatrolProjects | `lib/patrolApi.ts` | `GET /api/patrol/projects` | 登録済みプロジェクト一覧の取得 |
| registerPatrolProject | `lib/patrolApi.ts` | `POST /api/patrol/projects` | プロジェクトの巡回対象登録 |
| removePatrolProject | `lib/patrolApi.ts` | `POST /api/patrol/projects/remove` | プロジェクトの巡回対象解除 |
| startPatrol | `lib/patrolApi.ts` | `POST /api/patrol/start` | 巡回の開始 |
| stopPatrol | `lib/patrolApi.ts` | `POST /api/patrol/stop` | 巡回の停止 |
| sendPatrolAnswer | `lib/patrolApi.ts` | `POST /api/patrol/resume` | 承認待ちプロジェクトへの回答送信 |
| fetchPatrolStates | `lib/patrolApi.ts` | `GET /api/patrol/states` | 全プロジェクトの状態取得 |
| fetchPatrolScan | `lib/patrolApi.ts` | `POST /api/patrol/scan` | プロジェクトのスキャン実行 |
| startPolling | `lib/patrolApi.ts` | `POST /api/patrol/polling/start` | ポーリングの開始 |
| stopPolling | `lib/patrolApi.ts` | `POST /api/patrol/polling/stop` | ポーリングの停止 |

### SSE 接続（EventSource）

`/api/patrol/stream` に EventSource で常時接続し、プロジェクト状態の変更をリアルタイム受信する。

| 設定 | 値 |
|-----|-----|
| 初回リトライ待機 | 1秒 |
| 最大リトライ待機 | 30秒 |
| 最大リトライ回数 | 10回 |
| バックオフ方式 | 指数バックオフ（2^n * 初回待機） |

### SSE イベント種別

| イベント種別 | 説明 | 処理 |
|------------|------|------|
| project_started | プロジェクトの巡回が開始された | isRunning を true に、該当プロジェクトの state を更新 |
| project_question | プロジェクトが承認待ち状態になった | 該当プロジェクトの state を更新（質問情報を含む） |
| project_completed | プロジェクトの巡回が完了した | 該当プロジェクトの state を更新 |
| project_error | プロジェクトの巡回でエラーが発生した | 該当プロジェクトの state を更新 |
| scan_completed | 全プロジェクトのスキャンが完了した | 該当プロジェクトの state を更新 |

### SSE 接続状態インジケーター

| 状態 | ドット色 | 表示テキスト |
|-----|---------|------------|
| connected | 緑 | 接続済み |
| connecting | 黄色（点滅） | 接続中... |
| disconnected | 赤 | 切断 |

### プロジェクトステータスバッジ

| ステータス | 表示ラベル | バッジ色 |
|-----------|----------|---------|
| idle | 待機中 | グレー |
| running | 実行中 | 青（点滅ドット付き） |
| waiting_approval | 承認待ち | 黄 |
| queued | キュー待ち | グレー（破線ドット付き） |
| completed | 完了 | 緑 |
| error | エラー | 赤 |

### ProjectCard の表示内容

| 要素 | 表示条件 | 説明 |
|-----|---------|------|
| プロジェクト名 | 常時 | `project.name` を表示 |
| プロジェクトパス | 常時 | `project.path` を表示 |
| ステータスバッジ | 常時 | 上記ステータスバッジ |
| 解除ボタン | 常時 | 巡回対象からプロジェクトを解除 |
| 最近のコミット | `recent_commits` が存在する場合 | 最大5件のコミットメッセージを表示 |
| 実装待ちタスク数 | `pending_tasks > 0` の場合 | タスク数を表示 |
| エラーメッセージ | `error` が存在する場合 | エラー内容を赤背景で表示 |
| 回答フォーム | `status === "waiting_approval"` かつ `question` がある場合 | AnswerForm コンポーネントを表示 |

### AnswerForm の動作

| 操作 | 単一選択モード | 複数選択モード |
|-----|-------------|-------------|
| 選択肢ボタンクリック | 即座に選択値で回答を送信 | 選択/解除をトグル |
| 送信ボタンクリック | 自由テキスト入力値で回答を送信 | 選択済みの値をカンマ区切りで送信（自由テキスト入力値が優先） |
| テキスト入力 + Enter | 自由テキスト値で回答を送信 | 同上 |

### ProjectRegister の動作

| 操作 | 処理 |
|-----|------|
| 「追加」ボタンクリック | プロジェクト選択パネルを展開（バックエンドからプロジェクト一覧を取得） |
| プロジェクト選択 | 巡回対象に登録し、パネルを閉じる |
| 「閉じる」ボタンクリック | パネルを閉じる |

### 関連ファイル

| ファイル | 役割 |
|---------|------|
| `app/patrol/page.tsx` | ページエントリーポイント |
| `components/patrol/PatrolHeader.tsx` | 巡回制御ヘッダー |
| `components/patrol/ProjectCard.tsx` | プロジェクトステータスカード |
| `components/patrol/AnswerForm.tsx` | 承認待ち回答フォーム |
| `components/patrol/ProjectRegister.tsx` | プロジェクト登録UI |
| `hooks/usePatrol.ts` | 巡回状態管理フック |
| `hooks/usePatrolSSE.ts` | SSE 接続管理フック |
| `lib/patrolApi.ts` | 巡回 API 関数 |
| `types/patrol.ts` | `PatrolProject`, `PatrolProjectState`, `PatrolProjectStatus`, `PatrolSSEEvent`, `PatrolSSEEventType` 型定義 |

---

## 統括ダッシュボードページ

`/dashboard` は全プロジェクトの状態を横断把握する統括ダッシュボード。開発カンバンと運用状態をカード形式で一覧表示し、even-terminal 経由のチャットで指示・問い合わせを行える。応答は Web Speech API による音声読み上げに対応する。

トップ `/` のヘッダにある blue 系「統括」ボタンから遷移する。旧 `/patrol` URL も `/dashboard` へ 308 リダイレクトされ、ここに集約される。

### 構成要素

| 要素 | 役割 | 備考 |
|-----|------|------|
| エラー表示 | 集約エラーメッセージの表示 | チャット・ダッシュボード・TTS のいずれかでエラー発生時に表示 |
| DashboardHeader | ポーリングトグル、TTS トグル、「状況は？」ボタン | |
| DashboardCard 一覧 | 各プロジェクトのステータスカード | 開発カンバン、運用状態、未回答質問を表示 |
| ChatTranscript | even-terminal からの応答テキスト表示 | |
| ChatInput | テキスト入力による指示送信 | Enter キーで送信（Shift+Enter で改行） |
| 最終更新時刻 | ダッシュボードデータの取得時刻 | |

### コンポーネント

| コンポーネント | ファイル | 役割 |
|--------------|---------|------|
| DashboardHeader | `components/dashboard/DashboardHeader.tsx` | ヘッダー（タイトル、ポーリング・TTS トグル、把握ボタン） |
| DashboardCard | `components/dashboard/DashboardCard.tsx` | プロジェクトカード（カンバン、運用、未回答を集約表示） |
| AccentBar | `components/dashboard/AccentBar.tsx` | カード左端のアテンションカラーバー |
| DevSummary | `components/dashboard/DevSummary.tsx` | 開発カンバン件数（レビュー・待ち・実行中・完了）の表示 |
| OpsEntryComponent | `components/dashboard/OpsEntryComponent.tsx` | 運用エントリの状態表示（進捗、本日実績、stale 検知） |
| UnansweredList | `components/dashboard/UnansweredList.tsx` | 未回答質問の一覧と展開式回答フォーム |
| DashboardAnswerForm | `components/dashboard/DashboardAnswerForm.tsx` | 未回答質問への回答送信フォーム |
| ChatTranscript | `components/dashboard/ChatTranscript.tsx` | チャット応答テキストの表示（ステータス表示付き） |
| ChatInput | `components/dashboard/ChatInput.tsx` | テキスト入力フォーム（Enter 送信、Shift+Enter 改行） |
| PollingToggle | `components/dashboard/PollingToggle.tsx` | 自動更新 ON/OFF ボタン |
| TTSToggle | `components/dashboard/TTSToggle.tsx` | 音声読み上げ ON/OFF ボタン |
| ProgressGraspButton | `components/dashboard/ProgressGraspButton.tsx` | 「状況は？」ショートカットボタン |
| SessionPicker | `components/dashboard/SessionPicker.tsx` | チャットセッションの切替ドロップダウン + 「新規 session」ボタン |
| SidCopyButton | `components/dashboard/SidCopyButton.tsx` | 現在のセッション ID をクリップボードへコピー（フォールバック対応） |
| ConnectionIndicator | `components/dashboard/ConnectionIndicator.tsx` | SSE 接続状態（live / reconnecting / offline）のドット表示 |

### カスタムフック

| フック | ファイル | 役割 |
|-------|---------|------|
| useDashboard | `hooks/useDashboard.ts` | ダッシュボード状態のポーリング取得、自動更新の ON/OFF 制御、visibilitychange 連動 |
| useChat | `hooks/useChat.ts` | even-terminal への SSE 接続・プロンプト送信・応答テキスト蓄積、セッション自動復元 |
| useTTS | `hooks/useTTS.ts` | ElevenLabs サーバー TTS（主経路）と Web Speech API（フォールバック）の切り替え管理。`<audio>` 要素・AbortController・Blob URL のライフサイクル管理、iOS Safari の autoplay unlock（prime） |

#### TTS 関連の補助モジュール

useTTS は内部実装を `lib/tts/` 配下の純関数群に分離している。dashboard/page.tsx 側からは `useTTS` の公開 API（`speak`, `cancel`, `enabled`, `setEnabled`, `isSpeaking`, `error`, `prime`）のみが見える。

| ファイル | 役割 |
|---------|------|
| `lib/tts/elevenlabsClient.ts` | `POST /api/tts` を叩き、レスポンスを `audio/*` Blob として返す。HTTP エラー / Content-Type 欠落・不正 / 空 Body は `TTSError` を throw（`AbortError` はそのまま伝播） |
| `lib/tts/webSpeech.ts` | Web Speech API ラッパー。`speakWithWebSpeech` / `cancelWebSpeech` / `primeWebSpeech` を提供。日本語 voice 選択、`voiceschanged` 購読、iOS Safari の cancel→speak 50ms 遅延、prime 用の無音 utterance を含む |
| `lib/tts/silentMp3.ts` | iOS Safari の `<audio>` autoplay unlock 用、約 0.1 秒の無音 MP3 を data URL でエクスポート（`SILENT_MP3_DATA_URL`） |
| `lib/tts/errors.ts` | `TTSError` クラスと `TTSFallbackReason` union（`http_error` / `missing_content_type` / `invalid_content_type` / `empty_body` / `audio_error` / `network_error`） |
| `types/tts.ts` | `TTSRequest`（`{ text }`）型定義。voice_id / model_id は backend env で固定 |

### API 関数

| 関数 | ファイル | エンドポイント | 説明 |
|-----|---------|-------------|------|
| fetchDashboardState | `lib/dashboardApi.ts` | `GET /api/dashboard/state` | 全プロジェクトの集約状態を取得 |
| submitAnswer | `lib/dashboardApi.ts` | `POST /api/dashboard/answer` | 未回答質問への回答を書き戻し |
| listSessions | `lib/chatApi.ts` | `GET /api/sessions` | even-terminal のセッション一覧取得 |
| sendPrompt | `lib/chatApi.ts` | `POST /api/prompt` | even-terminal へプロンプト送信 |
| openEventStream | `lib/chatApi.ts` | `GET /api/events` | even-terminal の SSE イベントストリーム接続 |
| requestTTS | `lib/tts/elevenlabsClient.ts` | `POST /api/tts` | ElevenLabs TTS Blob（`audio/*`）取得。失敗時は `TTSError` を throw |

### API プロキシ（next.config.ts rewrites）

even-terminal（ポート 3456）と devtools バックエンド（ポート 8888）の API を Next.js の rewrites でプロキシする。

| ソース | プロキシ先 | 対象 |
|-------|----------|------|
| `/api/tts` | `http://localhost:8888/api/tts` | devtools バックエンド（ElevenLabs プロキシ）。catch-all より前に明示エントリを置き、将来 even-terminal へのルート分岐が増えた際の順序ミスを防ぐ |
| `/api/prompt` | `http://localhost:3456/api/prompt` | even-terminal |
| `/api/events` | `http://localhost:3456/api/events` | even-terminal |
| `/api/sessions` | `http://localhost:3456/api/sessions` | even-terminal |
| `/api/:path*` | `http://localhost:8888/api/:path*` | devtools バックエンド |

### DashboardCard の表示内容

| 要素 | 表示条件 | 説明 |
|-----|---------|------|
| プロジェクト名 | 常時 | `project.name` を表示。自身のプロジェクトは `(self)` ラベル付き |
| AccentBar | 常時 | アテンションレベルに応じた左端カラーバー |
| DevSummary | 常時 | カンバン件数（レビュー・待ち・実行中・完了） |
| 警告 | `warnings` が存在する場合 | オレンジ色で警告メッセージを表示 |
| 運用エントリ | `opsOptedIn` かつ `ops` が存在する場合 | 運用状態（進捗、本日実績、stale、連続エラー）を表示 |
| 未回答リスト | `unanswered` が存在する場合 | 展開式の回答フォーム付き未回答質問一覧 |

### AccentBar のカラー

| attention | 未回答あり | 色 |
|-----------|----------|---|
| required | - | 赤 |
| progress | はい | 黄 |
| progress | いいえ | 青 |
| watching | - | グレー |

### OpsEntryComponent の表示

| 要素 | 表示条件 | 説明 |
|-----|---------|------|
| 種別 / アカウント | 常時 | `kind / account` 形式で表示 |
| ステータスバッジ | 常時 | running は青、その他はグレー |
| 進捗 | `progress` がある場合 | `index/total` 形式 |
| 本日実績 | `today` がある場合 | `count/target` 形式 |
| 統計 | `stats` がある場合 | 実行・既存・skip・err の件数 |
| stale 警告 | `stale === true` | 赤文字で「N時間無更新（実行停止疑い）」 |
| 連続エラー | `consecutiveErrors >= 3` | 赤文字で「連続エラー: N回」 |

### ポーリング設定

| 項目 | 値 |
|-----|-----|
| ポーリング間隔 | 15秒 |
| 既定状態 | ON（自動更新） |
| 永続化 | localStorage に保存 |
| visibilitychange | タブ非表示でインターバル停止、表示復帰で即時取得＋インターバル再開 |

### チャット（even-terminal SSE）

| 項目 | 値 |
|-----|-----|
| 接続方式 | EventSource による SSE 常時接続 |
| セッション復元 | localStorage からセッションIDを復元、なければ一覧取得して最新を選択 |
| 再接続 | 最大10回、指数バックオフ（1秒-8秒） |
| 無音タイムアウト | 3秒間イベントなしで完了扱い |
| visibilitychange | タブ非表示で SSE 切断、表示復帰で再接続 |
| セッション無効時 | 4xx レスポンスでセッション一覧を再取得し、リトライ1回 |

### TTS（ElevenLabs 主経路 + Web Speech フォールバック）

主経路は ElevenLabs のサーバー TTS（`POST /api/tts` → `audio/mpeg` Blob → `<audio>` で再生）。失敗時は Web Speech API に自動降格し、`error` に `"ElevenLabs 接続失敗。Web Speech に降格しました"` をセットしてユーザーに通知する。

#### 主経路（ElevenLabs）

| 項目 | 値 |
|-----|-----|
| エンドポイント | `POST /api/tts`（`{ text }` を JSON で送信） |
| レスポンス | `audio/*` Blob（典型: `audio/mpeg`） |
| 再生方法 | `URL.createObjectURL(blob)` で `<audio>.src` に紐付け、`audio.play()` |
| 音声 | Romaco（voice_id / model_id は backend 側 env で固定） |
| 出力 | AirPods など Bluetooth デバイスにも乗る（`<audio>` 経路のため OS 出力ルーティング準拠） |
| キャンセル | `AbortController.abort()` で進行中の fetch を中断 |
| `<audio>` 要素 | マウント時に 1 つだけ生成して使い回す（iOS Safari の unlock 状態を維持するため） |

#### フォールバック経路（Web Speech）

| 項目 | 値 |
|-----|-----|
| 音声合成 | `SpeechSynthesisUtterance` |
| 音声選択 | 日本語（`ja` プレフィックス）の音声を自動選択、`voiceschanged` で再選択 |
| 起動条件 | ElevenLabs 経路の fetch 失敗 / HTTP 4xx-5xx / Content-Type 欠落・不正 / 空 Blob / `audio.onerror` 発火 / `audio.play()` の reject |
| iOS Safari 対策 | cancel → speak 間に 50ms の遅延を挿入 |
| 既知制約 | iOS Safari の SpeechSynthesis は内蔵スピーカー固定（Bluetooth に乗らない）。Romaco の声は AirPods に乗るが、フォールバック中は内蔵スピーカーから出る |

#### 共通仕様

| 項目 | 値 |
|-----|-----|
| 既定状態 | OFF |
| 永続化 | localStorage（`ghostrunner_tts_enabled`） |
| `isSpeaking` の更新 | ElevenLabs 経路は `<audio>.onplaying`（実再生開始）、Web Speech 経路は `utterance.onstart` |
| `error` のクリア | 次回 `speak()` で実再生が始まったタイミング（`<audio>.onplaying`）に `null` に戻る。`play` メソッドの resolve ではなく実再生開始イベントで判定 |
| `prime()` の役割 | iOS Safari の autoplay 制約対策。ユーザージェスチャ（タップ / トグル）の同期スコープで呼ぶ |

#### prime() の autoplay unlock パターン

ユーザージェスチャの同期スコープ内で、`<audio>` 要素と Web Speech の両方を unlock する。

1. `<audio>.muted = true` にする（無音再生時のクリック音漏れ防止）
2. `<audio>.src = SILENT_MP3_DATA_URL`（約 0.1 秒の無音 MP3 data URL）をセット
3. `<audio>.play()` を呼ぶ（同期スコープ内）
4. `play()` の resolve 後に `pause()` → `currentTime = 0` → `muted = false`
5. `play()` の reject 時は `muted = false` に戻し、`error` に「音声再生の準備に失敗しました」をセット
6. 並行して `primeWebSpeech()` を呼び、Web Speech 側も同期スコープ内で unlock（無音 utterance を `volume=0, rate=10, " "` で speak）

`prime()` 冒頭では `<audio>` の `onplaying` / `onended` / `onerror` ハンドラを `null` に戻し、残留 Blob URL を revoke する（前回 `speak()` のハンドラが残ったまま SILENT_MP3 のデコード失敗で `handleError` が発火し、`triggerFallback` が空文字テキストで二重発火するのを防ぐ）。

#### prime() の呼び出し箇所

| 箇所 | 役割 |
|-----|------|
| TTS トグル ON | `setEnabled(true)` 内で自動呼び出し |
| チャット送信ボタン | `handleChatSend` で `tts.prime()` → `chat.send(text)` |
| 「状況は？」ボタン | `handleGrasp` で `tts.prime()` → `chat.send("状況は？")` → `dashboard.refresh()` |

### SessionPicker

| 項目 | 値 |
|-----|-----|
| 表示形式 | Tailwind カスタムドロップダウン（`<select>` ではなくタップ最適化） |
| 各エントリ | `title`（なければ `id` 先頭 8 文字） + 相対時刻 + `status` |
| 「新規 session」 | 一覧最上部のボタン。押すと `startNewSession` が呼ばれ、次の `send` で sessionId 省略 POST が走る |
| 展開時の挙動 | `onOpen` で親が `fetchSessions` を呼び、最新の一覧で再描画 |
| 空リスト | 「セッションなし」を表示 |
| current ハイライト | 現在 session に `aria-selected="true"` と青系背景 |
| current 切替時 | `onSwitch(sid)` 経由で `useChat.switchSession`、その内部で `onSessionSwitch` コールバック（dashboard では `tts.cancel` を渡す）が走る |
| disabled | `chat.status === "busy"` の間は trigger ボタン disabled |

### SidCopyButton

| 項目 | 値 |
|-----|-----|
| 表示 | 通常「SID」、コピー成功直後の 2 秒間「Copied」 |
| 動作 | `navigator.clipboard.writeText` を試行 → 失敗時 `document.execCommand('copy')` フォールバック → さらに失敗で `<input readonly>` を露出 |
| disabled | `sessionId === null` のとき |
| secure context 制約 | HTTP（Tailscale 経由）では `navigator.clipboard` が動かないため、必ずフォールバックが必要 |

### ConnectionIndicator

| 状態 | ドット色 | ラベル | 説明 |
|------|---------|--------|------|
| live | 緑（実線） | 接続 | SSE open 中 |
| reconnecting | 黄（点滅） | 再接続 | `onerror` 発生後、指数バックオフで待機中 |
| offline | グレー | 切断 | バックオフ 10 回上限到達、または `visibilitychange` で hidden |

`useChat.connectionState` を購読し、ヘッダ右端に小さく表示する。

### iOS 背景復帰時の整合性（FE-17）

| 項目 | 値 |
|-----|-----|
| トリガー | `visibilitychange` で `visible` 復帰 |
| 動作 | `getHistory(sessionId, 5)` を 1 回叩き、取得したアシスタント応答テキストで transcript を上書き |
| TTS | 復帰時には呼ばない（autoplay 制約のため） |
| 履歴取得失敗時 | 黙ってスキップ（`error` にはしない、SSE 再接続は継続） |

### 永続化データ

| キー | 保存場所 | 内容 |
|-----|---------|------|
| `ghostrunner_polling_enabled` | localStorage | ポーリングトグル状態 |
| `ghostrunner_tts_enabled` | localStorage | TTS トグル状態 |
| `ghostrunner_active_session_id` | localStorage | チャットセッションID |

### 関連ファイル

| ファイル | 役割 |
|---------|------|
| `app/dashboard/page.tsx` | ページエントリーポイント |
| `components/dashboard/DashboardHeader.tsx` | ヘッダーコンポーネント |
| `components/dashboard/DashboardCard.tsx` | プロジェクトカードコンポーネント |
| `components/dashboard/AccentBar.tsx` | アテンションカラーバー |
| `components/dashboard/DevSummary.tsx` | 開発カンバンサマリー |
| `components/dashboard/OpsEntryComponent.tsx` | 運用エントリ表示 |
| `components/dashboard/UnansweredList.tsx` | 未回答質問一覧 |
| `components/dashboard/DashboardAnswerForm.tsx` | 回答フォーム |
| `components/dashboard/ChatTranscript.tsx` | チャット応答表示 |
| `components/dashboard/ChatInput.tsx` | チャット入力フォーム |
| `components/dashboard/PollingToggle.tsx` | ポーリングトグル |
| `components/dashboard/TTSToggle.tsx` | TTS トグル |
| `components/dashboard/ProgressGraspButton.tsx` | 把握ショートカットボタン |
| `components/dashboard/SessionPicker.tsx` | チャットセッション切替ドロップダウン |
| `components/dashboard/SidCopyButton.tsx` | セッション ID クリップボードコピー |
| `components/dashboard/ConnectionIndicator.tsx` | SSE 接続状態ドット |
| `hooks/useDashboard.ts` | ダッシュボードポーリングフック |
| `hooks/useChat.ts` | チャット SSE フック |
| `hooks/useTTS.ts` | TTS フック（ElevenLabs 主経路 + Web Speech フォールバック） |
| `lib/dashboardApi.ts` | ダッシュボード API 関数 |
| `lib/chatApi.ts` | チャット API 関数 |
| `lib/tts/elevenlabsClient.ts` | ElevenLabs TTS リクエスト（`POST /api/tts`、Blob 返却） |
| `lib/tts/webSpeech.ts` | Web Speech API ラッパー（`speakWithWebSpeech` / `cancelWebSpeech` / `primeWebSpeech`） |
| `lib/tts/silentMp3.ts` | autoplay unlock 用 無音 MP3 data URL |
| `lib/tts/errors.ts` | `TTSError` クラスと `TTSFallbackReason` union 型 |
| `types/dashboard.ts` | `DashboardState`, `ProjectCardData`, `KanbanCounts`, `UnansweredItem`, `OpsEntry`, `Attention` 型定義 |
| `types/chat.ts` | `ChatSession`, `PromptRequest`, `ChatStreamEvent` 型定義 |
| `types/tts.ts` | `TTSRequest` 型定義 |
| `lib/constants.ts` | `GHOSTRUNNER_CWD`, `DASHBOARD_POLL_INTERVAL_MS`, localStorage キー定義 |
| `next.config.ts` | `/api/tts` を devtools backend (`:8888`) にプロキシ（catch-all より前に明示エントリ）、even-terminal (`:3456`) へのプロキシ rewrites 設定 |
