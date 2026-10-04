import client, { unwrap, type PageData } from './client'

export interface DashboardStats {
  quoteCount: number
  draftCount: number
  sentCount: number
  wonCount: number
  templateCount: number
  monthTotalAmt: number
}

export interface QuoteTemplateLine {
  id?: number
  sort: number
  category: string
  partName: string
  hint?: string
}

export interface QuoteTemplate {
  id: number
  name: string
  kind?: string // layout | skeleton
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
  lines?: QuoteTemplateLine[]
}

export interface QuoteItem {
  id?: number
  sort: number
  source: string
  productId?: number | null
  skuId?: number | null
  templateLineId?: number | null
  category?: string
  partName?: string
  name: string
  specLabel: string
  imageUrl: string
  qty: number
  unit: string
  retailPrice: number
  costPrice: number
  /** 供货商填写的拿货价；有更新时间时参与成本合计 */
  supplyPrice?: number
  supplyPriceAt?: string | null
  /** 供货商备注 */
  supplyRemark?: string
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

export function saveTemplate(
  data: Partial<QuoteTemplate> & { name: string; lines?: QuoteTemplateLine[] },
  id?: number,
) {
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

export function ensureShareToken(id: number) {
  return client.post(`/quotes/${id}/share`).then((r) =>
    unwrap<{ shareToken: string; quoteNo: string; quoteId: number }>(r),
  )
}

export function ensureCustomerShareToken(id: number) {
  return client.post(`/quotes/${id}/customer-share`).then((r) =>
    unwrap<{ shareToken: string; quoteNo: string; quoteId: number }>(r),
  )
}

/** 有供货商拿货价更新时，用拿货价作为该规格成本 */
export function effectiveCostPrice(it: Pick<QuoteItem, 'costPrice' | 'supplyPrice' | 'supplyPriceAt'>): number {
  if (it.supplyPriceAt) return Number(it.supplyPrice || 0)
  return Number(it.costPrice || 0)
}

export function searchCustomers(keyword: string, page = 1, pageSize = 20) {
  return client.get('/customers/search', { params: { keyword, page, pageSize } }).then((r) => unwrap<PageData<CustomerHit>>(r))
}

export function searchProductSkus(keyword: string, page = 1, pageSize = 20) {
  return client.get('/product-skus/search', { params: { keyword, page, pageSize } }).then((r) => unwrap<PageData<SkuHit>>(r))
}

/** 与 Excel「精灵」结构对齐的组装车骨架预设 */
export const ASSEMBLE_SKELETON_PRESET: QuoteTemplateLine[] = [
  { sort: 10, category: '车架组', partName: '车架' },
  { sort: 20, category: '车架组', partName: '前叉' },
  { sort: 30, category: '车架组', partName: '座管' },
  { sort: 40, category: '车架组', partName: '弯把' },
  { sort: 50, category: '车架组', partName: '把立' },
  { sort: 60, category: '变速套件', partName: '手变前拨后拨' },
  { sort: 70, category: '变速套件', partName: '夹器' },
  { sort: 80, category: '变速套件', partName: '飞轮' },
  { sort: 90, category: '变速套件', partName: '链条' },
  { sort: 100, category: '变速套件', partName: '牙盘' },
  { sort: 110, category: '变速套件', partName: '碟片' },
  { sort: 120, category: '轮组', partName: '轮组' },
  { sort: 130, category: '轮组', partName: '外胎' },
  { sort: 140, category: '轮组', partName: '内胎' },
  { sort: 150, category: '其他', partName: '中轴' },
  { sort: 160, category: '其他', partName: '坐垫' },
  { sort: 170, category: '其他', partName: '脚踏' },
  { sort: 180, category: '其他', partName: '把带' },
  { sort: 190, category: '其他', partName: '水壶架' },
  { sort: 200, category: '服务', partName: '组装费' },
  { sort: 210, category: '服务', partName: '运费' },
]

export function isSkeletonTemplate(t?: QuoteTemplate | null): boolean {
  if (!t) return false
  if (t.kind === 'skeleton') return true
  return Array.isArray(t.lines) && t.lines.length > 0
}

export function seedItemsFromTemplate(t: QuoteTemplate): QuoteItem[] {
  const lines = [...(t.lines || [])].sort((a, b) => (a.sort || 0) - (b.sort || 0))
  return lines.map((ln, i) => ({
    sort: ln.sort || (i + 1) * 10,
    source: 'template',
    templateLineId: ln.id ?? null,
    category: ln.category || '',
    partName: ln.partName || '',
    name: '',
    specLabel: '',
    imageUrl: '',
    qty: 1,
    unit: '件',
    retailPrice: 0,
    costPrice: 0,
    quotePrice: 0,
    upgradeNote: '',
    paramsText: '',
    remark: '',
  }))
}
