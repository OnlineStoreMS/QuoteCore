<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import QuoteSheet from '../components/QuoteSheet.vue'
import QuoteItemsEditor from '../components/QuoteItemsEditor.vue'
import {
  fetchSecondEdit,
  saveSecondEdit,
  uploadSecondEditImage,
  uploadSecondEditImageFromUrl,
  type CustomerShareTemplate,
} from '../api/publicShare'
import { isBizQuoteItems, type QuoteItem, type QuoteTemplate } from '../api/quote'
import { copyElementAsImage, downloadElementAsPdf, downloadElementAsPng } from '../utils/exportQuote'
import { normalizeSpecLabel, pricedFlagsFromSaved, sumPriced } from '../utils/quotePrice'

const route = useRoute()
const token = computed(() => String(route.params.token || '').trim())

const loading = ref(true)
const saving = ref(false)
const exporting = ref(false)
const error = ref('')
const exportRef = ref<HTMLElement | null>(null)
const previewRef = ref<HTMLElement | null>(null)
const showPreview = ref(false)
const previewShowRetailPrice = ref(true)
const imagePreviewUrl = ref('')
const showImagePreview = ref(false)
const originNo = ref('')
const applicant = ref('')
const applicantPhone = ref('')
const quoteNo = ref('')

const form = reactive({
  title: '报价单',
  customerName: '',
  contactName: '',
  contactPhone: '',
  currency: 'CNY',
  validUntil: '',
  remark: '',
  discountAmt: 0,
  shippingAmt: 0,
  taxAmt: 0,
  items: [] as QuoteItem[],
})

const layout = ref<CustomerShareTemplate | null>(null)

const sheetTemplate = computed<QuoteTemplate | null>(() => {
  const t = layout.value
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
    showRetailPrice: previewShowRetailPrice.value,
    showSpecImage: t.showSpecImage !== false,
    showUpgrade: t.showUpgrade !== false,
    showParams: t.showParams !== false,
    showTotals: t.showTotals !== false,
    stylePreset: t.stylePreset || 'compare',
    lines: [],
  }
})

const skeletonMode = computed(() => isBizQuoteItems(form.items))

const priceFlags = computed(() => pricedFlagsFromSaved(form.items, true))
const subtotal = computed(() =>
  sumPriced(form.items, priceFlags.value, (it) => Number(it.qty || 0) * Number(it.quotePrice || 0)),
)
const total = computed(() =>
  Math.round((subtotal.value - Number(form.discountAmt || 0) + Number(form.shippingAmt || 0) + Number(form.taxAmt || 0)) * 100) / 100,
)

const previewQuote = computed(() => ({
  quoteNo: quoteNo.value,
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
  subtotalAmt: subtotal.value,
  totalAmt: total.value,
  items: form.items,
}))

function mapItems(items: SecondEditItemLike[]): QuoteItem[] {
  return items.map((it, i) => ({
    id: it.id,
    sort: it.sort || (i + 1) * 10,
    source: it.source || 'manual',
    productId: it.productId ?? null,
    skuId: it.skuId ?? null,
    templateLineId: it.templateLineId ?? null,
    category: it.category || '',
    partName: it.partName || '',
    name: it.name || '',
    specLabel: normalizeSpecLabel(it.specLabel),
    imageUrl: it.imageUrl || '',
    qty: Number(it.qty || 1),
    unit: it.unit || '件',
    retailPrice: Number(it.retailPrice || 0),
    costPrice: 0,
    quotePrice: Number(it.quotePrice || 0),
    upgradeNote: it.upgradeNote || '',
    paramsText: it.paramsText || '',
    remark: it.remark || '',
    customerSelected: !!it.customerSelected,
  }))
}

