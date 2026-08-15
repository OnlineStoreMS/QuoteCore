<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import QuoteSheet from '../components/QuoteSheet.vue'
import {
  getQuote,
  listTemplates,
  saveQuote,
  searchCustomers,
  searchProductSkus,
  type CustomerHit,
  type QuoteItem,
  type QuoteTemplate,
  type SkuHit,
} from '../api/quote'
import { uploadImage } from '../api/upload'
import { copyElementAsImage, downloadElementAsPdf } from '../utils/exportQuote'

const route = useRoute()
const router = useRouter()
const quoteId = computed(() => {
  const id = Number(route.params.id)
  return Number.isFinite(id) && id > 0 ? id : 0
})

const loading = ref(false)
const saving = ref(false)
const templates = ref<QuoteTemplate[]>([])
const activeTemplate = ref<QuoteTemplate | null>(null)
const previewRef = ref<HTMLElement | null>(null)
const showPreview = ref(false)
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
const total = computed(() =>
  Math.round((subtotal.value - Number(form.discountAmt || 0) + Number(form.shippingAmt || 0) + Number(form.taxAmt || 0)) * 100) / 100,
)

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

function emptyItem(): QuoteItem {
  return {
    sort: (form.items.length + 1) * 10,
    source: 'manual',
    name: '',
    specLabel: '',
    imageUrl: '',
    qty: 1,
    unit: '件',
    retailPrice: 0,
    quotePrice: 0,
    upgradeNote: '',
    paramsText: '',
    remark: '',
  }
}

function addManualRow() {
  form.items.push(emptyItem())
}

function removeRow(idx: number) {
  form.items.splice(idx, 1)
}

function moveRow(idx: number, dir: -1 | 1) {
  const j = idx + dir
  if (j < 0 || j >= form.items.length) return
  const tmp = form.items[idx]
  form.items[idx] = form.items[j]
  form.items[j] = tmp
}

async function loadTemplates() {
  templates.value = await listTemplates()
  const def = templates.value.find((t) => t.isDefault) || templates.value[0]
  if (def && !form.templateId) {
    form.templateId = def.id
    activeTemplate.value = def
  } else if (form.templateId) {
    activeTemplate.value = templates.value.find((t) => t.id === form.templateId) || def || null
  }
}

watch(
  () => form.templateId,
  (id) => {
    activeTemplate.value = templates.value.find((t) => t.id === id) || null
  },
)

