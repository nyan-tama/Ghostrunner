# Next.js移行 実装計画

## 概要

Gemini Live API実装に向けたNext.js移行（A案）の実装計画。既存の`web/index.html`（バニラJS、約1005行）をNext.js 15 + React 19 + TypeScriptに完全移植する。

### 選択した方針: A案

- 先にNext.js化してからGemini Live実装
- 既存`web/index.html`の全機能を移植
- TypeScriptの型安全性でオーディオ処理やWebSocket管理のバグを減らす

### 移植対象機能

- SSEストリーミングによるコマンド実行（plan, fullstack, go, nextjs, discuss, research）
- イベント表示（tool_use, text, thinking等）
- ファイルセレクター（5フォルダ対応）
- 質問UI（単一選択/複数選択、カスタム入力）
- 計画承認UI
- セッション継続機能
- コスト表示

---

## 懸念点と解決策

### 解決済み

| 懸念点 | 解決策 |
|--------|--------|
| SSEバッファ処理 | TextDecoderの`stream: true`オプション使用、不完全行のバッファリング（詳細は後述） |
| セッション状態管理 | page.tsxで一元管理し、propsで子コンポーネントに伝播（状態フロー図参照） |
| PlanApproval表示条件 | 出力テキストの文字列検出ロジック（詳細は後述） |

### 要確認（実装前にユーザー確認が必要）

| 懸念点 | 選択肢 |
|--------|--------|
| バックエンドURL | 開発時: localhost:8080、本番: 環境変数 `NEXT_PUBLIC_API_URL` で設定 |
| 認証 | 現在と同様、認証なしで進める |
| web/index.html | Next.js完成後は削除（A案の方針） |

---

## 修正範囲の全体像

```mermaid
flowchart TD
    subgraph "Frontend（新規作成）"
        PAGE[app/page.tsx]
        LAYOUT[app/layout.tsx]

        subgraph "Components"
            FORM[CommandForm]
            PROGRESS[ProgressContainer]
            EVENTS[EventList]
            QUESTION[QuestionSection]
            APPROVAL[PlanApproval]
        end

        subgraph "Hooks"
            SSE[useSSEStream]
            SESSION[useSessionManagement]
            FILES[useFileSelector]
        end

        subgraph "Types"
            TYPES[types/index.ts]
        end
    end

    subgraph "Backend（変更なし）"
        API[Go API Server]
    end

    PAGE --> FORM
    PAGE --> PROGRESS
    PROGRESS --> EVENTS
    PROGRESS --> QUESTION
    PROGRESS --> APPROVAL
    FORM --> FILES
    PROGRESS --> SSE
    SSE --> SESSION
    SSE --> API
    FILES --> API
```

---

## 状態管理フロー

```mermaid
flowchart TD
    subgraph "page.tsx（状態管理の中心）"
        STATE[State]
        STATE --> |sessionId| SID[currentSessionId]
        STATE --> |totalCost| COST[totalCost]
        STATE --> |events| EVT[events配列]
        STATE --> |isLoading| LOAD[isLoading]
        STATE --> |questions| Q[questions]
        STATE --> |result| RES[result]
        STATE --> |showApproval| APPR[showApproval]
    end

    subgraph "Props伝播"
        FORM2[CommandForm]
        FORM2 --> |onSubmit| PAGE2[page.tsx]

        PROGRESS2[ProgressContainer]
        PAGE2 --> |events, questions, result| PROGRESS2

        QUESTION2[QuestionSection]
        PROGRESS2 --> |questions, onAnswer| QUESTION2

        APPROVAL2[PlanApproval]
        PROGRESS2 --> |showApproval, onApprove, onReject| APPROVAL2
    end

    subgraph "フック"
        SSE2[useSSEStream]
        SSE2 --> |onEvent| STATE
        SSE2 --> |updateSessionId| SID
        SSE2 --> |updateCost| COST
    end
```

### 状態の責務

