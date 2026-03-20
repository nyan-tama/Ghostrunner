# devtools GUI プロジェクト作成画面 - バックエンド実装レポート

---

## 実装完了レポート

### 実装サマリー
- **実装日**: 2026-03-20
- **対象**: devtools/backend/ 配下のみ（フロントエンドは未実装）
- **変更ファイル数**: 7 files（新規6 + 修正1）
- **新規コード行数**: 1,030行（プロダクションコード） + 968行（テストコード）
- **テストケース数**: 77ケース（テーブル駆動テストのサブテスト含む）
- **テスト結果**: 全テスト パス、go vet / go fmt / go build 全パス

### 変更ファイル一覧

| ファイル | 変更種別 | 行数 | 変更内容 |
|---------|---------|------|---------|
| `internal/service/template.go` | 新規 | 477行 | TemplateService: テンプレートコピー（base + サービス別）、プレースホルダー置換（`{{PROJECT_NAME}}`）、docker-compose YAMLマージ、.envファイル生成、.claude資産コピー、不要エージェント削除、CLAUDE.md生成、devtoolsシンボリックリンク作成。ヘルパー関数として copyDir/copyFile/isBinaryFile/mergeYAMLMaps/buildClaudeMD を含む |
| `internal/service/create.go` | 新規 | 322行 | CreateProjectServiceインターフェース + CreateService実装: 10ステップのプロジェクト生成オーケストレーション、プロジェクト名バリデーション（正規表現 + ディレクトリ存在チェック）、VS Code起動、SSEイベント送信。CreateEvent/CreateRequest/ValidateResult の型定義を含む |
| `internal/handler/create.go` | 新規 | 231行 | CreateHandler: 3エンドポイント（validate/create-stream/open）のHTTPハンドラ。SSEストリーミング送信（writeCreateSSEEvents）、サービス名バリデーション（AllowedServicesマップ参照）、パストラバーサル防止（ホームディレクトリ配下のみ許可）。既存 sse.go の setSSEHeaders と sseKeepaliveInterval を再利用 |
| `internal/service/template_test.go` | 新規 | 369行 | TemplateServiceのテスト5関数: serviceTemplateDir（5ケース）、isBinaryFile（27ケース: バイナリ20 + テキスト7）、mergeYAMLMaps（5ケース: services/volumes マージ + 未存在時新規作成 + 対象外キー不変）、ReplacePlaceholders（3ケース: 置換/変更なし/複数置換）、buildClaudeMD（5ケース: 基本/DB/Storage/Cache/desc空） |
| `internal/service/create_test.go` | 新規 | 225行 | CreateServiceのテスト4関数: ValidateProjectName（12ケース: 正常3 + 空文字 + 大文字 + スペース + アンダースコア + ドット + 先頭ハイフン + 末尾ハイフン + 連続ハイフン + 重複）、ProjectBaseDir（1ケース）、CreateProject_ContextCancel（キャンセル時のエラーイベント検証）、CreateProject_SendsProgressEvents（進捗イベント送信検証） |
| `internal/handler/create_test.go` | 新規 | 374行 | CreateHandlerのテスト6関数: HandleValidate（3ケース）、HandleOpen（7ケース: 空パス/存在しないパス/パストラバーサル3パターン/不正JSON/正常）、HandleCreateStream_InvalidJSON、HandleCreateStream_InvalidService、HandleOpen_VSCodeError、validateServices（4ケース）。mockCreateProjectService によるインターフェースモック使用 |
| `cmd/server/main.go` | 修正 | 112行(全体) | DI追加（runtime.Caller(0) でGhostrunnerルート解決、os.UserHomeDir() でプロジェクト生成先解決、TemplateService/CreateService/CreateHandler の組み立て）、ルーティング3行追加（validate/create-stream/open） |

### 計画からの変更点

実装計画に記載がなかった判断・選択:

- **go.mod の変更は不要だった**: 計画では `gopkg.in/yaml.v3` の追加を記載していたが、既に indirect 依存として含まれていた。go.mod の修正は発生しなかった
- **Ghostrunnerルートの解決方法**: 計画には詳細がなかったが、`runtime.Caller(0)` でソースファイルの絶対パスを取得し、4階層上（`devtools/backend/cmd/server/main.go` -> Ghostrunner root）をルートとする方式を採用した
- **プロジェクト生成先ディレクトリ**: 計画に明記がなかったが、`os.UserHomeDir()` をベースディレクトリとし、`~/プロジェクト名` に生成する方式とした（レビュー指摘 W1 対応: ハードコードではなくコンストラクタ経由で注入）
- **SSEヘルパー関数の再利用**: 既存の `sse.go` にある `setSSEHeaders` と `sseKeepaliveInterval` を再利用し、`handler/create.go` 側は CreateEvent 用の `writeCreateSSEEvents` のみ新規実装とした。既存の `writeSSEEvents` は `service.StreamEvent` 型専用のため、新しい関数を作成
- **サーバー起動ステップ**: 計画では `docker-compose up + backend/frontend起動` と記載されていたが、実装では `make start-backend` のみを使用した。フロントエンド起動は含めていない（devtools のフロントエンドはこのステップでは不要なため）

