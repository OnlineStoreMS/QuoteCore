<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import QuoteSheet from '../components/QuoteSheet.vue'
import QuoteItemsEditor from '../components/QuoteItemsEditor.vue'
import {
  effectiveCostPrice,
  ensureShareToken,
  getQuote,
  isSkeletonTemplate,
  listTemplates,
  saveQuote,
  searchCustomers,
  searchProductSkus,
  seedItemsFromTemplate,
  type CustomerHit,
  type QuoteItem,
  type QuoteTemplate,
  type SkuHit,
} from '../api/quote'
import { copyElementAsImage, downloadElementAsPdf, downloadElementAsPng, waitForImages } from '../utils/exportQuote'

const route = useRoute()
const router = useRouter()
const quoteId = computed(() => {
  const id = Number(route.params.id)
  return Number.isFinite(id) && id > 0 ? id : 0
})

const loading = ref(false)
const saving = ref(false)
const autoSaveReady = ref(false)
const autoSaveStatus = ref<'idle' | 'pending' | 'saving' | 'saved' | 'error'>('idle')
const lastSavedAt = ref('')
let persistLock = false
/** 用户有未落库的改动 */
let formDirty = false
let dirtyGen = 0
/** 保存进行中又改了，结束后再存一次 */
let saveAgain = false
/** 应用服务端回写时跳过 watch */
let suppressAutoSave = false
/** 中文输入法选词中，失焦暂不保存 */
let imeComposing = false
const templates = ref<QuoteTemplate[]>([])
const activeTemplate = ref<QuoteTemplate | null>(null)
const previewRef = ref<HTMLElement | null>(null)
const exportRef = ref<HTMLElement | null>(null)
const showPreview = ref(false)
const previewShowRetailPrice = ref(true)
const imagePreviewUrl = ref('')
const showImagePreview = ref(false)
const itemsZoomed = ref(false)
const exporting = ref(false)

/** 预览/导出用的版式（可临时关闭零售价等） */
const sheetTemplate = computed<QuoteTemplate | null>(() => {
  const base = activeTemplate.value
  if (!base) {
    return {
      id: 0,
      name: '',
      isDefault: false,
      logoUrl: '',
      shopName: '报价中心',
      shopPhone: '',
      shopAddress: '',
      headerSubtitle: '',
      footerText: '',
      showLogo: true,
      showRetailPrice: previewShowRetailPrice.value,
      showSpecImage: true,
      showUpgrade: true,
      showParams: true,
      showTotals: true,
      stylePreset: 'compare',
    }
  }
  return { ...base, showRetailPrice: previewShowRetailPrice.value }
})

watch(
  activeTemplate,
  (t) => {
    previewShowRetailPrice.value = t?.showRetailPrice !== false
  },
  { immediate: true },
)

function openImagePreview(url?: string) {
  const u = (url || '').trim()
  if (!u) return
  imagePreviewUrl.value = u
  showImagePreview.value = true
}
const skuDrawer = ref(false)
const custDrawer = ref(false)
const skuKeyword = ref('')
const custKeyword = ref('')
const skuHits = ref<SkuHit[]>([])
const custHits = ref<CustomerHit[]>([])
const searching = ref(false)

const form = reactive({
  title: '报价单',
  status: 1,
  customerId: null as number | null,
  customerName: '',
  contactName: '',
  contactPhone: '',
  currency: 'CNY',
  validUntil: '' as string,
  remark: '',
  discountAmt: 0,
  shippingAmt: 0,
  taxAmt: 0,
  templateId: null as number | null,
  quoteNo: '',
  items: [] as QuoteItem[],
})

const subtotal = computed(() =>
  form.items.reduce((s, it) => s + Number(it.qty || 0) * Number(it.quotePrice || 0), 0),
)
const retailTotal = computed(() =>
  Math.round(form.items.reduce((s, it) => s + Number(it.qty || 0) * Number(it.retailPrice || 0), 0) * 100) / 100,
)
const costTotal = computed(() =>
  Math.round(
    form.items.reduce((s, it) => s + Number(it.qty || 0) * effectiveCostPrice(it), 0) * 100,
  ) / 100,
)
const total = computed(() =>
  Math.round((subtotal.value - Number(form.discountAmt || 0) + Number(form.shippingAmt || 0) + Number(form.taxAmt || 0)) * 100) / 100,
)
const profit = computed(() =>
  Math.round((total.value - Number(form.shippingAmt || 0) - costTotal.value) * 100) / 100,
)