| 状態 | 管理場所 | 更新タイミング |
|------|----------|---------------|
| `currentSessionId` | page.tsx | SSEイベント `init` 受信時 |
| `totalCost` | page.tsx | SSEイベント `complete` 受信時 |
| `events` | page.tsx | 各SSEイベント受信時 |
| `isLoading` | page.tsx | コマンド実行開始/完了時 |
| `questions` | page.tsx | SSEイベント `question` 受信時 |
| `result` | page.tsx | SSEイベント `complete` 受信時 |
| `showApproval` | page.tsx | result内容の文字列判定で決定 |
| `projectPath` | useSessionManagement | localStorageから復元/保存 |

---

## SSEバッファ処理の詳細設計

### 既存実装のロジック（移植対象）

```typescript
// useSSEStream.ts の実装方針

async function processStream(response: Response, onEvent: (event: StreamEvent) => void) {
  const reader = response.body!.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    // stream: true で途中のマルチバイト文字を正しく処理
    buffer += decoder.decode(value, { stream: true });

    // 改行で分割
    const lines = buffer.split('\n');

    // 最後の不完全な行をバッファに保持
    buffer = lines.pop() || '';

    for (const line of lines) {
      // 空行をスキップ
      if (!line.trim()) continue;

      // SSEの data: プレフィックスを除去
      if (!line.startsWith('data: ')) continue;

      const jsonStr = line.slice(6); // 'data: '.length

      try {
        const event: StreamEvent = JSON.parse(jsonStr);
        onEvent(event);
      } catch (e) {
        console.error('Failed to parse SSE event:', e);
      }
    }
  }
}
```

### バッファ処理の重要ポイント

1. **`stream: true`オプション**: マルチバイト文字（日本語等）が途中で切れた場合に正しく処理
2. **不完全行の保持**: `lines.pop()` で最後の行（改行で終わっていない可能性）をバッファに戻す
3. **`data: `プレフィックス**: SSEフォーマットに従い、このプレフィックスがない行は無視

---

## PlanApproval表示条件

### 判定ロジック

```typescript
// ProgressContainer.tsx または page.tsx での判定

function shouldShowApproval(result: CommandResult | null): boolean {
  if (!result?.output) return false;

  const approvalKeywords = [
    '承認をお待ち',
    'waiting for approval',
    'Ready for approval'
  ];

  return approvalKeywords.some(keyword =>
    result.output.includes(keyword)
  );
}
```

### 表示フロー

```mermaid
sequenceDiagram
    participant SSE as useSSEStream
    participant Page as page.tsx
    participant Progress as ProgressContainer
    participant Approval as PlanApproval

    SSE->>Page: onEvent(complete, result)
    Page->>Page: setResult(result)
    Page->>Page: setShowApproval(shouldShowApproval(result))
    Page->>Progress: props: {result, showApproval}
    Progress->>Approval: 条件付きレンダリング
    Note over Approval: showApproval=true の場合のみ表示
```

---

## ファイルパス結合ロジック

### CommandFormの仕様

```typescript
// CommandForm.tsx でのargs生成ロジック

function buildArgs(selectedFile: string, userArgs: string): string {
  if (selectedFile && userArgs) {
    return `${selectedFile} ${userArgs}`;
  } else if (selectedFile) {
    return selectedFile;
  } else {
    return userArgs;
  }
}
```

| ファイル選択 | 引数入力 | 結果のargs |
|------------|---------|-----------|
| 選択あり | 入力あり | `${file} ${args}` |
| 選択あり | 入力なし | `${file}` |
| 選択なし | 入力あり | `${args}` |
| 選択なし | 入力なし | 空文字 |

---

## 質問回答のmultiSelect対応

### QuestionSectionの仕様

