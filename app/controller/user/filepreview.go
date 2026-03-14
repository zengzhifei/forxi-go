package user

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"forxi.cn/forxi-go/app/resource"
	"forxi.cn/forxi-go/app/resource/rds"
	"forxi.cn/forxi-go/app/util"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const MaxFileSize = 50 * 1024 * 1024

const filePreviewCacheExpiry = 30 * 24 * time.Hour

var (
	supportedTypes = map[string]struct {
		ext      string
		fileType string
	}{
		"text/plain":             {".txt", "text"},
		"text/markdown":          {".md", "text"},
		"application/json":       {".json", "text"},
		"text/xml":               {".xml", "text"},
		"text/html":              {".html", "text"},
		"text/css":               {".css", "text"},
		"application/javascript": {".js", "text"},
		"application/typescript": {".ts", "text"},
		"image/jpeg":             {".jpg", "image"},
		"image/png":              {".png", "image"},
		"image/gif":              {".gif", "image"},
		"image/webp":             {".webp", "image"},
		"image/svg+xml":          {".svg", "image"},
		"image/x-icon":           {".ico", "image"},
		"image/bmp":              {".bmp", "image"},
		"application/pdf":        {".pdf", "pdf"},
		"video/mp4":              {".mp4", "video"},
		"video/webm":             {".webp", "video"},
		"video/x-msvideo":        {".avi", "video"},
		"video/quicktime":        {".mov", "video"},
		"audio/mpeg":             {".mp3", "audio"},
		"audio/wav":              {".wav", "audio"},
		"audio/flac":             {".flac", "audio"},
		"audio/aac":              {".aac", "audio"},
		"audio/ogg":              {".ogg", "audio"},
		"audio/mp4":              {".m4a", "audio"},
		"application/msword":     {".doc", "office"},
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": {".docx", "office"},
		"application/vnd.ms-excel": {".xls", "office"},
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         {".xlsx", "office"},
		"application/vnd.ms-powerpoint":                                             {".ppt", "office"},
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": {".pptx", "office"},
	}
)

type FilePreviewController struct{}

func NewFilePreviewController() *FilePreviewController {
	return &FilePreviewController{}
}

func (c *FilePreviewController) Online(ctx *gin.Context) {
	url := ctx.Query("url")
	if url == "" {
		util.BadRequest(ctx, "请提供文件URL")
		return
	}

	if !isValidURL(url) {
		util.BadRequest(ctx, "无效的URL格式")
		return
	}

	fileURL, contentType, _, fileType, err := c.downloadAndUpload(ctx, url)
	if err != nil {
		util.BadRequest(ctx, "文件处理失败: "+err.Error())
		return
	}

	util.Success(ctx, gin.H{
		"url":  fileURL,
		"name": filepath.Base(url),
		"type": fileType,
		"mime": contentType,
	})
}

func (c *FilePreviewController) Local(ctx *gin.Context) {
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		util.BadRequest(ctx, "请选择要上传的文件")
		return
	}
	defer file.Close()

	if header.Size > MaxFileSize {
		util.BadRequest(ctx, fmt.Sprintf("文件大小不能超过%dMB", MaxFileSize/1024/1024))
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		util.InternalServerError(ctx, "文件读取失败")
		return
	}

	fileURL, contentType, _, fileType, err := c.upload(ctx, data)
	if err != nil {
		util.InternalServerError(ctx, "文件处理失败")
		return
	}

	util.Success(ctx, gin.H{
		"url":  fileURL,
		"name": header.Filename,
		"type": fileType,
		"mime": contentType,
	})
}

func (c *FilePreviewController) downloadAndUpload(ctx context.Context, url string) (string, string, string, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", "", err
	}

	return c.upload(ctx, data)
}

func (c *FilePreviewController) upload(ctx context.Context, data []byte) (string, string, string, string, error) {
	if len(data) > int(MaxFileSize) {
		return "", "", "", "", fmt.Errorf("文件大小超过限制")
	}

	contentType, ext, fileType, err := detectFileInfo(data)
	if err != nil {
		return "", contentType, "", "", err
	}

	md5Hash := fmt.Sprintf("%x", md5.Sum(data))
	cacheKey := rds.Key(rds.KeyFilePreviewCache, md5Hash)

	cachedURL, err := resource.Redis.Get(ctx, cacheKey).Result()
	if err == nil && cachedURL != "" {
		return cachedURL, contentType, ext, fileType, nil
	}

	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	fileURL, err := resource.Storage.UploadReader(bytes.NewReader(data), "filereview/"+filename, nil)
	if err != nil {
		return "", contentType, "", "", err
	}

	resource.Redis.Set(ctx, cacheKey, fileURL, filePreviewCacheExpiry)

	return fileURL, contentType, ext, fileType, nil
}

func detectFileInfo(data []byte) (string, string, string, error) {
	mimeType := mimetype.Detect(data)
	contentType := mimeType.String()
	ext := mimeType.Extension()

	if info, ok := supportedTypes[contentType]; ok {
		return contentType, info.ext, info.fileType, nil
	}

	if strings.HasPrefix(contentType, "text/") {
		return contentType, ".txt", "text", nil
	}

	if info, ok := supportedTypes[ext]; ok {
		return contentType, info.ext, info.fileType, nil
	}

	return contentType, "", "", fmt.Errorf("不支持的文件类型")
}

func isValidURL(url string) bool {
	if len(url) > 2048 {
		return false
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return false
	}
	blocked := []string{"localhost", "127.0.0.1", "0.0.0.0", "192.168.", "10.", "172."}
	for _, d := range blocked {
		if strings.Contains(url, d) {
			return false
		}
	}
	return true
}
