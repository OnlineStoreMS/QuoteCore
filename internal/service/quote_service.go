package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"quotecore/internal/dto"
	"quotecore/internal/model"
	"quotecore/internal/repo"

	"gorm.io/gorm"
)

type QuoteService struct {
	repos *repo.Repos
}

func NewQuoteService(repos *repo.Repos) *QuoteService {
	return &QuoteService{repos: repos}
}

func boolOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func (s *QuoteService) DashboardStats(tenantID uint64) (*dto.DashboardStats, error) {
	tenantID = repo.NormalizeTenantID(tenantID)
	stats := &dto.DashboardStats{}
	db := s.repos.DB
	if err := db.Model(&model.Quote{}).Where("tenant_id = ?", tenantID).Count(&stats.QuoteCount).Error; err != nil {
		return nil, err
	}
	_ = db.Model(&model.Quote{}).Where("tenant_id = ? AND status = ?", tenantID, model.QuoteStatusDraft).Count(&stats.DraftCount).Error
	_ = db.Model(&model.Quote{}).Where("tenant_id = ? AND status = ?", tenantID, model.QuoteStatusSent).Count(&stats.SentCount).Error
	_ = db.Model(&model.Quote{}).Where("tenant_id = ? AND status = ?", tenantID, model.QuoteStatusWon).Count(&stats.WonCount).Error
	_ = db.Model(&model.QuoteTemplate{}).Where("tenant_id = ?", tenantID).Count(&stats.TemplateCount).Error
	start := time.Now().In(time.Local)
	monthStart := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())
	var sum *float64
	_ = db.Model(&model.Quote{}).
		Where("tenant_id = ? AND status <> ? AND created_at >= ?", tenantID, model.QuoteStatusVoid, monthStart).
		Select("COALESCE(SUM(total_amt),0)").Scan(&sum).Error
	if sum != nil {
		stats.MonthTotalAmt = *sum
	}
	return stats, nil
}

func templateWithLines(db *gorm.DB) *gorm.DB {
	return db.Preload("Lines", func(tx *gorm.DB) *gorm.DB {
		return tx.Order("sort ASC, id ASC")
	})
}

func (s *QuoteService) ListTemplates(tenantID uint64) ([]model.QuoteTemplate, error) {
	var list []model.QuoteTemplate
	err := templateWithLines(s.repos.ScopeTenant(tenantID)).Order("is_default DESC, id ASC").Find(&list).Error
	return list, err
}

func (s *QuoteService) GetTemplate(tenantID, id uint64) (*model.QuoteTemplate, error) {
	var item model.QuoteTemplate
	err := templateWithLines(s.repos.ScopeTenant(tenantID)).First(&item, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrNotFound
	}
	return &item, err
}

func normalizeTemplateKind(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == model.TemplateKindSkeleton {
		return model.TemplateKindSkeleton
	}
	return model.TemplateKindLayout
}

func buildTemplateLines(tenantID, templateID uint64, reqs []dto.TemplateLineReq) ([]model.QuoteTemplateLine, error) {
	lines := make([]model.QuoteTemplateLine, 0, len(reqs))
	for i, r := range reqs {
		part := strings.TrimSpace(r.PartName)
		if part == "" {
			return nil, fmt.Errorf("%w: 第 %d 行配件名称必填", ErrBadRequest, i+1)
		}
		sort := r.Sort
		if sort == 0 {
			sort = (i + 1) * 10
		}
		lines = append(lines, model.QuoteTemplateLine{
			TenantID:   tenantID,
			TemplateID: templateID,
			Sort:       sort,
			Category:   strings.TrimSpace(r.Category),
			PartName:   part,
			Hint:       strings.TrimSpace(r.Hint),
		})
	}
	return lines, nil
}

