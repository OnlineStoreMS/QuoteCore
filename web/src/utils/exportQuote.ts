import html2canvas from 'html2canvas'
import { jsPDF } from 'jspdf'

export async function elementToCanvas(el: HTMLElement) {
  return html2canvas(el, {
    scale: 2,
    useCORS: true,
    allowTaint: true,
    backgroundColor: '#ffffff',
  })
}

export async function copyElementAsImage(el: HTMLElement) {
  const canvas = await elementToCanvas(el)
  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/png'))
  if (!blob) throw new Error('生成图片失败')
  if (navigator.clipboard && 'ClipboardItem' in window) {
    await navigator.clipboard.write([new ClipboardItem({ 'image/png': blob })])
    return
  }
  // fallback download
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `quote-${Date.now()}.png`
  a.click()
  URL.revokeObjectURL(url)
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
  pdf.addImage(img, 'JPEG', (pageW - w) / 2, margin, w, h)
  pdf.save(filename.endsWith('.pdf') ? filename : `${filename}.pdf`)
}
