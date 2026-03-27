# {{PROJECT_NAME}}

macOS ネイティブアプリ（Swift + SwiftUI）

## 技術スタック

- Swift 6 / SwiftUI
- macOS 14+
- Swift Package Manager

## セットアップ

```bash
make build
```

## 開発コマンド

| コマンド | 説明 |
|---------|------|
| `make build` | ビルド |
| `make run` | ビルドして実行 |
| `make test` | テスト実行 |
| `make clean` | ビルド成果物を削除 |

## プロジェクト構造

```
Sources/App/
├── {{PROJECT_NAME}}App.swift  # エントリーポイント（DI 組み立て）
├── ContentView.swift          # メインビュー
├── Domain/
│   ├── Models/               # ドメインモデル
│   └── Protocols/            # リポジトリ Protocol
├── UseCases/                  # ユースケース
├── Infrastructure/            # Protocol の具体実装
└── Presentation/
    ├── ViewModels/            # @Observable ViewModel
    └── Views/                 # SwiftUI Views
```

## 実装済みの機能

- 基本的なウィンドウ表示