func (s *QuoteService) SaveTemplate(tenantID, id uint64, req dto.TemplateSaveReq) (*model.QuoteTemplate, error) {
	tenantID = repo.NormalizeTenantID(tenantID)
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: 模板名称必填", ErrBadRequest)
	}
	preset := strings.TrimSpace(req.StylePreset)
	if preset == "" {
		preset = "compare"
	}
	kind := normalizeTemplateKind(req.Kind)
	if kind == model.TemplateKindSkeleton && len(req.Lines) == 0 {
		return nil, fmt.Errorf("%w: 骨架模板至少需要一行配件", ErrBadRequest)
	}
	lines, err := buildTemplateLines(tenantID, 0, req.Lines)
	if err != nil {
		return nil, err
	}

	var item model.QuoteTemplate
	if id > 0 {
		if err := s.repos.ScopeTenant(tenantID).First(&item, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
	} else {
		item.TenantID = tenantID
	}
	item.Name = name
	item.Kind = kind
	if kind == model.TemplateKindSkeleton {
		item.IsDefault = false
	} else {
		item.IsDefault = req.IsDefault
	}
	item.LogoURL = strings.TrimSpace(req.LogoURL)
	item.ShopName = strings.TrimSpace(req.ShopName)
	item.ShopPhone = strings.TrimSpace(req.ShopPhone)
	item.ShopAddress = strings.TrimSpace(req.ShopAddress)
	item.HeaderSubtitle = strings.TrimSpace(req.HeaderSubtitle)
	item.FooterText = strings.TrimSpace(req.FooterText)
	item.ShowLogo = boolOr(req.ShowLogo, true)
	item.ShowRetailPrice = boolOr(req.ShowRetailPrice, true)
	item.ShowSpecImage = boolOr(req.ShowSpecImage, true)
	item.ShowUpgrade = boolOr(req.ShowUpgrade, true)
	item.ShowParams = boolOr(req.ShowParams, true)
	item.ShowTotals = boolOr(req.ShowTotals, true)
	item.StylePreset = preset

	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if item.IsDefault {
			if err := tx.Model(&model.QuoteTemplate{}).
				Where("tenant_id = ?", tenantID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if id == 0 {
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&item).Error; err != nil {
			return err
		}
		if err := tx.Where("template_id = ?", item.ID).Delete(&model.QuoteTemplateLine{}).Error; err != nil {
			return err
		}
		for i := range lines {
			lines[i].TemplateID = item.ID
			lines[i].TenantID = tenantID
		}
		if len(lines) > 0 {
			if err := tx.Create(&lines).Error; err != nil {
				return err
			}
		}
		item.Lines = lines
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetTemplate(tenantID, item.ID)
}

func (s *QuoteService) DeleteTemplate(tenantID, id uint64) error {
	return s.repos.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("tenant_id = ?", repo.NormalizeTenantID(tenantID)).Delete(&model.QuoteTemplate{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return tx.Where("template_id = ?", id).Delete(&model.QuoteTemplateLine{}).Error
	})
}

func defaultAssembleLines() []model.QuoteTemplateLine {
	pairs := [][2]string{
		{"车架组", "车架"},
		{"车架组", "前叉"},
		{"车架组", "座管"},
		{"车架组", "弯把"},
		{"车架组", "把立"},
		{"变速套件", "手变前拨后拨"},
		{"变速套件", "夹器"},
		{"变速套件", "飞轮"},
		{"变速套件", "链条"},
		{"变速套件", "牙盘"},
		{"变速套件", "碟片"},
		{"轮组", "轮组"},
		{"轮组", "外胎"},
		{"轮组", "内胎"},
		{"其他", "中轴"},
		{"其他", "坐垫"},
		{"其他", "脚踏"},
		{"其他", "把带"},
		{"其他", "水壶架"},
		{"服务", "组装费"},
		{"服务", "运费"},
	}
	lines := make([]model.QuoteTemplateLine, 0, len(pairs))
	for i, p := range pairs {
		lines = append(lines, model.QuoteTemplateLine{
			Sort:     (i + 1) * 10,
			Category: p[0],
			PartName: p[1],
		})
	}
	return lines
}

func (s *QuoteService) EnsureDefaultTemplate(tenantID uint64) error {
	tenantID = repo.NormalizeTenantID(tenantID)
	var n int64
	if err := s.repos.DB.Model(&model.QuoteTemplate{}).Where("tenant_id = ?", tenantID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		tpl := model.QuoteTemplate{
			TenantID:        tenantID,
			Name:            "默认报价模板",
			Kind:            model.TemplateKindLayout,
			IsDefault:       true,
			ShopName:        "报价中心",
			HeaderSubtitle:  "专业配件报价",
			FooterText:      "本报价单有效期内价格有效，最终解释权归报价方所有。",
			ShowLogo:        true,
			ShowRetailPrice: true,
			ShowSpecImage:   true,
			ShowUpgrade:     true,
			ShowParams:      true,
			ShowTotals:      true,
			StylePreset:     "compare",
		}
		if err := s.repos.DB.Create(&tpl).Error; err != nil {
			return err
		}
	}
	return s.ensureAssembleSkeleton(tenantID)
}

func (s *QuoteService) ensureAssembleSkeleton(tenantID uint64) error {
	var n int64
	if err := s.repos.DB.Model(&model.QuoteTemplate{}).
		Where("tenant_id = ? AND kind = ?", tenantID, model.TemplateKindSkeleton).
		Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	tpl := model.QuoteTemplate{
		TenantID:        tenantID,
		Name:            "组装车报价骨架",
		Kind:            model.TemplateKindSkeleton,
		IsDefault:       false,
		ShopName:        "报价中心",
		HeaderSubtitle:  "组车配件清单",
		FooterText:      "本报价单有效期内价格有效，最终解释权归报价方所有。",
		ShowLogo:        true,
		ShowRetailPrice: true,
		ShowSpecImage:   true,
		ShowUpgrade:     false,
		ShowParams:      false,
		ShowTotals:      true,
		StylePreset:     "compare",
	}
	return s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tpl).Error; err != nil {
			return err
		}
		lines := defaultAssembleLines()
		for i := range lines {
			lines[i].TenantID = tenantID
			lines[i].TemplateID = tpl.ID
		}
		return tx.Create(&lines).Error
	})
}

