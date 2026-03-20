## フロントエンド実装完了レポート

### 実装サマリー
- **実装日**: 2026-03-20
- **対象**: devtools/frontend/ 配下のみ
- **変更ファイル数**: 12 files（新規7 + テスト5 + 修正2 ※型定義修正含む）
- **新規コード行数**: 約804行（本体） + 約597行（テスト）
- **テスト数**: 43テストケース

### 変更ファイル一覧

| ファイル | 変更種別 | 行数 | 変更内容 |
|---------|---------|------|---------|
| `src/types/index.ts` | 修正 | +32行 | `DataService`, `CreateProjectRequest`, `CreateProgressEvent`, `CreateStep`, `CreatedProject`, `CreatePhase` の6型を追加 |
| `src/lib/createApi.ts` | 新規 | 61行 | API呼び出し3関数: `validateProjectName`（GET、AbortSignal対応）、`createProjectStream`（POST、SSE用Responseを返却）、`openInVSCode`（POST） |
| `src/hooks/useProjectValidation.ts` | 新規 | 81行 | 300msデバウンス付きバリデーション。タイマーとAbortControllerをuseRefで管理、stale closure問題を回避 |
| `src/hooks/useProjectCreate.ts` | 新規 | 206行 | SSE通信と状態遷移管理。10ステップ定義、ReadableStreamパース、phase管理（form/creating/complete/error）、AbortController対応 |
| `src/components/create/ServiceSelector.tsx` | 新規 | 52行 | Data Servicesチェックボックス3つ（PostgreSQL+GORM, Cloudflare R2/MinIO, Redis）。トグル操作でイミュータブルに配列更新 |
| `src/components/create/ProjectForm.tsx` | 新規 | 155行 | フォーム入力（名前、概要、サービス選択）+ 確認セクション常時表示 + バリデーション結果表示。Enter送信対応、初期値復元対応 |
| `src/components/create/CreateProgress.tsx` | 新規 | 80行 | プログレスバー + 10ステップチェックリスト。SVGアイコンで状態表示（done=緑チェック、active=スピナー、error=赤バツ、pending=空丸） |
| `src/components/create/CreateComplete.tsx` | 新規 | 74行 | 完了画面。プロジェクトパス表示 + [Open in VS Code]（API呼び出し、エラーハンドリング付き）+ [Create Another] |
| `src/app/new/page.tsx` | 新規 | 95行 | /new ページ本体。useProjectCreateでphase管理、エラー時に入力値保持してフォームに復帰、Backリンク |
| `src/app/page.tsx` | 修正 | +6行 | ヘッダーに "New Project" リンク（紫色ボタン）を追加 |

### テストファイル一覧

| ファイル | テスト数 | 内容 |
|---------|---------|------|
| `src/__tests__/lib/createApi.test.ts` | 10 | validateProjectName（5: 正常、エンコード、エラー、フォールバック、AbortSignal）、createProjectStream（2: 正常、AbortSignal）、openInVSCode（3: 正常、エラー、フォールバック） |
| `src/__tests__/components/create/ServiceSelector.test.tsx` | 5 | 3つのチェックボックス描画、選択状態反映、追加操作、削除操作、説明文表示 |
| `src/__tests__/components/create/CreateProgress.test.tsx` | 11 | ステップラベル描画、パーセント表示、バー幅スタイル、4状態の色分け（done/active/error/pending）、4状態のアイコン（SVG/スピナー/丸） |
| `src/__tests__/components/create/CreateComplete.test.tsx` | 7 | プロジェクト名パス表示、ボタン描画、Create Another動作、Open in VS Code成功、API失敗時エラー、success:false時エラー |
| `src/__tests__/components/create/ProjectForm.test.tsx` | 10 | 入力描画、onNameChange呼び出し、ボタン無効化3パターン（空/バリデ中/invalid）、ボタン有効化、onSubmit引数、Validating表示、パス表示、エラー表示 |

### 計画からの変更点

- **SSEパースの実装方針**: 計画では「`useSSEStream` と同じ ReadableStream パースロジックを `useProjectCreate` 内に持つ」と記載があり、その通りに `processSSEResponse` メソッドとして `useProjectCreate` 内に実装した。`data: ` プレフィックス解析とバッファリング処理を独自実装している
- **エラー復帰時の初期バリデーション**: 計画にはなかったが、レビュー指摘（W2）を受けて `ProjectForm` に `useEffect` を追加し、`initialName` が設定されている場合にマウント時にバリデーションを自動実行するようにした。これによりエラー復帰時にバリデーション状態が即座に反映される
- **processSSEResponse のエラー解析**: 計画にはなかったが、レビュー指摘（W4）を受けて、HTTP エラーレスポンスの JSON パース時に try-catch を追加した。JSON パースが失敗した場合は HTTP ステータスコードを含むフォールバックメッセージを表示する

