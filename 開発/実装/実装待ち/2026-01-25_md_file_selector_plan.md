# 開発フォルダMDファイル選択機能 実装計画

## 概要

Web UIのArgumentsフィールドに、開発フォルダ内のmdファイルをドロップダウンで選択できる機能を追加する。
コマンドごとに表示するフォルダの優先順位を設定し、ユーザーが目的のファイルを素早く選択できるようにする。

## 要件

### コマンド別フォルダ優先順位

| コマンド | 優先順位 |
|---------|---------|
| `/plan` | 1. 実装/実装待ち → 2. 実装/完了 |
| `/discuss` | 1. 検討中 → 2. 資料 |
| `/research` | 1. 検討中 → 2. 実装/実装待ち |
| その他 | ドロップダウン非表示（テキストエリアのみ） |

### UI仕様

- コマンド選択時にドロップダウンの内容を動的に更新
- フォルダ名をグループラベルとして表示（`<optgroup>`）
- ファイル選択時にArgumentsテキストエリアにファイルパスを設定
- 「選択なし」オプションで自由入力も可能

---

## バックエンド計画

### 新規API

#### `GET /api/files`

開発フォルダ内のmdファイル一覧を取得する。

**リクエストパラメータ:**
- `project` (query, 必須): プロジェクトのパス

**レスポンス:**
```json
{
  "success": true,
  "files": {
    "実装/実装待ち": [
      {"name": "2026-01-25_feature_plan.md", "path": "開発/実装/実装待ち/2026-01-25_feature_plan.md"}
    ],
    "実装/完了": [
      {"name": "2026-01-24_other_plan.md", "path": "開発/実装/完了/2026-01-24_other_plan.md"}
    ],
    "検討中": [],
    "資料": [
      {"name": "2026-01-25_research.md", "path": "開発/資料/2026-01-25_research.md"}
    ]
  }
}
```

**エラーレスポンス:**
```json
{
  "success": false,
  "error": "エラーメッセージ"
}
```
注: エラー時は `files` フィールドは省略される（`omitempty`）。

### 変更ファイル一覧

| ファイル | 変更内容 |
|---------|---------|
| `internal/handler/files.go` | 新規作成: FilesHandler |
| `cmd/server/main.go` | ルーティング追加: `GET /api/files` (20行目、26行目付近) |

### 実装ステップ

#### Step 1: FilesHandler の作成

`internal/handler/files.go` を新規作成:

```go
package handler

import (
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strings"

    "github.com/gin-gonic/gin"
)

// DevFolders はスキャン対象のフォルダ一覧（フロントエンドと共有）
var DevFolders = []string{
    "実装/実装待ち",
    "実装/完了",
    "検討中",
    "資料",
}

// FileInfo はファイル情報を表す
type FileInfo struct {
    Name string `json:"name"` // ファイル名
    Path string `json:"path"` // 相対パス（開発/から）
}

// FilesResponse は/api/filesレスポンスの構造体
type FilesResponse struct {
    Success bool                   `json:"success"`
    Files   map[string][]FileInfo  `json:"files,omitempty"`
    Error   string                 `json:"error,omitempty"`
}

// FilesHandler はファイル一覧のHTTPハンドラ
// 注: ファイルシステム操作のみのため外部依存なし
type FilesHandler struct{}

// NewFilesHandler は新しいFilesHandlerを生成
func NewFilesHandler() *FilesHandler

// Handle は/api/filesリクエストを処理
// GET /api/files?project=/path/to/project
// 注: GETリクエストのため c.Query() でパラメータ取得（POSTの既存APIは c.ShouldBindJSON）
func (h *FilesHandler) Handle(c *gin.Context)
```

**処理フロー:**
1. `c.Query("project")` でプロジェクトパスを取得
2. プロジェクトパスのバリデーション（既存の`validateProjectPath`を使用）
3. `開発/` ディレクトリの存在確認
4. `DevFolders` で定義されたフォルダをスキャン
5. 各フォルダ内の`.md`ファイルを収集（`.gitkeep`等は除外）
6. フォルダ別にグループ化してレスポンス