```typescript
// 単一選択（multiSelect: false）
// - 選択肢クリックで即座に回答送信
// - 選択後はUIを非表示

// 複数選択（multiSelect: true）
// - 選択肢クリックで選択状態をトグル
// - 「送信」ボタン押下で選択肢をカンマ区切りで結合して送信
// - 例: ["Option A", "Option C"] → "Option A, Option C"

// カスタム入力
// - テキスト入力フィールド + 「送信」ボタン
// - 入力内容をそのまま送信
```

---

## 変更ファイル一覧

| ファイル | 変更内容 | 影響度 |
|---------|---------|-------|
| `frontend/package.json` | プロジェクト設定（Next.js 15, React 19, TypeScript） | 高 |
| `frontend/tsconfig.json` | TypeScript設定 | 中 |
| `frontend/tailwind.config.ts` | Tailwind CSS設定 | 低 |
| `frontend/next.config.ts` | Next.js設定（APIプロキシ、環境変数） | 中 |
| `frontend/src/app/layout.tsx` | ルートレイアウト | 低 |
| `frontend/src/app/page.tsx` | メインページ（状態管理の中心） | 高 |
| `frontend/src/types/index.ts` | 型定義（StreamEvent, Question等） | 中 |
| `frontend/src/hooks/useSSEStream.ts` | SSEストリーム処理フック | 高 |
| `frontend/src/hooks/useSessionManagement.ts` | セッション・コスト管理フック | 中 |
| `frontend/src/hooks/useFileSelector.ts` | ファイル選択フック | 中 |
| `frontend/src/components/CommandForm.tsx` | フォームセクション | 高 |
| `frontend/src/components/ProgressContainer.tsx` | プログレス表示コンテナ | 高 |
| `frontend/src/components/EventList.tsx` | イベントリスト表示 | 中 |
| `frontend/src/components/EventItem.tsx` | 個別イベント表示（展開可能） | 中 |
| `frontend/src/components/QuestionSection.tsx` | 質問UI（multiSelect対応） | 高 |
| `frontend/src/components/PlanApproval.tsx` | 計画承認UI | 低 |
| `frontend/src/components/LoadingIndicator.tsx` | ローディング表示 | 低 |
| `frontend/src/lib/api.ts` | API呼び出しユーティリティ | 中 |
| `frontend/src/lib/constants.ts` | 定数（ALL_DEV_FOLDERS等） | 低 |

---

## 実装ステップ

### Phase 1: プロジェクト基盤構築

#### Step 1: Next.jsプロジェクト初期化
- `create-next-app@latest` で `frontend/` ディレクトリを作成
- App Router、TypeScript、Tailwind CSS、ESLint を有効化
- `src/` ディレクトリ使用
- 不要な初期ファイル（page.tsx のデフォルト内容等）を削除

#### Step 2: 型定義の作成
- `frontend/src/types/index.ts` を作成
- バックエンドの `internal/service/types.go` と整合性を取る
- StreamEvent, Question, Option, CommandRequest, FileInfo等

#### Step 3: 定数ファイルの作成
- `frontend/src/lib/constants.ts` を作成
- `ALL_DEV_FOLDERS` 配列（バックエンドと同期）
- 許可コマンドリスト

#### Step 4: Next.js設定（APIプロキシ）
- `frontend/next.config.ts` を編集
- `/api/*` をバックエンドにプロキシ
- 環境変数 `NEXT_PUBLIC_API_URL` のサポート

### Phase 2: カスタムフック実装

#### Step 5: useSSEStreamフック
- `frontend/src/hooks/useSSEStream.ts`
- fetch + ReadableStream でSSE処理
- バッファ処理（前述の詳細設計に従う）
- エラーハンドリング

#### Step 6: useSessionManagementフック
- `frontend/src/hooks/useSessionManagement.ts`
- localStorageでプロジェクトパスを永続化
- セッションIDとコストは page.tsx で管理（このフックは localStorage 操作のみ）

#### Step 7: useFileSelectorフック
- `frontend/src/hooks/useFileSelector.ts`
- GET /api/files を呼び出し
- フォルダ別ファイル一覧を取得

