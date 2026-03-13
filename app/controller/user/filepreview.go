package user

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"forxi.cn/forxi-go/app/config"
	"forxi.cn/forxi-go/app/database"
	"forxi.cn/forxi-go/app/util"

	"github.com/gin-gonic/gin"
)

var SupportedTypes = map[string]string{
	".txt":  "text/plain",
	".md":   "text/markdown",
	".json": "application/json",
	".log":  "text/plain",
	".xml":  "text/xml",
	".html": "text/html",
	".htm":  "text/html",
	".css":  "text/css",
	".js":   "application/javascript",
	".ts":   "application/typescript",
	".py":   "text/x-python",
	".java": "text/x-java",
	".go":   "text/x-go",
	".rs":   "text/x-rust",
	".c":    "text/x-c",
	".cpp":  "text/x-c++",
	".h":    "text/x-c-header",
	".hpp":  "text/x-c++-header",
	".cs":   "text/x-csharp",
	".php":  "text/x-php",
	".rb":   "text/x-ruby",
	".sh":   "text/x-shellscript",
	".sql":  "text/x-sql",
	".yaml": "text/x-yaml",
	".yml":  "text/x-yaml",
	".toml": "text/x-toml",
	".ini":  "text/x-ini",
	".conf": "text/plain",
	".cfg":  "text/plain",
	".env":  "text/plain",

	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
	".ico":  "image/x-icon",
	".bmp":  "image/bmp",

	".pdf": "application/pdf",

	".mp4":  "video/mp4",
	".webm": "video/webm",
	".avi":  "video/x-msvideo",
	".mov":  "video/quicktime",

	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
}

type PreviewResponse struct {
	Type    string `json:"type"`    // image, pdf, video, office, text
	Content string `json:"content"` // base64 content for non-office files
	URL     string `json:"url"`     // original url or office viewer url
	Name    string `json:"name"`    // file name
	Mime    string `json:"mime"`    // mime type
}

const CacheDir = "data/filecache"
const CacheKey = "filereview:file"

type FilePreviewController struct {
	config      *config.FilePreviewConfig
	redisPrefix string
}

func NewFilePreviewController(cfg *config.FilePreviewConfig, redisCfg *config.RedisConfig) *FilePreviewController {
	initCacheDir()
	go startCleanup(cfg, redisCfg.Prefix)
	return &FilePreviewController{config: cfg, redisPrefix: redisCfg.Prefix}
}

func initCacheDir() {
	os.MkdirAll(CacheDir, 0755)
}

func startCleanup(cfg *config.FilePreviewConfig, redisPrefix string) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cleanupCache(cfg, redisPrefix)
	}
}

func cleanupCache(_ *config.FilePreviewConfig, redisPrefix string) {
	ctx := context.Background()
	hashKey := redisPrefix + ":" + CacheKey
	result, err := database.RedisClient.HGetAll(ctx, hashKey).Result()
	if err != nil {
		return
	}
	for filename, expireTimeStr := range result {
		var expireTime int64
		fmt.Sscanf(expireTimeStr, "%d", &expireTime)
		if time.Now().Unix() > expireTime {
			filePath := filepath.Join(CacheDir, filename)
			_ = os.Remove(filePath)
			database.RedisClient.HDel(ctx, hashKey, filename)
		}
	}
}

func (c *FilePreviewController) Online(ctx *gin.Context) {
	url := ctx.Query("url")
	if url == "" {
		util.BadRequest(ctx, "请提供文件URL")
		return
	}
	c.handleURL(ctx, url)
}

func (c *FilePreviewController) Local(ctx *gin.Context) {
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		util.BadRequest(ctx, "请选择要上传的文件")
		return
	}
	c.handleUpload(ctx, file, header)
}

