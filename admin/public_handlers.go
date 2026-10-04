package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"quotecore/internal/dto"
	"quotecore/internal/pkg/response"
	"quotecore/internal/service"
	"quotecore/internal/storage"

	"github.com/gin-gonic/gin"
)

type PublicHandlers struct {
	svc   *service.QuoteService
	store storage.Storage
}

func NewPublicHandlers(svc *service.QuoteService, store storage.Storage) *PublicHandlers {
	return &PublicHandlers{svc: svc, store: store}
}

func (h *PublicHandlers) GetShare(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	view, err := h.svc.GetShareByToken(token)
	if err != nil {
		mapPublicError(c, err)
		return
	}
	response.OK(c, view)
}

func (h *PublicHandlers) GetCustomerShare(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	view, err := h.svc.GetCustomerShareByToken(token)
	if err != nil {
		mapPublicError(c, err)
		return
	}
	response.OK(c, view)
}

func (h *PublicHandlers) SaveCustomerPriced(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	var req dto.CustomerPricedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	view, err := h.svc.SaveCustomerPriced(token, req.Flags)
	if err != nil {
		mapPublicError(c, err)
		return
	}
	response.OK(c, view)
}

func (h *PublicHandlers) ApplySecondEdit(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	var req dto.SecondEditApplyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	view, err := h.svc.ApplySecondEdit(token, req)
	if err != nil {
		mapPublicError(c, err)
		return
	}
	response.OK(c, view)
}

func (h *PublicHandlers) OpenSecondEdit(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	var req dto.SecondEditOpenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	view, err := h.svc.OpenSecondEdit(token, req)
	if err != nil {
		mapPublicError(c, err)
		return
	}
	response.OK(c, view)
}

func (h *PublicHandlers) GetSecondEdit(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	view, err := h.svc.GetSecondEditByToken(token)
	if err != nil {
		mapPublicError(c, err)
		return
	}
	response.OK(c, view)
}

func (h *PublicHandlers) SaveSecondEdit(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	var req dto.QuoteSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	view, err := h.svc.SaveSecondEditByToken(token, req)
	if err != nil {
		mapPublicError(c, err)
		return
	}
	response.OK(c, view)
}

type submitSupplyReq struct {
	Items []service.SupplyPriceSubmitItem `json:"items"`
}

func (h *PublicHandlers) SubmitSupply(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	var req submitSupplyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.SubmitSupplyPrices(token, req.Items); err != nil {
		mapPublicError(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func mapPublicError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		response.Fail(c, http.StatusNotFound, "链接无效或报价单不存在")
	case errors.Is(err, service.ErrBadRequest):
		response.Fail(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrConflict):
		response.Fail(c, http.StatusConflict, err.Error())
	default:
		msg := err.Error()
		if strings.Contains(msg, "作废") || strings.Contains(msg, "拿货价") || strings.Contains(msg, "明细") {
			response.Fail(c, http.StatusBadRequest, msg)
			return
		}
		response.Fail(c, http.StatusInternalServerError, msg)
	}
}

func (h *PublicHandlers) UploadSecondEdit(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	tenantID, err := h.svc.SecondEditTenantID(token)
	if err != nil {
		mapPublicError(c, err)
		return
	}
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
		response.Fail(c, http.StatusBadRequest, "image too large (max 10MB)")
		return
	}
	subdir := fmt.Sprintf("quote/%d/%s", tenantID, time.Now().Format("200601"))
	url, err := h.store.Upload(file, subdir)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"url": url, "mediaType": kind})
}

func (h *PublicHandlers) UploadSecondEditFromURL(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	tenantID, err := h.svc.SecondEditTenantID(token)
	if err != nil {
		mapPublicError(c, err)
		return
	}
	var body uploadFromURLBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, "url required")
		return
	}
	subdir := fmt.Sprintf("items/%d/%s", tenantID, time.Now().Format("200601"))
	url, err := storage.UploadFromURL(h.store, body.URL, subdir)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, gin.H{"url": url})
}
