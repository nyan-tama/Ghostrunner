# Ghostrunner に Swift/macOS ネイティブアプリ対応を追加 - 実装計画

## 1. 仕様サマリー

Ghostrunner に Swift/macOS ネイティブアプリのプロジェクト生成機能を追加する。
Web アプリ（Go + Next.js）と同じワークフロー（plan → review → impl → review → test → docs → report）を Swift でも実現する。

**方針:**
- Protocol + Constructor Injection による疎結合（Go の interface + DI と同じ思想）
- SwiftUI + @Observable を採用（macOS 14+）
- Swift Package Manager（SPM）ベース（Claude Code がテキストファイルとして扱いやすい）
- MVP を作るように導く

## 2. 変更ファイル一覧

| ファイル | 変更内容 | 影響度 |
|---------|---------|-------|
| `.claude/agents/swift-planner.md` | 新規作成 | 中 |
| `.claude/agents/swift-plan-reviewer.md` | 新規作成 | 中 |
| `.claude/agents/swift-impl.md` | 新規作成 | 高 |
| `.claude/agents/swift-reviewer.md` | 新規作成 | 中 |
| `.claude/agents/swift-tester.md` | 新規作成 | 中 |
| `.claude/agents/swift-documenter.md` | 新規作成 | 低 |
| `templates/swift-macos/` | 新規作成（テンプレート一式） | 高 |
| `.claude/skills/init/SKILL.md` | macOS アプリ選択肢追加 | 高 |
| `.claude/skills/plan/SKILL.md` | Swift planner ルーティング追加 | 中 |
| `.claude/skills/coding/SKILL.md` | Swift ワークフロー追加 | 中 |
| `.claude/agents/test-planner.md` | Swift 固有の考慮事項追加 | 低 |
| `.claude/CLAUDE.md` | プロジェクト概要にSwift対応を追記 | 低 |

## 3. 実装ステップ

### Step 1: Swift 用エージェント6個の作成

既存の Go 系エージェントをベースに、Swift/SwiftUI 固有の規約に置き換える。

#### 1.1 swift-planner.md

go-planner.md をベースに以下を変更:
- Go 固有の分析（goroutine リーク、nil ポインタ等）→ Swift 固有の分析（Optional 安全性、Actor isolation、Sendable 準拠）
- Clean Architecture の層構成を Swift 版に（Domain/ → Protocols/, Models/, UseCases/ / Infrastructure/ → Repositories/ / Presentation/ → ViewModels/, Views/）
- 検索パターンを Swift ファイル構成に変更
- ドキュメント参照先を `docs/` に変更
- SPM ベースのプロジェクト構造を記載

#### 1.2 swift-plan-reviewer.md

go-plan-reviewer.md をベースに以下を変更:
- API設計チェック → Protocol 設計チェック（Interface Segregation、Sendable 準拠）
- データ構造チェック → Swift 型設計チェック（value type vs reference type、Optional の扱い）
- 実装配置チェック → Swift モジュール構造チェック
- macOS 固有チェック追加（entitlements、Info.plist、権限要求）

#### 1.3 swift-impl.md

go-impl.md をベースに以下を変更:
- コーディング規約を Swift/SwiftUI に全面書き換え
  - Protocol Oriented Programming
  - value type 優先（struct > class、class は @Observable ViewModel 等に限定）
  - `any` / `some` の使い分け
  - @Observable + @MainActor パターン
  - Constructor Injection による DI
  - Result 型 / throws によるエラーハンドリング
- プロジェクト構造を Swift Clean Architecture に
- ビルド・実行コマンドを `swift build` / `swift test` に
- ログ出力規約を `os.Logger` に

#### 1.4 swift-reviewer.md

go-reviewer.md をベースに以下を変更:
- Go 言語固有の検証 → Swift 固有の検証
  - Optional の安全な unwrap（force unwrap 禁止）
  - Actor isolation の正しさ
  - Sendable 準拠
  - メモリリーク（循環参照、strong reference cycle）
- 静的解析コマンドを `swift build` / SwiftLint に
- チェックリストを Swift 版に

