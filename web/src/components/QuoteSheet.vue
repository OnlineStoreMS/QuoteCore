<script setup lang="ts">
import { computed } from 'vue'
import type { QuoteItem, QuoteTemplate } from '../api/quote'
import { isSkeletonTemplate } from '../api/quote'
import { defaultPricedFlags, productGroupRange, selectOnlySpec, sumPriced } from '../utils/quotePrice'

const props = defineProps<{
  quote: {
    quoteNo?: string
    title: string
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
    items: QuoteItem[]
  }
  template?: QuoteTemplate | null
  /** 顾客分享页：规格图可点开预览 */
  interactive?: boolean
  /** 顾客分享页：勾选哪些规格计入合计 */
  pickPrices?: boolean
  priceFlags?: boolean[] | null
}>()

const emit = defineEmits<{
  'preview-image': [url: string]
  'update:priceFlags': [flags: boolean[]]
}>()

function emitPreview(url?: string) {
  const u = (url || '').trim()
  if (!u || !props.interactive) return
  emit('preview-image', u)
}

const skeleton = computed(() => isSkeletonTemplate(props.template))
const showRetail = computed(() => props.template?.showRetailPrice !== false)
const specColStyle = computed(() =>
  showRetail.value
    ? skeleton.value
      ? undefined
      : { width: '140px' }
    : skeleton.value
      ? { minWidth: '220px' }
      : { width: '220px' },
)

function money(v: number) {
  return '¥' + Number(v || 0).toFixed(2)
}

function sameProduct(a: QuoteItem, b: QuoteItem): boolean {
  if (a.productId && b.productId) return Number(a.productId) === Number(b.productId)
  return (a.name || '').trim() !== '' && (a.name || '').trim() === (b.name || '').trim()
}

function sameCategory(a: QuoteItem, b: QuoteItem): boolean {
  const ca = (a.category || '').trim()
  const cb = (b.category || '').trim()
  return ca !== '' && ca === cb
}

function isProductHead(idx: number): boolean {
  if (idx <= 0) return true
  return !sameProduct(props.quote.items[idx], props.quote.items[idx - 1])
}

function isCategoryHead(idx: number): boolean {
  if (idx <= 0) return true
  const cur = props.quote.items[idx]
  const prev = props.quote.items[idx - 1]
  if (!(cur.category || '').trim()) return true
  return !sameCategory(cur, prev)
}

const pricedFlags = computed(() => {
  const items = props.quote.items
  if (props.priceFlags && props.priceFlags.length === items.length) return props.priceFlags
  return defaultPricedFlags(items)
})

const hasOptionalSpec = computed(() => pricedFlags.value.some((on) => !on))

const displaySubtotal = computed(() =>
  sumPriced(props.quote.items, pricedFlags.value, (it) => Number(it.qty || 0) * Number(it.quotePrice || 0)),
)

const displayTotal = computed(() =>
  Math.round(
    (displaySubtotal.value
      - Number(props.quote.discountAmt || 0)
      + Number(props.quote.shippingAmt || 0)
      + Number(props.quote.taxAmt || 0)) * 100,
  ) / 100,
)

function groupHead(idx: number) {
  return productGroupRange(props.quote.items, idx)[0]
}

function choosePrice(idx: number) {
  emit('update:priceFlags', selectOnlySpec(props.quote.items, pricedFlags.value, idx))
}
</script>