type SecondEditItemLike = {
  id?: number
  sort?: number
  source?: string
  productId?: number | null
  skuId?: number | null
  templateLineId?: number | null
  category?: string
  partName?: string
  name?: string
  specLabel?: string
  imageUrl?: string
  qty?: number
  unit?: string
  retailPrice?: number
  quotePrice?: number
  upgradeNote?: string
  paramsText?: string
  remark?: string
  customerSelected?: boolean
}

function applyView(q: Awaited<ReturnType<typeof fetchSecondEdit>>) {
  quoteNo.value = q.quoteNo || ''
  originNo.value = q.originQuoteNo || ''
  applicant.value = q.secondEditApplicant || ''
  applicantPhone.value = q.secondEditApplicantPhone || ''
  form.title = q.title || '报价单'
  form.customerName = q.customerName || ''
  form.contactName = q.contactName || ''
  form.contactPhone = q.contactPhone || ''
  form.currency = q.currency || 'CNY'
  form.validUntil = q.validUntil ? String(q.validUntil).slice(0, 10) : ''
  form.remark = q.remark || ''
  form.discountAmt = q.discountAmt || 0
  form.shippingAmt = q.shippingAmt || 0
  form.taxAmt = q.taxAmt || 0
  form.items = mapItems(q.items || [])
  layout.value = q.template
  previewShowRetailPrice.value = q.template?.showRetailPrice !== false
  document.title = `二次编辑 · ${q.title || '报价单'}`
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
    applyView(await fetchSecondEdit(token.value))
  } catch (e) {
    error.value = (e as Error).message || '加载失败'
  } finally {
    loading.value = false
  }
}

function canPersist(): boolean {
  return form.items.length > 0 && form.items.every(
    (it) => !!(it.name || '').trim() || !!(it.category || '').trim() || !!(it.partName || '').trim(),
  )
}

