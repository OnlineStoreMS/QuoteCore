# QuoteCore · 报价中心

OSMS 报价中心：为客户选品或手填明细，生成可复制图片 / 下载 PDF 的报价单。

| 项 | 值 |
|----|----|
| Web | `5189`（路径 `/apps/quote/`） |
| API | `8105` |
| 权限 | `quote:read` / `quote:write` |

## 本地开发

```bash
# API
go run ./cmd/api -config configs/config.yaml

# Web
cd web && npm install && npm run dev
```

依赖：PostgreSQL、UserCore JWT、可选 CustomerCore (`8099`) / ProductCore (`8090`) 搜索代理。

## 功能（一期）

- 报价单 CRUD、复制、作废
- 客户中心搜索 / 手填客户
- 商品中心 SKU 搜索 / 手填明细
- 报价模板（Logo、店名、显示项；不绑定门店）
- 预览、复制图片、下载 PDF
