package dto

type TemplateLineReq struct {
	Sort     int    `json:"sort"`
	Category string `json:"category"`
	PartName string `json:"partName" binding:"required"`
	Hint     string `json:"hint"`
}

type TemplateSaveReq struct {
	Name            string            `json:"name" binding:"required"`
	Kind            string            `json:"kind"` // layout | skeleton
	IsDefault       bool              `json:"isDefault"`
	LogoURL         string            `json:"logoUrl"`
	ShopName        string            `json:"shopName"`
	ShopPhone       string            `json:"shopPhone"`
	ShopAddress     string            `json:"shopAddress"`
	HeaderSubtitle  string            `json:"headerSubtitle"`
	FooterText      string            `json:"footerText"`
	ShowLogo        *bool             `json:"showLogo"`
	ShowRetailPrice *bool             `json:"showRetailPrice"`
	ShowSpecImage   *bool             `json:"showSpecImage"`
	ShowUpgrade     *bool             `json:"showUpgrade"`
	ShowParams      *bool             `json:"showParams"`
	ShowTotals      *bool             `json:"showTotals"`
	StylePreset     string            `json:"stylePreset"`
	Lines           []TemplateLineReq `json:"lines"`
}

type QuoteItemReq struct {
	ID               *uint64 `json:"id"`
	Sort             int     `json:"sort"`
	Source           string  `json:"source"`
	ProductID        *uint64 `json:"productId"`
	SkuID            *uint64 `json:"skuId"`
	TemplateLineID   *uint64 `json:"templateLineId"`
	Category         string  `json:"category"`
	PartName         string  `json:"partName"`
	Name             string  `json:"name"`
	SpecLabel        string  `json:"specLabel"`
	ImageURL         string  `json:"imageUrl"`
	Qty              float64 `json:"qty"`
	Unit             string  `json:"unit"`
	RetailPrice      float64 `json:"retailPrice"`
	CostPrice        float64 `json:"costPrice"`
	SupplyPrice      float64 `json:"supplyPrice"`
	SupplyPriceAt    *string `json:"supplyPriceAt"` // RFC3339 or empty; admin round-trip
	SupplyRemark     string  `json:"supplyRemark"`
	QuotePrice       float64 `json:"quotePrice"`
	UpgradeNote      string  `json:"upgradeNote"`
	ParamsText       string  `json:"paramsText"`
	Remark           string  `json:"remark"`
	CustomerSelected *bool   `json:"customerSelected"`
}

type QuoteSaveReq struct {
	Title        string         `json:"title"`
	Status       *int8          `json:"status"`
	CustomerID   *uint64        `json:"customerId"`
	CustomerName string         `json:"customerName"`
	ContactName  string         `json:"contactName"`
	ContactPhone string         `json:"contactPhone"`
	Currency     string         `json:"currency"`
	ValidUntil   *string        `json:"validUntil"`
	Remark       string         `json:"remark"`
	DiscountAmt  float64        `json:"discountAmt"`
	ShippingAmt  float64        `json:"shippingAmt"`
	TaxAmt       float64        `json:"taxAmt"`
	TemplateID   *uint64        `json:"templateId"`
	Items        []QuoteItemReq `json:"items"`
}

type CustomerPricedReq struct {
	Flags []bool `json:"flags"`
}

type SecondEditApplyReq struct {
	ApplicantName  string `json:"applicantName"`
	ApplicantPhone string `json:"applicantPhone"`
	ApplicantNote  string `json:"applicantNote"`
}

type SecondEditOpenReq struct {
	ApplicantName  string `json:"applicantName"`
	ApplicantPhone string `json:"applicantPhone"`
}

type DashboardStats struct {
	QuoteCount           int64   `json:"quoteCount"`
	DraftCount           int64   `json:"draftCount"`
	SentCount            int64   `json:"sentCount"`
	WonCount             int64   `json:"wonCount"`
	TemplateCount        int64   `json:"templateCount"`
	MonthTotalAmt        float64 `json:"monthTotalAmt"`
	SecondEditApplyCount int64   `json:"secondEditApplyCount"`
}