func (s *QuoteService) ListQuotes(tenantID uint64, keyword, status string, page, pageSize int) ([]model.Quote, int64, error) {
	tenantID = repo.NormalizeTenantID(tenantID)
	q := s.repos.DB.Model(&model.Quote{}).Where("tenant_id = ?", tenantID)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("quote_no ILIKE ? OR title ILIKE ? OR customer_name ILIKE ? OR contact_phone ILIKE ?", like, like, like, like)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Quote
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *QuoteService) GetQuote(tenantID, id uint64) (*model.Quote, error) {
	var item model.Quote
	err := s.repos.ScopeTenant(tenantID).Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort ASC, id ASC")
	}).First(&item, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrNotFound
	}
	return &item, err
}

func parseValidUntil(s *string) (*time.Time, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(*s), time.Local)
	if err != nil {
		return nil, fmt.Errorf("%w: 有效期格式应为 YYYY-MM-DD", ErrBadRequest)
	}
	return &t, nil
}

func (s *QuoteService) nextQuoteNo(tx *gorm.DB, tenantID uint64) (string, error) {
	prefix := time.Now().Format("20060102")
	var count int64
	if err := tx.Model(&model.Quote{}).
		Where("tenant_id = ? AND quote_no LIKE ?", tenantID, "Q"+prefix+"%").
		Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("Q%s%04d", prefix, count+1), nil
}

func buildItems(tenantID, quoteID uint64, reqs []dto.QuoteItemReq) ([]model.QuoteItem, float64, error) {
	items := make([]model.QuoteItem, 0, len(reqs))
	var subtotal float64
	for i, r := range reqs {
		name := strings.TrimSpace(r.Name)
		category := strings.TrimSpace(r.Category)
		partName := strings.TrimSpace(r.PartName)
		if name == "" && category == "" && partName == "" {
			return nil, 0, fmt.Errorf("%w: 第 %d 行至少填写名称或配件", ErrBadRequest, i+1)
		}
		src := strings.TrimSpace(r.Source)
		if src == "" {
			if partName != "" {
				src = model.ItemSourceTemplate
			} else {
				src = model.ItemSourceManual
			}
		}
		qty := r.Qty
		if qty <= 0 {
			qty = 1
		}
		unit := strings.TrimSpace(r.Unit)
		if unit == "" {
			unit = "件"
		}
		lineTotal := round2(qty * r.QuotePrice)
		item := model.QuoteItem{
			ProductID: r.ProductID,
			Name:      name,
		}
		if n := len(items); n == 0 || !sameProductSpec(items[n-1], item) {
			subtotal += lineTotal
		}
		sort := r.Sort
		if sort == 0 {
			sort = (i + 1) * 10
		}
		var supplyAt *time.Time
		if r.SupplyPriceAt != nil && strings.TrimSpace(*r.SupplyPriceAt) != "" {
			raw := strings.TrimSpace(*r.SupplyPriceAt)
			if t, err := time.Parse(time.RFC3339, raw); err == nil {
				supplyAt = &t
			} else if t, err := time.ParseInLocation("2006-01-02 15:04:05", raw, time.Local); err == nil {
				supplyAt = &t
			}
		}
		items = append(items, model.QuoteItem{
			TenantID:       tenantID,
			QuoteID:        quoteID,
			Sort:           sort,
			Source:         src,
			ProductID:      r.ProductID,
			SkuID:          r.SkuID,
			TemplateLineID: r.TemplateLineID,
			Category:       category,
			PartName:       partName,
			Name:           name,
			SpecLabel:      strings.TrimSpace(r.SpecLabel),
			ImageURL:       strings.TrimSpace(r.ImageURL),
			Qty:            qty,
			Unit:           unit,
			RetailPrice:    round2(r.RetailPrice),
			CostPrice:      round2(r.CostPrice),
			SupplyPrice:    round2(r.SupplyPrice),
			SupplyPriceAt:  supplyAt,
			SupplyRemark:   strings.TrimSpace(r.SupplyRemark),
			QuotePrice:     round2(r.QuotePrice),
			LineTotal:      lineTotal,
			UpgradeNote:    strings.TrimSpace(r.UpgradeNote),
			ParamsText:     strings.TrimSpace(r.ParamsText),
			Remark:         strings.TrimSpace(r.Remark),
		})
	}
	return items, round2(subtotal), nil
}

