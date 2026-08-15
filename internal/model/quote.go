package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	QuoteStatusDraft     int8 = 1
	QuoteStatusSent      int8 = 2
	QuoteStatusWon       int8 = 3
	QuoteStatusVoid      int8 = 4

	ItemSourceProduct = "product"
	ItemSourceManual  = "manual"
)

// QuoteTemplate is a printable quote layout config (logo / shop info are local, not StoreCore-bound).
type QuoteTemplate struct {
	ID              uint64         `gorm:"primaryKey" json:"id"`
	TenantID        uint64         `gorm:"index;not null;default:1" json:"tenantId"`
	Name            string         `gorm:"size:128;not null" json:"name"`
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
	StylePreset     string         `gorm:"size:32;not null;default:compare" json:"stylePreset"` // simple | compare
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (QuoteTemplate) TableName() string { return "quote_templates" }

type Quote struct {
	ID           uint64         `gorm:"primaryKey" json:"id"`
	TenantID     uint64         `gorm:"index;not null;default:1" json:"tenantId"`
	QuoteNo      string         `gorm:"size:64;not null;index" json:"quoteNo"`
	Title        string         `gorm:"size:255;not null;default:报价单" json:"title"`
	Status       int8           `gorm:"not null;default:1;index" json:"status"`
	CustomerID   *uint64        `gorm:"index" json:"customerId"`
	CustomerName string         `gorm:"size:128" json:"customerName"`
	ContactName  string         `gorm:"size:64" json:"contactName"`
	ContactPhone string         `gorm:"size:32" json:"contactPhone"`
	Currency     string         `gorm:"size:8;not null;default:CNY" json:"currency"`
	ValidUntil   *time.Time     `json:"validUntil"`
	Remark       string         `gorm:"type:text" json:"remark"`
	DiscountAmt  float64        `gorm:"type:decimal(12,2);not null;default:0" json:"discountAmt"`
	ShippingAmt  float64        `gorm:"type:decimal(12,2);not null;default:0" json:"shippingAmt"`
	TaxAmt       float64        `gorm:"type:decimal(12,2);not null;default:0" json:"taxAmt"`
	SubtotalAmt  float64        `gorm:"type:decimal(12,2);not null;default:0" json:"subtotalAmt"`
	TotalAmt     float64        `gorm:"type:decimal(12,2);not null;default:0" json:"totalAmt"`
	TemplateID   *uint64        `json:"templateId"`
	TemplateSnap string         `gorm:"type:text" json:"templateSnap"`
	CreatedBy    uint64         `gorm:"not null;default:0" json:"createdBy"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	Items []QuoteItem `gorm:"foreignKey:QuoteID" json:"items,omitempty"`
}

func (Quote) TableName() string { return "quotes" }

type QuoteItem struct {
	ID           uint64         `gorm:"primaryKey" json:"id"`
	TenantID     uint64         `gorm:"index;not null;default:1" json:"tenantId"`
	QuoteID      uint64         `gorm:"index;not null" json:"quoteId"`
	Sort         int            `gorm:"not null;default:0" json:"sort"`
	Source       string         `gorm:"size:16;not null;default:manual" json:"source"`
	ProductID    *uint64        `json:"productId"`
	SkuID        *uint64        `json:"skuId"`
	Name         string         `gorm:"size:255;not null" json:"name"`
	SpecLabel    string         `gorm:"size:255" json:"specLabel"`
	ImageURL     string         `gorm:"size:512" json:"imageUrl"`
	Qty          float64        `gorm:"type:decimal(12,2);not null;default:1" json:"qty"`
	Unit         string         `gorm:"size:16;not null;default:件" json:"unit"`
	RetailPrice  float64        `gorm:"type:decimal(12,2);not null;default:0" json:"retailPrice"`
	QuotePrice   float64        `gorm:"type:decimal(12,2);not null;default:0" json:"quotePrice"`
	LineTotal    float64        `gorm:"type:decimal(12,2);not null;default:0" json:"lineTotal"`
	UpgradeNote  string         `gorm:"size:512" json:"upgradeNote"`
	ParamsText   string         `gorm:"type:text" json:"paramsText"`
	Remark       string         `gorm:"size:512" json:"remark"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (QuoteItem) TableName() string { return "quote_items" }
