<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import QuoteSheet from '../components/QuoteSheet.vue'
import { applySecondEdit, fetchCustomerShare, openSecondEdit, saveCustomerPriced, type CustomerShareQuote } from '../api/publicShare'
import type { QuoteItem, QuoteTemplate } from '../api/quote'
import { defaultPricedFlags, sumPriced } from '../utils/quotePrice'

const route = useRoute()
const router = useRouter()
const token = computed(() => String(route.params.token || '').trim())

const loading = ref(true)
const error = ref('')
const data = ref<CustomerShareQuote | null>(null)
const previewUrl = ref('')
const showPreview = ref(false)
const SHEET_W = 794
const priceFlags = ref<boolean[]>([])
const savingFlags = ref(false)
const saveHint = ref('')
const secondEditUsed = ref(false)
const secondEditApproved = ref(false)
const isSecondEditShare = ref(false)
const applyOpen = ref(false)
const applying = ref(false)
const applicant = reactive({ name: '', phone: '', note: '' })
let flagsReady = false
let saveTimer: number | null = null
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
  flagsReady = false
  if (!q) {
    priceFlags.value = []
    secondEditUsed.value = false
    secondEditApproved.value = false
    isSecondEditShare.value = false
    return
  }
  isSecondEditShare.value = !!q.isSecondEdit
  secondEditUsed.value = !!q.secondEditUsed && !q.isSecondEdit
  secondEditApproved.value = !!q.secondEditApproved && !q.isSecondEdit
  const flags = q.pricedFlags && q.pricedFlags.length === quote.value.items.length
    ? q.pricedFlags.slice()
    : defaultPricedFlags(quote.value.items)
  priceFlags.value = flags
  nextTick(() => {
    flagsReady = true
  })
})

async function persistFlags(manual = false) {
  if (!token.value || !priceFlags.value.length) return
  savingFlags.value = true
  try {
    const saved = await saveCustomerPriced(token.value, priceFlags.value)
    isSecondEditShare.value = !!saved.isSecondEdit
    secondEditUsed.value = !!saved.secondEditUsed && !saved.isSecondEdit
    secondEditApproved.value = !!saved.secondEditApproved && !saved.isSecondEdit
    saveHint.value = '已保存勾选'
    if (manual) ElMessage.success('规格勾选已保存')
  } catch (e) {
    saveHint.value = '保存失败'
    if (manual) ElMessage.error((e as Error).message || '保存失败')
  } finally {
    savingFlags.value = false
  }
}

function scheduleSaveFlags() {
  if (!flagsReady) return
  saveHint.value = '正在保存…'
  if (saveTimer) window.clearTimeout(saveTimer)
  saveTimer = window.setTimeout(() => {
    void persistFlags(false)
  }, 400)
}

watch(priceFlags, () => scheduleSaveFlags(), { deep: true })

function openApply() {
  applyOpen.value = true
}

async function submitOpen() {
  if (!applicant.phone.trim()) {
    ElMessage.warning('请填写申请时的电话')
    return
  }
  applying.value = true
  try {
    const res = await openSecondEdit(token.value, {
      applicantPhone: applicant.phone.trim(),
      applicantName: applicant.name.trim() || undefined,
    })
    if (!res?.token) throw new Error('核验成功但未返回编辑链接')
    applyOpen.value = false
    ElMessage.success('核验通过，正在打开二次编辑')
    await router.push(`/revise/${res.token}`)
  } catch (e) {
    ElMessage.error((e as Error).message || '核验失败')
  } finally {
    applying.value = false
  }
}

