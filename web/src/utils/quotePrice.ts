import type { QuoteItem } from '../api/quote'

/** 与报价单「同产品规格」分组一致：相邻行 productId 相同，或名称相同。 */
export function isSameProductSpec(a: QuoteItem, b: QuoteItem): boolean {
  if (a.productId && b.productId) return Number(a.productId) === Number(b.productId)
  const na = (a.name || '').trim()
  const nb = (b.name || '').trim()
  return na !== '' && na === nb
}

/** 每组同产品默认只勾选第一行计价。 */
export function defaultPricedFlags(items: QuoteItem[]): boolean[] {
  return items.map((it, idx) => idx === 0 || !isSameProductSpec(it, items[idx - 1]))
}

export function sumPriced(
  items: QuoteItem[],
  flags: boolean[],
  amount: (it: QuoteItem) => number,
): number {
  let s = 0
  items.forEach((it, i) => {
    if (flags[i] !== false) s += amount(it)
  })
  return Math.round(s * 100) / 100
}
