package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	QuoteStatusDraft int8 = 1
	QuoteStatusSent  int8 = 2
	QuoteStatusWon   int8 = 3
	QuoteStatusVoid  int8 = 4

	ItemSourceProduct  = "product"
	ItemSourceManual   = "manual"
	ItemSourceTemplate = "template"

	// TemplateKindLayout: 仅版式（店名/Logo）
	TemplateKindLayout = "layout"
	// TemplateKindSkeleton: 组车/组合报价骨架（产品+配件固定行）
	TemplateKindSkeleton = "skeleton"
)

// QuoteTemplate is printable layout + optional line skeleton (组装车等).
type QuoteTemplate struct {
	ID              uint64         `gorm:"primaryKey" json:"id"`
	TenantID        uint64         `gorm:"index;not null;default:1" json:"tenantId"`
	Name            string         `gorm:"size:128;not null" json:"name"`
	Kind            string         `gorm:"size:16;not null;default:layout" json:"kind"` // layout | skeleton
	IsDefault       bool           `gorm:"not null;default:false" json:"isDefault"`
	LogoURL         string         `gorm:"size:512" json:"logoUrl"`
	ShopName        string         `gorm:"size:128" json:"shopName"`
	ShopPhone       string         `gorm:"size:64" json:"shopPhone"`
	ShopAddress     string         `gorm:"size:255" json:"shopAddress"`
	HeaderSubtitle  string         `gorm:"size:255" json:"headerSubtitle"`
	FooterText      string         `gorm:"type:text" json:"footerText"`
	ShowLogo        bool           `gorm:"not null;default:true" json:"showLogo"`
	ShowRetailPrice bool           `gorm:"not null;default:true" json:"showRetailPrice"`
	ShowSpecImage   bool           `gorm:"not null;default:true" json:"showSpecImage"`
	ShowUpgrade     bool           `gorm:"not null;default:true" json:"showUpgrade"`
	ShowParams      bool           `gorm:"not null;default:true" json:"showParams"`
	ShowTotals      bool           `gorm:"not null;default:true" json:"showTotals"`
	StylePreset     string         `gorm:"size:32;not null;default:compare" json:"stylePreset"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	Lines []QuoteTemplateLine `gorm:"foreignKey:TemplateID" json:"lines,omitempty"`
}

func (QuoteTemplate) TableName() string { return "quote_templates" }

// QuoteTemplateLine is a fixed skeleton row: 产品(category) + 配件(partName).
type QuoteTemplateLine struct {
	ID         uint64         `gorm:"primaryKey" json:"id"`
	TenantID   uint64         `gorm:"index;not null;default:1" json:"tenantId"`
	TemplateID uint64         `gorm:"index;not null" json:"templateId"`
	Sort       int            `gorm:"not null;default:0" json:"sort"`
	Category   string         `gorm:"size:128" json:"category"`          // 产品/产品组，如「车架组」
	PartName   string         `gorm:"size:128;not null" json:"partName"` // 配件，如「车架」「前叉」
	Hint       string         `gorm:"size:255" json:"hint"`              // 填写提示
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (QuoteTemplateLine) TableName() string { return "quote_template_lines" }

type Quote struct {
	ID                 uint64         `gorm:"primaryKey" json:"id"`
	TenantID           uint64         `gorm:"index;not null;default:1" json:"tenantId"`
	QuoteNo            string         `gorm:"size:64;not null;index" json:"quoteNo"`
	Title              string         `gorm:"size:255;not null;default:报价单" json:"title"`
	Status             int8           `gorm:"not null;default:1;index" json:"status"`
	CustomerID         *uint64        `gorm:"index" json:"customerId"`
	CustomerName       string         `gorm:"size:128" json:"customerName"`
	ContactName        string         `gorm:"size:64" json:"contactName"`
	ContactPhone       string         `gorm:"size:32" json:"contactPhone"`
	Currency           string         `gorm:"size:8;not null;default:CNY" json:"currency"`
	ValidUntil         *time.Time     `json:"validUntil"`
	Remark             string         `gorm:"type:text" json:"remark"`
	DiscountAmt        float64        `gorm:"type:decimal(12,2);not null;default:0" json:"discountAmt"`
	ShippingAmt        float64        `gorm:"type:decimal(12,2);not null;default:0" json:"shippingAmt"`
	TaxAmt             float64        `gorm:"type:decimal(12,2);not null;default:0" json:"taxAmt"`
	SubtotalAmt        float64        `gorm:"type:decimal(12,2);not null;default:0" json:"subtotalAmt"`
	TotalAmt           float64        `gorm:"type:decimal(12,2);not null;default:0" json:"totalAmt"`
	TemplateID         *uint64        `json:"templateId"`
	TemplateSnap       string         `gorm:"type:text" json:"templateSnap"`
	ShareToken         string         `gorm:"size:64;index" json:"shareToken"`
	CustomerShareToken string         `gorm:"size:64;index" json:"customerShareToken"`
	CreatedBy          uint64         `gorm:"not null;default:0" json:"createdBy"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`

	Items []QuoteItem `gorm:"foreignKey:QuoteID" json:"items,omitempty"`
}

func (Quote) TableName() string { return "quotes" }

type QuoteItem struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	TenantID       uint64         `gorm:"index;not null;default:1" json:"tenantId"`
	QuoteID        uint64         `gorm:"index;not null" json:"quoteId"`
	Sort           int            `gorm:"not null;default:0" json:"sort"`
	Source         string         `gorm:"size:16;not null;default:manual" json:"source"`
	ProductID      *uint64        `json:"productId"`
	SkuID          *uint64        `json:"skuId"`
	TemplateLineID *uint64        `json:"templateLineId"`
	Category       string         `gorm:"size:128" json:"category"`                 // 产品/产品组
	PartName       string         `gorm:"size:128" json:"partName"`                 // 配件
	Name           string         `gorm:"size:255;not null;default:''" json:"name"` // 具体名称
	SpecLabel      string         `gorm:"size:255" json:"specLabel"`
	ImageURL       string         `gorm:"size:512" json:"imageUrl"`
	Qty            float64        `gorm:"type:decimal(12,2);not null;default:1" json:"qty"`
	Unit           string         `gorm:"size:16;not null;default:件" json:"unit"`
	RetailPrice    float64        `gorm:"type:decimal(12,2);not null;default:0" json:"retailPrice"`
	CostPrice      float64        `gorm:"type:decimal(12,2);not null;default:0" json:"costPrice"`   // 内部成本价
	SupplyPrice    float64        `gorm:"type:decimal(12,2);not null;default:0" json:"supplyPrice"` // 供货商拿货价
	SupplyPriceAt  *time.Time     `json:"supplyPriceAt"`                                            // 拿货价更新时间
	SupplyRemark   string         `gorm:"size:512" json:"supplyRemark"`                             // 供货商备注
	QuotePrice     float64        `gorm:"type:decimal(12,2);not null;default:0" json:"quotePrice"`
	LineTotal      float64        `gorm:"type:decimal(12,2);not null;default:0" json:"lineTotal"`
	UpgradeNote    string         `gorm:"size:512" json:"upgradeNote"`
	ParamsText     string         `gorm:"type:text" json:"paramsText"`
	Remark         string         `gorm:"size:512" json:"remark"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (QuoteItem) TableName() string { return "quote_items" }