### レビュー指摘と対応

| 分類 | 指摘内容 | 対応 |
|------|---------|------|
| C1 (Critical) | HandleOpen パストラバーサル脆弱性 | `os.UserHomeDir()` + `filepath.Clean()` + `strings.HasPrefix()` でホームディレクトリ配下のみ許可。テストに3パターンの攻撃ケース追加（/etc/passwd, /, ../相対パス混入） |
| C2 (Critical) | stepServerStart goroutineリーク | `go backendCmd.Wait()` でゾンビプロセス防止 + `select/time.After` で3秒待機（ctx.Done 対応） |
| W1 (Warning) | projectBaseDir ハードコード | `NewCreateService` のコンストラクタ引数で注入。main.go で `os.UserHomeDir()` を渡す |
| W2 (Warning) | ctx.Err() チェック不足 | ステップループの各ステップ実行前に `ctx.Err()` をチェックし、キャンセル時は即座にエラーイベント送信して終了 |

### コード品質の観察

#### 設計パターン
- **Clean Architecture**: handler -> service の依存方向が明確。CreateProjectService インターフェースによる抽象化
- **テーブル駆動テスト**: 全テスト関数でテーブル駆動パターンを使用（CLAUDE.md のテスト規約に準拠）
- **インターフェースモック**: handler テストで `mockCreateProjectService` を使用し、service 層との結合を切断

#### ファイルサイズ
- template.go（477行）は CLAUDE.md の推奨範囲（200-400行、最大600行）内に収まっている
- 他の全ファイルも400行以下で規約内

#### エラーハンドリング
- 全エラーが `fmt.Errorf("failed to X: %w", err)` でコンテキスト付きラップ
- API レスポンスに適切な HTTP ステータスコード（400/404/500）
- SSE では error イベントとしてクライアントに通知

### 実装時の課題

特になし

### 残存する懸念点

今後注意が必要な点:

- **Ghostrunnerルートのパス解決**: `runtime.Caller(0)` によるソースファイルパスベースの解決は、`go build` でバイナリ化した場合にビルド時のソースパスが埋め込まれるため、異なるマシンへのバイナリ配布時にはパスが不正になる。現在は `go run` での開発利用を前提としているため問題ないが、バイナリ配布時には環境変数等での指定が必要
- **サーバー起動ステップの信頼性**: `make start-backend` を `exec.Command.Start()` で非同期実行し、3秒の固定待機後に次のステップ（ヘルスチェック）に進む。低スペック環境では起動が間に合わない場合がある。ヘルスチェック（step 10）で最大10回x2秒=20秒のリトライがあるため実用上は問題ないが、タイムアウト値の調整が必要になる可能性がある
- **ヘルスチェックのポート固定**: `localhost:8080` がハードコードされている。生成されたプロジェクトのバックエンドポートが変更された場合には対応できない
- **VS Code の `code` コマンド依存**: `code` コマンドが PATH にない環境では OpenInVSCode が失敗する。エラーメッセージは返すが、ユーザーへの案内（PATH 設定方法等）は含めていない

### 動作確認フロー

```
1. make backend でdevtoolsバックエンドを起動
2. curl でバリデーションAPIを確認:
   curl "http://localhost:8080/api/projects/validate?name=my-project"
   -> {"valid":true,"path":"/Users/user/my-project"} が返ること
3. curl で不正な名前のバリデーションを確認:
   curl "http://localhost:8080/api/projects/validate?name=My-Project"
   -> {"valid":false,"error":"..."} が返ること
4. curl で既存ディレクトリ名のバリデーションを確認:
   curl "http://localhost:8080/api/projects/validate?name=Ghostrunner"
   -> {"valid":false,"error":"同名のディレクトリが既に存在します"} が返ること
5. SSE作成APIは結合テスト時にフロントエンドと合わせて確認
6. go test ./... で全テストがパスすること
```

### デプロイ後の確認事項

- [ ] `cd devtools/backend && go test ./...` で全テストがパスすること
- [ ] `cd devtools/backend && go vet ./...` で警告がないこと
- [ ] `cd devtools/backend && go build ./cmd/server` でビルドが通ること
- [ ] バリデーションAPIが正常に応答すること（`GET /api/projects/validate?name=test`）
- [ ] フロントエンド実装後にSSE作成API（`POST /api/projects/create/stream`）の結合テストを実施すること
- [ ] フロントエンド実装後にOpen API（`POST /api/projects/open`）のE2Eテストを実施すること
- [ ] パストラバーサル攻撃が拒否されること（`POST /api/projects/open` に `/etc/passwd` 等を送信）