// sameProductSpec matches the sheet grouping: adjacent rows of one product are alternative specs.
// Only the first spec is included in 商品小计 / 合计.
func sameProductSpec(a, b model.QuoteItem) bool {
	aID, bID := uint64(0), uint64(0)
	if a.ProductID != nil {
		aID = *a.ProductID
	}
	if b.ProductID != nil {
		bID = *b.ProductID
	}
	if aID > 0 && bID > 0 {
		return aID == bID
	}
	an := strings.TrimSpace(a.Name)
	bn := strings.TrimSpace(b.Name)
	return an != "" && an == bn
}

func (s *QuoteService) snapTemplate(tenantID uint64, templateID *uint64) (string, *uint64, error) {
	var tpl *model.QuoteTemplate
	if templateID != nil && *templateID > 0 {
		item, err := s.GetTemplate(tenantID, *templateID)
		if err != nil {
			return "", nil, err
		}
		tpl = item
	} else {
		var def model.QuoteTemplate
		err := s.repos.ScopeTenant(tenantID).Where("is_default = ?", true).First(&def).Error
		if err == nil {
			tpl = &def
		} else if err != gorm.ErrRecordNotFound {
			return "", nil, err
		}
	}
	if tpl == nil {
		return "", nil, nil
	}
	b, err := json.Marshal(tpl)
	if err != nil {
		return "", nil, err
	}
	id := tpl.ID
	return string(b), &id, nil
}

func (s *QuoteService) SaveQuote(tenantID, userID, id uint64, req dto.QuoteSaveReq) (*model.Quote, error) {
	tenantID = repo.NormalizeTenantID(tenantID)
	_ = s.EnsureDefaultTemplate(tenantID)
	validUntil, err := parseValidUntil(req.ValidUntil)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "报价单"
	}
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "CNY"
	}
	status := model.QuoteStatusDraft
	if req.Status != nil {
		status = *req.Status
	}

	var quote model.Quote
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if id > 0 {
			if err := tx.Where("tenant_id = ?", tenantID).First(&quote, id).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return ErrNotFound
				}
				return err
			}
		} else {
			no, err := s.nextQuoteNo(tx, tenantID)
			if err != nil {
				return err
			}
			quote = model.Quote{
				TenantID:  tenantID,
				QuoteNo:   no,
				CreatedBy: userID,
			}
		}

		items, subtotal, err := buildItems(tenantID, quote.ID, req.Items)
		if err != nil {
			return err
		}
		snap, tid, err := s.snapTemplate(tenantID, req.TemplateID)
		if err != nil {
			return err
		}

		quote.Title = title
		quote.Status = status
		quote.CustomerID = req.CustomerID
		quote.CustomerName = strings.TrimSpace(req.CustomerName)
		quote.ContactName = strings.TrimSpace(req.ContactName)
		quote.ContactPhone = strings.TrimSpace(req.ContactPhone)
		quote.Currency = currency
		quote.ValidUntil = validUntil
		quote.Remark = strings.TrimSpace(req.Remark)
		quote.DiscountAmt = round2(req.DiscountAmt)
		quote.ShippingAmt = round2(req.ShippingAmt)
		quote.TaxAmt = round2(req.TaxAmt)
		quote.SubtotalAmt = subtotal
		quote.TotalAmt = round2(subtotal - quote.DiscountAmt + quote.ShippingAmt + quote.TaxAmt)
		quote.TemplateID = tid
		quote.TemplateSnap = snap

		if id == 0 {
			if err := tx.Create(&quote).Error; err != nil {
				return err
			}
			for i := range items {
				items[i].QuoteID = quote.ID
				items[i].TenantID = tenantID
				items[i].ID = 0
			}
			if len(items) > 0 {
				if err := tx.Create(&items).Error; err != nil {
					return err
				}
			}
		} else {
			if err := tx.Save(&quote).Error; err != nil {
				return err
			}
			synced, err := syncQuoteItems(tx, tenantID, quote.ID, req.Items)
			if err != nil {
				return err
			}
			items = synced
		}
		quote.Items = items
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetQuote(tenantID, quote.ID)
}

