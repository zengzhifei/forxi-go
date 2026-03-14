package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"forxi.cn/forxi-go/app/config"

	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader"
)

var (
	storageOnce sync.Once
)

func Init(cfg *config.StorageConfig) error {
	var initErr error
	storageOnce.Do(func() {
		qiniu := NewQiniuStorage()
		initErr = qiniu.Init(cfg)
		if initErr == nil && cfg.Active == "qiniu" {
			storageInstance = qiniu
		}
	})
	return initErr
}

type QiniuStorage struct {
	cfg       *config.StorageConfigItem
	uploadMgr *uploader.UploadManager
	domain    string
}

func NewQiniuStorage() *QiniuStorage {
	return &QiniuStorage{}
}

func (q *QiniuStorage) Init(cfg *config.StorageConfig) error {
	if cfg == nil || cfg.Qiniu.AccessKey == "" || cfg.Qiniu.SecretKey == "" || cfg.Qiniu.Bucket == "" || cfg.Qiniu.Domain == "" {
		return fmt.Errorf("七牛云配置不完整")
	}

	q.cfg = &cfg.Qiniu
	q.domain = strings.TrimRight(cfg.Qiniu.Domain, "/")

	cred := credentials.NewCredentials(cfg.Qiniu.AccessKey, cfg.Qiniu.SecretKey)
	q.uploadMgr = uploader.NewUploadManager(&uploader.UploadManagerOptions{
		Options: http_client.Options{
			Credentials: cred,
		},
	})

	return nil
}

func (q *QiniuStorage) UploadFile(file string, objectKey string, options *UploadOptions) (string, error) {
	if q.cfg == nil || q.uploadMgr == nil {
		return "", fmt.Errorf("七牛云未初始化")
	}

	objectKey = strings.TrimLeft(objectKey, "/")

	_, err := os.Stat(file)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %v", err)
	}

	objectOptions := &uploader.ObjectOptions{
		BucketName: q.cfg.Bucket,
		ObjectName: &objectKey,
	}
	if options != nil {
		if options.CustomVars != nil {
			objectOptions.CustomVars = options.CustomVars
		}
		if options.FileName != "" {
			objectOptions.FileName = options.FileName
		}
	}

	err = q.uploadMgr.UploadFile(context.Background(), file, objectOptions, nil)
	if err != nil {
		return "", fmt.Errorf("文件上传storage失败: %v", err)
	}

	return fmt.Sprintf("%s/%s", q.domain, objectKey), nil
}

func (q *QiniuStorage) UploadReader(reader io.Reader, objectKey string, options *UploadOptions) (string, error) {
	if q.cfg == nil || q.uploadMgr == nil {
		return "", fmt.Errorf("七牛云未初始化")
	}

	objectKey = strings.TrimLeft(objectKey, "/")

	objectOptions := &uploader.ObjectOptions{
		BucketName: q.cfg.Bucket,
		ObjectName: &objectKey,
	}
	if options != nil {
		if options.CustomVars != nil {
			objectOptions.CustomVars = options.CustomVars
		}
		if options.FileName != "" {
			objectOptions.FileName = options.FileName
		}
	}

	err := q.uploadMgr.UploadReader(context.Background(), reader, objectOptions, nil)
	if err != nil {
		return "", fmt.Errorf("文件流上传storage失败: %v", err)
	}

	return fmt.Sprintf("%s/%s", q.domain, objectKey), nil
}

func (q *QiniuStorage) UploadDirectory(dir string, options *UploadOptions) error {
	if q.cfg == nil || q.uploadMgr == nil {
		return fmt.Errorf("七牛云未初始化")
	}

	directoryOptions := &uploader.DirectoryOptions{
		BucketName: q.cfg.Bucket,
	}
	if options != nil {
		if options.UpdateObjectName != nil {
			directoryOptions.UpdateObjectName = options.UpdateObjectName
		}
	}

	err := q.uploadMgr.UploadDirectory(context.Background(), dir, directoryOptions)
	if err != nil {
		return fmt.Errorf("目录上传storage失败: %v", err)
	}

	return nil
}