<template>
  <div class="sheet">
    <header class="head">
      <div class="brand">
        <img v-if="template?.showLogo !== false && template?.logoUrl" :src="template.logoUrl" class="logo" alt="logo" crossorigin="anonymous" />
        <div>
          <div class="shop">{{ template?.shopName || '报价中心' }}</div>
          <div class="sub">{{ template?.headerSubtitle || '' }}</div>
        </div>
      </div>
      <div class="meta">
        <div class="title">{{ quote.title || '报价单' }}</div>
        <div v-if="quote.quoteNo">单号：{{ quote.quoteNo }}</div>
        <div v-if="quote.validUntil">有效期：{{ String(quote.validUntil).slice(0, 10) }}</div>
      </div>
    </header>

    <section class="party">
      <div>客户：{{ quote.customerName || '—' }}</div>
      <div>联系人：{{ quote.contactName || '—' }}</div>
      <div>电话：{{ quote.contactPhone || '—' }}</div>
      <div v-if="template?.shopPhone">门店电话：{{ template.shopPhone }}</div>
      <div v-if="template?.shopAddress" class="full">地址：{{ template.shopAddress }}</div>
    </section>

    <table class="items">
      <thead>
        <tr v-if="skeleton">
          <th style="width:36px">#</th>
          <th style="width:72px">产品</th>
          <th style="width:150px">名称</th>
          <th style="width:72px">配件</th>
          <th :style="specColStyle">规格</th>
          <th style="width:72px">优惠价</th>
          <th v-if="showRetail" style="width:72px">零售价</th>
          <th v-if="template?.showSpecImage !== false" style="width:56px">图</th>
          <th style="width:90px">备注</th>
        </tr>
        <tr v-else>
          <th style="width:36px">#</th>
          <th v-if="template?.showSpecImage !== false" style="width:56px">图</th>
          <th>产品</th>
          <th :style="specColStyle">规格</th>
          <th style="width:56px">数量</th>
          <th v-if="showRetail" style="width:72px">零售价</th>
          <th style="width:72px">报价</th>
          <th style="width:88px">{{ pickPrices ? '计价' : '小计' }}</th>
        </tr>
      </thead>
      <tbody>
        <template v-if="skeleton">
          <tr
            v-for="(it, idx) in quote.items"
            :key="idx"
            :class="{ 'spec-row': !isCategoryHead(idx), 'product-head': isCategoryHead(idx) }"
          >
            <td class="cell-no" data-label="#">{{ idx + 1 }}</td>
            <td data-label="产品">{{ isCategoryHead(idx) ? (it.category || '—') : '' }}</td>
            <td data-label="名称">
              <template v-if="isCategoryHead(idx)">
                <div class="name">{{ it.name || '—' }}</div>
              </template>
            </td>
            <td data-label="配件">{{ it.partName || '—' }}</td>
            <td data-label="规格"><span class="spec-label">{{ it.specLabel || '—' }}</span></td>
            <td data-label="优惠价">{{ money(it.quotePrice) }}</td>
            <td v-if="showRetail" data-label="零售价">{{ money(it.retailPrice) }}</td>
            <td v-if="template?.showSpecImage !== false" class="cell-img" data-label="图">
              <button
                v-if="interactive && it.imageUrl"
                type="button"
                class="thumb-btn"
                aria-label="查看规格图"
                @click="emitPreview(it.imageUrl)"
              >
                <img :src="it.imageUrl" class="thumb" alt="" crossorigin="anonymous" />
              </button>
              <img v-else-if="it.imageUrl" :src="it.imageUrl" class="thumb" alt="" crossorigin="anonymous" />
            </td>
            <td data-label="备注">{{ it.remark || '' }}</td>
          </tr>
        </template>
        <template v-else>
          <tr
            v-for="(it, idx) in quote.items"
            :key="idx"
            :class="{ 'spec-row': !isProductHead(idx), 'product-head': isProductHead(idx), 'off-price': pickPrices && !pricedFlags[idx] }"
          >
            <td class="cell-no" data-label="#">{{ idx + 1 }}</td>
            <td v-if="template?.showSpecImage !== false" class="cell-img" data-label="图">
              <button
                v-if="interactive && it.imageUrl"
                type="button"
                class="thumb-btn"
                aria-label="查看规格图"
                @click="emitPreview(it.imageUrl)"
              >
                <img :src="it.imageUrl" class="thumb" alt="" crossorigin="anonymous" />
              </button>
              <img v-else-if="it.imageUrl" :src="it.imageUrl" class="thumb" alt="" crossorigin="anonymous" />
            </td>
            <td data-label="产品">
              <template v-if="isProductHead(idx)">
                <div class="name">{{ it.name }}</div>
                <div v-if="template?.showUpgrade !== false && it.upgradeNote" class="muted">升级：{{ it.upgradeNote }}</div>
                <div v-if="template?.showParams !== false && it.paramsText" class="muted">参数：{{ it.paramsText }}</div>
                <div v-if="it.remark" class="muted">备注：{{ it.remark }}</div>
              </template>
              <template v-else>
                <div class="spec-cont">└ 同产品规格</div>
                <div v-if="template?.showUpgrade !== false && it.upgradeNote" class="muted">升级：{{ it.upgradeNote }}</div>
                <div v-if="template?.showParams !== false && it.paramsText" class="muted">参数：{{ it.paramsText }}</div>
                <div v-if="it.remark" class="muted">备注：{{ it.remark }}</div>
              </template>
            </td>
            <td data-label="规格">
              <span class="spec-label">{{ it.specLabel || '—' }}</span>
            </td>
            <td data-label="数量">{{ it.qty }}{{ it.unit }}</td>
            <td v-if="showRetail" data-label="零售价">{{ money(it.retailPrice) }}</td>
            <td data-label="报价">{{ money(it.quotePrice) }}</td>
            <td class="line-total" :data-label="pickPrices ? '计价' : '小计'">
              <label v-if="pickPrices" class="pick">
                <input
                  type="radio"
                  :name="'spec-' + groupHead(idx)"
                  :checked="!!pricedFlags[idx]"
                  @change="choosePrice(idx)"
                />
                <span v-if="pricedFlags[idx]">{{ money(it.qty * it.quotePrice) }}</span>
                <span v-else class="skip">不计</span>
              </label>
              <template v-else>
                <span v-if="pricedFlags[idx]">{{ money(it.qty * it.quotePrice) }}</span>
                <span v-else class="skip">不计</span>
              </template>
            </td>
          </tr>
        </template>
        <tr v-if="!quote.items.length">
          <td colspan="9" class="empty">暂无明细</td>
        </tr>
      </tbody>
    </table>

    <section v-if="template?.showTotals !== false" class="totals">
      <div>商品小计：{{ money(displaySubtotal) }}</div>
      <div v-if="quote.discountAmt">折扣：-{{ money(quote.discountAmt) }}</div>
      <div v-if="quote.shippingAmt">运费：{{ money(quote.shippingAmt) }}</div>
      <div v-if="quote.taxAmt">税费：{{ money(quote.taxAmt) }}</div>
      <div class="grand">合计（{{ quote.currency || 'CNY' }}）：{{ money(displayTotal) }}</div>
      <div v-if="hasOptionalSpec" class="note">
        {{ pickPrices ? '同产品规格为单选，改选后合计自动更新。' : '同产品多规格默认只计第一个，其余不重复加总。' }}
      </div>
    </section>

    <section v-if="quote.remark" class="remark">整单备注：{{ quote.remark }}</section>
    <footer v-if="template?.footerText" class="foot">{{ template.footerText }}</footer>
  </div>
