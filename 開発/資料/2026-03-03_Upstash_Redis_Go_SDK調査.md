# 調査レポート: Upstash Redis Go SDK

## 概要

Go バックエンド（Gin）から Upstash Redis に接続するには、**`github.com/redis/go-redis/v9`** が推奨クライアントである。Upstash 独自の Go SDK は存在せず、標準的な go-redis を `REDIS_URL` 環境変数（`rediss://` スキーム）で接続する。ローカル開発は Docker Redis（`redis://`）、本番は Upstash（`rediss://`）と URL を切り替えるだけで同一コードが使える。

## 背景

プロジェクトスターター (`/init`) のテンプレートに Storage オプション（Redis キャッシュ/セッション管理）を追加する際、Upstash Redis を採用する選択肢が浮上している。サーバーレス・低コスト・ゼロインフラ運用という観点から Upstash は有力候補であり、Go バックエンドとの統合方法を調査する。

## 調査結果

### 公式ドキュメント

#### Go クライアントの選択肢

Upstash は Go 向けに **2つの接続方式** を公式にサポートしている。

| クライアント | パッケージ | プロトコル | 推奨用途 |
|-------------|-----------|-----------|---------|
| go-redis v9 | `github.com/redis/go-redis/v9` | TCP (Redis プロトコル) | 通常の Go サーバー（Gin 等） |
| REST API (HTTP) | 標準 `net/http` | HTTPS REST | サーバーレス関数・エッジ環境 |

**重要**: Upstash には `github.com/upstash/upstash-redis-go` のような公式 Go SDK は存在しない。Upstash 公式 SDK（`@upstash/redis`）は TypeScript/JavaScript および Python のみ提供されている。Go では go-redis を使うのが標準。

公式ドキュメント（"Connect Your Client"）でも Go 向けには **redigo** または **go-redis** が案内されており、go-redis v9 が現在の推奨（redigo は旧世代）。

#### 環境変数と接続 URL

**本番（Upstash）で使う環境変数:**

```
REDIS_URL=rediss://default:<PASSWORD>@<ENDPOINT>:<PORT>
```

Upstash コンソールから取得できる接続情報:

| 変数名 | 内容 | 用途 |
|--------|------|------|
| `REDIS_URL` | `rediss://default:xxx@xxx.upstash.io:PORT` | go-redis の `ParseURL()` に渡す |
| `UPSTASH_REDIS_REST_URL` | `https://xxx.upstash.io` | HTTP REST API 用（Go では通常不要） |
| `UPSTASH_REDIS_REST_TOKEN` | Bearer トークン | HTTP REST API 用（Go では通常不要） |

go-redis で使う場合は `REDIS_URL` のみ必要。`UPSTASH_REDIS_REST_*` は TypeScript/Python SDK や直接 HTTP を叩く場合に使う。

**URL スキームの違い:**

- `redis://` - TLS なし（ローカル Docker 向け）
- `rediss://` - TLS あり（Upstash 本番向け、末尾に `s` が付く）

Upstash は TLS が強制で無効化できない。go-redis の `ParseURL()` は `rediss://` スキームを自動検知して TLS を有効化する。

### サンプルコード

#### 推奨実装パターン（Go + Gin）

```go
package infrastructure

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/redis/go-redis/v9"
)

// NewRedisClient は環境変数 REDIS_URL からクライアントを生成する。
// ローカル: redis://localhost:6379
// Upstash:  rediss://default:<password>@<host>:<port>
func NewRedisClient() (*redis.Client, error) {
    url := os.Getenv("REDIS_URL")
    if url == "" {
        return nil, fmt.Errorf("REDIS_URL is not set")
    }

    opt, err := redis.ParseURL(url)
    if err != nil {
        return nil, fmt.Errorf("failed to parse REDIS_URL: %w", err)
    }

    client := redis.NewClient(opt)

    // 接続確認
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("failed to connect to Redis: %w", err)
    }

    return client, nil
}
```

#### 基本操作（Set / Get / Del）

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

