# 調査レポート: Cloudflare R2 を Go (aws-sdk-go-v2) で使用する方法

## 概要

Cloudflare R2 は S3 互換の API を提供するオブジェクトストレージサービスであり、AWS SDK for Go v2 (`aws-sdk-go-v2`) をそのまま利用してアクセスできる。エンドポイントを R2 に向け、リージョンを `auto` に設定するだけで、S3 と同様の操作（アップロード、ダウンロード、一覧、削除、署名付き URL 生成）が可能である。

## 背景

Go バックエンドからファイルストレージとして Cloudflare R2 を利用するために、必要な依存パッケージ、環境変数、具体的なコード例を調査した。

## 調査結果

### 必要な Go パッケージ (go.mod)

```
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/credentials
go get github.com/aws/aws-sdk-go-v2/service/s3
```

go.mod に追加される主要な依存:

```
require (
    github.com/aws/aws-sdk-go-v2        v1.36.x
    github.com/aws/aws-sdk-go-v2/config  v1.29.x
    github.com/aws/aws-sdk-go-v2/credentials v1.17.x
    github.com/aws/aws-sdk-go-v2/service/s3  v1.76.x
)
```

> **注意**: バージョン互換性の問題が報告されている（後述の「既知の問題」参照）。最新版で問題が発生した場合は、既知の動作確認済みバージョンへのピン留めを検討する。

### 環境変数

| 環境変数 | 説明 | 例 |
|---------|------|-----|
| `R2_ACCOUNT_ID` | Cloudflare アカウント ID | `abc123def456` |
| `R2_ACCESS_KEY_ID` | R2 API トークンのアクセスキー ID | `AKIAIOSFODNN7EXAMPLE` |
| `R2_ACCESS_KEY_SECRET` | R2 API トークンのシークレットアクセスキー | `wJalrXUtnFEMI/K7MDENG/...` |
| `R2_BUCKET_NAME` | R2 バケット名 | `my-bucket` |

エンドポイントは `https://<ACCOUNT_ID>.r2.cloudflarestorage.com` の形式で構成される。

R2 API トークンの作成手順:
1. Cloudflare ダッシュボード > R2 オブジェクトストレージ
2. 「Manage R2 API tokens」を選択
3. 権限を選択（Object Read & Write など）
4. トークン作成後、Access Key ID と Secret Access Key をコピー（Secret は一度しか表示されない）

### サンプルコード

以下に、R2 クライアントの初期化から各操作までの完全なコード例を示す。

#### 1. クライアント初期化

```go
package r2

import (
    "context"
    "fmt"
    "os"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
    S3         *s3.Client
    BucketName string
}

func NewClient() (*Client, error) {
    accountID := os.Getenv("R2_ACCOUNT_ID")
    accessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
    accessKeySecret := os.Getenv("R2_ACCESS_KEY_SECRET")
    bucketName := os.Getenv("R2_BUCKET_NAME")

    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider(accessKeyID, accessKeySecret, ""),
        ),
        config.WithRegion("auto"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }

    client := s3.NewFromConfig(cfg, func(o *s3.Options) {
        o.BaseEndpoint = aws.String(
            fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID),
        )
    })

    return &Client{
        S3:         client,
        BucketName: bucketName,
    }, nil
}
```

**重要なポイント**:
- リージョンは必ず `"auto"` を指定する。`us-east-1` や空文字でもエイリアスとして動作するが、`auto` が推奨。
- エンドポイントは `o.BaseEndpoint` で設定する（非推奨の `aws.EndpointResolverWithOptionsFunc` は使わない）。

#### 2. ファイルアップロード (PutObject)

