import client, { unwrap, type PageData } from './client'

export interface DashboardStats {
  quoteCount: number
  draftCount: number
  sentCount: number
  wonCount: number
  templateCount: number
  monthTotalAmt: number
}

export interface QuoteTemplate {
  id: number
  name: string
  isDefault: boolean
  logoUrl: string
  shopName: string
  shopPhone: string
  shopAddress: string
  headerSubtitle: string
  footerText: string
  showLogo: boolean
  showRetailPrice: boolean
  showSpecImage: boolean
  showUpgrade: boolean
  showParams: boolean
  showTotals: boolean
  stylePreset: string
}

export interface QuoteItem {
  id?: number
  sort: number
  source: string
  productId?: number | null
  skuId?: number | null
  name: string
  specLabel: string
  imageUrl: string
  qty: number
  unit: string
  retailPrice: number
  quotePrice: number
  lineTotal?: number
  upgradeNote: string
  paramsText: string
  remark: string
}

export interface Quote {
  id: number
  quoteNo: string
  title: string
  status: number
  customerId?: number | null
  customerName: string
  contactName: string
  contactPhone: string
  currency: string
  validUntil?: string | null
  remark: string
  discountAmt: number
  shippingAmt: number
  taxAmt: number
  subtotalAmt: number
  totalAmt: number
  templateId?: number | null
  templateSnap?: string
  items?: QuoteItem[]
  createdAt?: string
  updatedAt?: string
}

export interface QuoteSavePayload {
  title: string
  status?: number
  customerId?: number | null
  customerName: string
  contactName: string
  contactPhone: string
  currency: string
  validUntil?: string | null
  remark: string
  discountAmt: number
  shippingAmt: number
  taxAmt: number
  templateId?: number | null
  items: QuoteItem[]
}

export interface CustomerHit {
  id: number
  displayName: string
  primaryPhone: string
  status: number
  remark?: string
}

export interface SkuHit {
  productId: number
  productName: string
  productPic: string
  brandName: string
  skuId: number
  skuCode: string
  specLabel: string
  price: number
  stock: number
  pic: string
}

export function fetchDashboardStats() {
  return client.get('/dashboard/stats').then((r) => unwrap<DashboardStats>(r))
}

export function listTemplates() {
  return client.get('/quote-templates').then((r) => unwrap<QuoteTemplate[]>(r))
}

export function saveTemplate(data: Partial<QuoteTemplate> & { name: string }, id?: number) {
  if (id) return client.put(`/quote-templates/${id}`, data).then((r) => unwrap<QuoteTemplate>(r))
  return client.post('/quote-templates', data).then((r) => unwrap<QuoteTemplate>(r))
}

export function deleteTemplate(id: number) {
  return client.delete(`/quote-templates/${id}`).then((r) => unwrap(r))
}

export function listQuotes(params: { keyword?: string; status?: string | number; page?: number; pageSize?: number }) {
  return client.get('/quotes', { params }).then((r) => unwrap<PageData<Quote>>(r))
}

export function getQuote(id: number) {
  return client.get(`/quotes/${id}`).then((r) => unwrap<Quote>(r))
}

export function saveQuote(data: QuoteSavePayload, id?: number) {
  if (id) return client.put(`/quotes/${id}`, data).then((r) => unwrap<Quote>(r))
  return client.post('/quotes', data).then((r) => unwrap<Quote>(r))
}

export function deleteQuote(id: number) {
  return client.delete(`/quotes/${id}`).then((r) => unwrap(r))
}

export function voidQuote(id: number) {
  return client.post(`/quotes/${id}/void`).then((r) => unwrap<Quote>(r))
}

export function copyQuote(id: number) {
  return client.post(`/quotes/${id}/copy`).then((r) => unwrap<Quote>(r))
}

export function searchCustomers(keyword: string, page = 1, pageSize = 20) {
  return client.get('/customers/search', { params: { keyword, page, pageSize } }).then((r) => unwrap<PageData<CustomerHit>>(r))
}

export function searchProductSkus(keyword: string, page = 1, pageSize = 20) {
  return client.get('/product-skus/search', { params: { keyword, page, pageSize } }).then((r) => unwrap<PageData<SkuHit>>(r))
}
