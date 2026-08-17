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

export function submitSupplyPrices(
  token: string,
  items: { id: number; supplyPrice: number; supplyRemark?: string }[],
) {
  return publicClient
    .post(`/share/${encodeURIComponent(token)}/submit`, { items })
    .then((r) => (r.data as ApiResponse<{ ok: boolean }>).data)
}