async function onSave() {
  if (!canPersist()) {
    ElMessage.warning('每行明细需填写名称或配件')
    return
  }
  saving.value = true
  try {
    const saved = await saveSecondEdit(token.value, {
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
      items: form.items.map((it, i) => ({ ...it, sort: (i + 1) * 10 })),
    })
    applyView(saved)
    ElMessage.success('已保存')
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function readyExportEl() {
  await nextTick()
  const el = exportRef.value
  if (!el) throw new Error('预览区域未就绪')
  return el
}

async function doCopyImage() {
  exporting.value = true
  try {
    await copyElementAsImage(await readyExportEl())
    ElMessage.success('图片已复制')
  } catch (e) {
    ElMessage.error((e as Error).message || '复制失败')
  } finally {
    exporting.value = false
  }
}

async function doDownloadPng() {
  exporting.value = true
  try {
    await downloadElementAsPng(await readyExportEl(), quoteNo.value || `quote-${Date.now()}`)
    ElMessage.success('PNG 已下载')
  } catch (e) {
    ElMessage.error((e as Error).message || '导出失败')
  } finally {
    exporting.value = false
  }
}

async function doDownloadPdf() {
  exporting.value = true
  try {
    await downloadElementAsPdf(await readyExportEl(), quoteNo.value || `quote-${Date.now()}`)
    ElMessage.success('PDF 已下载')
  } catch (e) {
    ElMessage.error((e as Error).message || '导出失败')
  } finally {
    exporting.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div v-if="loading" class="state">正在打开二次编辑报价单…</div>
    <div v-else-if="error" class="state error">{{ error }}</div>
    <template v-else>
      <div class="topbar">
        <div class="left">
          <strong>二次编辑分享</strong>
          <span class="meta">原单 {{ originNo || '—' }} · 本单 {{ quoteNo || '—' }}</span>
        </div>
        <div class="right">
          <el-button :loading="exporting" @click="showPreview = true">预览</el-button>
          <el-button :loading="exporting" @click="doCopyImage">复制图片</el-button>
          <el-button :loading="exporting" @click="doDownloadPng">下载 PNG</el-button>
          <el-button :loading="exporting" @click="doDownloadPdf">下载 PDF</el-button>
          <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
        </div>
      </div>
      <p class="banner">
        申请人 {{ applicant || '—' }} {{ applicantPhone }}。可改标题、客户、联系人、电话、报价与规格。之后用原分享链接加申请电话可再次打开本单。
      </p>
      <el-form label-width="88px" class="head-form">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="标题"><el-input v-model="form.title" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="客户"><el-input v-model="form.customerName" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="联系人"><el-input v-model="form.contactName" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="电话"><el-input v-model="form.contactPhone" /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" /></el-form-item></el-col>
        </el-row>
      </el-form>
      <QuoteItemsEditor
        :items="form.items"
        :skeleton="skeletonMode"
        public-mode
        :upload-fn="(file) => uploadSecondEditImage(token, file)"
        :upload-from-url-fn="(url) => uploadSecondEditImageFromUrl(token, url)"
        @preview-image="(u) => { imagePreviewUrl = u; showImagePreview = true }"
      />
      <div class="sum">合计 ¥{{ total.toFixed(2) }}</div>
      <div class="export-host" aria-hidden="true">
        <div ref="exportRef" class="export-sheet-host">
          <QuoteSheet :quote="previewQuote" :template="sheetTemplate" :skeleton="skeletonMode" :price-flags="priceFlags" />
        </div>
      </div>
    </template>
    <el-dialog
      v-model="showPreview"
      title="报价预览"
      width="860px"
      top="4vh"
      append-to-body
    >
      <div class="preview-toolbar">
        <el-checkbox v-model="previewShowRetailPrice">显示零售价</el-checkbox>
        <span class="preview-tip">关闭后预览 / 复制图片 / PDF 均不显示零售价</span>
      </div>
      <div class="preview-wrap">
        <div ref="previewRef" class="preview-sheet-host">
          <QuoteSheet :quote="previewQuote" :template="sheetTemplate" :skeleton="skeletonMode" :price-flags="priceFlags" />
        </div>
      </div>
      <template #footer>
        <el-button :loading="exporting" @click="doCopyImage">复制图片</el-button>
        <el-button :loading="exporting" @click="doDownloadPng">下载 PNG</el-button>
        <el-button type="primary" :loading="exporting" @click="doDownloadPdf">下载 PDF</el-button>
      </template>
    </el-dialog>
    <el-dialog v-model="showImagePreview" title="图片预览" width="auto" append-to-body @closed="imagePreviewUrl = ''">
      <img v-if="imagePreviewUrl" :src="imagePreviewUrl" alt="预览" class="img-preview" />
    </el-dialog>
  </div>
</template>

<style scoped>
.page { min-height: 100%; padding: 12px 16px 40px; box-sizing: border-box; background: #f5f6f8; }
.topbar { display: flex; justify-content: space-between; gap: 12px; flex-wrap: wrap; align-items: center; margin-bottom: 10px; }
.left, .right { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.meta { color: #909399; font-size: 13px; }
.banner { margin: 0 0 12px; color: #646a73; font-size: 13px; }
.head-form { background: #fff; padding: 12px 12px 0; border-radius: 8px; margin-bottom: 12px; }
.sum { text-align: right; font-size: 18px; font-weight: 700; margin-top: 12px; }
.state { min-height: 40vh; display: flex; align-items: center; justify-content: center; color: #646a73; }
.state.error { color: #b42318; }
.export-host {
  position: fixed;
  left: -2400px;
  top: 0;
  width: 794px;
  pointer-events: none;
}
.img-preview { max-width: 80vw; max-height: 70vh; display: block; }
.preview-wrap { overflow: auto; max-height: 70vh; background: #eef0f3; padding: 16px; display: flex; justify-content: center; }
.preview-sheet-host { flex: 0 0 auto; max-width: 100%; }
.preview-toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; flex-wrap: wrap; }
.preview-tip { color: #909399; font-size: 12px; }
</style>
