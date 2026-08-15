package customercore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

type CustomerItem struct {
	ID           uint64 `json:"id"`
	DisplayName  string `json:"displayName"`
	PrimaryPhone string `json:"primaryPhone"`
	Status       int8   `json:"status"`
	Source       string `json:"source"`
	Remark       string `json:"remark"`
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

func (c *Client) SearchCustomers(ctx context.Context, authHeader, keyword string, page, pageSize int) ([]CustomerItem, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("pageSize", strconv.Itoa(pageSize))
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		q.Set("keyword", keyword)
	}
	var pageData pagePayload[CustomerItem]
	if err := c.get(ctx, authHeader, "/api/v1/admin/customers?"+q.Encode(), &pageData); err != nil {
		return nil, 0, err
	}
	if pageData.List == nil {
		pageData.List = []CustomerItem{}
	}
	return pageData.List, pageData.Total, nil
}

func (c *Client) get(ctx context.Context, authHeader, path string, dest any) error {
	if c.baseURL == "" {
		return fmt.Errorf("customercore url not configured")
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
		return fmt.Errorf("customercore request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("customercore http %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var wrapped apiBody
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return fmt.Errorf("customercore decode: %w", err)
	}
	if wrapped.Code != 200 {
		msg := wrapped.Message
		if msg == "" {
			msg = "customercore error"
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