#### 1.5 swift-tester.md

go-tester.md をベースに以下を変更:
- テスト規約を XCTest / Swift Testing に
- `@MainActor` テストパターン
- Mock の作り方（Protocol 準拠のモック struct）
- 実行コマンドを `swift test` に

#### 1.6 swift-documenter.md

go-documenter.md をベースに以下を変更:
- GoDoc → Swift DocC コメント形式
- doc.go → 該当なし（Swift にはパッケージドキュメントの慣習が異なる）
- ドキュメント更新対象を `docs/` 配下に

### Step 2: templates/swift-macos/ テンプレートの作成

SPM ベースの最小構成 macOS アプリテンプレートを作成する。

#### ディレクトリ構成

```
templates/swift-macos/
├── Package.swift                      # SPM マニフェスト
├── Sources/
│   └── App/
│       ├── {{PROJECT_NAME}}App.swift  # エントリーポイント（DI 組み立て）
│       ├── ContentView.swift          # メインビュー
│       ├── Domain/
│       │   ├── Models/               # ドメインモデル
│       │   │   └── .gitkeep
│       │   └── Protocols/            # リポジトリ Protocol
│       │       └── .gitkeep
│       ├── UseCases/                  # ユースケース
│       │   └── .gitkeep
│       ├── Infrastructure/            # Protocol の具体実装
│       │   └── .gitkeep
│       └── Presentation/
│           ├── ViewModels/            # @Observable ViewModel
│           │   └── .gitkeep
│           └── Views/                 # SwiftUI Views
│               └── .gitkeep
├── Tests/
│   └── AppTests/
│       └── .gitkeep
├── docs/
│   └── .gitkeep
├── .gitignore
├── Makefile
├── README.md
└── 開発/                              # 開発ドキュメント（base と共通）
```

#### 主要ファイルの内容

**Package.swift**: macOS 14+ ターゲット、Swift 6 言語モード
**App.swift**: DI 組み立て + WindowGroup
**ContentView.swift**: "Hello, World!" レベルの最小 View
**Makefile**: `make build`, `make test`, `make run`, `make clean`, `make lint`
**.gitignore**: .build/, .swiftpm/, *.xcodeproj（SPM 生成物）

### Step 3: /init スキルの拡張

SKILL.md に macOS アプリの選択肢を追加する。

**変更箇所:**

1. **Step 2 Q0/Q1 の後に分岐を追加**: ユーザーの回答から「macOS アプリ」「デスクトップアプリ」「ネイティブアプリ」等のキーワードを検知し、プロジェクトタイプを判定
   - Web アプリ → 既存フロー（変更なし）
   - macOS アプリ → Swift フロー

2. **Q2 の提案テンプレートを macOS 版に分岐**:
   - Docker / ポート / DB の質問をスキップ
   - 代わりに必要な権限（画面録画、カメラ等）を判断

3. **Step 3: テンプレートコピー**を分岐:
   - macOS: `cp -r ./templates/swift-macos/. ~/<プロジェクト名>/`

4. **Step 4: ポート割り当てをスキップ**（macOS アプリには不要）

5. **Step 5-6: .env / 依存解決をスキップ**

6. **Step 7: .claude/ 資産の生成を分岐**:
   - macOS: Go/Next.js 系エージェントを削除、Swift 系エージェントのみ残す
   - CLAUDE.md を Swift/SwiftUI 版で生成

7. **Step 8-10: Git 初期化 + 起動を分岐**:
   - macOS: Docker 起動なし、`swift build` で確認

8. **Step 12: MVP実装**:
   - `/plan` + `/coding` の Swift 版を実行

9. **Step 13: GETTING_STARTED.md を macOS 版で生成**

10. **Step 15: デプロイ準備をスキップ**（macOS は .app 配布、/init のスコープ外）

### Step 4: /plan スキルの拡張

**変更箇所:**

実行方法セクションに Swift ルーティングを追加:

```
- **Swift macOS アプリ** の実装計画 → `swift-planner` エージェント
```

レビューエージェント選択にも追加:

```
- **Swift macOS アプリ** の計画 → `swift-plan-reviewer` エージェント
```