### Phase 3: コンポーネント実装

#### Step 8: LoadingIndicatorコンポーネント
- `frontend/src/components/LoadingIndicator.tsx`
- Tailwind CSS でスピナーアニメーション
- シンプルな表示のみ

#### Step 9: EventItemコンポーネント
- `frontend/src/components/EventItem.tsx`
- ツール別の表示フォーマット（Read, Write, Edit, Bash等）
- 展開可能テキスト（200文字以上で折りたたみ）

#### Step 10: EventListコンポーネント
- `frontend/src/components/EventList.tsx`
- EventItem の配列をレンダリング
- 自動スクロール

#### Step 11: PlanApprovalコンポーネント
- `frontend/src/components/PlanApproval.tsx`
- 承認/拒否ボタン
- onApprove, onReject コールバック

#### Step 12: QuestionSectionコンポーネント
- `frontend/src/components/QuestionSection.tsx`
- 単一選択/複数選択対応
- カスタム入力フィールド

#### Step 13: ProgressContainerコンポーネント
- `frontend/src/components/ProgressContainer.tsx`
- EventList, QuestionSection, PlanApproval を統括
- 表示/非表示の制御

#### Step 14: CommandFormコンポーネント
- `frontend/src/components/CommandForm.tsx`
- プロジェクトパス入力
- コマンド選択ドロップダウン
- ファイルセレクター
- 引数入力
- 実行ボタン

### Phase 4: ページ統合

#### Step 15: メインページ実装
- `frontend/src/app/page.tsx`
- 状態管理の中心
- CommandForm と ProgressContainer を統合
- useSSEStream との連携

### Phase 5: 動作確認

#### Step 16: ローカル動作確認
- バックエンドを起動（`go run ./cmd/server`）
- フロントエンドを起動（`npm run dev`）
- 全6コマンドの動作確認
- ファイルセレクターの動作確認
- 質問応答の動作確認
- セッション継続の動作確認

---

## ディレクトリ構造（完成形）

```
frontend/
├── src/
│   ├── app/
│   │   ├── layout.tsx          # ルートレイアウト
│   │   └── page.tsx            # メインページ（状態管理中心）
│   ├── components/
│   │   ├── CommandForm.tsx     # フォームセクション
│   │   ├── ProgressContainer.tsx # プログレス表示
│   │   ├── EventList.tsx       # イベントリスト
│   │   ├── EventItem.tsx       # 個別イベント（展開可能）
│   │   ├── QuestionSection.tsx # 質問UI
│   │   ├── PlanApproval.tsx    # 計画承認
│   │   └── LoadingIndicator.tsx # ローディング
│   ├── hooks/
│   │   ├── useSSEStream.ts     # SSE処理
│   │   ├── useSessionManagement.ts # localStorage操作
│   │   └── useFileSelector.ts  # ファイル選択
│   ├── lib/
│   │   ├── api.ts              # API呼び出し
│   │   └── constants.ts        # 定数定義
│   └── types/
│       └── index.ts            # 型定義
├── next.config.ts              # Next.js設定
├── tailwind.config.ts          # Tailwind設定
├── tsconfig.json               # TypeScript設定
└── package.json                # 依存関係
```

---

## 設計判断

| 判断 | 選択した方法 | 理由 |
|-----|------------|------|
| SSE処理 | fetch + ReadableStream | EventSourceはPOST非対応 |
| 状態管理 | useState + props伝播 | 複雑度が低い、Context不要 |
| APIプロキシ | next.config.ts rewrites | CORS回避 |
| スタイリング | Tailwind CSS | CLAUDE.MDの規約 |

---

## MVP外（将来対応）

- エフェメラルトークンAPI（Gemini Live用）
- WebSocket接続（Gemini Live用）
- 音声処理フック（Gemini Live用）
- E2Eテスト
- ダークモード

---

## 次のステップ

承認後、`/nextjs` コマンドで実装を開始。
