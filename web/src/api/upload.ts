import client, { unwrap } from './client'

export async function uploadImage(file: File, subdir = 'quote'): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  form.append('subdir', subdir)
  const res = await client.post('/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 120000,
  })
  const data = unwrap<{ url: string }>(res)
  return data.url
}

/** 下载远程图片并上传到报价中心存储，返回本站 URL */
export async function uploadImageFromUrl(sourceUrl: string, subdir = 'items'): Promise<string> {
  const res = await client.post(
    '/upload/from-url',
    { url: sourceUrl, subdir },
    { timeout: 120000 },
  )
  const data = unwrap<{ url: string }>(res)
  return data.url
}

/** 已是报价中心 / MinIO 本站地址则无需再传 */
export function isQuoteStoredUrl(url: string): boolean {
  const u = (url || '').trim().toLowerCase()
  if (!u) return false
  return u.includes('/minio/quotecore/') || u.includes('/quotecore/uploads/') || u.includes('/uploads/')
}
