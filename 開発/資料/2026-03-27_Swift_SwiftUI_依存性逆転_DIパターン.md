# 調査レポート: Swift + SwiftUI における依存性逆転（Dependency Inversion）と疎結合の王道パターン

## 概要

2024-2025年の Swift コミュニティでは、SwiftUI + @Observable 時代の DI は「Protocol + Constructor Injection」を基本とし、SwiftUI View 層では `@Environment` を活用、View 層以外では Property Wrapper ベースの軽量 DI（Factory や swift-dependencies）を採用するのが王道パターンとなっている。Swinject のような Runtime DI コンテナは大規模 UIKit プロジェクト向けであり、モダンな SwiftUI プロジェクトでは Protocol ベース + 軽量ライブラリで十分である。

## 背景

Go の Clean Architecture では interface を定義し、Constructor Injection で依存を注入するパターンが定番である。Swift でも同様のアプローチが可能だが、SwiftUI の View ライフサイクル、@Observable マクロ、Swift 6 の Strict Concurrency など、Swift 固有の考慮事項がある。本調査は、Go の Clean Architecture 経験者が Swift/SwiftUI で同等の設計を実現するための王道パターンを整理するものである。

---

## 調査結果

### 1. Swift における DI の主要パターン

#### パターン A: Protocol + Constructor Injection（最も基本）

Go の interface + DI に最も近いパターン。Swift では Protocol がインターフェースに相当する。

```swift
// Protocol 定義（Go の interface に相当）
protocol UserRepositoryProtocol: Sendable {
    func fetchUser(id: String) async throws -> User
}

// 本番実装
struct APIUserRepository: UserRepositoryProtocol {
    func fetchUser(id: String) async throws -> User {
        // API 呼び出し
    }
}

// テスト用モック
struct MockUserRepository: UserRepositoryProtocol {
    var stubbedUser: User?
    func fetchUser(id: String) async throws -> User {
        guard let user = stubbedUser else { throw TestError.notFound }
        return user
    }
}

// ViewModel（依存を Constructor で受け取る）
@Observable
@MainActor
final class UserViewModel {
    private let repository: any UserRepositoryProtocol

    var user: User?
    var errorMessage: String?

    init(repository: any UserRepositoryProtocol) {
        self.repository = repository
    }

    func loadUser(id: String) async {
        do {
            user = try await repository.fetchUser(id: id)
        } catch {
            errorMessage = "Failed to load user: \(error.localizedDescription)"
        }
    }
}
```

**ポイント:**
- Swift 6 では existential type に `any` キーワードが必須（SE-0335）
- `@Observable` と `@MainActor` を組み合わせて Strict Concurrency に対応
- Protocol に `Sendable` を付与して Actor 境界を越えられるようにする

#### パターン B: @Environment を使った SwiftUI View 層の DI

Apple 公式が推奨する SwiftUI ネイティブの DI メカニズム。

```swift
// 1. EnvironmentKey を定義
struct UserRepositoryKey: EnvironmentKey {
    static let defaultValue: any UserRepositoryProtocol = APIUserRepository()
}

extension EnvironmentValues {
    var userRepository: any UserRepositoryProtocol {
        get { self[UserRepositoryKey.self] }
        set { self[UserRepositoryKey.self] = newValue }
    }
}

// 2. View で使用
struct UserProfileView: View {
    @Environment(\.userRepository) private var repository

    var body: some View {
        // repository を使用
        Text("User Profile")
    }
}

// 3. テスト・プレビューでモック注入
#Preview {
    UserProfileView()
        .environment(\.userRepository, MockUserRepository(stubbedUser: .preview))
}
```

**制約:**
- `@Environment` は SwiftUI View 内でしかアクセスできない
- View 以外の層（ViewModel、Service）では使えない
- 登録漏れ時にクラッシュするリスクがある（EnvironmentObject の場合）

#### パターン C: Property Wrapper ベースの DI（SwiftLee パターン）

View 層以外でも使える軽量 DI。サードパーティ不要。

