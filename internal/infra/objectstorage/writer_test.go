package objectstorage

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingPutObjectClient struct {
	bucketName string
	objectKey  string
	body       []byte
	size       int64
	options    minio.PutObjectOptions
	// PutObjectに失敗させたい原因エラー。nilのままなら成功を記録する。
	putErr error
}

func (c *recordingPutObjectClient) PutObject(
	_ context.Context,
	bucketName string,
	objectKey string,
	reader io.Reader,
	objectSize int64,
	options minio.PutObjectOptions,
) (minio.UploadInfo, error) {
	// 失敗伝播の検証では保存を試さず原因エラーをそのまま返す。
	if c.putErr != nil {
		return minio.UploadInfo{}, c.putErr
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return minio.UploadInfo{}, err
	}
	c.bucketName = bucketName
	c.objectKey = objectKey
	c.body = body
	c.size = objectSize
	c.options = options
	return minio.UploadInfo{Key: objectKey, Size: objectSize}, nil
}

func TestWriterPutJSON_JSON用メタデータでアップロードする(t *testing.T) {
	// Given
	client := &recordingPutObjectClient{}
	writer := newWriter(client, "song-snapshots")
	body := []byte(`{"songs":[]}`)

	// When
	err := writer.PutJSON(context.Background(), "v1/songs.json", body)

	// Then
	require.NoError(t, err)
	assert.Equal(t, "song-snapshots", client.bucketName)
	assert.Equal(t, "v1/songs.json", client.objectKey)
	assert.Equal(t, body, client.body)
	assert.Equal(t, int64(len(body)), client.size)
	assert.Equal(t, "application/json; charset=utf-8", client.options.ContentType)
	// 固定キーの上書きなので、ブラウザは短時間だけ再利用し、エッジはパージ失敗時の上限まで保持する。
	assert.Equal(t, "public, max-age=300, s-maxage=3600", client.options.CacheControl)
}

func TestWriterPutJSON_PutObject失敗を原因保持して伝播する(t *testing.T) {
	sentinelErr := errors.New("put object failed")
	tests := []struct {
		name string
		// Given: PutObjectに失敗させる原因エラー
		putErr error
	}{
		{
			name:   "固定の失敗原因をそのまま伝播する",
			putErr: sentinelErr,
		},
		{
			name:   "キャンセルの原因を保持する",
			putErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			client := &recordingPutObjectClient{putErr: tt.putErr}
			writer := newWriter(client, "song-snapshots")

			// When
			err := writer.PutJSON(context.Background(), "v1/songs.json", []byte(`{"songs":[]}`))

			// Then
			require.Error(t, err)
			// 原因エラーを文字列化せずerrors.Isで辿れることと、どの固定キーの失敗か分かることを検証する。
			assert.ErrorIs(t, err, tt.putErr)
			assert.Contains(t, err.Error(), "v1/songs.json")
		})
	}
}
