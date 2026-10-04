<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import QuoteSheet from '../components/QuoteSheet.vue'
import { fetchCustomerShare, type CustomerShareQuote } from '../api/publicShare'
import type { QuoteItem, QuoteTemplate } from '../api/quote'
import { defaultPricedFlags, sumPriced } from '../utils/quotePrice'

const route = useRoute()
const token = computed(() => String(route.params.token || '').trim())

const loading = ref(true)
const error = ref('')
const data = ref<CustomerShareQuote | null>(null)
const previewUrl = ref('')
const showPreview = ref(false)
const SHEET_W = 794
const priceFlags = ref<boolean[]>([])
const pageEl = ref<HTMLElement | null>(null)
const sheetEl = ref<HTMLElement | null>(null)
const scale = ref(Math.min(1, Math.max(280, window.innerWidth - 32) / SHEET_W))
const sheetH = ref(640)

const fitStyle = computed(() => ({
  width: `${Math.round(SHEET_W * scale.value)}px`,
  height: `${Math.round(sheetH.value * scale.value)}px`,
}))

const template = computed<QuoteTemplate | null>(() => {
  const t = data.value?.template
  if (!t) return null
  return {
    id: 0,
    name: t.shopName || '默认版式',
    kind: 'layout',
    isDefault: true,
    logoUrl: t.logoUrl || '',
    shopName: t.shopName || '报价中心',
    shopPhone: t.shopPhone || '',
    shopAddress: t.shopAddress || '',
    headerSubtitle: t.headerSubtitle || '',
    footerText: t.footerText || '',
    showLogo: t.showLogo !== false,
    showRetailPrice: t.showRetailPrice !== false,
    showSpecImage: t.showSpecImage !== false,
    showUpgrade: t.showUpgrade !== false,
    showParams: t.showParams !== false,
    showTotals: t.showTotals !== false,
    stylePreset: t.stylePreset || 'compare',
    lines: [],
  }
})

const quote = computed(() => {
  const q = data.value
  const items: QuoteItem[] = (q?.items || []).map((it, i) => ({
    sort: it.sort || (i + 1) * 10,
    source: 'manual',
    productId: it.productId ?? null,
    category: it.category || '',
    partName: it.partName || '',
    name: it.name || '',
    specLabel: it.specLabel || '',
    imageUrl: it.imageUrl || '',
    qty: Number(it.qty || 0),
    unit: it.unit || '件',
    retailPrice: Number(it.retailPrice || 0),
    costPrice: 0,
    quotePrice: Number(it.quotePrice || 0),
    upgradeNote: it.upgradeNote || '',
    paramsText: it.paramsText || '',
    remark: it.remark || '',
  }))
  return {
    quoteNo: q?.quoteNo || '',
    title: q?.title || '报价单',
    customerName: q?.customerName || '',
    contactName: q?.contactName || '',
    contactPhone: q?.contactPhone || '',
    currency: q?.currency || 'CNY',
    validUntil: q?.validUntil || null,
    remark: q?.remark || '',
    discountAmt: Number(q?.discountAmt || 0),
    shippingAmt: Number(q?.shippingAmt || 0),
    taxAmt: Number(q?.taxAmt || 0),
    subtotalAmt: Number(q?.subtotalAmt || 0),
    totalAmt: Number(q?.totalAmt || 0),
    items,
  }
})

const hasImage = computed(() =>
  quote.value.items.some((it) => !!(it.imageUrl || '').trim()) && template.value?.showSpecImage !== false,
)

const liveTotal = computed(() => {
  const sub = sumPriced(quote.value.items, priceFlags.value.length ? priceFlags.value : defaultPricedFlags(quote.value.items), (it) =>
    Number(it.qty || 0) * Number(it.quotePrice || 0),
  )
  return (
    Math.round(
      (sub - Number(quote.value.discountAmt || 0) + Number(quote.value.shippingAmt || 0) + Number(quote.value.taxAmt || 0)) * 100,
    ) / 100
  )
})

watch(data, (q) => {
  if (!q) {
    priceFlags.value = []
    return
  }
  priceFlags.value = defaultPricedFlags(quote.value.items)
})

function openPreview(url: string) {
  const u = (url || '').trim()
  if (!u) return
  previewUrl.value = u
  showPreview.value = true
}

function closePreview() {
  showPreview.value = false
  previewUrl.value = ''
}