```swift
// DI Key プロトコル
protocol InjectionKey {
    associatedtype Value
    static var currentValue: Value { get set }
}

// DI コンテナ
struct InjectedValues {
    private static var current = InjectedValues()

    static subscript<K>(key: K.Type) -> K.Value where K: InjectionKey {
        get { key.currentValue }
        set { key.currentValue = newValue }
    }

    static subscript<T>(keyPath: WritableKeyPath<InjectedValues, T>) -> T {
        get { current[keyPath: keyPath] }
        set { current[keyPath: keyPath] = newValue }
    }
}

// Property Wrapper
@propertyWrapper
struct Injected<T> {
    private let keyPath: WritableKeyPath<InjectedValues, T>
    var wrappedValue: T {
        get { InjectedValues[keyPath] }
        set { InjectedValues[keyPath] = newValue }
    }

    init(_ keyPath: WritableKeyPath<InjectedValues, T>) {
        self.keyPath = keyPath
    }
}

// 登録
struct UserRepositoryInjectionKey: InjectionKey {
    static var currentValue: any UserRepositoryProtocol = APIUserRepository()
}

extension InjectedValues {
    var userRepository: any UserRepositoryProtocol {
        get { Self[UserRepositoryInjectionKey.self] }
        set { Self[UserRepositoryInjectionKey.self] = newValue }
    }
}

// 使用
@Observable
@MainActor
final class UserViewModel {
    @ObservationIgnored
    @Injected(\.userRepository) private var repository

    var user: User?
    // ...
}
```

**注意:** `@Observable` クラス内で Property Wrapper を使う場合は `@ObservationIgnored` を付ける必要がある。Observation フレームワークが Property Wrapper を二重に解釈してしまうため。

### 2. SwiftUI @Observable (macOS 14+ / iOS 17+) と DI の組み合わせ

#### @Observable の基本

```swift
@Observable
@MainActor
final class ContentViewModel {
    var items: [Item] = []
    var isLoading = false

    private let service: any ItemServiceProtocol

    init(service: any ItemServiceProtocol) {
        self.service = service
    }
}

// View での使用
struct ContentView: View {
    @State private var viewModel: ContentViewModel

    init(service: any ItemServiceProtocol = ItemService()) {
        _viewModel = State(initialValue: ContentViewModel(service: service))
    }

    var body: some View {
        List(viewModel.items) { item in
            Text(item.name)
        }
    }
}
```

#### @Observable + @Environment の組み合わせ（iOS 17+）

```swift
@Observable
final class ThemeProvider {
    var primaryColor: Color = .blue
    var fontSize: CGFloat = 16
}

// @Observable オブジェクトを Environment に直接注入可能（iOS 17+）
struct MyApp: App {
    @State private var theme = ThemeProvider()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(theme)  // @Observable は直接渡せる
        }
    }
}

struct ContentView: View {
    @Environment(ThemeProvider.self) private var theme

    var body: some View {
        Text("Hello")
            .foregroundStyle(theme.primaryColor)
    }
}
```

### 3. テスタビリティのための Protocol 設計

#### Go の interface に相当する Swift Protocol の書き方

```swift
// Go: type UserRepository interface { ... }
// Swift:
protocol UserRepositoryProtocol: Sendable {
    func fetchUser(id: String) async throws -> User
    func saveUser(_ user: User) async throws
    func deleteUser(id: String) async throws
}

// Go: type UserUseCase struct { repo UserRepository }
// Swift:
@MainActor
final class UserUseCase {
    private let repository: any UserRepositoryProtocol

    init(repository: any UserRepositoryProtocol) {
        self.repository = repository
    }
}
```

#### テスト例

```swift
@MainActor
final class UserViewModelTests: XCTestCase {
    func testLoadUserSuccess() async {
        // Arrange
        let mockRepo = MockUserRepository(
            stubbedUser: User(id: "1", name: "Test User")
        )
        let viewModel = UserViewModel(repository: mockRepo)

        // Act
        await viewModel.loadUser(id: "1")

        // Assert
        XCTAssertEqual(viewModel.user?.name, "Test User")
        XCTAssertNil(viewModel.errorMessage)
    }

    func testLoadUserFailure() async {
        let mockRepo = MockUserRepository(stubbedUser: nil)
        let viewModel = UserViewModel(repository: mockRepo)

        await viewModel.loadUser(id: "1")

        XCTAssertNil(viewModel.user)
        XCTAssertNotNil(viewModel.errorMessage)
    }
}
```

#### Protocol 設計のベストプラクティス

1. **小さく保つ**: Interface Segregation Principle に従い、1 Protocol に 3-5 メソッド程度
2. **Sendable を付与**: Swift 6 の Strict Concurrency に対応
3. **`any` キーワード**: existential type として使う場合は `any Protocol` と書く
4. **`some` キーワード**: 具体型が1つに決まる場合は `some Protocol` でパフォーマンス向上

```swift
// existential（動的ディスパッチ、DI 向き）
func configure(repository: any UserRepositoryProtocol) { }

// opaque type（静的ディスパッチ、パフォーマンス重視）
func process(handler: some EventHandler) { }
```

### 4. Apple 公式の推奨パターン

Apple は特定の DI フレームワークを推奨していないが、以下のメカニズムを公式に提供している:

