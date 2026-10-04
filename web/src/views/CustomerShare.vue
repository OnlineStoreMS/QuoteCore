<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import QuoteSheet from '../components/QuoteSheet.vue'
import { fetchCustomerShare, type CustomerShareQuote } from '../api/publicShare'
import type { QuoteItem, QuoteTemplate } from '../api/quote'

const SHEET_W = 794

const route = useRoute()
const token = computed(() => String(route.params.token || '').trim())

const loading = ref(true)
const error = ref('')
const data = ref<CustomerShareQuote | null>(null)
const previewUrl = ref('')
const showPreview = ref(false)
const viewportEl = ref<HTMLElement | null>(null)
const sheetWrap = ref<HTMLElement | null>(null)
const scale = ref(1)
const sheetH = ref(640)

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

const fitStyle = computed(() => ({
  width: `${Math.round(SHEET_W * scale.value)}px`,
  height: `${Math.round(sheetH.value * scale.value)}px`,
}))

function measure() {
  const vw = viewportEl.value?.clientWidth || window.innerWidth
  const avail = Math.max(280, vw - 8)
  scale.value = Math.min(1, avail / SHEET_W)
  const h = sheetWrap.value?.offsetHeight
  if (h && h > 0) sheetH.value = h
}

let ro: ResizeObserver | null = null

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
    await nextTick()
    measure()
  }
}

onMounted(() => {
  ro = new ResizeObserver(() => measure())
  if (viewportEl.value) ro.observe(viewportEl.value)
  window.addEventListener('resize', measure)
  void load()
})

watch(sheetWrap, (el, prev) => {
  if (prev && ro) ro.unobserve(prev)
  if (el && ro) ro.observe(el)
  measure()
})

onBeforeUnmount(() => {
  ro?.disconnect()
  window.removeEventListener('resize', measure)
})
</script>

<template>
  <div ref="viewportEl" class="page">
    <div v-if="loading" class="state">正在打开报价单…</div>
    <div v-else-if="error" class="state error">{{ error }}</div>
    <template v-else>
      <p v-if="hasImage" class="hint">点击规格图片可查看大图</p>
      <div class="fit" :style="fitStyle">
        <div ref="sheetWrap" class="sheet-wrap" :style="{ transform: `scale(${scale})` }">
          <QuoteSheet
            :quote="quote"
            :template="template"
            interactive
            @preview-image="openPreview"
          />
        </div>
      </div>
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
  padding: 16px 8px 32px;
  box-sizing: border-box;
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
  text-align: center;
  color: #646a73;
  font-size: 13px;
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