**ログ出力:**
- リクエスト受信時: `[FilesHandler] Handle started: project=%s`
- エラー時: `[FilesHandler] Handle failed: %s, error=%v`
- 成功時: `[FilesHandler] Handle completed: project=%s, folders=%d, files=%d`

#### Step 2: ルーティング追加

`cmd/server/main.go` に追加（既存コードへの差分）:

```go
// ハンドラー生成（20行目付近、commandHandler の後に追加）
filesHandler := handler.NewFilesHandler()

// ルーティング（26行目付近、api.POST("/command", ...) の前に追加）
api.GET("/files", filesHandler.Handle)
```

---

## フロントエンド計画

### UI変更

#### 変更前
```
Command: [dropdown]
Arguments: [textarea]
```

#### 変更後
```
Command: [dropdown]
File: [dropdown] (plan/discuss/research時のみ表示)
Arguments: [textarea]
```

### 変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `web/index.html` | ファイル選択ドロップダウン追加、JS処理追加 |

### 実装ステップ

#### Step 1: HTML追加

Commandの下にファイル選択フィールドを追加:

```html
<div class="form-group" id="fileSelectGroup" style="display: none;">
    <label for="fileSelect">File (optional)</label>
    <select id="fileSelect" name="fileSelect">
        <option value="">-- Select a file or type below --</option>
    </select>
</div>
```

#### Step 2: JavaScript追加

**コマンド別フォルダ優先順位の定義:**

注: フォルダ名はバックエンドの `DevFolders` と同じ値を使用。変更時は両方を更新すること。

```javascript
const COMMAND_FOLDER_PRIORITY = {
    'plan': ['実装/実装待ち', '実装/完了'],
    'discuss': ['検討中', '資料'],
    'research': ['検討中', '実装/実装待ち']
};
```

**処理フロー:**

1. コマンド選択変更時:
   - `COMMAND_FOLDER_PRIORITY`に該当するコマンドか確認
   - 該当する場合: ファイル選択フィールドを表示し、`/api/files`を呼び出し
   - 該当しない場合: ファイル選択フィールドを非表示

2. ファイル一覧取得時:
   - プロジェクトパスをクエリパラメータに設定
   - レスポンスのfilesをコマンドの優先順位に従ってドロップダウンに追加
   - `<optgroup>`でフォルダ別にグループ化

3. ファイル選択時:
   - 選択したファイルのパスをArgumentsテキストエリアに設定
   - 既存の内容がある場合は上書き確認（今回はシンプルに上書き）

**主要な関数:**

```javascript
// ファイル一覧を取得
async function fetchDevFiles(project) {
    const response = await fetch(`/api/files?project=${encodeURIComponent(project)}`);
    return await response.json();
}

// ドロップダウンを更新
function updateFileDropdown(files, command) {
    const select = document.getElementById('fileSelect');
    select.innerHTML = '<option value="">-- Select a file or type below --</option>';

    const priorities = COMMAND_FOLDER_PRIORITY[command] || [];
    for (const folder of priorities) {
        const folderFiles = files[folder] || [];
        if (folderFiles.length === 0) continue;

        const optgroup = document.createElement('optgroup');
        optgroup.label = folder;
        for (const file of folderFiles) {
            const option = document.createElement('option');
            option.value = file.path;
            option.textContent = file.name;
            optgroup.appendChild(option);
        }
        select.appendChild(optgroup);
    }
}

// コマンド変更時のハンドラ
async function onCommandChange() {
    const command = document.getElementById('command').value;
    const fileGroup = document.getElementById('fileSelectGroup');

    if (COMMAND_FOLDER_PRIORITY[command]) {
        fileGroup.style.display = 'block';
        const project = document.getElementById('project').value;
        if (project) {
            const data = await fetchDevFiles(project);
            if (data.success) {
                updateFileDropdown(data.files, command);
            }
        }
    } else {
        fileGroup.style.display = 'none';
    }
}

// ファイル選択時のハンドラ
function onFileSelect() {
    const fileSelect = document.getElementById('fileSelect');
    const argsTextarea = document.getElementById('args');
    if (fileSelect.value) {
        argsTextarea.value = fileSelect.value;
    }
}
```

---

## テスト計画

### バックエンド

