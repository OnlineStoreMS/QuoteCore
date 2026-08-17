package admin

import (
	"errors"
	"net/http"
	"strings"

	"quotecore/internal/pkg/response"
	"quotecore/internal/service"

	"github.com/gin-gonic/gin"
)

type PublicHandlers struct {
	svc *service.QuoteService
}

func NewPublicHandlers(svc *service.QuoteService) *PublicHandlers {
	return &PublicHandlers{svc: svc}
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
	default:
		msg := err.Error()
		if strings.Contains(msg, "作废") || strings.Contains(msg, "拿货价") || strings.Contains(msg, "明细") {
			response.Fail(c, http.StatusBadRequest, msg)
			return
		}
		response.Fail(c, http.StatusInternalServerError, msg)
	}
}