1. **@Environment / @EnvironmentObject**: SwiftUI View 層での DI メカニズム
2. **@Observable (iOS 17+)**: ObservableObject を置き換えるモダンな状態管理
3. **Swift Testing Framework**: テスト容易性を重視した設計

WWDC セッションでは一貫して「Protocol を使って依存を抽象化し、@Environment で注入する」パターンがサンプルコードに使われている。ただし、View 層以外の DI については公式のガイドラインは特にない。

### 5. Go Clean Architecture との対応関係

```
Go                          Swift
─────────────────────────── ───────────────────────────────
interface                   protocol
struct (implements)         class / struct (conforms to)
constructor func            init()
context.Context             Task / actor isolation
error                       throws / Result<T, Error>
*sql.DB                     any DatabaseProtocol
http.Handler                any RouterProtocol
```

#### Clean Architecture 層構成の対応

```
Go パッケージ構成              Swift モジュール構成
─────────────────────────── ───────────────────────────────
domain/                     Domain/
  entity/user.go              Models/User.swift
  repository/user.go          Protocols/UserRepositoryProtocol.swift
  usecase/user.go             UseCases/UserUseCase.swift

infrastructure/             Infrastructure/
  persistence/user_repo.go    Repositories/APIUserRepository.swift
  api/handler.go              (SwiftUI View が handler に相当)

cmd/server/main.go          App.swift (DI の組み立て)
```

#### App.swift での DI 組み立て（Go の main.go に相当）

```swift
@main
struct MyApp: App {
    // DI の組み立て（Go の main() に相当）
    @State private var userViewModel = UserViewModel(
        repository: APIUserRepository(
            client: URLSessionHTTPClient()
        )
    )

    var body: some Scene {
        WindowGroup {
            ContentView(viewModel: userViewModel)
        }
    }
}
```

---

## 比較表

### DI パターン比較

| 項目 | Constructor Injection | @Environment | Property Wrapper DI | Factory ライブラリ | swift-dependencies |
|------|----------------------|--------------|---------------------|-------------------|-------------------|
| 型安全性 | コンパイル時 | コンパイル時（Key 定義あり） | コンパイル時 | コンパイル時 | コンパイル時 |
| View 層以外で使用 | 可能 | 不可 | 可能 | 可能 | 可能 |
| @Observable 対応 | 問題なし | ネイティブ対応 | @ObservationIgnored 必要 | @InjectedObservable あり | @ObservationIgnored 必要 |
| テスタビリティ | 優秀 | View テストが難しい | 優秀 | 優秀 | 非常に優秀 |
| サードパーティ依存 | なし | なし | なし | あり | あり |
| 学習コスト | 低 | 低 | 中 | 中 | 中-高 |
| 適用規模 | 全規模 | 小-中規模 | 中-大規模 | 中-大規模 | 中-大規模 |
| Swift 6 対応 | 問題なし | 問題なし | 要注意（Sendable） | 対応済み | 対応済み |

### DI フレームワーク比較

| 項目 | Protocol ベース（手動） | Factory | swift-dependencies | Swinject | Needle |
|------|------------------------|---------|-------------------|----------|--------|
| 推奨対象 | 小規模/学習 | 小-中規模 SwiftUI | TCA / 中規模 | 大規模 UIKit | 大規模（Uber 社内） |
| DI 解決 | コンパイル時 | コンパイル時 | コンパイル時 | ランタイム | コード生成 |
| SwiftUI 統合 | 手動 | ネイティブ対応 | 良好 | 限定的 | 限定的 |
| メンテナンス状況 | N/A | 活発 | 活発（Point-Free） | 低下傾向 | メンテナンスモード |
| コミュニティ規模 | N/A | 大 | 大 | 大（レガシー） | 中 |

---

## 既知の問題・注意点

- **@Observable 内の Property Wrapper**: `@Observable` マクロが Property Wrapper と干渉するため、DI 用の Property Wrapper には `@ObservationIgnored` が必須。これを忘れるとコンパイルエラーまたは予期しない動作が発生する
- **Swift 6 Strict Concurrency**: 依存オブジェクトが Actor 境界を越える場合は `Sendable` 準拠が必要。DI コンテナがグローバルな static 変数を持つ場合、data race の警告が出る可能性がある
- **@Environment の制約**: SwiftUI View 階層内でしかアクセスできないため、ViewModel や Service 層から直接使えない。これが「View 層以外の DI」が別途必要になる主な理由
- **EnvironmentObject のクラッシュ**: `@EnvironmentObject` は注入漏れ時にランタイムクラッシュする。`@Environment` + カスタムキーの方が defaultValue を持てるため安全

