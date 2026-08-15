<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ASSEMBLE_SKELETON_PRESET,
  deleteTemplate,
  listTemplates,
  saveTemplate,
  type QuoteTemplate,
  type QuoteTemplateLine,
} from '../api/quote'
import { uploadImage, uploadImageFromUrl, isQuoteStoredUrl } from '../api/upload'

const loading = ref(false)
const list = ref<QuoteTemplate[]>([])
const dialog = ref(false)
const editingId = ref<number | undefined>()
const form = reactive({
  name: '',
  kind: 'layout' as 'layout' | 'skeleton',
  isDefault: false,
  logoUrl: '',
  shopName: '',
  shopPhone: '',
  shopAddress: '',
  headerSubtitle: '',
  footerText: '',
  showLogo: true,
  showRetailPrice: true,
  showSpecImage: true,
  showUpgrade: true,
  showParams: true,
  showTotals: true,
  stylePreset: 'compare',
  lines: [] as QuoteTemplateLine[],
})

const isSkeleton = computed(() => form.kind === 'skeleton')

async function load() {
  loading.value = true
  try {
    list.value = await listTemplates()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

function resetForm(partial?: Partial<typeof form>) {
  Object.assign(form, {
    name: '新模板',
    kind: 'layout',
    isDefault: false,
    logoUrl: '',
    shopName: '',
    shopPhone: '',
    shopAddress: '',
    headerSubtitle: '',
    footerText: '本报价单有效期内价格有效。',
    showLogo: true,
    showRetailPrice: true,
    showSpecImage: true,
    showUpgrade: true,
    showParams: true,
    showTotals: true,
    stylePreset: 'compare',
    lines: [] as QuoteTemplateLine[],
    ...partial,
  })
}

function openCreate() {
  editingId.value = undefined
  resetForm()
  logoLinkDraft.value = ''
  dialog.value = true
}

function openCreateSkeleton() {
  editingId.value = undefined
  resetForm({
    name: '组装车报价骨架',
    kind: 'skeleton',
    headerSubtitle: '组车配件清单',
    showUpgrade: false,
    showParams: false,
    lines: ASSEMBLE_SKELETON_PRESET.map((x) => ({ ...x })),
  })
  logoLinkDraft.value = ''
  dialog.value = true
}

function openEdit(row: QuoteTemplate) {
  editingId.value = row.id
  resetForm({
    name: row.name,
    kind: row.kind === 'skeleton' || (row.lines && row.lines.length) ? 'skeleton' : 'layout',
    isDefault: row.isDefault,
    logoUrl: row.logoUrl || '',
    shopName: row.shopName || '',
    shopPhone: row.shopPhone || '',
    shopAddress: row.shopAddress || '',
    headerSubtitle: row.headerSubtitle || '',
    footerText: row.footerText || '',
    showLogo: row.showLogo,
    showRetailPrice: row.showRetailPrice,
    showSpecImage: row.showSpecImage,
    showUpgrade: row.showUpgrade,
    showParams: row.showParams,
    showTotals: row.showTotals,
    stylePreset: row.stylePreset || 'compare',
    lines: (row.lines || []).map((ln) => ({
      sort: ln.sort,
      category: ln.category || '',
      partName: ln.partName || '',
      hint: ln.hint || '',
    })),
  })
  logoLinkDraft.value = row.logoUrl || ''
  dialog.value = true
}

const logoLinkDraft = ref('')
const logoUploading = ref(false)

async function onUpload(opt: { file: File }) {
  try {
    form.logoUrl = await uploadImage(opt.file, 'logo')
    logoLinkDraft.value = form.logoUrl
    ElMessage.success('Logo 已上传')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function applyLogoLink() {
  const raw = logoLinkDraft.value.trim()
  if (!raw) {
    form.logoUrl = ''
    return
  }
  if (isQuoteStoredUrl(raw)) {
    form.logoUrl = raw
    ElMessage.success('已使用本站图片')
    return
  }
  if (!/^https?:\/\//i.test(raw)) {
    ElMessage.error('请粘贴以 http(s):// 开头的图片链接')
    return
  }
  logoUploading.value = true
  try {
    form.logoUrl = await uploadImageFromUrl(raw, 'logo')
    logoLinkDraft.value = form.logoUrl
    ElMessage.success('已上传到报价中心')
  } catch (e) {
    ElMessage.error((e as Error).message || '上传失败')
  } finally {
    logoUploading.value = false
  }
}

function addLine() {
  form.lines.push({
    sort: (form.lines.length + 1) * 10,
    category: form.lines.length ? form.lines[form.lines.length - 1].category : '',
    partName: '',
    hint: '',
  })
}

function removeLine(idx: number) {
  form.lines.splice(idx, 1)
  form.lines.forEach((ln, i) => {
    ln.sort = (i + 1) * 10
  })
}

function loadPreset() {
  form.kind = 'skeleton'
  form.lines = ASSEMBLE_SKELETON_PRESET.map((x) => ({ ...x }))
  ElMessage.success('已载入组装车预设（精灵表结构）')
}

async function onSave() {
  if (form.kind === 'skeleton') {
    const bad = form.lines.findIndex((ln) => !(ln.partName || '').trim())
    if (bad >= 0) {
      ElMessage.warning(`第 ${bad + 1} 行配件名称必填`)
      return
    }
    if (!form.lines.length) {
      ElMessage.warning('骨架模板至少需要一行配件')
      return
    }
  }
  try {
    await saveTemplate(
      {
        ...form,
        lines: form.kind === 'skeleton' ? form.lines : [],
      },
      editingId.value,
    )
    ElMessage.success('已保存')
    dialog.value = false
    load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function onDelete(row: QuoteTemplate) {
  await ElMessageBox.confirm(`删除模板「${row.name}」？`, '提示', { type: 'warning' })
  try {
    await deleteTemplate(row.id)
    ElMessage.success('已删除')
    load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <el-button type="primary" @click="openCreate">新建版式模板</el-button>
      <el-button @click="openCreateSkeleton">新建组装车骨架</el-button>
      <span class="hint">版式管 Logo/店名；骨架固定「产品+配件」行，报价时只填名称/规格/价格/图片。</span>
    </div>

    <el-table v-loading="loading" :data="list" border>
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column label="类型" width="110">
        <template #default="{ row }">
          <el-tag v-if="row.kind === 'skeleton' || (row.lines && row.lines.length)" type="warning" size="small">骨架</el-tag>
          <el-tag v-else size="small">版式</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="骨架行" width="80">
        <template #default="{ row }">{{ row.lines?.length || 0 }}</template>
      </el-table-column>
      <el-table-column label="默认" width="80">
        <template #default="{ row }">
          <el-tag v-if="row.isDefault" type="success" size="small">默认</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="shopName" label="店名" width="140" />
      <el-table-column prop="shopPhone" label="电话" width="120" />
      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialog" :title="editingId ? '编辑模板' : '新建模板'" width="820px" destroy-on-close>
      <el-form label-width="100px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="form.kind">
            <el-radio-button value="layout">版式</el-radio-button>
            <el-radio-button value="skeleton">组车骨架</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="默认模板"><el-switch v-model="form.isDefault" /></el-form-item>
        <el-form-item label="样式">
          <el-radio-group v-model="form.stylePreset">
            <el-radio-button value="compare">对比价</el-radio-button>
            <el-radio-button value="simple">简洁</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="Logo">
          <div class="logo-row">
            <el-image v-if="form.logoUrl" :src="form.logoUrl" style="width: 64px; height: 64px" fit="contain" />
            <el-upload :show-file-list="false" :http-request="onUpload as any" accept="image/*">
              <el-button>上传</el-button>
            </el-upload>
          </div>
          <div class="logo-link">
            <el-input v-model="logoLinkDraft" clearable placeholder="粘贴 Logo 图片链接，将上传到报价中心" />
            <el-button type="primary" :loading="logoUploading" @click="applyLogoLink">上传到报价中心</el-button>
          </div>
        </el-form-item>
        <el-form-item label="店名"><el-input v-model="form.shopName" /></el-form-item>
        <el-form-item label="电话"><el-input v-model="form.shopPhone" /></el-form-item>
        <el-form-item label="地址"><el-input v-model="form.shopAddress" /></el-form-item>
        <el-form-item label="副标题"><el-input v-model="form.headerSubtitle" /></el-form-item>
        <el-form-item label="页脚条款"><el-input v-model="form.footerText" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="显示项">
          <el-checkbox v-model="form.showLogo">Logo</el-checkbox>
          <el-checkbox v-model="form.showRetailPrice">零售价</el-checkbox>
          <el-checkbox v-model="form.showSpecImage">规格图</el-checkbox>
          <el-checkbox v-model="form.showUpgrade">升级款</el-checkbox>
          <el-checkbox v-model="form.showParams">参数</el-checkbox>
          <el-checkbox v-model="form.showTotals">合计</el-checkbox>
        </el-form-item>

        <template v-if="isSkeleton">
          <el-form-item label="骨架明细">
            <div class="lines-toolbar">
              <el-button size="small" @click="addLine">加一行</el-button>
              <el-button size="small" type="primary" plain @click="loadPreset">载入组装车预设</el-button>
              <span class="hint">产品=组名（可空）；配件=固定列。报价时只填名称/规格/价/图/备注。</span>
            </div>
            <el-table :data="form.lines" border size="small" class="lines-table">
              <el-table-column label="产品（组）" min-width="120">
                <template #default="{ row }"><el-input v-model="row.category" placeholder="如 车架组" /></template>
              </el-table-column>
              <el-table-column label="配件" min-width="140">
                <template #default="{ row }"><el-input v-model="row.partName" placeholder="如 车架" /></template>
              </el-table-column>
              <el-table-column label="填写提示" min-width="140">
                <template #default="{ row }"><el-input v-model="row.hint" placeholder="可选" /></template>
              </el-table-column>
              <el-table-column label="操作" width="70">
                <template #default="{ $index }">
                  <el-button link type="danger" @click="removeLine($index)">删</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="onSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; flex-wrap: wrap; }
.hint { color: #909399; font-size: 13px; }
.logo-row { display: flex; align-items: center; gap: 12px; }
.logo-link { display: flex; gap: 8px; margin-top: 8px; align-items: center; }
.logo-link .el-input { flex: 1; }
.lines-toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-wrap: wrap; width: 100%; }
.lines-table { width: 100%; }
</style>
