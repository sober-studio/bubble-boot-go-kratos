package oss

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
)

type qiniuStorage struct {
	mac  *qbox.Mac
	cfg  *storage.Config
	conf *conf.Data_Oss
	log  *log.Helper
}

func NewQiniuStorage(c *conf.Data_Oss, logger log.Logger) Storage {
	mac := qbox.NewMac(c.AccessKeyId, c.AccessKeySecret)

	cfg := storage.Config{}
	// cfg.Zone = &storage.ZoneHuadong
	cfg.UseHTTPS = c.UseHttps
	cfg.UseCdnDomains = false

	s := &qiniuStorage{
		mac:  mac,
		cfg:  &cfg,
		conf: c,
		log:  log.NewHelper(logger),
	}
	s.fillZone(&cfg)
	return s
}

func (s *qiniuStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string, isPrivate bool) (string, error) {
	putPolicy := storage.PutPolicy{
		Scope: s.conf.Bucket,
	}
	upToken := putPolicy.UploadToken(s.mac)

	formUploader := storage.NewFormUploader(s.cfg)
	ret := storage.PutRet{}

	putExtra := storage.PutExtra{
		MimeType: contentType,
	}

	err := formUploader.Put(ctx, &ret, upToken, key, reader, size, &putExtra)
	if err != nil {
		return "", err
	}

	return key, nil
}

func (s *qiniuStorage) Delete(ctx context.Context, key string) error {
	bucketManager := storage.NewBucketManager(s.mac, s.cfg)
	return bucketManager.Delete(s.conf.Bucket, key)
}

func (s *qiniuStorage) GenerateURL(ctx context.Context, key string, isPrivate bool, expires time.Duration) string {
	schema := "http"
	if s.conf.UseHttps {
		schema = "https"
	}

	// 构建公开 URL
	// 七牛通常绑定自定义域名
	publicURL := fmt.Sprintf("%s://%s/%s", schema, s.conf.Domain, key)

	if !isPrivate {
		return publicURL
	}

	// 私有文件签名
	deadline := time.Now().Add(expires).Unix()
	return storage.MakePrivateURL(s.mac, s.conf.Domain, key, deadline)
}

// fillZone 区域映射逻辑
func (s *qiniuStorage) fillZone(cfg *storage.Config) {
	switch s.conf.Region {
	case "z0":
		cfg.Zone = &storage.ZoneHuadong
	case "z1":
		cfg.Zone = &storage.ZoneHuabei
	case "z2":
		cfg.Zone = &storage.ZoneHuanan
	case "na0":
		cfg.Zone = &storage.ZoneBeimei
	case "as0":
		cfg.Zone = &storage.ZoneXinjiapo
	default:
		cfg.Zone = &storage.ZoneHuadong
	}
}
