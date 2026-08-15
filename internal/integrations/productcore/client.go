package productcore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type SkuSearchItem struct {
	ProductID    uint64            `json:"productId"`
	ProductName  string            `json:"productName"`
	MaterialCode string            `json:"materialCode"`
	ProductSn    string            `json:"productSn"`
	ProductPic   string            `json:"productPic"`
	BrandName    string            `json:"brandName"`
	CategoryName string            `json:"categoryName"`
	SkuID        uint64            `json:"skuId"`
	SkuCode      string            `json:"skuCode"`
	Specs        map[string]string `json:"specs"`
	SpecLabel    string            `json:"specLabel"`
	Price        float64           `json:"price"`
	Stock        int               `json:"stock"`
	Pic          string            `json:"pic"`
}

type pagePayload[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

type apiBody struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) SearchSkus(ctx context.Context, authHeader, keyword string, page, pageSize int) ([]SkuSearchItem, int64, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, 0, fmt.Errorf("keyword required")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := url.Values{}
	q.Set("keyword", keyword)
	q.Set("page", strconv.Itoa(page))
	q.Set("pageSize", strconv.Itoa(pageSize))
	var pageData pagePayload[SkuSearchItem]
	if err := c.get(ctx, authHeader, "/api/v1/admin/super-search?"+q.Encode(), &pageData); err != nil {
		return nil, 0, err
	}
	if pageData.List == nil {
		pageData.List = []SkuSearchItem{}
	}
	for i := range pageData.List {
		pageData.List[i].SpecLabel = SpecValuesLabel(pageData.List[i].Specs, pageData.List[i].SpecLabel, pageData.List[i].SkuCode)
	}
	return pageData.List, pageData.Total, nil
}

// SpecValuesLabel 仅保留规格值（如「盒装 HG400-9飞轮 11-34T」），不含「颜色分类:」等规格名。
func SpecValuesLabel(specs map[string]string, fallback, skuCode string) string {
	if len(specs) > 0 {
		keys := make([]string, 0, len(specs))
		for k := range specs {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			v := strings.TrimSpace(specs[k])
			if v == "" {
				continue
			}
			parts = append(parts, v)
		}
		if len(parts) > 0 {
			return strings.Join(parts, " / ")
		}
	}
	label := stripSpecNamePrefixes(fallback)
	if label != "" {
		return label
	}
	return strings.TrimSpace(skuCode)
}

func stripSpecNamePrefixes(label string) string {
	label = strings.TrimSpace(label)
	if label == "" || label == "-" {
		return ""
	}
	parts := strings.Split(label, " / ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if v, ok := cutSpecValue(p); ok {
			out = append(out, v)
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, " / ")
}

func cutSpecValue(part string) (string, bool) {
	for _, sep := range []string{": ", "：", ":"} {
		if i := strings.Index(part, sep); i >= 0 {
			v := strings.TrimSpace(part[i+len(sep):])
			if v != "" {
				return v, true
			}
		}
	}
	return "", false
}

func (c *Client) get(ctx context.Context, authHeader, path string, dest any) error {
	if c.baseURL == "" {
		return fmt.Errorf("productcore url not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("productcore request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("productcore http %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var wrapped apiBody
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return fmt.Errorf("productcore decode: %w", err)
	}
	if wrapped.Code != 200 {
		msg := wrapped.Message
		if msg == "" {
			msg = "productcore error"
		}
		return fmt.Errorf("%s", msg)
	}
	return json.Unmarshal(wrapped.Data, dest)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
