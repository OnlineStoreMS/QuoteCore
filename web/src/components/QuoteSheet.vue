<script setup lang="ts">
import type { QuoteItem, QuoteTemplate } from '../api/quote'

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
}>()

function money(v: number) {
  return Number(v || 0).toFixed(2)
}

function sameProduct(a: QuoteItem, b: QuoteItem): boolean {
  if (a.productId && b.productId) return Number(a.productId) === Number(b.productId)
  return (a.name || '').trim() !== '' && (a.name || '').trim() === (b.name || '').trim()
}

function isProductHead(idx: number): boolean {
  if (idx <= 0) return true
  return !sameProduct(props.quote.items[idx], props.quote.items[idx - 1])
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
        <tr>
          <th style="width:36px">#</th>
          <th v-if="template?.showSpecImage !== false" style="width:56px">图</th>
          <th>产品</th>
          <th style="width:140px">规格</th>
          <th style="width:56px">数量</th>
          <th v-if="template?.showRetailPrice !== false" style="width:72px">零售价</th>
          <th style="width:72px">报价</th>
          <th style="width:80px">小计</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(it, idx) in quote.items"
          :key="idx"
          :class="{ 'spec-row': !isProductHead(idx), 'product-head': isProductHead(idx) }"
        >
          <td>{{ idx + 1 }}</td>
          <td v-if="template?.showSpecImage !== false">
            <img v-if="it.imageUrl" :src="it.imageUrl" class="thumb" alt="" crossorigin="anonymous" />
          </td>
          <td>
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
          <td>
            <span class="spec-label">{{ it.specLabel || '—' }}</span>
          </td>
          <td>{{ it.qty }}{{ it.unit }}</td>
          <td v-if="template?.showRetailPrice !== false">{{ money(it.retailPrice) }}</td>
          <td>{{ money(it.quotePrice) }}</td>
          <td>{{ money(it.qty * it.quotePrice) }}</td>
        </tr>
        <tr v-if="!quote.items.length">
          <td :colspan="template?.showRetailPrice === false && template?.showSpecImage === false ? 5 : 8" class="empty">暂无明细</td>
        </tr>
      </tbody>
    </table>

    <section v-if="template?.showTotals !== false" class="totals">
      <div>商品小计：¥{{ money(quote.subtotalAmt) }}</div>
      <div v-if="quote.discountAmt">折扣：-¥{{ money(quote.discountAmt) }}</div>
      <div v-if="quote.shippingAmt">运费：¥{{ money(quote.shippingAmt) }}</div>
      <div v-if="quote.taxAmt">税费：¥{{ money(quote.taxAmt) }}</div>
      <div class="grand">合计（{{ quote.currency || 'CNY' }}）：¥{{ money(quote.totalAmt) }}</div>
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
.thumb { width: 40px; height: 40px; max-width: 40px; max-height: 40px; object-fit: cover; border-radius: 4px; display: block; }
.name { font-weight: 600; }
.spec-cont { color: #8f959e; font-size: 11px; }
.spec-row td { background: #fafbfc; }
.spec-row .spec-label { font-weight: 600; }
.muted { color: #8f959e; font-size: 11px; margin-top: 2px; }
.empty { text-align: center; color: #8f959e; padding: 24px !important; }
.totals { margin-top: 14px; text-align: right; }
.grand { font-size: 16px; font-weight: 700; margin-top: 4px; }
.remark { margin-top: 12px; color: #4e5969; }
.foot { margin-top: 18px; padding-top: 10px; border-top: 1px dashed #d0d3d6; color: #8f959e; white-space: pre-wrap; }
</style>