### Step 5: /coding スキルの拡張

**変更箇所:**

Swift macOS アプリの場合のワークフローを追加:

```
Swift macOS:
  swift-impl → swift-reviewer → swift-tester → swift-documenter → コミット
```

フロントエンド/バックエンドの2層ではなく、単一アプリとして1サイクルで完了。

### Step 6: test-planner エージェントの拡張

Swift 固有の考慮セクションを追加:

```
## Swift 固有の考慮

- XCTest / Swift Testing を使用
- @MainActor テストパターン
- Protocol モック（struct で Protocol 準拠）
- async/await テスト
- UI テストは基本不要（ViewModel のロジックテストを優先）
```

### Step 7: CLAUDE.md の更新

プロジェクト概要の構成セクションを更新:
- エージェント数を更新（23 → 29）
- テンプレートに `swift-macos` を追加

## 4. 設計判断とトレードオフ

| 判断 | 選択した方法 | 理由 | 他の選択肢 |
|-----|------------|------|----------|
| プロジェクト構成 | SPM ベース | Claude Code がテキストファイルとして扱いやすい。.xcodeproj はバイナリで差分管理困難 | Xcode プロジェクト（entitlements 管理は楽だが自動生成に不向き） |
| DI パターン | Protocol + Constructor Injection | Go の interface + DI と同じ思想。サードパーティ不要。小-中規模で十分 | Factory ライブラリ（学習コスト増）、swift-dependencies（TCA 向け） |
| 最小 macOS バージョン | macOS 14+ | @Observable が使える最小バージョン | macOS 15+（最新だが互換性低下）、macOS 13（@Observable 使えない） |
| Swift 言語バージョン | Swift 6 | Strict Concurrency を活用し、安全なコードを生成 | Swift 5.x（緩いが将来的に移行必要） |
| /init の分岐方法 | Q0/Q1 の回答からプロジェクトタイプを自動判定 | 追加の質問を増やさない。非エンジニア向け | 明示的にプロジェクトタイプを質問（1ステップ増える） |

## 5. 懸念点と対応方針

### 要確認（実装前に解決が必要）

| 懸念点 | 詳細 | 確認事項 |
|-------|------|---------|
| SPM で entitlements を管理できるか | macOS アプリが画面録画等の権限を必要とする場合、entitlements ファイルが必要。SPM 単体では Info.plist / entitlements の管理が限定的 | テンプレートに entitlements ファイルを含め、Package.swift で参照する方式で対応可能か検証が必要。必要に応じて `swift package generate-xcodeproj` で Xcode プロジェクトを生成するステップを追加 |

### 注意（実装時に考慮が必要）

| 懸念点 | 対応方針 |
|-------|---------|
| /init スキルの複雑化 | 分岐を明確にセクション分けし、Web フローと macOS フローを独立させる |
| settings.json のフック | SwiftFormat / SwiftLint のフックが必要だが、ユーザーの環境に依存。MVP ではフックなしで開始し、後から追加 |
| Go 系エージェントとの重複 | 共通部分（MVP原則、レビュー方針等）を両方に記載。エージェント間で共有テンプレートは作らない（各エージェントが独立して動作するため） |

## 6. 次回実装（MVP外）

以下はMVP範囲外とし、次回以降に実装:

- **with-coredata テンプレート**: Core Data / SwiftData 対応テンプレート（with-db の Swift 版）
- **with-network テンプレート**: URLSession + API クライアント テンプレート
- **SwiftLint / SwiftFormat フック**: settings.json への自動追加
- **App Store デプロイ対応**: コード署名、notarization
- **iOS テンプレート**: iPhone/iPad 対応
- **settings.json の Swift 用フック**: swiftformat 自動実行

## 7. 確認事項

1. SPM ベースで進めてよいか？（entitlements が必要な場合は Xcode プロジェクト生成ステップを追加する方針）
2. macOS 14+ / Swift 6 で問題ないか？
3. /init のプロジェクトタイプ判定は「ユーザーの回答から自動判定」で良いか？（明示的な質問にするか）