function emptyItem(partial?: Partial<QuoteItem>): QuoteItem {
  return {
    sort: (form.items.length + 1) * 10,
    source: 'manual',
    category: '',
    partName: '',
    name: '',
    specLabel: '',
    imageUrl: '',
    qty: 1,
    unit: '件',
    retailPrice: 0,
    costPrice: 0,
    quotePrice: 0,
    upgradeNote: '',
    paramsText: '',
    remark: '',
    ...partial,
  }
}

const previewQuote = computed(() => ({
  quoteNo: form.quoteNo,
  title: form.title,
  customerName: form.customerName,
  contactName: form.contactName,
  contactPhone: form.contactPhone,
  currency: form.currency,
  validUntil: form.validUntil || null,
  remark: form.remark,
  discountAmt: form.discountAmt,
  shippingAmt: form.shippingAmt,
  taxAmt: form.taxAmt,
  subtotalAmt: Math.round(subtotal.value * 100) / 100,
  totalAmt: total.value,
  items: form.items,
}))

const selectedBizId = ref<number | null>(null)

const skeletonMode = computed(() => {
  if (selectedBizId.value) return true
  return form.items.some((it) => !!(it.partName || '').trim() || !!(it.category || '').trim())
})

const layoutTemplates = computed(() => templates.value.filter((t) => !isSkeletonTemplate(t)))
const bizTemplates = computed(() => templates.value.filter((t) => isSkeletonTemplate(t)))

function pickDefaultLayout(): QuoteTemplate | null {
  return layoutTemplates.value.find((t) => t.isDefault) || layoutTemplates.value[0] || null
}

function useLayoutTemplate(tpl: QuoteTemplate | null) {
  if (!tpl) {
    activeTemplate.value = null
    form.templateId = null
    return
  }
  form.templateId = tpl.id
  activeTemplate.value = tpl
}

function itemsAreBlank(): boolean {
  if (!form.items.length) return true
  return form.items.every(
    (it) =>
      !(it.name || '').trim() &&
      !(it.specLabel || '').trim() &&
      !(it.partName || '').trim() &&
      !Number(it.quotePrice || 0) &&
      !Number(it.retailPrice || 0) &&
      !Number(it.costPrice || 0),
  )
}

async function applySkeleton(force = false) {
  const tpl = selectedBizId.value
    ? bizTemplates.value.find((t) => t.id === selectedBizId.value)
    : undefined
  if (!tpl || !isSkeletonTemplate(tpl)) {
    ElMessage.warning('请先选择业务模板')
    return
  }
  if (!force && !itemsAreBlank()) {
    try {
      await ElMessageBox.confirm('将用业务模板覆盖当前明细，未保存内容会丢失。继续？', '套用业务模板', {
        type: 'warning',
      })
    } catch {
      return
    }
  }
  selectedBizId.value = tpl.id
  form.items = seedItemsFromTemplate(tpl)
  if (!form.templateId || isSkeletonTemplate(activeTemplate.value)) {
    useLayoutTemplate(pickDefaultLayout())
  }
  ElMessage.success(`已套用业务模板（${form.items.length} 行）`)
}

function findLastSameProductIndex(productId?: number | null, name?: string): number {
  for (let i = form.items.length - 1; i >= 0; i--) {
    const it = form.items[i]
    if (productId && it.productId && Number(it.productId) === Number(productId)) return i
    if (!productId && name && (it.name || '').trim() === name.trim()) return i
  }
  return -1
}

function renumberSort() {
  form.items.forEach((it, i) => {
    it.sort = (i + 1) * 10
  })
}

function addManualRow() {
  form.items.push(emptyItem())
  renumberSort()
}

async function loadTemplates() {
  templates.value = await listTemplates()
  useLayoutTemplate(pickDefaultLayout())
}