```go
package r2

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "os"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

// UploadFile はローカルファイルを R2 にアップロードする
func (c *Client) UploadFile(ctx context.Context, key string, filePath string, contentType string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("failed to open file %s: %w", filePath, err)
    }
    defer file.Close()

    _, err = c.S3.PutObject(ctx, &s3.PutObjectInput{
        Bucket:      aws.String(c.BucketName),
        Key:         aws.String(key),
        Body:        file,
        ContentType: aws.String(contentType),
    })
    if err != nil {
        return fmt.Errorf("failed to upload file to R2 (key=%s): %w", key, err)
    }

    return nil
}

// UploadBytes はバイト列を R2 にアップロードする
func (c *Client) UploadBytes(ctx context.Context, key string, data []byte, contentType string) error {
    _, err := c.S3.PutObject(ctx, &s3.PutObjectInput{
        Bucket:      aws.String(c.BucketName),
        Key:         aws.String(key),
        Body:        bytes.NewReader(data),
        ContentType: aws.String(contentType),
    })
    if err != nil {
        return fmt.Errorf("failed to upload bytes to R2 (key=%s): %w", key, err)
    }

    return nil
}

// UploadReader は io.Reader から R2 にアップロードする
// 注意: io.ReadSeeker を実装している必要がある（署名計算のため）
func (c *Client) UploadReader(ctx context.Context, key string, body io.ReadSeeker, contentType string) error {
    _, err := c.S3.PutObject(ctx, &s3.PutObjectInput{
        Bucket:      aws.String(c.BucketName),
        Key:         aws.String(key),
        Body:        body,
        ContentType: aws.String(contentType),
    })
    if err != nil {
        return fmt.Errorf("failed to upload reader to R2 (key=%s): %w", key, err)
    }

    return nil
}
```

#### 3. ファイルダウンロード (GetObject)

```go
package r2

import (
    "context"
    "fmt"
    "io"
    "os"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

// DownloadFile は R2 からファイルをダウンロードしてローカルに保存する
func (c *Client) DownloadFile(ctx context.Context, key string, destPath string) error {
    output, err := c.S3.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(c.BucketName),
        Key:    aws.String(key),
    })
    if err != nil {
        return fmt.Errorf("failed to get object from R2 (key=%s): %w", key, err)
    }
    defer output.Body.Close()

    file, err := os.Create(destPath)
    if err != nil {
        return fmt.Errorf("failed to create file %s: %w", destPath, err)
    }
    defer file.Close()

    _, err = io.Copy(file, output.Body)
    if err != nil {
        return fmt.Errorf("failed to write object to file: %w", err)
    }

    return nil
}

// GetObjectBytes は R2 からオブジェクトをバイト列として取得する
func (c *Client) GetObjectBytes(ctx context.Context, key string) ([]byte, error) {
    output, err := c.S3.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(c.BucketName),
        Key:    aws.String(key),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get object from R2 (key=%s): %w", key, err)
    }
    defer output.Body.Close()

    data, err := io.ReadAll(output.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read object body: %w", err)
    }

    return data, nil
}
```

#### 4. ファイル一覧取得 (ListObjectsV2)

```go
package r2

import (
    "context"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ListObjects は指定したプレフィックスのオブジェクト一覧を取得する
func (c *Client) ListObjects(ctx context.Context, prefix string) ([]types.Object, error) {
    var objects []types.Object

    input := &s3.ListObjectsV2Input{
        Bucket: aws.String(c.BucketName),
    }
    if prefix != "" {
        input.Prefix = aws.String(prefix)
    }

    // ページネーション対応
    paginator := s3.NewListObjectsV2Paginator(c.S3, input)
    for paginator.HasMorePages() {
        page, err := paginator.NextPage(ctx)
        if err != nil {
            return nil, fmt.Errorf("failed to list objects from R2 (prefix=%s): %w", prefix, err)
        }
        objects = append(objects, page.Contents...)
    }

    return objects, nil
}

// ListObjectKeys は指定したプレフィックスのオブジェクトキー一覧を取得する
func (c *Client) ListObjectKeys(ctx context.Context, prefix string) ([]string, error) {
    objects, err := c.ListObjects(ctx, prefix)
    if err != nil {
        return nil, err
    }

    keys := make([]string, 0, len(objects))
    for _, obj := range objects {
        if obj.Key != nil {
            keys = append(keys, *obj.Key)
        }
    }

    return keys, nil
}
```

#### 5. ファイル削除 (DeleteObject)

