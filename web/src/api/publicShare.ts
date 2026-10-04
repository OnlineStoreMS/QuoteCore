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
  template: CustomerShareTemplate
  items: CustomerShareItem[]
}

export function fetchCustomerShare(token: string) {
  return publicClient
    .get(`/customer/${encodeURIComponent(token)}`)
    .then((r) => (r.data as ApiResponse<CustomerShareQuote>).data as CustomerShareQuote)
}

export function submitSupplyPrices(
  token: string,
  items: { id: number; supplyPrice: number; supplyRemark?: string }[],
) {
  return publicClient
    .post(`/share/${encodeURIComponent(token)}/submit`, { items })
    .then((r) => (r.data as ApiResponse<{ ok: boolean }>).data)
}