</template>

<style scoped>
.sheet {
  width: 794px;
  min-height: 500px;
  background: #fff;
  color: #1f2329;
  padding: 28px 32px;
  box-sizing: border-box;
  font-size: 12px;
  line-height: 1.5;
}
.head { display: flex; justify-content: space-between; gap: 16px; border-bottom: 2px solid #1f2329; padding-bottom: 12px; }
.brand { display: flex; gap: 12px; align-items: center; }
.logo { width: 56px; height: 56px; object-fit: contain; display: block; }
.shop { font-size: 18px; font-weight: 700; }
.sub { color: #646a73; }
.meta { text-align: right; }
.title { font-size: 20px; font-weight: 700; margin-bottom: 4px; }
.party { display: grid; grid-template-columns: 1fr 1fr; gap: 4px 16px; margin: 14px 0; }
.party .full { grid-column: 1 / -1; }
.items { width: 100%; border-collapse: collapse; table-layout: fixed; }
.items th, .items td { border: 1px solid #d0d3d6; padding: 6px 8px; vertical-align: top; }
.items th { background: #f5f6f7; text-align: left; }
.items .name { font-weight: 600; }
.items .muted, .spec-cont { color: #8f959e; font-size: 11px; margin-top: 2px; }
.items .spec-label { word-break: break-all; }
.items .thumb { width: 40px; height: 40px; object-fit: cover; display: block; border-radius: 2px; }
.thumb-btn {
  display: block;
  border: 0;
  padding: 0;
  margin: 0;
  background: transparent;
  cursor: zoom-in;
  line-height: 0;
}
.thumb-btn:focus-visible { outline: 2px solid #3d6b4f; outline-offset: 1px; }
.items .empty { text-align: center; color: #8f959e; }
.items tr.spec-row td { background: #fafbfc; }
.totals { margin-top: 12px; text-align: right; }
.totals .grand { font-size: 16px; font-weight: 700; margin-top: 4px; }
.totals .note { margin-top: 6px; color: #8f959e; font-size: 11px; font-weight: 400; }
.line-total .skip { color: #8f959e; }
.pick {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 28px;
  cursor: pointer;
}
.pick input { width: 18px; height: 18px; margin: 0; flex: 0 0 auto; accent-color: #1f2329; }
tr.off-price td { color: #8f959e; }
.remark { margin-top: 12px; color: #646a73; }
.foot { margin-top: 16px; padding-top: 10px; border-top: 1px dashed #d0d3d6; color: #8f959e; white-space: pre-wrap; }
</style>