func main() {
    ctx := context.Background()

    rdb, err := NewRedisClient()
    if err != nil {
        panic(err)
    }
    defer rdb.Close()

    // Set（TTL あり）
    err = rdb.Set(ctx, "session:abc123", "user-42", 1*time.Hour).Err()
    if err != nil {
        panic(fmt.Errorf("failed to set: %w", err))
    }

    // Get
    val, err := rdb.Get(ctx, "session:abc123").Result()
    if err == redis.Nil {
        fmt.Println("key does not exist")
    } else if err != nil {
        panic(fmt.Errorf("failed to get: %w", err))
    } else {
        fmt.Printf("value: %s\n", val)
    }

    // Del
    deleted, err := rdb.Del(ctx, "session:abc123").Result()
    if err != nil {
        panic(fmt.Errorf("failed to del: %w", err))
    }
    fmt.Printf("deleted %d key(s)\n", deleted)

    // Exists チェック
    exists, err := rdb.Exists(ctx, "session:abc123").Result()
    if err != nil {
        panic(fmt.Errorf("failed to check exists: %w", err))
    }
    fmt.Printf("key exists: %v\n", exists > 0)
}
```

#### go.mod 設定

```
require (
    github.com/redis/go-redis/v9 v9.x.x
)
```

インストール:

```bash
go get github.com/redis/go-redis/v9
```

### ローカル開発とプロダクションの共存

#### 接続 URL の違い

| 環境 | REDIS_URL | TLS |
|------|-----------|-----|
| ローカル Docker | `redis://localhost:6379` | なし |
| Upstash 本番 | `rediss://default:PWD@host.upstash.io:PORT` | 強制あり |

`redis.ParseURL()` がスキームを自動判定するため、**アプリケーションコードは変更不要**。環境変数 `REDIS_URL` の値を切り替えるだけ。

#### docker-compose.yml（ローカル開発）

```yaml
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    command: redis-server --save "" --appendonly no
```

#### .env.local（ローカル）

```
REDIS_URL=redis://localhost:6379
```

#### .env.production（Upstash 本番）

```
REDIS_URL=rediss://default:YOUR_PASSWORD@YOUR_ENDPOINT.upstash.io:PORT
```

#### 注意点

ローカル Docker Redis は TLS なし（`redis://`）で動作するが、go-redis は同一コードで両対応できる。ただし Upstash の `rediss://` URL を TLS なしのサーバーに使おうとするとエラーになる（逆も然り）。環境ごとに正しい URL スキームを使うこと。

### Upstash CLI

#### インストール

**npm（推奨）:**

```bash
npm install -g @upstash/cli
```

**バイナリ直接ダウンロード:**
GitHub Releases（https://github.com/upstash/cli/releases）から macOS/Linux/Windows 向けバイナリを取得可能。

**Homebrew tap は存在しない**（2026年3月時点で確認）。

最新バージョン: **v0.3.0**（2024年12月リリース）

#### 認証

```bash
upstash auth login
# メールアドレスと API キーを入力
# または環境変数で設定:
export UPSTASH_EMAIL=your@email.com
export UPSTASH_API_KEY=your-api-key
```

#### Redis データベース作成

```bash
# データベース作成
upstash redis create --name=my-app-cache --region=ap-northeast-1

# データベース一覧
upstash redis list

# データベース情報取得（JSON 出力）
upstash redis get --id=<database-id> --json
```

利用可能なリージョン例:
- `us-east-1` (バージニア)
- `eu-west-1` (アイルランド)
- `ap-northeast-1` (東京)
- `eu-central-1` (フランクフルト)

作成後、接続情報（endpoint, password, port）がターミナルに表示される。

### Free Tier の制限

2025年3月12日以降の新料金プラン:

| 項目 | 無料プラン |
|------|-----------|
| データベース数 | **1つ（無料）**、以降 $0.5/DB（最大 100 DB）|
| データサイズ | **256 MB** |
| コマンド数 | **500K コマンド/月** |
| 帯域幅 | **10 GB/月** |
| TLS | 強制（無効化不可） |

旧プランでは 10K コマンド/日（= 約 300K/月）だったが、改定により 500K/月に増加。開発・テスト・小規模本番ワークロードに適している。

## 比較表

