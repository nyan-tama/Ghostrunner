# File Selector 全フォルダ対応 実装計画

## 概要

Web UIの「File (optional)」ドロップダウンを、コマンドに関係なく開発フォルダ内のすべてのファイルを選べるように修正する。

## 現状

現在の実装では、コマンドごとに表示するフォルダが限定されている：

| コマンド | 表示されるフォルダ |
|---------|------------------|
| `/plan` | 実装/実装待ち, 実装/完了 |
| `/discuss` | 検討中, 資料 |
| `/research` | 検討中, 実装/実装待ち |
| その他 | ドロップダウン非表示 |

**問題点**: `/plan` 選択時に「アーカイブ」フォルダのファイルを参照したい場合など、目的のファイルが見つからない。

## 要件

- **すべてのコマンドで**ファイルセレクターを表示する
- **開発フォルダ内のすべてのサブフォルダ**を対象にする
- 優先順位の概念を廃止し、フォルダをアルファベット順（または固定順）で表示

## 変更方針

### バックエンド変更

`internal/handler/files.go` の `DevFolders` を拡張：

```go
var DevFolders = []string{
    "実装/実装待ち",
    "実装/完了",
    "検討中",
    "資料",
    "アーカイブ",  // 追加
}
```

### フロントエンド変更

`web/index.html` の変更点：

1. **`COMMAND_FOLDER_PRIORITY` の削除/変更** - すべてのフォルダを全コマンドで表示

2. **条件分岐の削除** - `if (COMMAND_FOLDER_PRIORITY[command])` を削除し、常にドロップダウンを表示

3. **表示フォルダ順序の固定** - 全フォルダを固定順序で表示

---

## バックエンド計画

### 変更ファイル一覧

| ファイル | 変更内容 |
|---------|---------|
| `internal/handler/files.go` | `DevFolders` に「アーカイブ」を追加 |

### 実装ステップ

#### Step 1: DevFolders 拡張

`internal/handler/files.go` の `DevFolders` 変数を変更：

**変更前:**
```go
var DevFolders = []string{
    "実装/実装待ち",
    "実装/完了",
    "検討中",
    "資料",
}
```

**変更後:**
```go
var DevFolders = []string{
    "実装/実装待ち",
    "実装/完了",
    "検討中",
    "資料",
    "アーカイブ",
}
```

---

## フロントエンド計画

### 変更ファイル一覧

| ファイル | 変更内容 |
|---------|---------|
| `web/index.html` | ファイルセレクター表示ロジックの変更 |

### 実装ステップ

#### Step 1: COMMAND_FOLDER_PRIORITY を ALL_DEV_FOLDERS に変更

**変更前:**
```javascript
const COMMAND_FOLDER_PRIORITY = {
    'plan': ['実装/実装待ち', '実装/完了'],
    'discuss': ['検討中', '資料'],
    'research': ['検討中', '実装/実装待ち']
};
```

**変更後:**
```javascript
// 開発フォルダ内の全フォルダ（表示順）
// バックエンドの DevFolders と同じ順序
const ALL_DEV_FOLDERS = [
    '実装/実装待ち',
    '実装/完了',
    '検討中',
    '資料',
    'アーカイブ'
];
```

#### Step 2: updateFileDropdown 関数の変更

**変更前:**
```javascript
function updateFileDropdown(files, command) {
    // ...
    const priorities = COMMAND_FOLDER_PRIORITY[command] || [];
    for (const folder of priorities) {
        // ...
    }
}
```

**変更後:**
```javascript
function updateFileDropdown(files) {
    fileSelect.innerHTML = '<option value="">-- Select a file or type below --</option>';

    for (const folder of ALL_DEV_FOLDERS) {
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
        fileSelect.appendChild(optgroup);
    }
}
```

#### Step 3: onCommandChange 関数の変更

**変更前:**
```javascript
async function onCommandChange() {
    const command = commandSelect.value;

    if (COMMAND_FOLDER_PRIORITY[command]) {
        fileSelectGroup.style.display = 'block';
        const project = document.getElementById('project').value.trim();
        if (project) {
            const data = await fetchDevFiles(project);
            if (data.success) {
                updateFileDropdown(data.files, command);
            }
        }
    } else {
        fileSelectGroup.style.display = 'none';
    }
}
```

**変更後:**
```javascript
async function onCommandChange() {
    // 常にファイルセレクターを表示
    fileSelectGroup.style.display = 'block';
    const project = document.getElementById('project').value.trim();
    if (project) {
        const data = await fetchDevFiles(project);
        if (data.success) {
            updateFileDropdown(data.files);
        }
    }
}
```

#### 注意事項

- 初期化処理（ページロード時の `onCommandChange()` 呼び出し）は変更不要
- `// Command-folder priority mapping` コメントも削除すること

---

## テスト計画

### 手動テスト