func (c *FilePreviewController) handleUpload(ctx *gin.Context, file multipart.File, header *multipart.FileHeader) {
	defer file.Close()

	if header.Size > c.config.MaxFileSize {
		util.BadRequest(ctx, fmt.Sprintf("文件大小不能超过%dMB", c.config.MaxFileSize/1024/1024))
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if SupportedTypes[ext] == "" {
		util.BadRequest(ctx, "不支持的文件类型")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		util.InternalServerError(ctx, "文件读取失败")
		return
	}

	isOffice := ext == ".doc" || ext == ".docx" || ext == ".xls" || ext == ".xlsx" || ext == ".ppt" || ext == ".pptx"

	if isOffice {
		hash := getFileHash(data)
		filename := fmt.Sprintf("%s%s", hash, ext)
		filePath := filepath.Join(CacheDir, filename)
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			util.InternalServerError(ctx, "文件保存失败")
			return
		}
		redisCtx := context.Background()
		hashKey := c.redisPrefix + ":" + CacheKey
		expireTime := time.Now().Add(time.Duration(c.config.CacheExpiry) * time.Second).Unix()
		database.RedisClient.HSet(redisCtx, hashKey, filename, expireTime)
		serveURL := fmt.Sprintf("/api/filereview/cache?file=%s", filename)
		resp := c.buildPreviewResponse(data, ext, header.Filename, serveURL)
		util.Success(ctx, resp)
		return
	}

	resp := c.buildPreviewResponse(data, ext, header.Filename, "")
	util.Success(ctx, resp)
}

func (c *FilePreviewController) handleURL(ctx *gin.Context, url string) {
	if !isValidURL(url) {
		util.BadRequest(ctx, "无效的URL格式")
		return
	}

	ext := getExtFromURL(url)
	filename := fmt.Sprintf("url_%s%s", uuidShort(), ext)
	filePath := filepath.Join(CacheDir, filename)

	if err := downloadFile(url, filePath); err != nil {
		util.BadRequest(ctx, "文件下载失败: "+err.Error())
		return
	}
	defer os.Remove(filePath)

	info, err := os.Stat(filePath)
	if err != nil || info.Size() > c.config.MaxFileSize {
		util.BadRequest(ctx, fmt.Sprintf("文件大小不能超过%dMB", c.config.MaxFileSize/1024/1024))
		return
	}

	ext = strings.ToLower(filepath.Ext(filename))
	if SupportedTypes[ext] == "" {
		util.BadRequest(ctx, "不支持的文件类型")
		return
	}

	data, _ := os.ReadFile(filePath)

	filename = filepath.Base(url)

	isOffice := ext == ".doc" || ext == ".docx" || ext == ".xls" || ext == ".xlsx" || ext == ".ppt" || ext == ".pptx"

	if isOffice {
		encodedURL := strings.ReplaceAll(url, "?", "%3F")
		encodedURL = strings.ReplaceAll(encodedURL, "&", "%26")
		encodedURL = strings.ReplaceAll(encodedURL, "#", "%23")
		resp := c.buildPreviewResponse(nil, ext, filename, encodedURL)
		util.Success(ctx, resp)
		return
	}

	resp := c.buildPreviewResponse(data, ext, filename, "")
	util.Success(ctx, resp)
}

