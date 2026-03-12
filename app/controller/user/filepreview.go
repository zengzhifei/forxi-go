package user

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var SupportedTypes = map[string]string{
	".txt":  "text/plain",
	".md":   "text/markdown",
	".json": "application/json",
	".log":  "text/plain",

	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",

	".pdf": "application/pdf",

	".mp4":  "video/mp4",
	".webm": "video/webm",

	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
}

const MaxFileSize = 200 * 1024 * 1024
const CacheDir = "data/filecache"
const CacheExpiry = 24 * time.Hour

type FileCache struct {
	Path       string
	AccessTime time.Time
	RefCount   int
}

var fileCache = make(map[string]*FileCache)

type FilePreviewController struct{}

func NewFilePreviewController() *FilePreviewController {
	initCacheDir()
	go startCleanup()
	return &FilePreviewController{}
}

func initCacheDir() {
	os.MkdirAll(CacheDir, 0755)
}

func startCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cleanupCache()
	}
}

func cleanupCache() {
	now := time.Now()
	for hash, cache := range fileCache {
		if now.Sub(cache.AccessTime) > CacheExpiry {
			if cache.RefCount <= 0 {
				os.Remove(cache.Path)
				delete(fileCache, hash)
			}
		}
	}
}

func (c *FilePreviewController) Online(ctx *gin.Context) {
	url := ctx.Query("url")
	if url == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请提供文件URL"})
		return
	}
	c.handleURL(ctx, url)
}

func (c *FilePreviewController) Local(ctx *gin.Context) {
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的文件"})
		return
	}
	c.handleUpload(ctx, file, header)
}

func (c *FilePreviewController) handleUpload(ctx *gin.Context, file multipart.File, header *multipart.FileHeader) {
	defer file.Close()

	if header.Size > MaxFileSize {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("文件大小不能超过%dMB", MaxFileSize/1024/1024)})
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if SupportedTypes[ext] == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型"})
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "文件读取失败"})
		return
	}

	hash := getFileHash(data)
	filename := fmt.Sprintf("%s%s", hash, ext)
	filePath := filepath.Join(CacheDir, filename)

	cache, exists := fileCache[hash]
	if !exists {
		err := os.WriteFile(filePath, data, 0644)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败"})
			return
		}
		fileCache[hash] = &FileCache{
			Path:       filePath,
			AccessTime: time.Now(),
			RefCount:   1,
		}
	} else {
		cache.AccessTime = time.Now()
		cache.RefCount++
	}

	c.servePreview(ctx, filePath, ext, header.Filename, hash)
}

func (c *FilePreviewController) handleURL(ctx *gin.Context, url string) {
	if !isValidURL(url) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的URL格式"})
		return
	}

	ext := getExtFromURL(url)
	filename := fmt.Sprintf("url_%s%s", uuidShort(), ext)
	filePath := filepath.Join(CacheDir, filename)

	if err := downloadFile(url, filePath); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "文件下载失败: " + err.Error()})
		return
	}
	defer os.Remove(filePath)

	info, err := os.Stat(filePath)
	if err != nil || info.Size() > MaxFileSize {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("文件大小不能超过%dMB", MaxFileSize/1024/1024)})
		return
	}

	ext = strings.ToLower(filepath.Ext(filename))
	if SupportedTypes[ext] == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型"})
		return
	}

	data, _ := os.ReadFile(filePath)
	hash := getFileHash(data)
	filename = fmt.Sprintf("%s%s", hash, ext)
	cachedPath := filepath.Join(CacheDir, filename)

	cache, exists := fileCache[hash]
	if !exists {
		os.Rename(filePath, cachedPath)
		fileCache[hash] = &FileCache{
			Path:       cachedPath,
			AccessTime: time.Now(),
			RefCount:   1,
		}
	} else {
		cache.AccessTime = time.Now()
		cache.RefCount++
		os.Remove(filePath)
	}

	c.servePreview(ctx, cachedPath, ext, filepath.Base(url), hash)
}

