package svc

import (
	"github.com/go-playground/validator/v10"
	"github.com/google/wire"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/swiftbit/know-agent/internal/config"
)

var ProviderSet = wire.NewSet(
	NewServiceContext,
)

type ServiceContext struct {
	Config      config.Config
	Validate    *validator.Validate
	MinioClient *minio.Client
}

func NewServiceContext(c config.Config, validate *validator.Validate) *ServiceContext {
	return &ServiceContext{
		Config:   c,
		Validate: validate,
	}
}

// NewMinioClient 创建 Minio 客户端
func (s *ServiceContext) NewMinioClient() *minio.Client {
	endpoint := s.Config.Minio.Endpoint
	accessKeyID := s.Config.Minio.AccessKeyID
	accessKey := s.Config.Minio.SecretAccessKey
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, accessKey, ""),
		Secure: s.Config.Minio.UseSSL,
	})
	if err != nil {
		return nil
	}
	return minioClient
}
