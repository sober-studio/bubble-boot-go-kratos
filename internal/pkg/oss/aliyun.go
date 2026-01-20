package oss

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
)

type aliyunStorage struct {
	client *oss.Client
	bucket *oss.Bucket
	conf   *conf.Data_Oss
	log    *log.Helper
}

func NewAliyunStorage(c *conf.Data_Oss, logger log.Logger) Storage {
	client, err := oss.New(c.Endpoint, c.AccessKeyId, c.AccessKeySecret)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize Aliyun OSS client: %v", err))
	}

	bucket, err := client.Bucket(c.Bucket)
	if err != nil {
		panic(fmt.Sprintf("Failed to get Aliyun OSS bucket: %v", err))
	}

	return &aliyunStorage{
		client: client,
		bucket: bucket,
		conf:   c,
		log:    log.NewHelper(logger),
	}
}

func (s *aliyunStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, isPrivate bool) (string, error) {
	// 设置访问权限
	var options []oss.Option
	if isPrivate {
		options = append(options, oss.ObjectACL(oss.ACLPrivate))
	} else {
		options = append(options, oss.ObjectACL(oss.ACLPublicRead))
	}

	err := s.bucket.PutObject(key, reader, options...)
	if err != nil {
		return "", err
	}

	return key, nil
}

func (s *aliyunStorage) Delete(ctx context.Context, key string) error {
	return s.bucket.DeleteObject(key)
}

func (s *aliyunStorage) GenerateURL(ctx context.Context, key string, isPrivate bool, expires time.Duration) string {
	if !isPrivate {
		schema := "http"
		if s.conf.UseHttps {
			schema = "https"
		}
		// 公开文件直接拼接URL
		// 注意：如果配置了自定义域名(CDN)，则使用Domain；否则使用 endpoint + bucket 的形式
		// 这里简单处理，优先使用 Domain
		return fmt.Sprintf("%s://%s/%s", schema, s.conf.Domain, key)
	}

	// 私有文件生成签名 URL
	signedURL, err := s.bucket.SignURL(key, oss.HTTPGet, int64(expires.Seconds()))
	if err != nil {
		s.log.Errorf("Failed to sign URL: %v", err)
		return ""
	}
	return signedURL
}