// syncQuoteItems updates existing rows by id (保留供货商拿货价关联的行 id)，新建无 id 行，删除未提交行。
func syncQuoteItems(tx *gorm.DB, tenantID, quoteID uint64, reqs []dto.QuoteItemReq) ([]model.QuoteItem, error) {
	var existing []model.QuoteItem
	if err := tx.Where("quote_id = ?", quoteID).Find(&existing).Error; err != nil {
		return nil, err
	}
	existMap := make(map[uint64]model.QuoteItem, len(existing))
	for _, e := range existing {
		existMap[e.ID] = e
	}
	built, _, err := buildItems(tenantID, quoteID, reqs)
	if err != nil {
		return nil, err
	}
	keep := make(map[uint64]bool)
	out := make([]model.QuoteItem, 0, len(built))
	for i := range built {
		it := built[i]
		it.QuoteID = quoteID
		it.TenantID = tenantID
		reqID := uint64(0)
		if i < len(reqs) && reqs[i].ID != nil {
			reqID = *reqs[i].ID
		}
		if reqID > 0 {
			if old, ok := existMap[reqID]; ok {
				it.ID = old.ID
				// 前端未带回拿货价时间戳时，保留库中供货商填写结果，避免自动保存冲掉
				if reqs[i].SupplyPriceAt == nil {
					it.SupplyPrice = old.SupplyPrice
					it.SupplyPriceAt = old.SupplyPriceAt
					it.SupplyRemark = old.SupplyRemark
				}
				if err := tx.Save(&it).Error; err != nil {
					return nil, err
				}
				keep[it.ID] = true
				out = append(out, it)
				continue
			}
		}
		it.ID = 0
		if err := tx.Create(&it).Error; err != nil {
			return nil, err
		}
		keep[it.ID] = true
		out = append(out, it)
	}
	for id := range existMap {
		if !keep[id] {
			if err := tx.Delete(&model.QuoteItem{}, id).Error; err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

func (s *QuoteService) DeleteQuote(tenantID, id uint64) error {
	return s.repos.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("tenant_id = ?", repo.NormalizeTenantID(tenantID)).Delete(&model.Quote{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return tx.Where("quote_id = ?", id).Delete(&model.QuoteItem{}).Error
	})
}

func (s *QuoteService) VoidQuote(tenantID, id uint64) (*model.Quote, error) {
	var quote model.Quote
	if err := s.repos.ScopeTenant(tenantID).First(&quote, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	quote.Status = model.QuoteStatusVoid
	if err := s.repos.DB.Save(&quote).Error; err != nil {
		return nil, err
	}
	return s.GetQuote(tenantID, id)
}

func (s *QuoteService) CopyQuote(tenantID, userID, id uint64) (*model.Quote, error) {
	src, err := s.GetQuote(tenantID, id)
	if err != nil {
		return nil, err
	}
	items := make([]dto.QuoteItemReq, 0, len(src.Items))
	for _, it := range src.Items {
		items = append(items, dto.QuoteItemReq{
			Sort:           it.Sort,
			Source:         it.Source,
			ProductID:      it.ProductID,
			SkuID:          it.SkuID,
			TemplateLineID: it.TemplateLineID,
			Category:       it.Category,
			PartName:       it.PartName,
			Name:           it.Name,
			SpecLabel:      it.SpecLabel,
			ImageURL:       it.ImageURL,
			Qty:            it.Qty,
			Unit:           it.Unit,
			RetailPrice:    it.RetailPrice,
			CostPrice:      it.CostPrice,
			SupplyPrice:    it.SupplyPrice,
			SupplyRemark:   it.SupplyRemark,
			QuotePrice:     it.QuotePrice,
			UpgradeNote:    it.UpgradeNote,
			ParamsText:     it.ParamsText,
			Remark:         it.Remark,
		})
		if it.SupplyPriceAt != nil {
			v := it.SupplyPriceAt.Format(time.RFC3339)
			items[len(items)-1].SupplyPriceAt = &v
		}
	}
	var valid *string
	if src.ValidUntil != nil {
		v := src.ValidUntil.Format("2006-01-02")
		valid = &v
	}
	st := model.QuoteStatusDraft
	return s.SaveQuote(tenantID, userID, 0, dto.QuoteSaveReq{
		Title:        src.Title + "（副本）",
		Status:       &st,
		CustomerID:   src.CustomerID,
		CustomerName: src.CustomerName,
		ContactName:  src.ContactName,
		ContactPhone: src.ContactPhone,
		Currency:     src.Currency,
		ValidUntil:   valid,
		Remark:       src.Remark,
		DiscountAmt:  src.DiscountAmt,
		ShippingAmt:  src.ShippingAmt,
		TaxAmt:       src.TaxAmt,
		TemplateID:   src.TemplateID,
		Items:        items,
	})
}

func randomShareToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// EnsureShareToken creates a share token if missing and returns the quote.
func (s *QuoteService) EnsureShareToken(tenantID, id uint64) (*model.Quote, error) {
	q, err := s.GetQuote(tenantID, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(q.ShareToken) != "" {
		return q, nil
	}
	tok, err := randomShareToken()
	if err != nil {
		return nil, err
	}
	if err := s.repos.DB.Model(&model.Quote{}).Where("id = ? AND tenant_id = ?", id, repo.NormalizeTenantID(tenantID)).
		Update("share_token", tok).Error; err != nil {
		return nil, err
	}
	q.ShareToken = tok
	return q, nil
}

// EnsureCustomerShareToken creates a customer-facing share token if missing.
// It is separate from the supplier token so the public quote page cannot submit supply prices.
func (s *QuoteService) EnsureCustomerShareToken(tenantID, id uint64) (*model.Quote, error) {
	q, err := s.GetQuote(tenantID, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(q.CustomerShareToken) != "" {
		return q, nil
	}
	tok, err := randomShareToken()
	if err != nil {
		return nil, err
	}
	if err := s.repos.DB.Model(&model.Quote{}).Where("id = ? AND tenant_id = ?", id, repo.NormalizeTenantID(tenantID)).
		Update("customer_share_token", tok).Error; err != nil {
		return nil, err
	}
	q.CustomerShareToken = tok
	return q, nil
}

type ShareQuoteView struct {
	Title   string          `json:"title"`
	QuoteNo string          `json:"quoteNo"`
	Remark  string          `json:"remark"`
	Items   []ShareItemView `json:"items"`
}

type ShareItemView struct {
	ID           uint64  `json:"id"`
	Sort         int     `json:"sort"`
	Category     string  `json:"category"`
	PartName     string  `json:"partName"`
	Name         string  `json:"name"`
	SpecLabel    string  `json:"specLabel"`
	ImageURL     string  `json:"imageUrl"`
	SupplyPrice  float64 `json:"supplyPrice"`
	SupplyRemark string  `json:"supplyRemark"`
}

type SupplyPriceSubmitItem struct {
	ID           uint64  `json:"id"`
	SupplyPrice  float64 `json:"supplyPrice"`
	SupplyRemark string  `json:"supplyRemark"`
}

func (s *QuoteService) GetShareByToken(token string) (*ShareQuoteView, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrNotFound
	}
	var q model.Quote
	err := s.repos.DB.Where("share_token = ?", token).Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort ASC, id ASC")
	}).First(&q).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if q.Status == model.QuoteStatusVoid {
		return nil, fmt.Errorf("%w: 报价单已作废", ErrBadRequest)
	}
	view := &ShareQuoteView{
		Title:   q.Title,
		QuoteNo: q.QuoteNo,
		Remark:  q.Remark,
		Items:   make([]ShareItemView, 0, len(q.Items)),
	}
	for _, it := range q.Items {
		view.Items = append(view.Items, ShareItemView{
			ID:           it.ID,
			Sort:         it.Sort,
			Category:     it.Category,
			PartName:     it.PartName,
			Name:         it.Name,
			SpecLabel:    it.SpecLabel,
			ImageURL:     it.ImageURL,
			SupplyPrice:  it.SupplyPrice,
			SupplyRemark: it.SupplyRemark,
		})
	}
	return view, nil
}

func (s *QuoteService) SubmitSupplyPrices(token string, rows []SupplyPriceSubmitItem) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrNotFound
	}
	var q model.Quote
	err := s.repos.DB.Where("share_token = ?", token).First(&q).Error
	if err == gorm.ErrRecordNotFound {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if q.Status == model.QuoteStatusVoid {
		return fmt.Errorf("%w: 报价单已作废", ErrBadRequest)
	}
	if len(rows) == 0 {
		return fmt.Errorf("%w: 请填写至少一行拿货价", ErrBadRequest)
	}
	now := time.Now()
	return s.repos.DB.Transaction(func(tx *gorm.DB) error {
		updated := 0
		for _, row := range rows {
			if row.ID == 0 {
				continue
			}
			price := round2(row.SupplyPrice)
			if price < 0 {
				return fmt.Errorf("%w: 拿货价不能为负", ErrBadRequest)
			}
			res := tx.Model(&model.QuoteItem{}).
				Where("id = ? AND quote_id = ? AND deleted_at IS NULL", row.ID, q.ID).
				Updates(map[string]any{
					"supply_price":    price,
					"supply_price_at": now,
					"supply_remark":   strings.TrimSpace(row.SupplyRemark),
					"updated_at":      now,
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return fmt.Errorf("%w: 明细不存在 id=%d", ErrBadRequest, row.ID)
			}
			updated++
		}
		if updated == 0 {
			return fmt.Errorf("%w: 请填写至少一行拿货价", ErrBadRequest)
		}
		return nil
	})
}

// CustomerShareTemplate is the default layout template shown on the customer page.
type CustomerShareTemplate struct {
	LogoURL         string `json:"logoUrl"`
	ShopName        string `json:"shopName"`
	ShopPhone       string `json:"shopPhone"`
	ShopAddress     string `json:"shopAddress"`
	HeaderSubtitle  string `json:"headerSubtitle"`
	FooterText      string `json:"footerText"`
	ShowLogo        bool   `json:"showLogo"`
	ShowRetailPrice bool   `json:"showRetailPrice"`
	ShowSpecImage   bool   `json:"showSpecImage"`
	ShowUpgrade     bool   `json:"showUpgrade"`
	ShowParams      bool   `json:"showParams"`
	ShowTotals      bool   `json:"showTotals"`
	StylePreset     string `json:"stylePreset"`
	Kind            string `json:"kind"`
}

// CustomerShareItem is a customer-safe line. Cost and supplier prices are omitted.
type CustomerShareItem struct {
	Sort        int     `json:"sort"`
	ProductID   *uint64 `json:"productId,omitempty"`
	Category    string  `json:"category"`
	PartName    string  `json:"partName"`
	Name        string  `json:"name"`
	SpecLabel   string  `json:"specLabel"`
	ImageURL    string  `json:"imageUrl"`
	Qty         float64 `json:"qty"`
	Unit        string  `json:"unit"`
	RetailPrice float64 `json:"retailPrice"`
	QuotePrice  float64 `json:"quotePrice"`
	UpgradeNote string  `json:"upgradeNote"`
	ParamsText  string  `json:"paramsText"`
	Remark      string  `json:"remark"`
}

// CustomerShareView is the public quote page for a customer.
type CustomerShareView struct {
	Title        string                `json:"title"`
	QuoteNo      string                `json:"quoteNo"`
	CustomerName string                `json:"customerName"`
	ContactName  string                `json:"contactName"`
	ContactPhone string                `json:"contactPhone"`
	Currency     string                `json:"currency"`
	ValidUntil   *string               `json:"validUntil"`
	Remark       string                `json:"remark"`
	DiscountAmt  float64               `json:"discountAmt"`
	ShippingAmt  float64               `json:"shippingAmt"`
	TaxAmt       float64               `json:"taxAmt"`
	SubtotalAmt  float64               `json:"subtotalAmt"`
	TotalAmt     float64               `json:"totalAmt"`
	Template     CustomerShareTemplate `json:"template"`
	Items        []CustomerShareItem   `json:"items"`
}

func (s *QuoteService) defaultLayoutTemplate(tenantID uint64) (model.QuoteTemplate, error) {
	tenantID = repo.NormalizeTenantID(tenantID)
	var tpl model.QuoteTemplate
	err := s.repos.DB.Where("tenant_id = ? AND is_default = ? AND kind <> ?", tenantID, true, model.TemplateKindSkeleton).
		Order("id ASC").First(&tpl).Error
	if err == nil {
		return tpl, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.QuoteTemplate{}, err
	}
	err = s.repos.DB.Where("tenant_id = ? AND kind <> ?", tenantID, model.TemplateKindSkeleton).
		Order("is_default DESC, id ASC").First(&tpl).Error
	if err == nil {
		return tpl, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.QuoteTemplate{}, err
	}
	return model.QuoteTemplate{
		Kind:            model.TemplateKindLayout,
		ShopName:        "报价中心",
		ShowLogo:        true,
		ShowRetailPrice: true,
		ShowSpecImage:   true,
		ShowUpgrade:     true,
		ShowParams:      true,
		ShowTotals:      true,
		StylePreset:     "compare",
	}, nil
}

func layoutTemplateView(tpl model.QuoteTemplate) CustomerShareTemplate {
	kind := tpl.Kind
	if kind == "" || kind == model.TemplateKindSkeleton {
		kind = model.TemplateKindLayout
	}
	preset := strings.TrimSpace(tpl.StylePreset)
	if preset == "" {
		preset = "compare"
	}
	shop := strings.TrimSpace(tpl.ShopName)
	if shop == "" {
		shop = "报价中心"
	}
	return CustomerShareTemplate{
		LogoURL:         strings.TrimSpace(tpl.LogoURL),
		ShopName:        shop,
		ShopPhone:       strings.TrimSpace(tpl.ShopPhone),
		ShopAddress:     strings.TrimSpace(tpl.ShopAddress),
		HeaderSubtitle:  strings.TrimSpace(tpl.HeaderSubtitle),
		FooterText:      strings.TrimSpace(tpl.FooterText),
		ShowLogo:        tpl.ShowLogo,
		ShowRetailPrice: tpl.ShowRetailPrice,
		ShowSpecImage:   tpl.ShowSpecImage,
		ShowUpgrade:     tpl.ShowUpgrade,
		ShowParams:      tpl.ShowParams,
		ShowTotals:      tpl.ShowTotals,
		StylePreset:     preset,
		Kind:            kind,
	}
}

func (s *QuoteService) GetCustomerShareByToken(token string) (*CustomerShareView, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrNotFound
	}
	var q model.Quote
	err := s.repos.DB.Where("customer_share_token = ?", token).Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort ASC, id ASC")
	}).First(&q).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if q.Status == model.QuoteStatusVoid {
		return nil, fmt.Errorf("%w: 报价单已作废", ErrBadRequest)
	}
	tpl, err := s.defaultLayoutTemplate(q.TenantID)
	if err != nil {
		return nil, err
	}
	view := &CustomerShareView{
		Title:        q.Title,
		QuoteNo:      q.QuoteNo,
		CustomerName: q.CustomerName,
		ContactName:  q.ContactName,
		ContactPhone: q.ContactPhone,
		Currency:     q.Currency,
		Remark:       q.Remark,
		DiscountAmt:  q.DiscountAmt,
		ShippingAmt:  q.ShippingAmt,
		TaxAmt:       q.TaxAmt,
		SubtotalAmt:  q.SubtotalAmt,
		TotalAmt:     q.TotalAmt,
		Template:     layoutTemplateView(tpl),
		Items:        make([]CustomerShareItem, 0, len(q.Items)),
	}
	if q.ValidUntil != nil {
		v := q.ValidUntil.Format("2006-01-02")
		view.ValidUntil = &v
	}
	for _, it := range q.Items {
		view.Items = append(view.Items, CustomerShareItem{
			Sort:        it.Sort,
			ProductID:   it.ProductID,
			Category:    it.Category,
			PartName:    it.PartName,
			Name:        it.Name,
			SpecLabel:   it.SpecLabel,
			ImageURL:    it.ImageURL,
			Qty:         it.Qty,
			Unit:        it.Unit,
			RetailPrice: it.RetailPrice,
			QuotePrice:  it.QuotePrice,
			UpgradeNote: it.UpgradeNote,
			ParamsText:  it.ParamsText,
			Remark:      it.Remark,
		})
	}
	return view, nil
}
