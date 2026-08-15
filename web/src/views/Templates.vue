<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { deleteTemplate, listTemplates, saveTemplate, type QuoteTemplate } from '../api/quote'
import { uploadImage, uploadImageFromUrl, isQuoteStoredUrl } from '../api/upload'

const loading = ref(false)
const list = ref<QuoteTemplate[]>([])
const dialog = ref(false)
const editingId = ref<number | undefined>()
const form = reactive({
  name: '',
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
})

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

function openCreate() {
  editingId.value = undefined
  Object.assign(form, {
    name: '新模板',
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
  })
  logoLinkDraft.value = ''
  dialog.value = true
}

function openEdit(row: QuoteTemplate) {
  editingId.value = row.id
  Object.assign(form, {
    name: row.name,
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

async function onSave() {
  try {
    await saveTemplate({ ...form }, editingId.value)
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
      <el-button type="primary" @click="openCreate">新建模板</el-button>
      <span class="hint">Logo / 店名等信息仅用于报价单版式，不绑定门店管理。</span>
    </div>

    <el-table v-loading="loading" :data="list" border>
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column label="默认" width="80">
        <template #default="{ row }">
          <el-tag v-if="row.isDefault" type="success" size="small">默认</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="shopName" label="店名" width="140" />
      <el-table-column prop="shopPhone" label="电话" width="120" />
      <el-table-column label="样式" width="100">
        <template #default="{ row }">{{ row.stylePreset === 'simple' ? '简洁' : '对比价' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialog" :title="editingId ? '编辑模板' : '新建模板'" width="640px" destroy-on-close>
      <el-form label-width="100px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
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
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="onSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
.hint { color: #909399; font-size: 13px; }
.logo-row { display: flex; align-items: center; gap: 12px; }
.logo-link { display: flex; gap: 8px; margin-top: 8px; align-items: center; }
.logo-link .el-input { flex: 1; }
</style>
