package admin

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"quotecore/internal/pkg/authcontext"
	"quotecore/internal/pkg/response"
	"quotecore/internal/storage"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	store storage.Storage
}

func NewUploadHandler(store storage.Storage) *UploadHandler {
	return &UploadHandler{store: store}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "file required")
		return
	}
	kind, maxSize, ok := classifyUploadFile(file)
	if !ok {
		response.Fail(c, http.StatusBadRequest, "unsupported file type")
		return
	}
	if file.Size > maxSize {
		if kind == "video" {
			response.Fail(c, http.StatusBadRequest, "video too large (max 100MB)")
		} else {
			response.Fail(c, http.StatusBadRequest, "image too large (max 10MB)")
		}
		return
	}
	subdir := c.DefaultPostForm("subdir", "quote")
	subdir = strings.Trim(subdir, "/")
	if subdir == "" {
		subdir = "quote"
	}
	tenantID := authcontext.TenantID(c)
	month := time.Now().Format("200601")
	subdir = fmt.Sprintf("%s/%d/%s", subdir, tenantID, month)

	url, err := h.store.Upload(file, subdir)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{
		"url":       url,
		"mediaType": kind,
		"fileName":  file.Filename,
		"sizeBytes": file.Size,
		"mime":      file.Header.Get("Content-Type"),
	})
}

type uploadFromURLBody struct {
	URL    string `json:"url" binding:"required"`
	Subdir string `json:"subdir"`
}

// UploadFromURL 下载远程图片并保存到报价中心存储。
func (h *UploadHandler) UploadFromURL(c *gin.Context) {
	var body uploadFromURLBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, "url required")
		return
	}
	subdir := strings.Trim(body.Subdir, "/")
	if subdir == "" {
		subdir = "items"
	}
	tenantID := authcontext.TenantID(c)
	month := time.Now().Format("200601")
	subdir = fmt.Sprintf("%s/%d/%s", subdir, tenantID, month)

	url, err := storage.UploadFromURL(h.store, body.URL, subdir)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, gin.H{"url": url})
}