1. `/plan` コマンドを選択 → 5つのフォルダ全てが表示されることを確認
2. `/fullstack` コマンドを選択 → 同様に5つのフォルダ全てが表示されることを確認
3. `/go` コマンドを選択 → 同様に5つのフォルダ全てが表示されることを確認
4. アーカイブフォルダのファイルを選択 → Argumentsに正しくパスが設定されることを確認

### APIテスト

```bash
# アーカイブフォルダが含まれることを確認
curl "http://localhost:8080/api/files?project=/Users/user/Ghostrunner"
```

---

## 懸念点と対応

### 1. 既存のユーザー体験の変化

**懸念**: コマンドごとに関連フォルダだけを見せていた方がユーザーにとって便利だったかもしれない

**対応**: ユーザーの要望に従い、全フォルダを表示する。将来的に必要なら「お気に入り」「最近使用」などの機能を追加検討

### 2. フォルダ順序

**懸念**: どの順序でフォルダを表示するか

**対応**: バックエンドの `DevFolders` と同じ順序で表示（実装/実装待ち → 実装/完了 → 検討中 → 資料 → アーカイブ）

### 3. バックエンドとフロントエンドの同期

**懸念**: フォルダ名がバックエンドとフロントエンドで重複定義されている

**対応**: 既存設計を踏襲。コメントで「バックエンドの DevFolders と同期が必要」と明記

---

## 変更サマリー

| 箇所 | 変更内容 |
|-----|---------|
| バックエンド | `DevFolders` に「アーカイブ」追加（1行追加） |
| フロントエンド | `COMMAND_FOLDER_PRIORITY` を `ALL_DEV_FOLDERS` に変更 |
| フロントエンド | `onCommandChange` の条件分岐削除（常に表示） |
| フロントエンド | `updateFileDropdown` の引数から `command` 削除 |

**影響範囲**: 軽微。既存機能の拡張のみ。

---

## 計画レビュー結果

### レビュー日: 2026-01-25

**結果**: Critical な問題なし。実装に進んでよい。

**Warning（軽微）**:
1. Step 4 の記述が Step 2 と重複 → 「注意事項」に整理
2. プロジェクトパス空の場合のエッジケース → 現状動作で問題なし（空ドロップダウン表示）
3. 初期化動作について明記 → 計画書に追記済み

### 変更箇所サマリー（コード行番号付き）

| ファイル | 行番号 | 変更内容 |
|---------|--------|---------|
| `internal/handler/files.go` | 15-20行 | `DevFolders` に「アーカイブ」追加 |
| `web/index.html` | 457-463行 | `COMMAND_FOLDER_PRIORITY` を `ALL_DEV_FOLDERS` に変更 |
| `web/index.html` | 499-517行 | `updateFileDropdown` から `command` 引数削除 |
| `web/index.html` | 520-535行 | `onCommandChange` の条件分岐削除 |
| `web/index.html` | 529行 | `updateFileDropdown` 呼び出しから `command` 引数削除 |

---

## バックエンド実装完了レポート

### 実装サマリー
- **実装日**: 2026-01-25
- **変更ファイル数**: 3 files

### 変更ファイル一覧

| ファイル | 変更内容 |
|---------|---------|
| `internal/handler/files.go` | `DevFolders` 変数に「アーカイブ」フォルダを追加（1行追加） |
| `docs/BACKEND_API.md` | スキャン対象フォルダ一覧とレスポンス例に「アーカイブ」を追加 |
| `internal/handler/doc.go` | スキャン対象フォルダ一覧とレスポンス例に「アーカイブ」を追加 |

### 実装内容の詳細

#### 1. `internal/handler/files.go` (15-21行)

```go
var DevFolders = []string{
    "実装/実装待ち",
    "実装/完了",
    "検討中",
    "資料",
    "アーカイブ",  // 追加
}
```

計画書通り、`DevFolders` スライスに「アーカイブ」を追加。

#### 2. `docs/BACKEND_API.md`

Files API のスキャン対象フォルダ一覧とレスポンス例に「アーカイブ」を追加。

#### 3. `internal/handler/doc.go`

パッケージドキュメントのスキャン対象フォルダ一覧とレスポンス例に「アーカイブ」を追加。

### 計画からの変更点

特になし。計画書通りの実装。

### 実装時の課題

特になし。

### レビュー結果

| 項目 | 結果 |
|------|------|
| Critical | なし |
| Warning | なし |
| `go build` | PASS |
| `go vet` | PASS |
| `go fmt` | PASS |

### 動作確認フロー

```
1. バックエンドサーバーを起動
   cd /Users/user/Ghostrunner && go run ./cmd/server

2. Files API を呼び出し
   curl "http://localhost:8080/api/files?project=/Users/user/Ghostrunner"

3. レスポンスに「アーカイブ」フォルダが含まれることを確認
   - "アーカイブ": [...] がレスポンスJSONに存在すること
```

### 残存する懸念点

特になし。

### 次のステップ

フロントエンド実装（`web/index.html` の変更）を実施する。