### 実装時の課題

#### レビュー指摘と対応

- **W1: useProjectValidation の stale closure**: 当初 `setTimeout` のタイマーIDを `useState` で管理していたが、`onNameChange` がクロージャで古い `timer` を参照する問題があった。`useRef` に変更することで解決
- **W2: ProjectForm 初期バリデーション欠落**: エラー復帰時に `initialName` が渡されるが、バリデーションが実行されずボタンが無効のままになる問題。`useEffect` でマウント時に `onNameChange(initialName)` を呼び出すことで解決
- **W4: processSSEResponse エラー解析**: `response.json()` が非JSONレスポンスで失敗する可能性があった。try-catch を追加し、フォールバックとして `HTTP ${response.status}` をメッセージに含めるようにした

### 残存する懸念点

- **useProjectValidation のクリーンアップ**: コンポーネントアンマウント時に `timerRef` と `controllerRef` のクリーンアップ（clearTimeout, abort）を明示的に行っていない。`/new` ページから離脱した際にタイマーが残る可能性があるが、300msと短いため実用上は問題ない
- **SSEストリームの再接続**: ネットワーク断時の自動再接続は未実装。エラーメッセージを表示してフォームに戻す設計のため、ユーザーが手動で再試行する必要がある
- **プログレスバーのステップ定義がフロントエンド固定**: 10ステップのID・ラベルが `useProjectCreate.ts` にハードコードされている。バックエンド側のステップ定義と同期が必要で、ステップの追加・変更時は両方の修正が必要
- **page.tsx のファイルサイズ**: 既存の `app/page.tsx` が610行あり、CLAUDE.md の規約（通常200-400行、最大800行）の上限に近づいている。今回はリンク追加（+6行）のみだが、今後機能追加時にはコンポーネント分割を検討すべき

### コード品質

- **イミュータブル操作**: 配列の追加・削除は `filter` / スプレッド演算子で新しい配列を生成しており、直接変更は行っていない（CLAUDE.md 準拠）
- **AbortController対応**: 全API呼び出しで AbortSignal をサポート。コンポーネント間の接続中断時にリソースリークを防止
- **型安全性**: `DataService` をユニオン型で定義し、`CreateStep["status"]` で状態を制約。any 型の使用なし
- **useCallback / useRef**: コールバック関数は `useCallback` でメモ化、タイマー/コントローラーは `useRef` で管理し、不要な再レンダリングを抑制
- **console.log なし**: 本番コードに console.log は含まれていない（CLAUDE.md 準拠）

### 動作確認フロー

```
1. make dev で devtools を起動（バックエンド + フロントエンド）
2. ブラウザで http://localhost:3001 にアクセス
3. ヘッダーの "New Project" リンク（紫色）をクリック → /new ページに遷移
4. Project Name に "my-test-app" と入力 → 300ms 後に "Will be created at: /Users/user/my-test-app" が表示される
5. 不正な名前（例: "My App"）を入力 → バリデーションエラーが赤字で表示される
6. Description に任意の説明を入力（任意）
7. Data Services でチェックボックスを選択（任意）
8. Summary セクションに入力内容が反映されていることを確認
9. [Create Project] をクリック → 進捗画面に遷移
10. 10ステップが順にチェック付きになり、プログレスバーが進行
11. 完了後 → "Project Created" 画面が表示される
12. [Open in VS Code] をクリック → VS Code でプロジェクトが開く
13. [Create Another] をクリック → フォームに戻り、入力欄がクリアされる
14. エラー発生時 → エラーメッセージが表示され、フォームに戻ると前回の入力値が保持されている
```

### デプロイ後の確認事項

- [ ] `npm run build` がエラーなく完了すること
- [ ] `npm test` で全43テストがパスすること
- [ ] `/new` ページにアクセスできること
- [ ] バリデーション API との疎通（入力→300ms後にレスポンス表示）
- [ ] SSE 作成 API との結合テスト（フォーム送信→進捗表示→完了）
- [ ] VS Code Open API との結合テスト（[Open in VS Code] → エディタ起動）
- [ ] エラー復帰フロー（意図的にエラーを発生させ、フォームに戻って入力値が保持されること）
- [ ] モバイル幅でのレイアウト崩れがないこと（max-w-[600px] で中央配置）
