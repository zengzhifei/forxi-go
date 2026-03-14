package user

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"forxi.cn/forxi-go/app/resource"
	"forxi.cn/forxi-go/app/resource/storage"
	"forxi.cn/forxi-go/app/util"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UploadScene struct {
	AllowedTypes []string
	MaxSize      int64
	RequireAuth  bool
}

var uploadScenes = map[string]UploadScene{
	"avatar": {
		AllowedTypes: []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
		MaxSize:      2 * 1024 * 1024,
		RequireAuth:  true,
	},
}

type UploadController struct{}

func NewUploadController() *UploadController {
	return &UploadController{}
}

func (c *UploadController) Upload(ctx *gin.Context) {
	scene := ctx.PostForm("scene")
	if scene == "" {
		util.BadRequest(ctx, "scene参数不能为空")
		return
	}

	sceneConfig, exists := uploadScenes[scene]
	if !exists {
		util.BadRequest(ctx, "不支持的上传场景")
		return
	}

	if sceneConfig.RequireAuth {
		if _, exists := ctx.Get("user_id"); !exists {
			util.Unauthorized(ctx, "该场景需要登录")
			return
		}
	}

	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		util.BadRequest(ctx, "请选择要上传的文件")
		return
	}
	defer file.Close()

	if header.Size > sceneConfig.MaxSize {
		util.BadRequest(ctx, fmt.Sprintf("文件大小不能超过%dMB", sceneConfig.MaxSize/1024/1024))
		return
	}

	contentType := header.Header.Get("Content-Type")
	allowed := false
	for _, t := range sceneConfig.AllowedTypes {
		if t == contentType {
			allowed = true
			break
		}
	}
	if !allowed {
		util.BadRequest(ctx, "不支持的文件类型")
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	objectKey := fmt.Sprintf("%s/%s%s", scene, uuid.New().String(), ext)

	data, err := io.ReadAll(file)
	if err != nil {
		util.InternalServerError(ctx, "读取文件失败")
		return
	}

	url, err := resource.Storage.UploadReader(bytes.NewReader(data), objectKey, &storage.UploadOptions{
		FileName: header.Filename,
	})
	if err != nil {
		resource.Logger.Error("upload failed", zap.String("error", err.Error()))
		util.InternalServerError(ctx, "文件上传失败")
		return
	}

	util.Success(ctx, gin.H{"url": url})
}
