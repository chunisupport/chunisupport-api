package objectstorage

import (
	"context"
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
}

func (c *recordingPutObjectClient) PutObject(
	_ context.Context,
	bucketName string,
	objectKey string,
	reader io.Reader,
	objectSize int64,
	options minio.PutObjectOptions,
) (minio.UploadInfo, error) {
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
}