1. **正常系**: プロジェクトパスを指定してmdファイル一覧を取得
2. **異常系**: 不正なプロジェクトパス
3. **異常系**: 開発フォルダが存在しない場合
4. **境界値**: 空のフォルダがある場合

### フロントエンド

1. コマンド切り替え時にドロップダウンが正しく表示/非表示
2. ファイル一覧がフォルダ別にグループ化されて表示
3. ファイル選択時にArgumentsに正しくパスが設定
4. プロジェクトパス変更時にファイル一覧が再取得

---

## 懸念点と対応

### 1. プロジェクトパス未入力時の挙動

**懸念**: コマンド選択時にプロジェクトパスが空の場合

**対応**: プロジェクトパスが空の場合はAPIを呼び出さず、ドロップダウンは空のまま表示

### 2. ファイル数が多い場合

**懸念**: 開発フォルダに大量のmdファイルがある場合のUI

**対応**: 今回は対応不要（開発フォルダは少数のファイル想定）。将来的に必要なら検索機能を追加

### 3. アーカイブフォルダの扱い

**懸念**: `開発/アーカイブ/`フォルダは表示するか

**対応**: コマンド別優先順位に含まれていないため非表示。必要なら後で追加可能

---

## バックエンド実装レポート

### 実装サマリー

- **実装日**: 2026-01-25
- **変更ファイル数**: 4 files

`GET /api/files` APIを新規作成し、開発フォルダ内のmdファイル一覧を取得する機能を実装した。
計画通りの実装を完了。

### 変更ファイル一覧

| ファイル | 変更内容 |
|---------|---------|
| `internal/handler/files.go` | 新規作成: FilesHandler実装（163行）- ファイル一覧取得API |
| `cmd/server/main.go` | ルーティング追加: `GET /api/files` エンドポイント登録 |
| `docs/BACKEND_API.md` | Files API仕様の追加（GET /api/files のドキュメント） |
| `internal/handler/doc.go` | パッケージドキュメント更新: FilesHandlerの説明追加 |

### 実装内容

#### FilesHandler (`internal/handler/files.go`)

- `FilesHandler` 構造体と `NewFilesHandler()` コンストラクタ
- `Handle(c *gin.Context)` メソッド: GETリクエスト処理
- `DevFolders` 変数: スキャン対象フォルダ一覧（4フォルダ）
- `FileInfo` / `FilesResponse` 構造体: レスポンス用の型定義

**処理フロー:**
1. `c.Query("project")` でプロジェクトパス取得
2. `validateProjectPath()` で入力検証（既存関数を再利用）
3. `開発/` ディレクトリの存在確認
4. 各フォルダをスキャンして `.md` ファイルを収集
5. フォルダ別にグループ化したレスポンスを返却

**対応するHTTPステータスコード:**
- 200: 成功
- 400: projectパラメータ未指定、無効なパス
- 404: 開発ディレクトリが存在しない
- 500: フォルダ読み取りエラー

#### ルーティング (`cmd/server/main.go`)

```go
filesHandler := handler.NewFilesHandler()
api.GET("/files", filesHandler.Handle)
```

### 動作確認手順

1. バックエンドサーバー起動
   ```bash
   cd backend
   go run ./cmd/server
   ```

2. APIリクエスト送信
   ```bash
   curl "http://localhost:8080/api/files?project=/Users/user/Ghostrunner"
   ```

3. 期待されるレスポンス
   ```json
   {
     "success": true,
     "files": {
       "実装/実装待ち": [
         {"name": "2026-01-25_md_file_selector_plan.md", "path": "開発/実装/実装待ち/2026-01-25_md_file_selector_plan.md"}
       ],
       "実装/完了": [...],
       "検討中": [...],
       "資料": [...]
     }
   }
   ```

4. エラーケースの確認
   ```bash
   # projectパラメータなし -> 400
   curl "http://localhost:8080/api/files"

   # 存在しないプロジェクト -> 400
   curl "http://localhost:8080/api/files?project=/nonexistent"
   ```

### 計画からの変更点

特になし。計画通りに実装を完了した。

### 課題・注意点

特になし。

### 残存する懸念点

- フロントエンド実装が未完了のため、統合テストは後続の実装完了後に実施が必要