async function submitApply() {
  if (!applicant.name.trim() || !applicant.phone.trim()) {
    ElMessage.warning('请填写申请人姓名和电话')
    return
  }
  applying.value = true
  try {
    await persistFlags(false)
    await applySecondEdit(token.value, {
      applicantName: applicant.name.trim(),
      applicantPhone: applicant.phone.trim(),
      applicantNote: applicant.note.trim(),
    })
    secondEditUsed.value = true
    secondEditApproved.value = false
    applyOpen.value = false
    ElMessage.success('申请已提交，等待后台审核。通过后可用本分享链接加申请电话打开编辑。')
  } catch (e) {
    ElMessage.error((e as Error).message || '申请失败')
    if (String((e as Error).message || '').includes('已用完')) secondEditUsed.value = true
  } finally {
    applying.value = false
  }
}

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
      <p class="hint">
        {{ isSecondEditShare ? '勾选规格后自动保存，合计会跟着变。双指放大可看清表格。' : '同产品多个规格只能选一个，勾选后自动保存。双指放大可看清表格。' }}
        {{ hasImage ? '点击规格图片查看大图。' : '' }}
      </p>
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
      <div class="live-total">
        <div class="live-actions">
          <button v-if="!isSecondEditShare" type="button" class="ghost" @click="openApply">
            {{ secondEditUsed ? (secondEditApproved ? '查看我的二次编辑单' : '二次编辑审核中') : '二次编辑分享' }}
          </button>
          <button type="button" class="save" :disabled="savingFlags" @click="persistFlags(true)">
            {{ savingFlags ? '保存中…' : '保存勾选' }}
          </button>
        </div>
        <div class="live-sum">
          <span v-if="saveHint" class="save-hint">{{ saveHint }}</span>
          合计 ¥{{ liveTotal.toFixed(2) }}
        </div>
      </div>
    </template>

    <div v-if="applyOpen" class="apply-mask" @click.self="applyOpen = false">
      <div class="apply-box">
        <template v-if="secondEditUsed">
          <h3>{{ secondEditApproved ? '查看二次编辑单' : '二次编辑审核中' }}</h3>
          <p v-if="secondEditApproved">请填写申请时的电话，核验通过后打开你的二次编辑报价单。</p>
          <p v-else>申请已提交，后台通过后可用申请电话打开编辑。现在核验会提示尚未通过。</p>
          <label>申请人姓名<input v-model="applicant.name" placeholder="可选，更准确" /></label>
          <label>申请人电话<input v-model="applicant.phone" placeholder="必填" /></label>
          <div class="apply-actions">
            <button type="button" class="ghost" @click="applyOpen = false">取消</button>
            <button type="button" class="save" :disabled="applying" @click="submitOpen">
              {{ applying ? '核验中…' : '打开二次编辑' }}
            </button>
          </div>
        </template>
        <template v-else>
          <h3>二次编辑分享申请</h3>
          <p>提交后需后台审核。通过后可用本分享链接加申请电话打开编辑，每个报价单只能申请一次。</p>
          <label>申请人姓名<input v-model="applicant.name" placeholder="必填" /></label>
          <label>申请人电话<input v-model="applicant.phone" placeholder="必填，回访时核验" /></label>
          <label>备注<input v-model="applicant.note" placeholder="可选" /></label>
          <div class="apply-actions">
            <button type="button" class="ghost" @click="applyOpen = false">取消</button>
            <button type="button" class="save" :disabled="applying" @click="submitApply">
              {{ applying ? '提交中…' : '提交申请并编辑' }}
            </button>
          </div>
        </template>
      </div>
    </div>

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
  padding: 10px 16px calc(10px + env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.96);
  border-top: 1px solid #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  color: #1f2329;
}
.live-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.live-sum { font-size: 18px; font-weight: 700; margin-left: auto; }
.save-hint { font-size: 12px; font-weight: 400; color: #67c23a; margin-right: 10px; }
button.save, button.ghost {
  border: 0;
  border-radius: 8px;
  padding: 8px 14px;
  font-size: 14px;
  cursor: pointer;
}
button.save { background: #1f6feb; color: #fff; }
button.ghost { background: #eef2f6; color: #303133; }
button.save:disabled, button.ghost:disabled { opacity: 0.5; cursor: not-allowed; }
.apply-mask {
  position: fixed;
  inset: 0;
  z-index: 30;
  background: rgba(16, 24, 40, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}
.apply-box {
  width: min(420px, 100%);
  background: #fff;
  border-radius: 12px;
  padding: 18px 16px;
}
.apply-box h3 { margin: 0 0 8px; font-size: 17px; }
.apply-box p { margin: 0 0 12px; color: #646a73; font-size: 13px; line-height: 1.5; }
.apply-box label { display: block; margin-bottom: 10px; font-size: 13px; color: #303133; }
.apply-box input {
  display: block;
  width: 100%;
  margin-top: 4px;
  box-sizing: border-box;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  padding: 8px 10px;
}
.apply-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 8px; }
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