watch(
  () => form.templateId,
  (id) => {
    const tpl = layoutTemplates.value.find((t) => t.id === id) || null
    // 禁止选中业务模板作为版式
    if (id && !tpl) {
      useLayoutTemplate(pickDefaultLayout())
      return
    }
    activeTemplate.value = tpl
  },
)

async function loadQuote() {
  if (!quoteId.value) {
    const bizTemplateId = Number(route.query.bizTemplateId || 0)
    if (bizTemplateId > 0) {
      const biz = bizTemplates.value.find((t) => t.id === bizTemplateId)
      if (biz) {
        selectedBizId.value = biz.id
        if (!form.title || form.title === '报价单') {
          form.title = biz.name || '报价单'
        }
        await applySkeleton(true)
        router.replace({ path: '/quotes/new' })
        return
      }
    }
    addManualRow()
    return
  }
  loading.value = true
  try {
    const q = await getQuote(quoteId.value)
    form.title = q.title
    form.status = q.status
    form.customerId = q.customerId || null
    form.customerName = q.customerName || ''
    form.contactName = q.contactName || ''
    form.contactPhone = q.contactPhone || ''
    form.currency = q.currency || 'CNY'
    form.validUntil = q.validUntil ? String(q.validUntil).slice(0, 10) : ''
    form.remark = q.remark || ''
    form.discountAmt = q.discountAmt || 0
    form.shippingAmt = q.shippingAmt || 0
    form.taxAmt = q.taxAmt || 0
    form.quoteNo = q.quoteNo || ''
    form.items = (q.items || []).map((it, i) => ({
      id: it.id,
      sort: it.sort || (i + 1) * 10,
      source: it.source || 'manual',
      productId: it.productId,
      skuId: it.skuId,
      templateLineId: it.templateLineId,
      category: it.category || '',
      partName: it.partName || '',
      name: it.name,
      specLabel: it.specLabel || '',
      imageUrl: it.imageUrl || '',
      qty: it.qty || 1,
      unit: it.unit || '件',
      retailPrice: it.retailPrice || 0,
      costPrice: it.costPrice || 0,
      supplyPrice: it.supplyPrice || 0,
      supplyPriceAt: it.supplyPriceAt || null,
      supplyRemark: it.supplyRemark || '',
      quotePrice: it.quotePrice || 0,
      upgradeNote: it.upgradeNote || '',
      paramsText: it.paramsText || '',
      remark: it.remark || '',
    }))
    // 版式统一用版式模板：历史若绑了业务模板，改回默认版式
    const bound = q.templateId ? templates.value.find((t) => t.id === q.templateId) : null
    if (bound && !isSkeletonTemplate(bound)) {
      useLayoutTemplate(bound)
    } else if (q.templateSnap) {
      try {
        const snap = JSON.parse(q.templateSnap) as QuoteTemplate
        if (!isSkeletonTemplate(snap) && snap.id) {
          const live = layoutTemplates.value.find((t) => t.id === snap.id)
          useLayoutTemplate(live || snap)
        } else {
          useLayoutTemplate(pickDefaultLayout())
        }
      } catch {
        useLayoutTemplate(pickDefaultLayout())
      }
    } else {
      useLayoutTemplate(pickDefaultLayout())
    }
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

function canPersistItems(): boolean {
  if (!form.items.length) return false
  return form.items.every(
    (it) =>
      !!(it.name || '').trim() || !!(it.category || '').trim() || !!(it.partName || '').trim(),
  )
}

function buildSavePayload() {
  return {
    title: form.title,
    status: form.status,
    customerId: form.customerId,
    customerName: form.customerName,
    contactName: form.contactName,
    contactPhone: form.contactPhone,
    currency: form.currency,
    validUntil: form.validUntil || null,
    remark: form.remark,
    discountAmt: form.discountAmt,
    shippingAmt: form.shippingAmt,
    taxAmt: form.taxAmt,
    templateId: form.templateId,
    items: form.items.map((it, i) => ({ ...it, sort: (i + 1) * 10 })),
  }
}

function mapSavedItem(it: QuoteItem, i: number): QuoteItem {
  return {
    id: it.id,
    sort: it.sort || (i + 1) * 10,
    source: it.source || 'manual',
    productId: it.productId,
    skuId: it.skuId,
    templateLineId: it.templateLineId,
    category: it.category || '',
    partName: it.partName || '',
    name: it.name,
    specLabel: it.specLabel || '',
    imageUrl: it.imageUrl || '',
    qty: it.qty || 1,
    unit: it.unit || '件',
    retailPrice: it.retailPrice || 0,
    costPrice: it.costPrice || 0,
    supplyPrice: it.supplyPrice || 0,
    supplyPriceAt: it.supplyPriceAt || null,
    supplyRemark: it.supplyRemark || '',
    quotePrice: it.quotePrice || 0,
    upgradeNote: it.upgradeNote || '',
    paramsText: it.paramsText || '',
    remark: it.remark || '',
  }
}

/** 静默保存只回填 id / 拿货价，不整表替换，避免打断正在输入 */
function patchItemsFromSaved(savedItems: QuoteItem[]) {
  suppressAutoSave = true
  try {
    const n = Math.min(form.items.length, savedItems.length)
    for (let i = 0; i < n; i++) {
      const cur = form.items[i]
      const sv = savedItems[i]
      if (sv.id && cur.id !== sv.id) cur.id = sv.id
      if (sv.supplyPriceAt) {
        cur.supplyPrice = sv.supplyPrice || 0
        cur.supplyPriceAt = sv.supplyPriceAt
        cur.supplyRemark = sv.supplyRemark || ''
      }
    }
  } finally {
    nextTick(() => {
      suppressAutoSave = false
    })
  }
}

async function onSave(silent = false) {
  if (silent && imeComposing) return
  if (!form.items.length) {
    if (!silent) ElMessage.warning('请至少添加一行明细')
    return
  }
  if (!canPersistItems()) {
    if (!silent) ElMessage.warning('每行明细需填写名称或配件')
    return
  }
  if (persistLock || saving.value) {
    if (silent) saveAgain = true
    return
  }
  persistLock = true
  saving.value = true
  if (silent) autoSaveStatus.value = 'saving'
  const saveGen = dirtyGen
  try {
    const wasNew = !quoteId.value
    const saved = await saveQuote(buildSavePayload(), quoteId.value || undefined)
    form.quoteNo = saved.quoteNo
    if (saved.items?.length) {
      if (silent && !wasNew) {
        patchItemsFromSaved(saved.items)
      } else {
        suppressAutoSave = true
        form.items = saved.items.map((it, i) => mapSavedItem(it, i))
        await nextTick()
        suppressAutoSave = false
      }
    }
    if (dirtyGen === saveGen) {
      formDirty = false
      autoSaveStatus.value = 'saved'
      lastSavedAt.value = new Date().toLocaleTimeString()
    } else {
      formDirty = true
      autoSaveStatus.value = 'pending'
    }
    if (!silent) ElMessage.success('已保存')
    if (wasNew) {
      autoSaveReady.value = false
      await router.replace(`/quotes/${saved.id}`)
      await nextTick()
      autoSaveReady.value = true
    }
  } catch (e) {
    autoSaveStatus.value = 'error'
    ElMessage.error(silent ? `自动保存失败：${(e as Error).message}` : (e as Error).message)
  } finally {
    saving.value = false
    persistLock = false
    if (saveAgain && formDirty) {
      saveAgain = false
      void onSave(true)
    } else {
      saveAgain = false
    }
  }
}

function onEditorCompositionStart() {
  imeComposing = true
}

function onEditorCompositionEnd() {
  imeComposing = false
}

/** 离开输入框后自动保存（不做定时轮询） */
function onEditorFocusOut() {
  if (!formDirty || !autoSaveReady.value || loading.value || suppressAutoSave) return
  queueMicrotask(() => {
    if (imeComposing || !formDirty || !autoSaveReady.value || loading.value) return
    void onSave(true)
  })
}

const autoSaveHint = computed(() => {
  if (autoSaveStatus.value === 'pending') return '有改动，离开输入框后自动保存'
  if (autoSaveStatus.value === 'saving') return '自动保存中…'
  if (autoSaveStatus.value === 'saved') return lastSavedAt.value ? `已自动保存 ${lastSavedAt.value}` : '已自动保存'
  if (autoSaveStatus.value === 'error') return '自动保存失败'
  return ''
})

watch(
  () => form,
  () => {
    if (!autoSaveReady.value || loading.value || suppressAutoSave) return
    formDirty = true
    dirtyGen += 1
    autoSaveStatus.value = 'pending'
  },
  { deep: true },
)

async function searchSku() {
  if (!skuKeyword.value.trim()) return
  searching.value = true
  try {
    const data = await searchProductSkus(skuKeyword.value.trim())
    skuHits.value = data.list || []
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    searching.value = false
  }
}

async function searchCust() {
  searching.value = true
  try {
    const data = await searchCustomers(custKeyword.value.trim())
    custHits.value = data.list || []
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    searching.value = false
  }
}

function pickCustomer(c: CustomerHit) {
  form.customerId = c.id
  form.customerName = c.displayName || ''
  form.contactName = c.displayName || ''
  form.contactPhone = c.primaryPhone || ''
  custDrawer.value = false
}

function pickSku(s: SkuHit) {
  const existingIdx = findLastSameProductIndex(s.productId)
  const head = existingIdx >= 0 ? form.items[existingIdx] : null
  // 同一商品再次选取：沿用首次产品名，只追加规格行
  const row = emptyItem({
    source: 'product',
    productId: s.productId,
    skuId: s.skuId,
    name: head?.name || s.productName,
    specLabel: s.specLabel || s.skuCode || '',
    imageUrl: s.pic || s.productPic || '',
    retailPrice: s.price || 0,
    quotePrice: s.price || 0,
    paramsText: head?.paramsText || (s.brandName ? `品牌：${s.brandName}` : ''),
  })
  if (existingIdx >= 0) {
    form.items.splice(existingIdx + 1, 0, row)
  } else {
    form.items.push(row)
  }
  renumberSort()
  ElMessage.success(head ? `已追加规格：${row.specLabel || '—'}` : '已加入明细（可继续点选多规格）')
}

async function readyExportEl() {
  await nextTick()
  const el = exportRef.value
  if (!el) throw new Error('导出区域未就绪')
  await waitForImages(el)
  return el
}

async function doCopyImage() {
  if (exporting.value) return
  exporting.value = true
  try {
    const el = await readyExportEl()
    const mode = await copyElementAsImage(el, form.quoteNo || `quote-${Date.now()}`)
    if (mode === 'clipboard') {
      ElMessage.success('已复制图片到剪贴板，可直接粘贴')
    } else {
      ElMessage.warning('浏览器限制无法写入剪贴板，已改为下载 PNG')
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '复制失败')
  } finally {
    exporting.value = false
  }
}

async function doDownloadPng() {
  if (exporting.value) return
  exporting.value = true
  try {
    const el = await readyExportEl()
    await downloadElementAsPng(el, form.quoteNo || `quote-${Date.now()}`)
    ElMessage.success('PNG 已下载')
  } catch (e) {
    ElMessage.error((e as Error).message || '下载失败')
  } finally {
    exporting.value = false
  }
}

async function doDownloadPdf() {
  if (exporting.value) return
  exporting.value = true
  try {
    const el = await readyExportEl()
    await downloadElementAsPdf(el, form.quoteNo || `quote-${Date.now()}`)
    ElMessage.success('PDF 已下载')
  } catch (e) {
    ElMessage.error((e as Error).message || '导出失败')
  } finally {
    exporting.value = false
  }
}

const sharing = ref(false)

async function shareToSupplier() {
  if (!quoteId.value) {
    ElMessage.warning('请先保存报价单后再分享')
    return
  }
  sharing.value = true
  try {
    // 先保存，确保明细 id 稳定，供货商提交能对上
    if (canPersistItems()) {
      await onSave(true)
    }
    const data = await ensureShareToken(quoteId.value)
    const base = (import.meta.env.BASE_URL || '/').replace(/\/?$/, '/')
    const url = `${window.location.origin}${base}share/${data.shareToken}`
    const title = (form.title || '').trim() || '报价单'
    const quoteNo = (form.quoteNo || data.quoteNo || '').trim()
    const text = [
      '您好，',
      `麻烦看一下报价单「${title}」${quoteNo ? `（${quoteNo}）` : ''}的拿货价。`,
      url,
    ].join('\n')
    try {
      await navigator.clipboard.writeText(text)
      ElMessage.success('分享文案已复制，可直接发给供货商')
    } catch {
      await ElMessageBox.alert(text, '请复制分享内容', {
        confirmButtonText: '知道了',
        customClass: 'share-copy-box',
      })
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '生成分享链接失败')
  } finally {
    sharing.value = false
  }
}

onMounted(async () => {
  autoSaveReady.value = false
  try {
    await loadTemplates()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
  await loadQuote()
  await nextTick()
  // 等初始赋值完成后再开启自动保存，避免一进页就触发
  setTimeout(() => {
    autoSaveReady.value = true
  }, 400)
})

onBeforeUnmount(() => {
  autoSaveReady.value = false
})
</script>

<template>
  <div
    v-loading="loading"
    class="edit"
    @compositionstart.capture="onEditorCompositionStart"
    @compositionend.capture="onEditorCompositionEnd"
    @focusout.capture="onEditorFocusOut"
  >
    <div class="topbar">
      <div class="left">
        <el-button @click="router.push('/quotes')">返回</el-button>
        <strong>{{ quoteId ? `编辑 ${form.quoteNo}` : '新建报价' }}</strong>
        <span v-if="autoSaveHint" class="autosave-hint" :class="autoSaveStatus">{{ autoSaveHint }}</span>
      </div>
      <div class="right">
        <el-button :loading="sharing" :disabled="!quoteId" @click="shareToSupplier">分享给供货商</el-button>
        <el-button :loading="exporting" @click="showPreview = true">预览</el-button>
        <el-button :loading="exporting" @click="doCopyImage">复制图片</el-button>
        <el-button :loading="exporting" @click="doDownloadPng">下载 PNG</el-button>
        <el-button :loading="exporting" @click="doDownloadPdf">下载 PDF</el-button>
        <el-button type="primary" :loading="saving" @click="onSave(false)">保存</el-button>
      </div>
    </div>

    <el-row :gutter="16">
      <el-col :span="16">
        <el-card shadow="never" class="block">
          <template #header>客户与单据</template>
          <el-form label-width="88px" class="head-form">
            <el-row :gutter="12">
              <el-col :span="12"><el-form-item label="标题"><el-input v-model="form.title" /></el-form-item></el-col>
              <el-col :span="12">
                <el-form-item label="状态">
                  <el-select v-model="form.status" style="width: 100%">
                    <el-option label="草稿" :value="1" />
                    <el-option label="已发送" :value="2" />
                    <el-option label="已成交" :value="3" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="客户">
                  <div class="inline">
                    <el-input v-model="form.customerName" placeholder="手填或从客户中心选取" />
                    <el-button @click="custDrawer = true; searchCust()">选取</el-button>
                  </div>
                </el-form-item>
              </el-col>
              <el-col :span="12"><el-form-item label="联系人"><el-input v-model="form.contactName" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="电话"><el-input v-model="form.contactPhone" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="有效期"><el-date-picker v-model="form.validUntil" type="date" value-format="YYYY-MM-DD" style="width:100%" /></el-form-item></el-col>
              <el-col :span="24"><el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" /></el-form-item></el-col>
            </el-row>
          </el-form>
        </el-card>

        <el-card shadow="never" class="block">
          <template #header>
            <div class="card-head">
              <span>报价明细{{ skeletonMode ? '（业务模板）' : '' }}</span>
              <div class="card-actions">
                <el-button @click="itemsZoomed = true">放大编辑</el-button>
                <el-button v-if="skeletonMode" type="warning" plain @click="applySkeleton()">套用业务模板</el-button>
                <el-button v-if="!skeletonMode" type="primary" @click="skuDrawer = true">从商品中心添加</el-button>
                <el-button v-if="!skeletonMode" @click="addManualRow">手填一行</el-button>
              </div>
            </div>
          </template>

          <QuoteItemsEditor
            :items="form.items"
            :skeleton="skeletonMode"
            @preview-image="openImagePreview"
          />
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card shadow="never" class="block">
          <template #header>模板与合计</template>
          <el-form label-width="88px">
            <el-form-item label="版式">
              <el-select v-model="form.templateId" style="width:100%" placeholder="选择版式模板">
                <el-option v-for="t in layoutTemplates" :key="t.id" :label="t.name" :value="t.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="预览项">
              <el-checkbox v-model="previewShowRetailPrice">显示零售价</el-checkbox>
            </el-form-item>
            <el-form-item label="业务模板">
              <el-select
                v-model="selectedBizId"
                clearable
                style="width:100%"
                placeholder="可选：套用产品/配件骨架"
              >
                <el-option v-for="t in bizTemplates" :key="t.id" :label="t.name" :value="t.id" />
              </el-select>
            </el-form-item>
            <el-form-item v-if="selectedBizId">
              <el-button type="warning" plain style="width:100%" @click="applySkeleton()">套用业务明细</el-button>
            </el-form-item>
            <el-form-item label="折扣"><el-input-number v-model="form.discountAmt" :min="0" :precision="2" style="width:100%" /></el-form-item>
            <el-form-item label="运费"><el-input-number v-model="form.shippingAmt" :min="0" :precision="2" style="width:100%" /></el-form-item>
            <el-form-item label="税费"><el-input-number v-model="form.taxAmt" :min="0" :precision="2" style="width:100%" /></el-form-item>
          </el-form>
          <div class="sum">
            <div>零售价合计：¥{{ retailTotal.toFixed(2) }}</div>
            <div>成本合计：¥{{ costTotal.toFixed(2) }} <span class="sum-tip">有拿货价时按拿货价</span></div>
            <div>报价小计：¥{{ subtotal.toFixed(2) }}</div>
            <div class="profit">预估利润：¥{{ profit.toFixed(2) }}</div>
            <div class="grand">合计：¥{{ total.toFixed(2) }}</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-drawer v-model="skuDrawer" title="从商品中心选 SKU" size="560px" append-to-body>
      <p class="drawer-hint">可连续点选多个规格；同一商品会沿用首次产品名，并追加为新规格行。</p>
      <div class="drawer-search">
        <el-input v-model="skuKeyword" placeholder="关键词" @keyup.enter="searchSku" />
        <el-button type="primary" :loading="searching" @click="searchSku">搜索</el-button>
        <el-button @click="skuDrawer = false">完成</el-button>
      </div>
      <el-table :data="skuHits" size="small" @row-click="pickSku">
        <el-table-column label="图" width="56">
          <template #default="{ row }">
            <img v-if="row.pic || row.productPic" :src="row.pic || row.productPic" class="sku-thumb" alt="" />
          </template>
        </el-table-column>
        <el-table-column prop="productName" label="商品" min-width="140" />
        <el-table-column prop="specLabel" label="规格" min-width="120" />
        <el-table-column label="售价" width="80">
          <template #default="{ row }">¥{{ Number(row.price||0).toFixed(2) }}</template>
        </el-table-column>
      </el-table>
    </el-drawer>

    <el-drawer v-model="custDrawer" title="选择客户" size="420px" append-to-body>
      <div class="drawer-search">
        <el-input v-model="custKeyword" placeholder="姓名/电话" @keyup.enter="searchCust" />
        <el-button type="primary" :loading="searching" @click="searchCust">搜索</el-button>
      </div>
      <el-table :data="custHits" size="small" @row-click="pickCustomer">
        <el-table-column prop="displayName" label="客户" />
        <el-table-column prop="primaryPhone" label="电话" width="120" />
      </el-table>
    </el-drawer>

    <el-dialog
      v-model="itemsZoomed"
      title="报价明细 · 放大编辑"
      fullscreen
      append-to-body
      destroy-on-close
      class="items-zoom-dialog"
    >
      <div class="zoom-toolbar">
        <div class="zoom-actions">
          <el-button v-if="skeletonMode" type="warning" plain @click="applySkeleton()">套用骨架</el-button>
          <el-button v-if="!skeletonMode" type="primary" @click="skuDrawer = true">从商品中心添加</el-button>
          <el-button v-if="!skeletonMode" @click="addManualRow">手填一行</el-button>
        </div>
        <div class="zoom-sum">
          零售 ¥{{ retailTotal.toFixed(2) }}　成本 ¥{{ costTotal.toFixed(2) }}　合计 ¥{{ total.toFixed(2) }}
        </div>
      </div>
      <div class="zoom-body">
        <QuoteItemsEditor
          :items="form.items"
          :skeleton="skeletonMode"
          large
          @preview-image="openImagePreview"
        />
      </div>
      <template #footer>
        <el-button type="primary" @click="itemsZoomed = false">完成编辑</el-button>
      </template>
    </el-dialog>

    <!-- 离屏导出：不依赖预览弹窗，避免 Dialog transform 裁切与剪贴板手势丢失 -->
    <div class="export-host" aria-hidden="true">
      <div ref="exportRef" class="export-sheet-host">
        <QuoteSheet :quote="previewQuote" :template="sheetTemplate" />
      </div>
    </div>

    <el-dialog
      v-model="showPreview"
      title="报价预览"
      width="860px"
      top="4vh"
      append-to-body
      class="quote-preview-dialog"
    >
      <div class="preview-toolbar">
        <el-checkbox v-model="previewShowRetailPrice">显示零售价</el-checkbox>
        <span class="preview-tip">关闭后预览 / 复制图片 / PDF 均不显示零售价</span>
      </div>
      <div class="preview-wrap">
        <div ref="previewRef" class="preview-sheet-host">
          <QuoteSheet :quote="previewQuote" :template="sheetTemplate" />
        </div>
      </div>
      <template #footer>
        <el-button :loading="exporting" @click="doCopyImage">复制图片</el-button>
        <el-button :loading="exporting" @click="doDownloadPng">下载 PNG</el-button>
        <el-button type="primary" :loading="exporting" @click="doDownloadPdf">下载 PDF</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showImagePreview"
      title="图片预览"
      width="auto"
      append-to-body
      align-center
      destroy-on-close
      class="image-preview-dialog"
      @closed="imagePreviewUrl = ''"
    >
      <div class="image-preview-body">
        <img v-if="imagePreviewUrl" :src="imagePreviewUrl" alt="预览" />
      </div>
    </el-dialog>
  </div>
</template>

<style scoped>
.topbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; gap: 12px; flex-wrap: wrap; }
.topbar .left, .topbar .right { display: flex; gap: 8px; align-items: center; }
.autosave-hint { color: #909399; font-size: 12px; font-weight: 400; margin-left: 4px; }
.autosave-hint.saving, .autosave-hint.pending { color: #e6a23c; }
.autosave-hint.saved { color: #67c23a; }
.autosave-hint.error { color: #f56c6c; }
.block { margin-bottom: 12px; }
.card-head { display: flex; justify-content: space-between; align-items: center; gap: 8px; flex-wrap: wrap; }
.card-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.inline { display: flex; gap: 8px; width: 100%; }
.sum { margin-top: 8px; text-align: right; line-height: 1.8; }
.sum-tip { font-size: 12px; color: #909399; font-weight: normal; margin-left: 4px; }
.profit { color: #67c23a; }
.grand { font-size: 18px; font-weight: 700; }
.drawer-search { display: flex; gap: 8px; margin-bottom: 12px; }
.drawer-hint { margin: 0 0 10px; color: #8f959e; font-size: 12px; line-height: 1.5; }
.preview-wrap { overflow: auto; max-height: 70vh; background: #eef0f3; padding: 16px; display: flex; justify-content: center; }
.preview-sheet-host { flex: 0 0 auto; max-width: 100%; }
.preview-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}
.preview-tip { color: #909399; font-size: 12px; }
.export-host {
  position: fixed;
  left: -12000px;
  top: 0;
  width: 794px;
  pointer-events: none;
  opacity: 1;
  z-index: -1;
}
.export-sheet-host { width: 794px; background: #fff; }
.sku-thumb { width: 36px; height: 36px; object-fit: cover; border-radius: 4px; display: block; }
.zoom-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}
.zoom-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.zoom-sum { color: #606266; font-size: 14px; }
.zoom-body { overflow: auto; max-height: calc(100vh - 160px); }
.image-preview-body {
  display: flex;
  align-items: center;
  justify-content: center;
  max-width: min(90vw, 960px);
  max-height: 80vh;
  overflow: auto;
}
.image-preview-body img {
  max-width: 100%;
  max-height: 80vh;
  width: auto;
  height: auto;
  object-fit: contain;
  display: block;
}
</style>