async function loadQuote() {
  if (!quoteId.value) {
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
    form.templateId = q.templateId || null
    form.quoteNo = q.quoteNo || ''
    form.items = (q.items || []).map((it, i) => ({
      sort: it.sort || (i + 1) * 10,
      source: it.source || 'manual',
      productId: it.productId,
      skuId: it.skuId,
      name: it.name,
      specLabel: it.specLabel || '',
      imageUrl: it.imageUrl || '',
      qty: it.qty || 1,
      unit: it.unit || '件',
      retailPrice: it.retailPrice || 0,
      quotePrice: it.quotePrice || 0,
      upgradeNote: it.upgradeNote || '',
      paramsText: it.paramsText || '',
      remark: it.remark || '',
    }))
    if (q.templateSnap) {
      try {
        activeTemplate.value = JSON.parse(q.templateSnap)
      } catch {
        /* ignore */
      }
    }
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function onSave() {
  if (!form.items.length) {
    ElMessage.warning('请至少添加一行明细')
    return
  }
  saving.value = true
  try {
    const saved = await saveQuote(
      {
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
      },
      quoteId.value || undefined,
    )
    ElMessage.success('已保存')
    form.quoteNo = saved.quoteNo
    if (!quoteId.value) {
      router.replace(`/quotes/${saved.id}`)
    }
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

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
  form.items.push({
    sort: (form.items.length + 1) * 10,
    source: 'product',
    productId: s.productId,
    skuId: s.skuId,
    name: s.productName,
    specLabel: s.specLabel || s.skuCode || '',
    imageUrl: s.pic || s.productPic || '',
    qty: 1,
    unit: '件',
    retailPrice: s.price || 0,
    quotePrice: s.price || 0,
    upgradeNote: '',
    paramsText: s.brandName ? `品牌：${s.brandName}` : '',
    remark: '',
  })
  skuDrawer.value = false
  ElMessage.success('已加入明细')
}

async function uploadRowImage(idx: number, opt: { file: File }) {
  try {
    form.items[idx].imageUrl = await uploadImage(opt.file, 'items')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function doCopyImage() {
  showPreview.value = true
  await new Promise((r) => setTimeout(r, 80))
  const el = previewRef.value
  if (!el) return
  try {
    await copyElementAsImage(el)
    ElMessage.success('已复制到剪贴板（若不支持则已下载 PNG）')
  } catch (e) {
    ElMessage.error((e as Error).message || '复制失败')
  }
}

async function doDownloadPdf() {
  showPreview.value = true
  await new Promise((r) => setTimeout(r, 80))
  const el = previewRef.value
  if (!el) return
  try {
    await downloadElementAsPdf(el, form.quoteNo || `quote-${Date.now()}`)
    ElMessage.success('PDF 已下载')
  } catch (e) {
    ElMessage.error((e as Error).message || '导出失败')
  }
}

onMounted(async () => {
  try {
    await loadTemplates()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
  await loadQuote()
})
</script>

<template>
  <div v-loading="loading" class="edit">
    <div class="topbar">
      <div class="left">
        <el-button @click="router.push('/quotes')">返回</el-button>
        <strong>{{ quoteId ? `编辑 ${form.quoteNo}` : '新建报价' }}</strong>
      </div>
      <div class="right">
        <el-button @click="showPreview = true">预览</el-button>
        <el-button @click="doCopyImage">复制图片</el-button>
        <el-button @click="doDownloadPdf">下载 PDF</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
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
              <span>报价明细</span>
              <div>
                <el-button type="primary" @click="skuDrawer = true">从商品中心添加</el-button>
                <el-button @click="addManualRow">手填一行</el-button>
              </div>
            </div>
          </template>

          <el-table :data="form.items" border size="small">
            <el-table-column label="产品" min-width="160">
              <template #default="{ row }"><el-input v-model="row.name" /></template>
            </el-table-column>
            <el-table-column label="规格" width="130">
              <template #default="{ row }"><el-input v-model="row.specLabel" /></template>
            </el-table-column>
            <el-table-column label="图" width="90">
              <template #default="{ row, $index }">
                <div class="img-cell">
                  <el-image v-if="row.imageUrl" :src="row.imageUrl" style="width:36px;height:36px" fit="cover" />
                  <el-upload :show-file-list="false" :http-request="(o:any) => uploadRowImage($index, o)" accept="image/*">
                    <el-button link type="primary">传图</el-button>
                  </el-upload>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="数量" width="90">
              <template #default="{ row }"><el-input-number v-model="row.qty" :min="0.01" :step="1" controls-position="right" style="width:100%" /></template>
            </el-table-column>
            <el-table-column label="零售价" width="100">
              <template #default="{ row }"><el-input-number v-model="row.retailPrice" :min="0" :precision="2" controls-position="right" style="width:100%" /></template>
            </el-table-column>
            <el-table-column label="报价" width="100">
              <template #default="{ row }"><el-input-number v-model="row.quotePrice" :min="0" :precision="2" controls-position="right" style="width:100%" /></template>
            </el-table-column>
            <el-table-column label="小计" width="90" align="right">
              <template #default="{ row }">{{ (Number(row.qty||0)*Number(row.quotePrice||0)).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="升级款" width="120">
              <template #default="{ row }"><el-input v-model="row.upgradeNote" /></template>
            </el-table-column>
            <el-table-column label="参数" width="140">
              <template #default="{ row }"><el-input v-model="row.paramsText" /></template>
            </el-table-column>
            <el-table-column label="备注" width="120">
              <template #default="{ row }"><el-input v-model="row.remark" /></template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ $index }">
                <el-button link @click="moveRow($index, -1)">上</el-button>
                <el-button link @click="moveRow($index, 1)">下</el-button>
                <el-button link type="danger" @click="removeRow($index)">删</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card shadow="never" class="block">
          <template #header>模板与合计</template>
          <el-form label-width="88px">
            <el-form-item label="模板">
              <el-select v-model="form.templateId" style="width:100%" placeholder="选择模板">
                <el-option v-for="t in templates" :key="t.id" :label="t.name" :value="t.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="折扣"><el-input-number v-model="form.discountAmt" :min="0" :precision="2" style="width:100%" /></el-form-item>
            <el-form-item label="运费"><el-input-number v-model="form.shippingAmt" :min="0" :precision="2" style="width:100%" /></el-form-item>
            <el-form-item label="税费"><el-input-number v-model="form.taxAmt" :min="0" :precision="2" style="width:100%" /></el-form-item>
          </el-form>
          <div class="sum">
            <div>小计：¥{{ subtotal.toFixed(2) }}</div>
            <div class="grand">合计：¥{{ total.toFixed(2) }}</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-drawer v-model="skuDrawer" title="从商品中心选 SKU" size="520px">
      <div class="drawer-search">
        <el-input v-model="skuKeyword" placeholder="关键词" @keyup.enter="searchSku" />
        <el-button type="primary" :loading="searching" @click="searchSku">搜索</el-button>
      </div>
      <el-table :data="skuHits" size="small" @row-click="pickSku">
        <el-table-column label="图" width="56">
          <template #default="{ row }"><el-image :src="row.pic || row.productPic" style="width:36px;height:36px" fit="cover" /></template>
        </el-table-column>
        <el-table-column prop="productName" label="商品" min-width="140" />
        <el-table-column prop="specLabel" label="规格" width="100" />
        <el-table-column label="售价" width="80">
          <template #default="{ row }">¥{{ Number(row.price||0).toFixed(2) }}</template>
        </el-table-column>
      </el-table>
    </el-drawer>

    <el-drawer v-model="custDrawer" title="选择客户" size="420px">
      <div class="drawer-search">
        <el-input v-model="custKeyword" placeholder="姓名/电话" @keyup.enter="searchCust" />
        <el-button type="primary" :loading="searching" @click="searchCust">搜索</el-button>
      </div>
      <el-table :data="custHits" size="small" @row-click="pickCustomer">
        <el-table-column prop="displayName" label="客户" />
        <el-table-column prop="primaryPhone" label="电话" width="120" />
      </el-table>
    </el-drawer>

    <el-dialog v-model="showPreview" title="报价预览" width="860px" top="4vh" destroy-on-close>
      <div class="preview-wrap">
        <div ref="previewRef">
          <QuoteSheet :quote="previewQuote" :template="activeTemplate" />
        </div>
      </div>
      <template #footer>
        <el-button @click="doCopyImage">复制图片</el-button>
        <el-button type="primary" @click="doDownloadPdf">下载 PDF</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.topbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; gap: 12px; flex-wrap: wrap; }
.topbar .left, .topbar .right { display: flex; gap: 8px; align-items: center; }
.block { margin-bottom: 12px; }
.card-head { display: flex; justify-content: space-between; align-items: center; }
.inline { display: flex; gap: 8px; width: 100%; }
.img-cell { display: flex; flex-direction: column; align-items: center; gap: 2px; }
.sum { margin-top: 8px; text-align: right; line-height: 1.8; }
.grand { font-size: 18px; font-weight: 700; }
.drawer-search { display: flex; gap: 8px; margin-bottom: 12px; }
.preview-wrap { overflow: auto; max-height: 70vh; background: #eef0f3; padding: 16px; display: flex; justify-content: center; }
</style>