| 項目 | go-redis v9 | REST API (net/http) |
|------|-------------|---------------------|
| プロトコル | TCP (Redis) | HTTPS |
| Go SDK | `github.com/redis/go-redis/v9` | 標準 `net/http` |
| ローカル Docker 互換 | **そのまま使える** | 不可（REST エミュレーター必要） |
| サーバーレス適性 | 低（TCP 接続維持コスト） | 高（ステートレス HTTP） |
| Go + Gin での推奨 | **推奨** | 非推奨（複雑） |
| パフォーマンス | 高速（ネイティブ Redis） | 低速（HTTP オーバーヘッド） |

## 既知の問題・注意点

- **TLS 強制**: Upstash は TLS を無効化できない。`redis://`（TLS なし）での接続は不可。`rediss://` を必ず使うこと。
- **接続タイムアウト**: Upstash は一定時間アイドル状態が続くと接続を切断することがある（Fly.io コミュニティで報告あり）。`PoolTimeout` や再接続設定を適切に行うこと。
- **go-redis v8 → v9 移行**: 旧ドキュメントでは `github.com/go-redis/redis/v8` を使用しているサンプルがあるが、現在は `github.com/redis/go-redis/v9` が正しいパッケージパス（Redis 公式 org に移管）。
- **Free Tier は 1 DB のみ**: 複数プロジェクトで Upstash Free を共有する場合、1アカウント1DB制限に注意。

## コミュニティ事例

- AWS Lambda + Go + Upstash Redis の公式ブログ事例では `go-redis/v8` + `redis.ParseURL()` で `UPSTASH_REDIS_URL` を読み込むパターンが実証されている。
- Koyeb、Heroku などのホスティングプラットフォームとの連携ドキュメントでも go-redis + `REDIS_URL` パターンが標準として採用されている。
- ローカル開発で Upstash REST API 互換サーバーが必要な場合は `github.com/hiett/serverless-redis-http` (Docker イメージ) が選択肢だが、go-redis を使う場合は不要（Docker Redis をそのまま使える）。

## 結論・推奨

**Go + Gin バックエンドでの推奨構成:**

1. **クライアント**: `github.com/redis/go-redis/v9` を使用する（Upstash 独自 Go SDK は存在しない）
2. **環境変数**: `REDIS_URL` 1つで管理する
   - ローカル: `redis://localhost:6379`
   - 本番: `rediss://default:PWD@host.upstash.io:PORT`
3. **ローカル開発**: Docker `redis:7-alpine` をそのまま使用、コード変更不要
4. **CLI**: `npm install -g @upstash/cli` でインストール、Homebrew tap は存在しない
5. **Free Tier**: 1 DB / 256 MB / 500K コマンド/月（2025年3月改定後）

この構成により、ローカル・本番で同一コードを維持しながら、Upstash のサーバーレス Redis を低コストで利用できる。

## ソース一覧

- [Connect Your Client - Upstash Documentation](https://upstash.com/docs/redis/howto/connectclient) - 公式ドキュメント
- [AWS Lambda + Upstash Redis + Go - Upstash Blog](https://upstash.com/blog/aws-lambda-go-redis) - 公式ブログ（Go 実装例）
- [Distributed tracing with go-redis and Upstash - Upstash Blog](https://upstash.com/blog/go-redis-opentelemetry) - 公式ブログ（go-redis 接続例）
- [Serverless Golang API with Redis - Upstash Documentation](https://upstash.com/docs/redis/tutorials/goapi) - 公式チュートリアル
- [Pricing & Limits - Upstash Documentation](https://upstash.com/docs/redis/overall/pricing) - 公式料金ページ
- [New Pricing and Increased Limits - Upstash Blog](https://upstash.com/blog/redis-new-pricing) - 2025年3月料金改定
- [GitHub - upstash/cli](https://github.com/upstash/cli) - Upstash CLI 公式リポジトリ
- [redis/go-redis - GitHub](https://github.com/redis/go-redis) - go-redis 公式リポジトリ
- [Use Upstash Redis from Go - GitHub Gist](https://gist.github.com/lambrospetrou/53962e22939341ab04926b7fb45aa3b9) - REST vs Native 比較コード
- [Connect with upstash-redis - Upstash Documentation](https://upstash.com/docs/redis/howto/connectwithupstashredis) - 公式ドキュメント（SDK 概要）

## 関連資料

- このレポートを参照: `/plan` でストレージ統合テンプレートの設計に活用
