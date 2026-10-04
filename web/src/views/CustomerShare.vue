<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
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
const priceFlags = ref<boolean[]>([])

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
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="page">
    <div v-if="loading" class="state">正在打开报价单…</div>
    <div v-else-if="error" class="state error">{{ error }}</div>
    <template v-else>
      <p class="hint">同产品多个规格只能选一个，默认选中第一个。{{ hasImage ? '点击规格图片可查看大图。' : '' }}</p>
      <div class="sheet-scroll">
        <div class="sheet-wrap">
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
  overflow-x: auto;
}
.sheet-wrap {
  width: 794px;
  margin: 0 auto;
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
  background: rgba(0, 0, 0, 0.88);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 16px 16px;
  box-sizing: border-box;
}
.lightbox img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}
.close {
  position: absolute;
  top: 12px;
  right: 12px;
  border: 0;
  background: rgba(255, 255, 255, 0.16);
  color: #fff;
  border-radius: 999px;
  padding: 6px 14px;
  font-size: 14px;
  cursor: pointer;
}
</style>
