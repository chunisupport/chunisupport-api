package objectstorage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const jsonContentType = "application/json; charset=utf-8"

type putObjectClient interface {
	PutObject(
		ctx context.Context,
		bucketName string,
		objectName string,
		reader io.Reader,
		objectSize int64,
		opts minio.PutObjectOptions,
	) (minio.UploadInfo, error)
}

// Writer はminio-goを使用してオブジェクトストレージへJSONを保存します。
type Writer struct {
	client     putObjectClient
	bucketName string
}

// NewWriter はS3互換エンドポイントへ接続するJSON Writerを生成します。
func NewWriter(cfg config.ObjectStorageConfig) (*Writer, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure:       cfg.Secure,
		Region:       "auto",
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create object storage client: %w", err)
	}
	return newWriter(client, cfg.BucketName), nil
}

func newWriter(client putObjectClient, bucketName string) *Writer {
	return &Writer{client: client, bucketName: bucketName}
}

// PutJSON は指定した固定キーへJSONを上書きします。
// PutObjectが成功するまで既存オブジェクトは置き換わらないため、失敗時は前回のJSONが残ります。
func (w *Writer) PutJSON(ctx context.Context, objectKey string, body []byte) error {
	_, err := w.client.PutObject(
		ctx,
		w.bucketName,
		objectKey,
		bytes.NewReader(body),
		int64(len(body)),
		minio.PutObjectOptions{ContentType: jsonContentType},
	)
	if err != nil {
		return fmt.Errorf("failed to put object storage object %s: %w", objectKey, err)
	}
	return nil
}
