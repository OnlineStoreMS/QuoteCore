import type { QuoteItem } from '../api/quote'

/** 与报价单「同产品规格」分组一致：相邻行 productId 相同，或名称相同。 */
export function isSameProductSpec(a: QuoteItem, b: QuoteItem): boolean {
  if (a.productId && b.productId) return Number(a.productId) === Number(b.productId)
  const na = (a.name || '').trim()
  const nb = (b.name || '').trim()
  return na !== '' && na === nb
}

/** 每组同产品默认只选中第一行计价。 */
export function defaultPricedFlags(items: QuoteItem[]): boolean[] {
  return items.map((it, idx) => idx === 0 || !isSameProductSpec(it, items[idx - 1]))
}

/** 同产品规格所在的连续行范围。 */
export function productGroupRange(items: QuoteItem[], idx: number): [number, number] {
  let head = idx
  while (head > 0 && isSameProductSpec(items[head], items[head - 1])) head -= 1
  let end = head
  while (end + 1 < items.length && isSameProductSpec(items[end + 1], items[end])) end += 1
  return [head, end]
}

/** 一组同产品里只保留 idx 这一行计价。 */
export function selectOnlySpec(items: QuoteItem[], flags: boolean[], idx: number): boolean[] {
  const next = flags.length === items.length ? flags.slice() : defaultPricedFlags(items)
  const [head, end] = productGroupRange(items, idx)
  for (let i = head; i <= end; i++) next[i] = i === idx
  return next
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
