package infrastructure

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Storage は S3 互換オブジェクトストレージを管理する構造体。
// Cloudflare R2（本番）と MinIO（ローカル開発）の両方に対応する。
type Storage struct {
	client     *s3.Client
	bucketName string
}

// NewStorage は S3 互換ストレージへの接続を確立し、Storage 構造体を返す。
// usePathStyle は MinIO 等のローカル S3 互換ストレージで true にする。
func NewStorage(endpoint, accessKeyID, accessKeySecret, bucketName string, usePathStyle bool) (*Storage, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, accessKeySecret, ""),
		),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = usePathStyle
	})

	return &Storage{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// Upload はファイルを R2 にアップロードする。
func (s *Storage) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}

	if _, err := s.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("failed to upload object %s: %w", key, err)
	}
	return nil
}

// Download は R2 からファイルをダウンロードする。
func (s *Storage) Download(ctx context.Context, key string) (io.ReadCloser, string, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to download object %s: %w", key, err)
	}

	contentType := ""
	if output.ContentType != nil {
		contentType = *output.ContentType
	}
	return output.Body, contentType, nil
}

// FileInfo はファイルのメタ情報。
type FileInfo struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified"`
}

// List は R2 バケット内のファイル一覧を返す。
func (s *Storage) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	output, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	files := make([]FileInfo, 0, len(output.Contents))
	for _, obj := range output.Contents {
		files = append(files, FileInfo{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: obj.LastModified.Format("2006-01-02T15:04:05Z"),
		})
	}
	return files, nil
}

// Delete は R2 からファイルを削除する。
func (s *Storage) Delete(ctx context.Context, key string) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}); err != nil {
		return fmt.Errorf("failed to delete object %s: %w", key, err)
	}
	return nil
}
