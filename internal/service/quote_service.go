package service

import (
	"encoding/json"
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
	item.IsDefault = req.IsDefault
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
		subtotal += lineTotal
		sort := r.Sort
		if sort == 0 {
			sort = (i + 1) * 10
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
			QuotePrice:     round2(r.QuotePrice),
			LineTotal:      lineTotal,
			UpgradeNote:    strings.TrimSpace(r.UpgradeNote),
			ParamsText:     strings.TrimSpace(r.ParamsText),
			Remark:         strings.TrimSpace(r.Remark),
		})
	}
	return items, round2(subtotal), nil
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
		} else {
			if err := tx.Save(&quote).Error; err != nil {
				return err
			}
			if err := tx.Where("quote_id = ?", quote.ID).Delete(&model.QuoteItem{}).Error; err != nil {
				return err
			}
		}
		for i := range items {
			items[i].QuoteID = quote.ID
			items[i].TenantID = tenantID
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		quote.Items = items
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetQuote(tenantID, quote.ID)
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
			QuotePrice:     it.QuotePrice,
			UpgradeNote:    it.UpgradeNote,
			ParamsText:     it.ParamsText,
			Remark:         it.Remark,
		})
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