```go
package r2

import (
    "context"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// DeleteObject は R2 からオブジェクトを削除する
func (c *Client) DeleteObject(ctx context.Context, key string) error {
    _, err := c.S3.DeleteObject(ctx, &s3.DeleteObjectInput{
        Bucket: aws.String(c.BucketName),
        Key:    aws.String(key),
    })
    if err != nil {
        return fmt.Errorf("failed to delete object from R2 (key=%s): %w", key, err)
    }

    return nil
}

// DeleteObjects は R2 から複数のオブジェクトを一括削除する
func (c *Client) DeleteObjects(ctx context.Context, keys []string) error {
    if len(keys) == 0 {
        return nil
    }

    objectIDs := make([]types.ObjectIdentifier, len(keys))
    for i, key := range keys {
        objectIDs[i] = types.ObjectIdentifier{
            Key: aws.String(key),
        }
    }

    _, err := c.S3.DeleteObjects(ctx, &s3.DeleteObjectsInput{
        Bucket: aws.String(c.BucketName),
        Delete: &types.Delete{
            Objects: objectIDs,
            Quiet:   aws.Bool(true),
        },
    })
    if err != nil {
        return fmt.Errorf("failed to delete objects from R2: %w", err)
    }

    return nil
}
```

#### 6. 署名付き URL (Presigned URL) の生成

```go
package r2

import (
    "context"
    "fmt"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

// GeneratePresignedDownloadURL はダウンロード用の署名付き URL を生成する
func (c *Client) GeneratePresignedDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
    presignClient := s3.NewPresignClient(c.S3)

    result, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(c.BucketName),
        Key:    aws.String(key),
    }, s3.WithPresignExpires(expiry))
    if err != nil {
        return "", fmt.Errorf("failed to generate presigned download URL (key=%s): %w", key, err)
    }

    return result.URL, nil
}

// GeneratePresignedUploadURL はアップロード用の署名付き URL を生成する
func (c *Client) GeneratePresignedUploadURL(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error) {
    presignClient := s3.NewPresignClient(c.S3)

    input := &s3.PutObjectInput{
        Bucket: aws.String(c.BucketName),
        Key:    aws.String(key),
    }
    if contentType != "" {
        input.ContentType = aws.String(contentType)
    }

    result, err := presignClient.PresignPutObject(ctx, input, s3.WithPresignExpires(expiry))
    if err != nil {
        return "", fmt.Errorf("failed to generate presigned upload URL (key=%s): %w", key, err)
    }

    return result.URL, nil
}
```

**署名付き URL のポイント**:
- 有効期限は 1 秒 ~ 7 日間（最大 604,800 秒）
- GET, HEAD, PUT, DELETE をサポート（POST はサポートされない）
- URL の生成はクライアントサイドで完結し、R2 への通信は発生しない（AWS Signature Version 4 署名）
- 署名付き URL はベアラートークンとして扱う（URL を知っている人は誰でもアクセス可能）
- カスタムドメインでは使用できない（S3 API ドメインが必要）

#### 7. 使用例 (main 関数)

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "yourmodule/internal/infrastructure/r2"
)