func (c *FilePreviewController) servePreview(ctx *gin.Context, filePath, ext, filename, hash string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "文件读取失败"})
		return
	}

	if cache, exists := fileCache[hash]; exists {
		cache.AccessTime = time.Now()
	}

	contentType := SupportedTypes[ext]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	scheme := "http"
	if ctx.Request.TLS != nil {
		scheme = "https"
	}
	serverBaseURL := fmt.Sprintf("%s://%s", scheme, ctx.Request.Host)

	html := generatePreviewHTML(data, ext, filename, contentType, filePath, serverBaseURL)
	ctx.Header("Content-Type", "text/html; charset=utf-8")
	ctx.Writer.Write([]byte(html))
}

func (c *FilePreviewController) Download(ctx *gin.Context) {
	filename := ctx.Query("file")
	if filename == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "缺少file参数"})
		return
	}

	filePath := filepath.Join(CacheDir, filename)
	info, err := os.Stat(filePath)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "文件不存在或已过期"})
		return
	}

	hash := strings.TrimSuffix(filename, filepath.Ext(filename))
	if cache, exists := fileCache[hash]; exists {
		cache.AccessTime = time.Now()
	}

	if time.Since(info.ModTime()) > CacheExpiry {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "文件不存在或已过期"})
		return
	}

	ext := strings.ToLower(filepath.Ext(filename))
	contentType := SupportedTypes[ext]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx.Header("Content-Type", contentType)
	ctx.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", filename))
	ctx.File(filePath)
}

