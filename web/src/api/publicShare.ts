import axios from 'axios'
import type { ApiResponse } from './client'

const publicClient = axios.create({
  baseURL: (import.meta.env.BASE_URL || '/') + 'api/v1/public',
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
})

publicClient.interceptors.response.use(
  (res) => {
    const body = res.data as ApiResponse
    if (body.code !== 200) {
      return Promise.reject(new Error(body.message || '请求失败'))
    }
    return res
  },
  (err) => {
    const msg = err.response?.data?.message || err.message || '请求失败'
    return Promise.reject(new Error(msg))
  },
)

export interface ShareItem {
  id: number
  sort: number
  category: string
  partName: string
  name: string
  specLabel: string
  imageUrl: string
  supplyPrice: number
  supplyRemark: string
}

export interface ShareQuote {
  title: string
  quoteNo: string
  remark: string
  items: ShareItem[]
}

export function fetchShareQuote(token: string) {
  return publicClient.get(`/share/${encodeURIComponent(token)}`).then((r) => (r.data as ApiResponse<ShareQuote>).data as ShareQuote)
}

export interface CustomerShareTemplate {
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
  kind: string
}

export interface CustomerShareItem {
  id?: number
  sort: number
  productId?: number | null
  category: string
  partName: string
  name: string
  specLabel: string
  imageUrl: string
  qty: number
  unit: string
  retailPrice: number
  quotePrice: number
  upgradeNote: string
  paramsText: string
  remark: string
  selected?: boolean
}

export interface CustomerShareQuote {
  title: string
  quoteNo: string
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
  pricedSaved?: boolean
  pricedFlags?: boolean[]
  secondEditUsed?: boolean
  isSecondEdit?: boolean
  template: CustomerShareTemplate
  items: CustomerShareItem[]
}

export function fetchCustomerShare(token: string) {
  return publicClient
    .get(`/customer/${encodeURIComponent(token)}`)
    .then((r) => (r.data as ApiResponse<CustomerShareQuote>).data as CustomerShareQuote)
}

export function saveCustomerPriced(token: string, flags: boolean[]) {
  return publicClient
    .post(`/customer/${encodeURIComponent(token)}/priced`, { flags })
    .then((r) => (r.data as ApiResponse<CustomerShareQuote>).data as CustomerShareQuote)
}

export function applySecondEdit(
  token: string,
  payload: { applicantName: string; applicantPhone: string; applicantNote?: string },
) {
  return publicClient
    .post(`/customer/${encodeURIComponent(token)}/second-edit`, payload)
    .then((r) => (r.data as ApiResponse<{ token: string; quoteNo: string; originQuoteNo: string }>).data!)
}

export function openSecondEdit(
  token: string,
  payload: { applicantPhone: string; applicantName?: string },
) {
  return publicClient
    .post(`/customer/${encodeURIComponent(token)}/second-edit/open`, payload)
    .then((r) => (r.data as ApiResponse<{ token: string; quoteNo: string; originQuoteNo: string }>).data!)
}

export interface SecondEditQuote {
  id: number
  quoteNo: string
  originQuoteNo: string
  title: string
  status: number
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
  template: CustomerShareTemplate
  secondEditApplicant: string
  secondEditApplicantPhone: string
  secondEditApplicantNote: string
  items: {
    id: number
    sort: number
    source: string
    productId?: number | null
    skuId?: number | null
    templateLineId?: number | null
    category: string
    partName: string
    name: string
    specLabel: string
    imageUrl: string
    qty: number
    unit: string
    retailPrice: number
    quotePrice: number
    upgradeNote: string
    paramsText: string
    remark: string
    customerSelected?: boolean
  }[]
}

export function fetchSecondEdit(token: string) {
  return publicClient
    .get(`/revise/${encodeURIComponent(token)}`)
    .then((r) => (r.data as ApiResponse<SecondEditQuote>).data as SecondEditQuote)
}

export function saveSecondEdit(token: string, payload: Record<string, unknown>) {
  return publicClient
    .put(`/revise/${encodeURIComponent(token)}`, payload)
    .then((r) => (r.data as ApiResponse<SecondEditQuote>).data as SecondEditQuote)
}

export async function uploadSecondEditImage(token: string, file: File, subdir = 'quote'): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  form.append('subdir', subdir)
  const r = await publicClient.post(`/revise/${encodeURIComponent(token)}/upload`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 120000,
  })
  return ((r.data as ApiResponse<{ url: string }>).data as { url: string }).url
}

export async function uploadSecondEditImageFromUrl(token: string, url: string, subdir = 'items'): Promise<string> {
  const r = await publicClient.post(
    `/revise/${encodeURIComponent(token)}/upload-from-url`,
    { url, subdir },
    { timeout: 120000 },
  )
  return ((r.data as ApiResponse<{ url: string }>).data as { url: string }).url
}

export function submitSupplyPrices(
  token: string,
  items: { id: number; supplyPrice: number; supplyRemark?: string }[],
) {
  return publicClient
    .post(`/share/${encodeURIComponent(token)}/submit`, { items })
    .then((r) => (r.data as ApiResponse<{ ok: boolean }>).data)
}
