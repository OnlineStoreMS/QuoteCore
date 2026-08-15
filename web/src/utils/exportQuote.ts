import html2canvas from 'html2canvas'
import { jsPDF } from 'jspdf'
import { nextTick } from 'vue'

export async function waitForImages(root: HTMLElement, timeoutMs = 8000) {
  const imgs = Array.from(root.querySelectorAll('img'))
  if (!imgs.length) return
  await Promise.all(
    imgs.map(
      (img) =>
        new Promise<void>((resolve) => {
          if (img.complete && img.naturalWidth > 0) {
            resolve()
            return
          }
          const done = () => resolve()
          img.addEventListener('load', done, { once: true })
          img.addEventListener('error', done, { once: true })
          setTimeout(done, timeoutMs)
        }),
    ),
  )
}

/** 打开预览后等到 DOM 与图片就绪 */
export async function prepareExportElement(getEl: () => HTMLElement | null, open: () => void) {
  open()
  await nextTick()
  await nextTick()
  // 等 dialog 动画 / 布局
  await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))
  await new Promise((r) => setTimeout(r, 120))

  let el: HTMLElement | null = null
  for (let i = 0; i < 20; i++) {
    el = getEl()
    if (el) break
    await new Promise((r) => setTimeout(r, 50))
  }
  if (!el) throw new Error('预览区域未就绪，请先点「预览」再复制')
  await waitForImages(el)
  await new Promise((r) => setTimeout(r, 60))
  return el
}

export async function elementToCanvas(el: HTMLElement) {
  // 禁止 allowTaint：否则 toBlob / 剪贴板会失败
  return html2canvas(el, {
    scale: 2,
    useCORS: true,
    allowTaint: false,
    backgroundColor: '#ffffff',
    logging: false,
    imageTimeout: 15000,
    onclone: (_doc, cloned) => {
      cloned.querySelectorAll('img').forEach((img) => {
        img.crossOrigin = 'anonymous'
        // 避免克隆节点仍在懒加载态
        if (img.loading === 'lazy') img.loading = 'eager'
      })
    },
  })
}

function canvasToPngBlob(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (blob) resolve(blob)
        else reject(new Error('生成图片失败（可能含跨域图片，请改用本站上传图）'))
      },
      'image/png',
    )
  })
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

export type CopyImageResult = 'clipboard' | 'download'

export async function copyElementAsImage(el: HTMLElement): Promise<CopyImageResult> {
  const canvas = await elementToCanvas(el)
  const blob = await canvasToPngBlob(canvas)

  const canClipboard =
    typeof navigator !== 'undefined' &&
    !!navigator.clipboard &&
    typeof ClipboardItem !== 'undefined' &&
    window.isSecureContext

  if (canClipboard) {
    try {
      // 部分浏览器要求 ClipboardItem 值为 Promise
      await navigator.clipboard.write([
        new ClipboardItem({
          'image/png': Promise.resolve(blob),
        }),
      ])
      return 'clipboard'
    } catch {
      // Safari / 权限不足时回退下载
      downloadBlob(blob, `quote-${Date.now()}.png`)
      return 'download'
    }
  }

  downloadBlob(blob, `quote-${Date.now()}.png`)
  return 'download'
}

export async function downloadElementAsPdf(el: HTMLElement, filename: string) {
  const canvas = await elementToCanvas(el)
  const img = canvas.toDataURL('image/jpeg', 0.92)
  const pdf = new jsPDF({ orientation: 'portrait', unit: 'pt', format: 'a4' })
  const pageW = pdf.internal.pageSize.getWidth()
  const pageH = pdf.internal.pageSize.getHeight()
  const margin = 24
  const maxW = pageW - margin * 2
  const maxH = pageH - margin * 2
  const ratio = Math.min(maxW / canvas.width, maxH / canvas.height)
  const w = canvas.width * ratio
  const h = canvas.height * ratio
  // 多页：超出一页时继续追加
  let remain = h
  let srcY = 0
  const pageContentH = maxH
  const sliceCanvas = document.createElement('canvas')
  const sliceCtx = sliceCanvas.getContext('2d')
  if (!sliceCtx || h <= pageContentH) {
    pdf.addImage(img, 'JPEG', (pageW - w) / 2, margin, w, h)
    pdf.save(filename.endsWith('.pdf') ? filename : `${filename}.pdf`)
    return
  }

  const scale = canvas.width / w
  while (remain > 0) {
    const sliceH = Math.min(pageContentH, remain)
    sliceCanvas.width = canvas.width
    sliceCanvas.height = Math.max(1, Math.floor(sliceH * scale))
    sliceCtx.clearRect(0, 0, sliceCanvas.width, sliceCanvas.height)
    sliceCtx.drawImage(
      canvas,
      0,
      Math.floor(srcY * scale),
      canvas.width,
      sliceCanvas.height,
      0,
      0,
      canvas.width,
      sliceCanvas.height,
    )
    const sliceData = sliceCanvas.toDataURL('image/jpeg', 0.92)
    pdf.addImage(sliceData, 'JPEG', (pageW - w) / 2, margin, w, sliceH)
    remain -= sliceH
    srcY += sliceH
    if (remain > 0) pdf.addPage()
  }
  pdf.save(filename.endsWith('.pdf') ? filename : `${filename}.pdf`)
}
