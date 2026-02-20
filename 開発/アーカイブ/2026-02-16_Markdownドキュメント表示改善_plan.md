# Markdownドキュメント表示改善 実装計画

## フロントエンド計画

### 1. 仕様サマリー

`/docs` ページのMarkdownドキュメント表示において、`@tailwindcss/typography` プラグインが Tailwind v4 で有効化されていないため、`MarkdownViewer.tsx` の `prose` クラスが全て無視されている。テーブル、見出し、リスト、コードブロック、引用、水平線等のスタイルが未適用。

案Bとして、typography プラグインの有効化と、`rehype-highlight` 用の highlight.js テーマCSS のインポートを行う。

### 2. 変更ファイル一覧

| ファイル | 変更内容 | 影響度 |
|---------|---------|-------|
| `frontend/src/app/globals.css` | `@plugin "@tailwindcss/typography"` の追加 + highlight.js テーマCSS のインポート追加 | 高 |
| `frontend/package.json` | `highlight.js` を明示的な依存に追加 | 低 |

### 3. 実装ステップ

#### Step 0: highlight.js を明示的な依存に追加

現在 `highlight.js` は `rehype-highlight` の間接依存としてのみインストールされている。CSS を直接 import するため、明示的な依存に追加する。

```bash
cd frontend && npm install highlight.js
```

#### Step 1: globals.css への typography プラグイン有効化と highlight.js テーマCSS の追加

**対象**: `frontend/src/app/globals.css`

**追加するもの**:
- `@plugin "@tailwindcss/typography"` ディレクティブ（`@import "tailwindcss"` の直後に配置）
- highlight.js の `github` テーマCSS の import

**配置順序**:
1. `@import "tailwindcss"`
2. `@plugin "@tailwindcss/typography"`（Tailwind plugin ディレクティブ。`@import` の直後に配置）
3. `@import "highlight.js/styles/github.css"`（CSS import）
4. 既存の `:root`、`@theme inline` 等

**注意点**:
- `@plugin` は Tailwind のビルドプロセスで処理されるディレクティブであり、`@import "tailwindcss"` の直後に配置するのが最も安全
- テーマは `github`（ライト系）を選択。背景が白系（`bg-gray-100` + `bg-white`）のため
- `MarkdownViewer.tsx` の変更は不要。`prose` クラス指定は既に正しい

#### Step 2: 動作確認

以下のMarkdown要素が全て正しく表示されることを確認:
- テーブル（罫線、ヘッダー背景）
- 見出し（h2-h6のサイズ・余白）
- 順序付き/順序なしリスト（マーカー・インデント）
- コードブロック（構文ハイライト含む）
- 引用（blockquote）
- 水平線（hr）
- リンク色
- 段落間余白

### 4. 設計判断とトレードオフ

| 判断 | 選択した方法 | 理由 | 他の選択肢 |
|-----|------------|------|----------|
| highlight.js テーマ | `github` (ライト系) | 背景が白系のため自然に調和 | `atom-one-light`（同等だが github の方が馴染み深い） |
| CSS の読み込み場所 | `globals.css` | 既に全ての CSS import が集約されている場所 | `layout.tsx` での JS import |
| 変更範囲 | `globals.css` のみ | `MarkdownViewer.tsx` の `prose` クラス指定は既に正しい | コンポーネント側のカスタマイズ（不要） |

### 5. 懸念点と対応方針

| 懸念点 | 対応方針 |
|-------|---------|
| `prose` のコードブロック背景色/パディングと highlight.js テーマの競合 | `prose` は `pre` と `code` に背景色・パディングを適用し、highlight.js の `github.css` も `.hljs` に `background` と `padding` を設定する。二重適用が発生した場合は `prose-pre:p-0` 等のオーバーライドクラスを `MarkdownViewer.tsx` の article に追加して対処する |
| `prose` のインラインコードスタイルと `MarkdownViewer.tsx` のカスタムスタイルの競合 | `MarkdownViewer.tsx` でインラインコードに `bg-gray-100 px-1 py-0.5 rounded text-sm` を直接適用しているが、`prose` 有効化後に `prose` のインラインコードスタイルと競合する可能性がある。問題が発生した場合は `prose-code:bg-transparent prose-code:p-0 prose-code:text-inherit` 等で prose のインラインコードスタイルを無効化する |
| 他のページへの影響 | `prose` クラスを使用しているのは `MarkdownViewer.tsx` のみ。highlight.js の CSS は `.hljs` プレフィックス付きクラスのみに適用。他ページへの影響なし |
| `@plugin` と CSS `@import` の記述順序 | `@plugin` は `@import "tailwindcss"` の直後に配置する（他の CSS `@import` より前） |
| `highlight.js` の間接依存リスク | CSS を直接 import するため、`highlight.js` を明示的な依存に追加する（Step 0） |

### 6. MVP外（次回以降）

- テーブルの横スクロール対応
- 見出しへのアンカーリンク付与
- 目次（TOC）の自動生成
- ダークモード対応

## テストプラン

### テスト方針

変更は CSS 設定のみ（`globals.css` への2行追加）であり、ロジック変更がないためユニットテストの追加は不要。手動での表示確認で十分。

### 手動確認チェックリスト