---

## コミュニティ事例

### 推奨される構成パターン（2024-2025年のコンセンサス）

1. **小規模アプリ**: Protocol + Constructor Injection のみで十分。フレームワーク不要
2. **中規模アプリ**: Factory ライブラリ or swift-dependencies を導入。@Environment と併用
3. **大規模アプリ / モジュラーアーキテクチャ**: Factory + Protocol ベースの Clean Architecture

### コミュニティでの議論ポイント

- Swinject は「UIKit 時代のフレームワーク」という認識が広がっており、新規 SwiftUI プロジェクトでは Factory が推奨される傾向
- Point-Free の swift-dependencies は TCA（The Composable Architecture）利用者に人気だが、TCA 非利用者には学習コストが高い
- 「Protocol ベースで十分」vs「DI コンテナを使うべき」の議論は続いているが、小-中規模では Protocol ベースが主流

---

## 結論・推奨

### Go Clean Architecture 経験者への推奨パターン

**基本方針: Protocol + Constructor Injection を軸にし、SwiftUI View 層では @Environment を併用する**

1. **Domain 層**: Protocol で依存を定義（Go の interface と同じ）
2. **Infrastructure 層**: Protocol の具体実装を配置
3. **Presentation 層**: @Observable ViewModel に Constructor Injection
4. **View 層**: @Environment で ViewModel や設定を注入
5. **App.swift**: 全ての DI を組み立てる（Go の main.go と同じ役割）

### DI コンテナについて

- **小-中規模**: Protocol ベースで十分。サードパーティ不要
- **中規模以上で依存が増えてきたら**: Factory を導入（学習コストが低く SwiftUI 親和性が高い）
- **Swinject は不要**: 新規 SwiftUI プロジェクトでは採用しない

### Swift 6 対応のチェックリスト

- [ ] Protocol に `Sendable` を付与しているか
- [ ] existential type に `any` キーワードを使っているか
- [ ] @Observable ViewModel に `@MainActor` を付けているか
- [ ] DI コンテナの static 変数が Sendable に準拠しているか
- [ ] テストクラスに `@MainActor` を付けているか

---

## ソース一覧

- [Dependency Injection in Swift (2025): Clean Architecture, Better Testing](https://medium.com/@varunbhola1991/dependency-injection-in-swift-2025-clean-architecture-better-testing-7228f971446c) - Medium 記事
- [Managing Dependencies in the Age of SwiftUI: Part I](https://lucasvandongen.dev/dependency_injection_swift_swiftui.php) - Lucas van Dongen ブログ
- [Comparing Four different approaches towards Dependency Injection](https://lucasvandongen.dev/di_frameworks_compared.php) - DI フレームワーク比較
- [Dependency Injection in Swift using latest Swift features - SwiftLee](https://www.avanderlee.com/swift/dependency-injection/) - Property Wrapper パターン
- [Factory - GitHub](https://github.com/hmlongco/Factory) - モダン DI コンテナライブラリ
- [swift-dependencies - GitHub](https://github.com/pointfreeco/swift-dependencies) - Point-Free の DI ライブラリ
- [swift-dependencies + @Observable Discussion](https://github.com/pointfreeco/swift-dependencies/discussions/99) - @Observable との組み合わせ
- [Using @Environment in SwiftUI - SwiftLee](https://www.avanderlee.com/swiftui/environment-property-wrapper/) - @Environment 活用法
- [Dependency Inversion Principle in iOS Swift](https://arifinfrds.com/2025/03/13/dependency-inversion-principle-dip-in-ios-swift/) - DIP 解説
- [Swinject - GitHub](https://github.com/Swinject/Swinject) - Swinject フレームワーク
- [Swinject vs Factory: The Definitive iOS DI Framework Comparison](https://medium.com/@thakurneeshu280/swinject-vs-factory-the-definitive-ios-di-framework-comparison-a3a0bd80e7c8) - フレームワーク比較
- [Adopting strict concurrency in Swift 6 apps - Apple Developer](https://developer.apple.com/documentation/swift/adoptingswift6) - Apple 公式
- [Existential any (SE-0335)](https://github.com/swiftlang/swift-evolution/blob/main/proposals/0335-existential-any.md) - Swift Evolution
- [SwiftUI Views and @MainActor](https://fatbobman.com/en/posts/swiftui-views-and-mainactor/) - @MainActor 解説
- [Approachable Concurrency in Swift 6.2](https://www.avanderlee.com/concurrency/approachable-concurrency-in-swift-6-2-a-clear-guide/) - Swift 6.2 Concurrency

## 関連資料

- このレポートを参照: /discuss, /plan で活用
