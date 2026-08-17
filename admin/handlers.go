package admin

import (
	"errors"
	"net/http"
	"strings"

	"quotecore/internal/dto"
	"quotecore/internal/integrations/customercore"
	"quotecore/internal/integrations/productcore"
	"quotecore/internal/pkg/authcontext"
	"quotecore/internal/pkg/httputil"
	"quotecore/internal/pkg/response"
	"quotecore/internal/service"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	svc *service.QuoteService
	pc  *productcore.Client
	cc  *customercore.Client
}

func NewHandlers(svc *service.QuoteService, pc *productcore.Client, cc *customercore.Client) *Handlers {
	return &Handlers{svc: svc, pc: pc, cc: cc}
}

func (h *Handlers) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		response.Fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrBadRequest):
		response.Fail(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrConflict):
		response.Fail(c, http.StatusConflict, err.Error())
	default:
		response.Fail(c, http.StatusInternalServerError, err.Error())
	}
}

func (h *Handlers) DashboardStats(c *gin.Context) {
	stats, err := h.svc.DashboardStats(authcontext.TenantID(c))
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, stats)
}

func (h *Handlers) ListTemplates(c *gin.Context) {
	_ = h.svc.EnsureDefaultTemplate(authcontext.TenantID(c))
	list, err := h.svc.ListTemplates(authcontext.TenantID(c))
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *Handlers) GetTemplate(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.svc.GetTemplate(authcontext.TenantID(c), id)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CreateTemplate(c *gin.Context) {
	var req dto.TemplateSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.SaveTemplate(authcontext.TenantID(c), 0, req)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) UpdateTemplate(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req dto.TemplateSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.SaveTemplate(authcontext.TenantID(c), id, req)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) DeleteTemplate(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.DeleteTemplate(authcontext.TenantID(c), id); err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *Handlers) ListQuotes(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.svc.ListQuotes(authcontext.TenantID(c), c.Query("keyword"), c.Query("status"), page, pageSize)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) GetQuote(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.svc.GetQuote(authcontext.TenantID(c), id)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CreateQuote(c *gin.Context) {
	var req dto.QuoteSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.SaveQuote(authcontext.TenantID(c), authcontext.UserID(c), 0, req)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) UpdateQuote(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req dto.QuoteSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.SaveQuote(authcontext.TenantID(c), authcontext.UserID(c), id, req)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) DeleteQuote(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.DeleteQuote(authcontext.TenantID(c), id); err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *Handlers) VoidQuote(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.svc.VoidQuote(authcontext.TenantID(c), id)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CopyQuote(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.svc.CopyQuote(authcontext.TenantID(c), authcontext.UserID(c), id)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) EnsureShare(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.svc.EnsureShareToken(authcontext.TenantID(c), id)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, gin.H{
		"shareToken": item.ShareToken,
		"quoteNo":    item.QuoteNo,
		"quoteId":    item.ID,
	})
}

func (h *Handlers) SearchCustomers(c *gin.Context) {
	if h.cc == nil {
		response.Fail(c, http.StatusServiceUnavailable, "客户中心未配置")
		return
	}
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.cc.SearchCustomers(c.Request.Context(), authHeader(c), c.Query("keyword"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) SearchProductSkus(c *gin.Context) {
	if h.pc == nil {
		response.Fail(c, http.StatusServiceUnavailable, "商品中心未配置")
		return
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		response.Fail(c, http.StatusBadRequest, "keyword required")
		return
	}
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.pc.SearchSkus(c.Request.Context(), authHeader(c), keyword, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func authHeader(c *gin.Context) string {
	if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return auth
	}
	if tok := authcontext.BearerToken(c); tok != "" {
		return "Bearer " + tok
	}
	return ""
}