1. `/docs` ページで既存のMarkdownファイルを開く
2. 以下の要素が正しく表示されることを確認:
   - テーブル: 罫線あり、ヘッダー行の区別あり
   - 見出し: h2-h6 のサイズ差あり、適切な余白
   - リスト: マーカー（ul: ドット、ol: 番号）あり、インデントあり
   - コードブロック: 背景色あり、構文ハイライト色あり、パディングの二重適用がないこと
   - インラインコード: 背景色・パディングが意図通り（prose との二重スタイル適用がないこと）
   - 引用: 左ボーダーあり
   - 水平線: 表示あり
   - リンク: 青色で表示
3. `/docs` 以外のページ（ダッシュボード等）に表示崩れがないこと
4. `npm run build` がエラーなく成功すること

---

## 実装完了レポート

### 実装サマリー
- **実装日**: 2026-02-16
- **変更ファイル数**: 3 files（`globals.css`, `package.json`, `package-lock.json`）
- **実装方針**: 計画書の案B（typographyプラグイン有効化 + 構文ハイライトCSS追加）を忠実に実施

### 変更ファイル一覧

| ファイル | 変更内容 |
|---------|---------|
| `frontend/src/app/globals.css` | `@plugin "@tailwindcss/typography"` と `@import "highlight.js/styles/github.css"` の2行を追加。`@import "tailwindcss"` の直後に配置 |
| `frontend/package.json` | `highlight.js` (`^11.11.1`) を `dependencies` に明示的に追加 |
| `frontend/package-lock.json` | `highlight.js` の依存関係をロックファイルに反映 |

### 変更内容の詳細

#### `frontend/src/app/globals.css`

```css
@import "tailwindcss";
@plugin "@tailwindcss/typography";       /* 追加: Tailwind v4 での typography プラグイン有効化 */
@import "highlight.js/styles/github.css"; /* 追加: コードブロック構文ハイライトテーマ */

:root {
  /* 以下既存コードは変更なし */
```

計画書の Step 1 で定義した配置順序（`@import "tailwindcss"` -> `@plugin` -> `@import highlight.js` -> 既存CSS）の通りに実装されている。

#### `frontend/package.json`

`dependencies` に `"highlight.js": "^11.11.1"` を追加。計画書 Step 0 の通り、間接依存から明示的な依存に昇格。

#### `frontend/src/components/docs/MarkdownViewer.tsx`

変更なし。計画書の判断通り、既存の `prose` クラス指定が正しいため修正不要。以下のクラスが typography プラグイン有効化により機能するようになった:

```
prose prose-lg max-w-none prose-headings:text-gray-800 prose-a:text-blue-600 prose-code:before:content-none prose-code:after:content-none
```

### 計画からの変更点

特になし。計画書の Step 0 -> Step 1 の順序通りに実装されており、追加の判断や仕様外の対応は発生していない。

### 実装時の課題

#### ビルド・テストで苦戦した点

特になし。以下の確認が全て成功:
- `npm run build`: OK
- `tsc --noEmit`: OK
- ESLint: 既存エラー1件のみ（`useSessionManagement.ts` の set-state-in-effect。本変更と無関係）

#### 技術的に難しかった点

特になし。CSS設定のみの変更であり、計画書の指示通りに2行追加するだけで完了。

### 残存する懸念点

今後注意が必要な点:

- **prose とインラインコードのスタイル競合の可能性**: `MarkdownViewer.tsx` でインラインコードに `bg-gray-100 px-1 py-0.5 rounded text-sm` を直接適用している。`prose` のインラインコードスタイルとの競合が発生した場合は、計画書に記載の通り `prose-code:bg-transparent prose-code:p-0 prose-code:text-inherit` 等のオーバーライドで対処可能。現時点では `prose-code:before:content-none prose-code:after:content-none` が既に設定されており、バッククォート装飾は抑制済み
- **prose とコードブロック背景の二重適用**: `prose` の `pre`/`code` 背景色と highlight.js の `.hljs` 背景色が二重に適用される可能性がある。問題が確認された場合は `prose-pre:p-0` 等で対処可能
- **highlight.js CSS のグローバル影響**: `github.css` は `.hljs` プレフィックス付きクラスにのみ適用されるため他ページへの影響は限定的だが、将来 `/docs` 以外でコードブロックを使用する場合はスタイルが適用される点に留意

### 動作確認フロー

```
1. make restart-frontend-logs でフロントエンドを起動
2. ブラウザで /docs ページを開く
3. 既存のMarkdownファイル（実装計画書や検討書など）を選択
4. 以下の要素が正しく表示されることを確認:
   - テーブル: 罫線あり、ヘッダー行の区別あり
   - 見出し: h2-h6 のサイズ差あり、適切な余白
   - リスト: マーカー（ul: ドット、ol: 番号）あり、インデントあり
   - コードブロック: 背景色あり、構文ハイライト色あり
   - インラインコード: 背景色・パディングが意図通り
   - 引用: 左ボーダーあり
   - 水平線: 表示あり
   - リンク: 青色で表示
5. /docs 以外のページ（ダッシュボード等）に表示崩れがないことを確認
```

### デプロイ後の確認事項

- [ ] `/docs` ページでテーブル・見出し・リスト・コードブロックが正しく表示されること
- [ ] コードブロックに構文ハイライト（色分け）が適用されていること
- [ ] `/docs` 以外のページに表示崩れがないこと
- [ ] Mermaid図が引き続き正常にレンダリングされること
