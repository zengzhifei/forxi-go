package storage

import (
	"io"

	"forxi.cn/forxi-go/app/config"
)

type UploadOptions struct {
	FileName          string
	CustomVars        map[string]string
	UpdateObjectName  func(file string) string
	ObjectConcurrency int
}

type Storage interface {
	Init(cfg *config.StorageConfig) error
	UploadFile(file string, objectKey string, options *UploadOptions) (string, error)
	UploadReader(reader io.Reader, objectKey string, options *UploadOptions) (string, error)
	UploadDirectory(dir string, options *UploadOptions) error
}