func (c *FilePreviewController) buildPreviewResponse(data []byte, ext, filename, originalURL string) PreviewResponse {
	resp := PreviewResponse{
		Name: filename,
		Mime: SupportedTypes[ext],
	}

	isOffice := ext == ".doc" || ext == ".docx" || ext == ".xls" || ext == ".xlsx" || ext == ".ppt" || ext == ".pptx"
	isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" || ext == ".svg"
	isVideo := ext == ".mp4" || ext == ".webm" || ext == ".avi" || ext == ".mov"
	isText := ext == ".txt" || ext == ".md" || ext == ".json" || ext == ".log" ||
		ext == ".xml" || ext == ".html" || ext == ".htm" || ext == ".css" || ext == ".js" || ext == ".ts" ||
		ext == ".py" || ext == ".java" || ext == ".go" || ext == ".rs" ||
		ext == ".c" || ext == ".cpp" || ext == ".h" || ext == ".hpp" || ext == ".cs" ||
		ext == ".php" || ext == ".rb" || ext == ".sh" || ext == ".sql" ||
		ext == ".yaml" || ext == ".yml" || ext == ".toml" || ext == ".ini" || ext == ".conf" || ext == ".cfg" || ext == ".env"

	if isOffice {
		resp.Type = "office"
		resp.URL = originalURL
	} else if isImage {
		resp.Type = "image"
		compressed := compressImage(data)
		if compressed != nil {
			resp.Content = base64.StdEncoding.EncodeToString(compressed)
		} else {
			resp.Content = base64.StdEncoding.EncodeToString(data)
		}
	} else if isVideo {
		resp.Type = "video"
		resp.Content = base64.StdEncoding.EncodeToString(data)
	} else if ext == ".pdf" {
		resp.Type = "pdf"
		resp.Content = base64.StdEncoding.EncodeToString(data)
	} else if isText {
		resp.Type = "text"
		resp.Content = string(data)
	} else {
		resp.Type = "unknown"
	}

	return resp
}

func compressImage(data []byte) []byte {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	maxDim := 1920
	if width > maxDim || height > maxDim {
		scale := float64(maxDim) / float64(width)
		if height > width {
			scale = float64(maxDim) / float64(height)
		}
		newWidth := int(float64(width) * scale)
		newHeight := int(float64(height) * scale)
		img = resize(img, newWidth, newHeight)
	}

	var buf bytes.Buffer
	if format == "png" {
		err = png.Encode(&buf, img)
	} else {
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	}
	if err != nil {
		return nil
	}

	return buf.Bytes()
}

func resize(img image.Image, width, height int) image.Image {
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	xScale := float64(img.Bounds().Dx()) / float64(width)
	yScale := float64(img.Bounds().Dy()) / float64(height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float64(x) * xScale)
			srcY := int(float64(y) * yScale)
			newImg.Set(x, y, img.At(srcX, srcY))
		}
	}
	return newImg
}

func (c *FilePreviewController) Cache(ctx *gin.Context) {
	filename := ctx.Query("file")
	if filename == "" {
		util.BadRequest(ctx, "缺少file参数")
		return
	}

	filePath := filepath.Join(CacheDir, filename)
	if _, err := os.Stat(filePath); err != nil {
		util.NotFound(ctx, "文件不存在或已过期")
		return
	}

	redisCtx := context.Background()
	hashKey := c.redisPrefix + ":" + CacheKey
	expireTimeStr, err := database.RedisClient.HGet(redisCtx, hashKey, filename).Result()
	if err != nil {
		_ = os.Remove(filePath)
		util.NotFound(ctx, "文件不存在或已过期")
		return
	}
	var expireTime int64
	fmt.Sscanf(expireTimeStr, "%d", &expireTime)
	if time.Now().Unix() > expireTime {
		_ = os.Remove(filePath)
		database.RedisClient.HDel(redisCtx, hashKey, filename)
		util.NotFound(ctx, "文件不存在或已过期")
		return
	}
	newExpireTime := time.Now().Add(time.Duration(c.config.CacheExpiry) * time.Second).Unix()
	database.RedisClient.HSet(redisCtx, hashKey, filename, newExpireTime)

	ext := strings.ToLower(filepath.Ext(filename))
	contentType := SupportedTypes[ext]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	util.Download(ctx, filePath, filename, contentType)
}

func getFileHash(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

func uuidShort() string {
	hash := md5.Sum([]byte(time.Now().String()))
	return hex.EncodeToString(hash[:8])
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
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

func getExtFromURL(url string) string {
	ext := filepath.Ext(url)
	if ext == "" {
		return ".bin"
	}
	return ext
}
