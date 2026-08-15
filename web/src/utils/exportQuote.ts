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

function blobToDataURL(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const fr = new FileReader()
    fr.onload = () => resolve(String(fr.result || ''))
    fr.onerror = () => reject(fr.error || new Error('read failed'))
    fr.readAsDataURL(blob)
  })
}

/** 把图片转成 dataURL，避免跨域/CORS 污染 canvas */
async function inlineImagesAsDataURL(root: HTMLElement) {
  const imgs = Array.from(root.querySelectorAll('img'))
  await Promise.all(
    imgs.map(async (img) => {
      const src = (img.currentSrc || img.getAttribute('src') || '').trim()
      if (!src || src.startsWith('data:') || src.startsWith('blob:')) return
      try {
        const res = await fetch(src, { mode: 'cors', credentials: 'omit', cache: 'no-cache' })
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        const blob = await res.blob()
        if (!blob.type.startsWith('image/') && blob.size === 0) throw new Error('empty image')
        img.src = await blobToDataURL(blob)
        img.removeAttribute('crossorigin')
        img.loading = 'eager'
      } catch {
        // 同源失败时尝试去掉 crossorigin 再等原图（可能污染 canvas，后面 toBlob 会兜底）
        img.removeAttribute('crossorigin')
      }
    }),
  )
}

/** 打开预览后等到 DOM 与图片就绪（兼容旧调用） */
export async function prepareExportElement(getEl: () => HTMLElement | null, open: () => void) {
  open()
  await nextTick()
  await nextTick()
  await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))
  await new Promise((r) => setTimeout(r, 160))

  let el: HTMLElement | null = null
  for (let i = 0; i < 30; i++) {
    el = getEl()
    if (el && el.offsetWidth > 0) break
    await new Promise((r) => setTimeout(r, 50))
  }
  if (!el) throw new Error('预览区域未就绪，请先点「预览」再复制')
  await waitForImages(el)
  await new Promise((r) => setTimeout(r, 60))
  return el
}

export async function elementToCanvas(el: HTMLElement) {
  const width = Math.max(el.scrollWidth, el.offsetWidth, 794)
  const clone = el.cloneNode(true) as HTMLElement
  clone.setAttribute('data-quote-export-clone', '1')
  clone.style.cssText = [
    'position:fixed',
    'left:-12000px',
    'top:0',
    `width:${width}px`,
    'margin:0',
    'padding:0',
    'background:#ffffff',
    'z-index:-1',
    'pointer-events:none',
    'opacity:1',
    'transform:none',
    'overflow:visible',
  ].join(';')
  document.body.appendChild(clone)

  try {
    await inlineImagesAsDataURL(clone)
    await waitForImages(clone, 12000)
    await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))

    return await html2canvas(clone, {
      scale: 2,
      useCORS: true,
      allowTaint: false,
      backgroundColor: '#ffffff',
      logging: false,
      imageTimeout: 20000,
      width,
      windowWidth: width,
      scrollX: 0,
      scrollY: 0,
      onclone: (_doc, cloned) => {
        cloned.querySelectorAll('img').forEach((img) => {
          if (img.loading === 'lazy') img.loading = 'eager'
        })
      },
    })
  } finally {
    clone.remove()
  }
}

function canvasToPngBlob(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) => {
    try {
      canvas.toBlob(
        (blob) => {
          if (blob) resolve(blob)
          else reject(new Error('生成图片失败（画布可能被跨域图片污染，请确认图片已上传到报价中心）'))
        },
        'image/png',
      )
    } catch (e) {
      reject(e instanceof Error ? e : new Error('生成图片失败'))
    }
  })
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 2000)
}

export type CopyImageResult = 'clipboard' | 'download'

async function tryWriteClipboard(blob: Blob): Promise<boolean> {
  if (typeof navigator === 'undefined' || !navigator.clipboard || typeof ClipboardItem === 'undefined') {
    return false
  }
  if (!window.isSecureContext) return false
  try {
    window.focus()
  } catch {
    /* ignore */
  }

  // Chromium: Blob 直接写入更稳；部分环境要求 Promise
  try {
    await navigator.clipboard.write([new ClipboardItem({ 'image/png': blob })])
    return true
  } catch {
    /* fallthrough */
  }
  try {
    await navigator.clipboard.write([
      new ClipboardItem({
        'image/png': Promise.resolve(blob),
      }),
    ])
    return true
  } catch {
    return false
  }
}

export async function copyElementAsImage(el: HTMLElement, filename?: string): Promise<CopyImageResult> {
  const canvas = await elementToCanvas(el)
  const blob = await canvasToPngBlob(canvas)
  const name = filename || `quote-${Date.now()}.png`

  if (await tryWriteClipboard(blob)) {
    return 'clipboard'
  }
  downloadBlob(blob, name.endsWith('.png') ? name : `${name}.png`)
  return 'download'
}

export async function downloadElementAsPng(el: HTMLElement, filename: string) {
  const canvas = await elementToCanvas(el)
  const blob = await canvasToPngBlob(canvas)
  downloadBlob(blob, filename.endsWith('.png') ? filename : `${filename}.png`)
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