func generatePreviewHTML(data []byte, ext, filename, contentType, filePath, serverBaseURL string) string {
	isText := ext == ".txt" || ext == ".md" || ext == ".json" || ext == ".log"
	isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" || ext == ".svg"
	isVideo := ext == ".mp4" || ext == ".webm"
	isOffice := ext == ".doc" || ext == ".docx" || ext == ".xls" || ext == ".xlsx" || ext == ".ppt" || ext == ".pptx"

	var contentHTML string

	if isText {
		content := string(data)
		if len(content) > 500000 {
			content = content[:500000] + "\n\n... 文件过大，仅显示前500KB ..."
		}
		escaped := escapeHTML(content)
		contentHTML = fmt.Sprintf(`<div class="text-preview"><pre>%s</pre></div>`, escaped)
	} else if isImage {
		base64Data := base64.StdEncoding.EncodeToString(data)
		contentHTML = fmt.Sprintf(`<div class="image-preview"><img id="preview" src="data:%s;base64,%s"></div>`, contentType, base64Data)
	} else if isVideo {
		base64Data := base64.StdEncoding.EncodeToString(data)
		contentHTML = fmt.Sprintf(`<div class="video-preview"><video controls><source src="data:%s;base64,%s" type="%s">您的浏览器不支持视频播放</video></div>`, contentType, base64Data, contentType)
	} else if ext == ".pdf" {
		base64Data := base64.StdEncoding.EncodeToString(data)
		contentHTML = fmt.Sprintf(`<div class="pdf-preview"><div id="pdf-container"></div></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/pdf.js/3.11.174/pdf.min.js"></script>
<script>
pdfjsLib.GlobalWorkerOptions.workerSrc = 'https://cdnjs.cloudflare.com/ajax/libs/pdf.js/3.11.174/pdf.worker.min.js';
const pdfData = atob('%s');
const pdfDataArray = new Uint8Array(pdfData.length);
for (let i = 0; i < pdfData.length; i++) pdfDataArray[i] = pdfData.charCodeAt(i);
pdfjsLib.getDocument({data: pdfDataArray}).promise.then(function(pdf) {
    const container = document.getElementById('pdf-container');
    for (let i = 1; i <= pdf.numPages; i++) {
        pdf.getPage(i).then(function(page) {
            const scale = 1.5;
            const viewport = page.getViewport({scale: scale});
            const canvas = document.createElement('canvas');
            const context = canvas.getContext('2d');
            canvas.height = viewport.height;
            canvas.width = viewport.width;
            canvas.style.display = 'block';
            canvas.style.margin = '10px auto';
            container.appendChild(canvas);
            page.render({canvasContext: context, viewport: viewport});
        });
    }
});
</script>`, base64Data)
	} else if isOffice {
		fileURL := fmt.Sprintf("%s/api/filereview/download?file=%s", serverBaseURL, filepath.Base(filePath))
		encodedURL := strings.ReplaceAll(fileURL, "?", "%3F")
		encodedURL = strings.ReplaceAll(encodedURL, "&", "%26")
		officeViewerURL := fmt.Sprintf("https://view.officeapps.live.com/op/embed.aspx?src=%s", encodedURL)
		contentHTML = fmt.Sprintf(`<div class="office-preview"><iframe src="%s"></iframe></div>`, officeViewerURL)
	} else {
		contentHTML = `<div class="unsupported"><p>不支持此文件类型的预览</p></div>`
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>文件预览 - %s</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { background: #1e1e1e; color: #d4d4d4; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; min-height: 100vh; }
        .toolbar { position: fixed; top: 0; left: 0; right: 0; background: #252526; padding: 12px 20px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #3c3c3c; z-index: 1000; }
        .toolbar h3 { color: #569cd6; font-size: 16px; }
        .toolbar-actions { display: flex; gap: 10px; }
        .toolbar button { background: #0e639c; color: white; border: none; padding: 8px 16px; cursor: pointer; border-radius: 4px; font-size: 14px; }
        .toolbar button:hover { background: #1177bb; }
        .content { padding-top: 60px; min-height: calc(100vh - 60px); }
        .text-preview { padding: 20px; overflow: auto; max-height: calc(100vh - 60px); }
        .text-preview pre { background: #1e1e1e; color: #d4d4d4; font-family: 'Consolas', 'Monaco', monospace; font-size: 14px; line-height: 1.6; white-space: pre-wrap; word-wrap: break-word; }
        .image-preview { display: flex; align-items: center; justify-content: center; padding: 20px; min-height: calc(100vh - 60px); overflow: auto; }
        .image-preview img { max-width: 100%%; max-height: calc(100vh - 100px); object-fit: contain; border-radius: 8px; }
        .pdf-preview { width: 100%%; min-height: calc(100vh - 60px); background: #525252; padding: 20px; }
        .pdf-preview canvas { display: block; margin: 10px auto; background: white; box-shadow: 0 2px 10px rgba(0,0,0,0.3); }
        .video-preview { display: flex; align-items: center; justify-content: center; padding: 20px; }
        .video-preview video { max-width: 100%%; max-height: calc(100vh - 100px); border-radius: 8px; }
        .office-preview { width: 100%%; height: calc(100vh - 60px); }
        .office-preview iframe { width: 100%%; height: 100%%; border: none; }
        .unsupported { display: flex; align-items: center; justify-content: center; height: calc(100vh - 60px); font-size: 18px; color: #888; }
    </style>
</head>
<body>
    <div class="toolbar">
        <h3>📄 %s</h3>
        <div class="toolbar-actions">
            <button onclick="zoomIn()">放大</button>
            <button onclick="zoomOut()">缩小</button>
        </div>
    </div>
    <div class="content">
        %s
    </div>
    <script>
        let scale = 1;
        function zoomIn() { scale += 0.2; applyScale(); }
        function zoomOut() { scale = Math.max(0.2, scale - 0.2); applyScale(); }
        function applyScale() {
            const img = document.getElementById('preview');
            if (img) img.style.transform = 'scale(' + scale + ')';
        }
    </script>
</body>
</html>`, filename, filename, contentHTML)
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

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

var _ = func() interface{} {
	_ = multipart.Form{}
	return nil
}()
