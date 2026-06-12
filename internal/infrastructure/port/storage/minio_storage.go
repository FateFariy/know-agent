package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/condition"
	"github.com/minio/minio-go/v7"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/svc"
)

type MinioStorage struct {
	MinioClient *minio.Client
	Config      config.MinioConf
}

var _ adapter.Storage = (*MinioStorage)(nil)

func NewMinioStorage(svcCtx *svc.ServiceContext) *MinioStorage {
	return &MinioStorage{
		MinioClient: svcCtx.MinioClient,
		Config:      svcCtx.Config.Minio,
	}
}

// UploadOriginalFile 上传原始文件
func (s *MinioStorage) UploadOriginalFile(ctx context.Context, documentID int64, fileName string, bytes []byte, contentType string) (*vo.StoredObjectInfo, error) {
	objectName := fmt.Sprintf("%s/%d/%d-%s", s.Config.ObjectPrefix, documentID, time.Now().Unix(), fileName)
	if err := s.upload(ctx, objectName, bytes, contentType); err != nil {
		return nil, err
	}
	return &vo.StoredObjectInfo{BucketName: s.Config.BucketName, ObjectName: objectName, ObjectUrl: s.buildObjectUrl(objectName)}, nil
}

// UploadParsedText 上传解析后的文本
func (s *MinioStorage) UploadParsedText(ctx context.Context, documentID int64, parsedText string) (string, error) {
	objectName := fmt.Sprintf("%s/%d/%d.txt", s.Config.ParsedTextPrefix, documentID, time.Now().Unix())
	if err := s.upload(ctx, objectName, []byte(parsedText), "text/plain;charset=UTF-8"); err != nil {
		return "", err
	}
	return objectName, nil
}

// DownloadObject 下载对象
func (s *MinioStorage) DownloadObject(ctx context.Context, objectName string) ([]byte, error) {
	object, err := s.MinioClient.GetObject(ctx, s.Config.BucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, common.WrapErr(err, errorx.ErrDocumentStorageFailed.Code, "下载 MinIO 文件失败: "+err.Error())
	}
	defer func() {
		_ = object.Close()
	}()

	b, err := io.ReadAll(object)
	if err != nil {
		return nil, common.WrapErr(err, errorx.ErrDocumentStorageFailed.Code, "下载 MinIO 文件失败: "+err.Error())
	}
	return b, nil
}

// DownloadText 下载文本
func (s *MinioStorage) DownloadText(ctx context.Context, objectName string) (string, error) {
	b, err := s.DownloadObject(ctx, objectName)
	return string(b), err
}

// DeleteObjects 删除对象
func (s *MinioStorage) DeleteObjects(ctx context.Context, objectNameList []string) error {
	validObjectNameList := make([]string, 0)
	seen := make(map[string]bool)
	for _, name := range objectNameList {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			validObjectNameList = append(validObjectNameList, trimmed)
		}
	}

	if len(validObjectNameList) == 0 {
		return nil
	}

	exists, err := s.MinioClient.BucketExists(ctx, s.Config.BucketName)
	if !exists {
		return nil
	}

	for _, objectName := range validObjectNameList {
		if err = s.MinioClient.RemoveObject(ctx, s.Config.BucketName, objectName, minio.RemoveObjectOptions{}); err != nil {
			return common.WrapErr(err, errorx.ErrDocumentStorageFailed.Code, "删除 MinIO 文件失败: "+err.Error())
		}
	}

	return nil
}

// upload 上传对象
func (s *MinioStorage) upload(ctx context.Context, objectName string, b []byte, contentType string) error {
	err := s.ensureBucketExists(ctx)
	if err != nil {
		return common.WrapErr(err, errorx.ErrDocumentStorageFailed.Code, "上传 MinIO 文件失败: "+err.Error())
	}

	options := minio.PutObjectOptions{
		ContentType: condition.Ternary(contentType == "", "application/octet-stream", contentType),
	}
	if _, err = s.MinioClient.PutObject(ctx, s.Config.BucketName, objectName, bytes.NewReader(b), int64(len(b)), options); err != nil {
		return common.WrapErr(err, errorx.ErrDocumentStorageFailed.Code, "上传 MinIO 文件失败: "+err.Error())
	}

	return nil
}

// ensureBucketExists 确保 Bucket 存在
func (s *MinioStorage) ensureBucketExists(ctx context.Context) error {
	exists, err := s.MinioClient.BucketExists(ctx, s.Config.BucketName)
	if err != nil {
		return err
	}
	if !exists {
		return s.MinioClient.MakeBucket(ctx, s.Config.BucketName, minio.MakeBucketOptions{})
	}
	return nil
}

// buildObjectUrl 构建对象 URL
func (s *MinioStorage) buildObjectUrl(objectName string) string {
	endpoint := s.Config.Endpoint
	if strings.HasSuffix(endpoint, "/") {
		endpoint = endpoint[:len(endpoint)-1]
	}
	return fmt.Sprintf("%s/%s/%s", endpoint, s.Config.BucketName, objectName)
}