function measure() {
  const avail = Math.max(280, (pageEl.value?.clientWidth || window.innerWidth) - 16)
  scale.value = Math.min(1, avail / SHEET_W)
  const h = sheetEl.value?.offsetHeight
  if (h && h > 0) sheetH.value = h
}

let ro: ResizeObserver | null = null

async function load() {
  if (!token.value) {
    error.value = '链接无效'
    loading.value = false
    return
  }
  loading.value = true
  error.value = ''
  try {
    data.value = await fetchCustomerShare(token.value)
    const shop = (data.value.template?.shopName || '').trim() || '报价单'
    document.title = `${shop} · ${data.value.title || '报价单'}`
  } catch (e) {
    error.value = (e as Error).message || '加载失败'
    data.value = null
  } finally {
    loading.value = false
    await nextTick()
    measure()
  }
}

onMounted(() => {
  document.querySelector('meta[name="viewport"]')?.setAttribute(
    'content',
    'width=device-width, initial-scale=1, maximum-scale=5, user-scalable=yes',
  )
  ro = new ResizeObserver(() => measure())
  if (pageEl.value) ro.observe(pageEl.value)
  window.addEventListener('resize', measure)
  void load()
})

watch(sheetEl, (el, prev) => {
  if (prev && ro) ro.unobserve(prev)
  if (el && ro) ro.observe(el)
  measure()
})

onBeforeUnmount(() => {
  ro?.disconnect()
  window.removeEventListener('resize', measure)
  document.querySelector('meta[name="viewport"]')?.setAttribute('content', 'width=device-width, initial-scale=1.0')
})
</script>

<template>
  <div ref="pageEl" class="page">
    <div v-if="loading" class="state">正在打开报价单…</div>
    <div v-else-if="error" class="state error">{{ error }}</div>
    <template v-else>
      <p class="hint">同产品多个规格只能选一个，默认选中第一个。双指放大可看清表格。{{ hasImage ? '点击规格图片查看大图。' : '' }}</p>
      <div class="sheet-scroll">
        <div class="fit" :style="fitStyle">
          <div ref="sheetEl" class="sheet-wrap" :style="{ transform: `scale(${scale})` }">
            <QuoteSheet
              v-model:price-flags="priceFlags"
              :quote="quote"
              :template="template"
              interactive
              pick-prices
              @preview-image="openPreview"
            />
          </div>
        </div>
      </div>
      <div class="live-total">合计 ¥{{ liveTotal.toFixed(2) }}</div>
    </template>

    <teleport to="body">
      <div v-if="showPreview" class="lightbox" @click="closePreview">
        <button type="button" class="close" @click="closePreview">关闭</button>
        <img :src="previewUrl" alt="规格图" @click.stop />
      </div>
    </teleport>
  </div>
</template>

<style scoped>
.page {
  height: 100%;
  overflow: auto;
  background: #eef0f3;
  padding: 16px 8px 72px;
  box-sizing: border-box;
}
.sheet-scroll {
  display: flex;
  justify-content: center;
}
.fit {
  margin: 0 auto;
}
.sheet-wrap {
  width: 794px;
  transform-origin: top left;
  box-shadow: 0 8px 28px rgba(16, 24, 40, 0.08);
}
.hint {
  margin: 0 auto 10px;
  max-width: 794px;
  text-align: center;
  color: #646a73;
  font-size: 13px;
  line-height: 1.5;
}
.live-total {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 20;
  padding: 12px 16px calc(12px + env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.96);
  border-top: 1px solid #e5e7eb;
  text-align: right;
  font-size: 18px;
  font-weight: 700;
  color: #1f2329;
}
.state {
  min-height: 40vh;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #646a73;
  font-size: 15px;
}
.state.error {
  color: #b42318;
}
.lightbox {
  position: fixed;
  inset: 0;
  z-index: 4000;
  width: 100vw;
  height: 100dvh;
  background: rgba(0, 0, 0, 0.92);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 56px 12px 16px;
  box-sizing: border-box;
}
.lightbox img {
  max-width: 100%;
  max-height: 100%;
  width: auto;
  height: auto;
  object-fit: contain;
}
.close {
  position: absolute;
  top: max(12px, env(safe-area-inset-top));
  right: 12px;
  border: 0;
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
  border-radius: 999px;
  padding: 8px 16px;
  font-size: 15px;
  cursor: pointer;
}
</style>