func main() {
    ctx := context.Background()

    // R2 クライアントの初期化
    client, err := r2.NewClient()
    if err != nil {
        log.Fatalf("failed to create R2 client: %v", err)
    }

    // アップロード
    err = client.UploadFile(ctx, "documents/report.pdf", "/tmp/report.pdf", "application/pdf")
    if err != nil {
        log.Fatalf("upload failed: %v", err)
    }
    fmt.Println("Upload completed")

    // 一覧取得
    objects, err := client.ListObjects(ctx, "documents/")
    if err != nil {
        log.Fatalf("list failed: %v", err)
    }
    for _, obj := range objects {
        fmt.Printf("Key: %s, Size: %d, LastModified: %v\n", *obj.Key, *obj.Size, *obj.LastModified)
    }

    // ダウンロード用署名付き URL の生成（有効期限: 1時間）
    url, err := client.GeneratePresignedDownloadURL(ctx, "documents/report.pdf", 1*time.Hour)
    if err != nil {
        log.Fatalf("presign failed: %v", err)
    }
    fmt.Printf("Download URL: %s\n", url)

    // 削除
    err = client.DeleteObject(ctx, "documents/report.pdf")
    if err != nil {
        log.Fatalf("delete failed: %v", err)
    }
    fmt.Println("Delete completed")
}
```

### R2 の S3 API 互換性

R2 が対応している主な S3 API 操作:

| カテゴリ | 操作 | 対応状況 |
|---------|------|---------|
| バケット | ListBuckets, HeadBucket, CreateBucket, DeleteBucket | 対応 |
| オブジェクト | PutObject, GetObject, HeadObject, DeleteObject, DeleteObjects | 対応 |
| 一覧 | ListObjectsV2, ListObjects | 対応 |
| コピー | CopyObject | 対応 |
| マルチパート | CreateMultipartUpload, UploadPart, CompleteMultipartUpload, AbortMultipartUpload | 対応 |
| 署名付き URL | GET, HEAD, PUT, DELETE | 対応 |
| CORS | GetBucketCors, PutBucketCors, DeleteBucketCors | 対応 |
| ACL | - | 非対応 |
| バケットポリシー | - | 非対応 |
| バージョニング | - | 非対応 |
| タグ付け | - | 非対応 |
| オブジェクトロック | - | 非対応 |

## 既知の問題・注意点

### 1. aws-sdk-go-v2 のバージョン互換性問題

AWS SDK は頻繁に更新されるが、新しいバージョンで R2 が未実装のパラメータが追加されることがあり、互換性が壊れるケースが報告されている。

- [Cloudflare should say which versions of the SDK R2 is supposed to work with](https://community.cloudflare.com/t/cloudflare-should-say-which-versions-of-the-sdk-r2-is-supposed-to-work-with-or-just-fork-the-aws-sdk/590192): SDK バージョンの明示を求めるコミュニティの議論
- [Outdated code example for R2 with aws-sdk-go-v2 (Issue #12043)](https://github.com/cloudflare/cloudflare-docs/issues/12043): 公式ドキュメントのコード例が古く、`Invalid region` エラーが発生していた問題。`BaseEndpoint` 方式への修正が必要だった
- [Issue #12179](https://github.com/cloudflare/cloudflare-docs/issues/12179): SDK バージョン互換性に関する Issue

**動作確認済みバージョンの組み合わせ**（問題が起きた場合に参照）:

組み合わせ A:
```
github.com/aws/aws-sdk-go-v2        v1.18.0
github.com/aws/aws-sdk-go-v2/config  v1.18.18
github.com/aws/aws-sdk-go-v2/credentials v1.13.17
github.com/aws/aws-sdk-go-v2/service/s3  v1.29.0
```

組み合わせ B (より新しい):
```
github.com/aws/aws-sdk-go-v2        v1.24.0
github.com/aws/aws-sdk-go-v2/config  v1.26.1
github.com/aws/aws-sdk-go-v2/credentials v1.16.12
github.com/aws/aws-sdk-go-v2/service/s3  v1.47.5
```

### 2. エンドポイント設定の非推奨パターン

以前のドキュメントで使用されていた `aws.EndpointResolverWithOptionsFunc` は非推奨。現在は `s3.Options.BaseEndpoint` を使用する:

```go
// 推奨（現在）
client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.BaseEndpoint = aws.String("https://<ACCOUNT_ID>.r2.cloudflarestorage.com")
})

// 非推奨（古いドキュメントに記載されていたパターン）
// cfg, _ := config.LoadDefaultConfig(context.TODO(),
//     config.WithEndpointResolverWithOptions(...)
// )
```

### 3. マルチパートアップロードの問題

`aws-sdk-go-v2` の `feature/s3/manager` を使った大容量ファイルのマルチパートアップロードで失敗するケースが報告されている。

- [R2 multipart upload with aws-sdk-go-v2 always fails](https://community.cloudflare.com/t/r2-multipart-upload-with-aws-sdk-go-v2-always-fails/794423): `UploadPart` エラーで最大リトライ回数を超過

回避策:
- `PartSize` を大きく設定する（例: 100MB）
- 小さなファイル（< 100MB）では `PutObject` を直接使用する
- マルチパートが必要な場合は `Concurrency` を低めに設定する

### 4. Body の io.ReadSeeker 要件

`PutObject` の `Body` フィールドは `io.ReadSeeker` を実装している必要がある（署名計算とコンテンツ長の決定に使用される）。`bytes.Reader`、`os.File`、`strings.NewReader` はいずれも `io.ReadSeeker` を実装している。ストリーミングデータの場合は注意が必要。

### 5. R2 の制限事項

- オブジェクトサイズ上限: 5 TB（マルチパートアップロード時）
- 単一 PUT リクエスト上限: 5 GB
- バケット名: 3-63 文字、小文字英数字とハイフンのみ
- 署名付き URL はカスタムドメインで使用不可（S3 API ドメインが必要）
- POST 操作（マルチパートフォームアップロード）の署名付き URL は非対応

## コミュニティ事例

- [PutObjects Cloudflare R2 Go (Cloudflare Community)](https://community.cloudflare.com/t/putobjects-cloudflare-r2-go/635926): Go での PutObject 実装に関する質問と回答
- [go-r2 (GitHub)](https://github.com/rneko26/go-r2): コミュニティ製の R2 用 Go ライブラリ。設定構造体とAPIの使い方を提供
- [Create PresignedURL in R2 + Golang (Medium)](https://medium.com/@humamalamin13/create-presignedurl-in-r2-golang-3cc4c9d09a4d): Go での署名付き URL 生成の解説記事

## 結論・推奨

1. **依存パッケージ**: `aws-sdk-go-v2` とそのサブパッケージ（config, credentials, service/s3）を使用する。専用の R2 SDK は不要。

2. **クライアント初期化**: `config.LoadDefaultConfig` で静的クレデンシャルとリージョン `auto` を設定し、`s3.NewFromConfig` で `BaseEndpoint` を R2 エンドポイントに指定する。

3. **環境変数**: `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_ACCESS_KEY_SECRET`, `R2_BUCKET_NAME` の 4 つを設定する。

4. **エンドポイント設定**: 非推奨の `EndpointResolver` ではなく `BaseEndpoint` を使用する。

5. **バージョン管理**: 最新の `aws-sdk-go-v2` で基本的には動作するが、互換性問題が発生した場合は動作確認済みバージョンにピン留めする。

6. **大容量ファイル**: 100MB 未満のファイルには `PutObject` を直接使用する。それ以上の場合は `feature/s3/manager` の `Uploader` を使用するが、R2 との互換性問題に注意する。

7. **署名付き URL**: クライアントからの直接アップロード/ダウンロードには署名付き URL を活用する。有効期限は短めに設定し、ベアラートークンとして扱う。

## ソース一覧

- [aws-sdk-go - Cloudflare R2 docs](https://developers.cloudflare.com/r2/examples/aws/aws-sdk-go/) - 公式ドキュメント（Go SDK 例）
- [S3 API compatibility - Cloudflare R2 docs](https://developers.cloudflare.com/r2/api/s3/api/) - 公式ドキュメント（S3 API 互換性）
- [Presigned URLs - Cloudflare R2 docs](https://developers.cloudflare.com/r2/api/s3/presigned-urls/) - 公式ドキュメント（署名付き URL）
- [Authentication - Cloudflare R2 docs](https://developers.cloudflare.com/r2/api/tokens/) - 公式ドキュメント（API トークン）
- [Upload objects - Cloudflare R2 docs](https://developers.cloudflare.com/r2/objects/upload-objects/) - 公式ドキュメント（アップロード）
- [Get started with S3 - Cloudflare R2 docs](https://developers.cloudflare.com/r2/get-started/s3/) - 公式ドキュメント（S3 入門）
- [Outdated code example - Issue #12043](https://github.com/cloudflare/cloudflare-docs/issues/12043) - GitHub Issue（コード例の修正）
- [SDK version compatibility - Issue #12179](https://github.com/cloudflare/cloudflare-docs/issues/12179) - GitHub Issue（バージョン互換性）
- [SDK version compatibility - Cloudflare Community](https://community.cloudflare.com/t/cloudflare-should-say-which-versions-of-the-sdk-r2-is-supposed-to-work-with-or-just-fork-the-aws-sdk/590192) - コミュニティ議論
- [R2 multipart upload fails - Cloudflare Community](https://community.cloudflare.com/t/r2-multipart-upload-with-aws-sdk-go-v2-always-fails/794423) - コミュニティ（マルチパート問題）
- [go-r2 - GitHub](https://github.com/rneko26/go-r2) - コミュニティ製ライブラリ
- [Amazon S3 examples using SDK for Go V2](https://docs.aws.amazon.com/code-library/latest/ug/go_2_s3_code_examples.html) - AWS 公式（Go SDK v2 例）

## 関連資料

- このレポートを参照: /discuss, /plan で活用
